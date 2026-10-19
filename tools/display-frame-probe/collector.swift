import CoreImage
import CoreMedia
import Darwin
import Foundation
import ImageIO
@preconcurrency import ScreenCaptureKit
import UniformTypeIdentifiers

func metadataRect(_ value: Any?) -> CGRect? {
  if let rect = value as? CGRect { return rect }
  if let dictionary = value as? NSDictionary { return CGRect(dictionaryRepresentation: dictionary) }
  return nil
}

func captureDeadline(start: UInt64, seconds: Double, numerator: UInt32, denominator: UInt32) throws
  -> UInt64
{
  guard start > 0, seconds.isFinite, seconds >= 0.1, seconds <= 5, numerator > 0, denominator > 0
  else {
    throw ProbeError("Invalid recording interval or Mach timebase")
  }
  let span = seconds * 1_000_000_000 * Double(denominator) / Double(numerator)
  guard span.isFinite, span >= 1, span < Double(UInt64.max) else {
    throw ProbeError("Recording deadline overflow")
  }
  let (deadline, overflow) = start.addingReportingOverflow(UInt64(span.rounded(.down)))
  guard !overflow else { throw ProbeError("Recording deadline overflow") }
  return deadline
}

// Agreement between two supplied representations, not proof of scanout or
// receipt latency. SCK display time can be ahead of callback receipt; retain it.
func matchingHostPresentation(_ presentation: CMTime, displayed: UInt64) -> Bool {
  presentation.isValid && presentation.isNumeric && presentation.epoch == 0
    && CMTimeCompare(presentation, CMClockMakeHostTimeFromSystemUnits(displayed)) == 0
}

// Queue-confined state. Receipt checks are injected only for offline synthetic
// tests; the capture entry point always supplies independent CG target checks.
final class DisplayCollector: NSObject, SCStreamOutput, SCStreamDelegate, @unchecked Sendable {
  let queue = DispatchQueue(label: "elab.display-frame-probe.samples")
  let deadline: UInt64
  private let output: URL
  private let journal: FileHandle
  private let plan: CapturePlan
  private let verifyReceipt: @Sendable () throws -> Void
  private let context = CIContext(options: [.cacheIntermediates: false])
  private var timebase = mach_timebase_info_data_t()
  private var timer: DispatchSourceTimer?
  private var failure: String?
  private var closeFailure: String?
  private var frameCount = 0
  private var completeCount = 0
  private var afterDeadline = 0
  private var storedBytes = 0
  private var images: [String: String] = [:]
  private var markerIDs = Set<String>()
  private var latestMarker: UInt64 = 0
  private var previousDisplay: UInt64 = 0
  private var futureDisplayCount = 0
  private var outsidePresentationWindow = 0
  private var finalReport: [String: Any]?

  init(
    output: URL, plan: CapturePlan, start: UInt64, seconds: Double,
    verifyReceipt: @escaping @Sendable () throws -> Void
  ) throws {
    var ratio = mach_timebase_info_data_t()
    guard mach_timebase_info(&ratio) == KERN_SUCCESS else {
      throw ProbeError("Cannot read Mach timebase")
    }
    deadline = try captureDeadline(
      start: start, seconds: seconds, numerator: ratio.numer, denominator: ratio.denom)
    self.output = output
    self.plan = plan
    self.verifyReceipt = verifyReceipt
    let path = output.appendingPathComponent("frames.jsonl")
    let fd = path.path.withCString {
      Darwin.open($0, O_WRONLY | O_CREAT | O_EXCL | O_CLOEXEC, mode_t(0o600))
    }
    guard fd >= 0 else {
      throw ProbeError("Cannot exclusively create frame journal (errno \(errno))")
    }
    journal = FileHandle(fileDescriptor: fd, closeOnDealloc: true)
    timebase = ratio
    super.init()
  }

  private func append(_ row: [String: Any]) throws {
    var data = try JSONSerialization.data(withJSONObject: row, options: [.sortedKeys])
    data.append(10)
    try journal.write(contentsOf: data)
  }

  func startMarkers() {
    queue.sync {
      let timer = DispatchSource.makeTimerSource(queue: queue)
      timer.schedule(deadline: .now(), repeating: .milliseconds(5))
      timer.setEventHandler { [weak self] in self?.readMarker() }
      self.timer = timer
      timer.resume()
    }
  }

  private func readMarker() {
    guard failure == nil, finalReport == nil, mach_absolute_time() <= deadline else { return }
    let request = output.appendingPathComponent("marker.request")
    guard let attributes = try? FileManager.default.attributesOfItem(atPath: request.path),
      attributes[.type] as? FileAttributeType == .typeRegular,
      let size = attributes[.size] as? NSNumber, size.intValue <= 32,
      let bytes = try? Data(contentsOf: request), let id = String(data: bytes, encoding: .utf8),
      validMarker(id), !markerIDs.contains(id)
    else { return }
    do {
      let ticks = mach_absolute_time()
      guard ticks <= deadline else { return }
      let row: [String: Any] = [
        "event": "marker", "id": id, "mach_ticks": String(ticks), "frame_count_before": frameCount,
      ]
      try append(row)
      try journal.synchronize()
      try writeJSON(row, to: output.appendingPathComponent("marker-\(id).json"))
      markerIDs.insert(id)
      latestMarker = ticks
    } catch { failure = "Marker journal/acknowledgment failed: \(error)" }
  }

  func stream(
    _ stream: SCStream, didOutputSampleBuffer sampleBuffer: CMSampleBuffer,
    of type: SCStreamOutputType
  ) {
    guard type == .screen else { return }
    recordSample(sampleBuffer, received: mach_absolute_time())
  }

  func recordSample(_ sample: CMSampleBuffer, received: UInt64) {
    guard failure == nil, finalReport == nil else { return }
    // Pixels/metadata after the fixed output deadline are never inspected or
    // stored, even if asynchronous server stop takes longer to acknowledge.
    guard received <= deadline else {
      afterDeadline += 1
      return
    }
    guard frameCount < 300 else {
      failure = "300-callback budget exceeded"
      return
    }
    frameCount += 1
    let metadata =
      (CMSampleBufferGetSampleAttachmentsArray(sample, createIfNecessary: false)
      as? [[SCStreamFrameInfo: Any]])?.first
    let status = (metadata?[.status] as? NSNumber)?.intValue
    let displayed = (metadata?[.displayTime] as? NSNumber)?.uint64Value
    let scale = (metadata?[.scaleFactor] as? NSNumber)?.doubleValue
    let contentScale = (metadata?[.contentScale] as? NSNumber)?.doubleValue
    let content = metadataRect(metadata?[.contentRect])
    let screen = metadataRect(metadata?[.screenRect])
    var row: [String: Any] = [
      "event": "frame", "index": frameCount, "received_mach_ticks": String(received),
      "status": status as Any? ?? NSNull(),
      "display_mach_ticks": displayed.map(String.init) as Any? ?? NSNull(),
      "scale_factor": recordedNumber(scale), "content_scale": recordedNumber(contentScale),
      "content_rect_points": content.map(rectangle) as Any? ?? NSNull(),
      "screen_rect_points": screen.map(rectangle) as Any? ?? NSNull(),
      "eligible_for_latency": false,
    ]
    let presentation = CMSampleBufferGetPresentationTimeStamp(sample)
    if presentation.isValid && presentation.isNumeric {
      row["presentation_value"] = String(presentation.value)
      row["presentation_timescale"] = presentation.timescale
    }
    if let displayed {
      let ahead = displayed > received
      row["display_time_after_receipt"] = ahead
      if ahead {
        futureDisplayCount += 1
        row["display_lead_at_receipt_ms"] =
          Double(displayed - received) * Double(timebase.numer) / Double(timebase.denom) / 1_000_000
      }
    }
    var complete = false
    do {
      try verifyReceipt()
      row["target_geometry_matches_at_receipt"] = true
      guard let status, let frameStatus = SCFrameStatus(rawValue: status) else {
        throw ProbeError("Missing or unknown frame status")
      }
      if frameStatus == .complete {
        guard let displayed, displayed > 0,
          displayed >= previousDisplay,
          let buffer = CMSampleBufferGetImageBuffer(sample)
        else { throw ProbeError("Missing image or invalid/nonmonotonic display clock") }
        guard matchingHostPresentation(presentation, displayed: displayed) else {
          throw ProbeError("Presentation timestamp disagrees with the converted display clock")
        }
        row["presentation_matches_converted_display_time"] = true
        let w = CVPixelBufferGetWidth(buffer)
        let h = CVPixelBufferGetHeight(buffer)
        row["width_pixels"] = w
        row["height_pixels"] = h
        try plan.validateFrame(
          width: w, height: h, content: content, screen: screen, scale: scale,
          contentScale: contentScale)
        if displayed > deadline {
          // A prospective frame beyond the predeclared recording interval is
          // retained as metadata, never as pixels or a timing observation.
          row["outside_presentation_window"] = true
          outsidePresentationWindow += 1
        } else {
          let image = CIImage(cvPixelBuffer: buffer)
          guard let cg = context.createCGImage(image, from: image.extent) else {
            throw ProbeError("Cannot encode cropped image")
          }
          let data = NSMutableData()
          guard
            let writer = CGImageDestinationCreateWithData(
              data, UTType.png.identifier as CFString, 1, nil)
          else {
            throw ProbeError("Cannot create PNG encoder")
          }
          CGImageDestinationAddImage(writer, cg, nil)
          guard CGImageDestinationFinalize(writer) else {
            throw ProbeError("Cannot finish PNG encoder")
          }
          let (name, digest) = try recordPNG(data as Data)
          row["png"] = name
          row["png_sha256"] = digest
          row["frame_geometry_validated"] = true
          if let bound = upperBoundMilliseconds(
            marker: latestMarker, displayed: displayed, numerator: timebase.numer,
            denominator: timebase.denom)
          {
            row["since_acknowledged_marker_ms"] = bound
          }
          complete = true
        }
        previousDisplay = displayed
      } else if frameStatus != .idle && frameStatus != .started {
        throw ProbeError("Noncomplete stream status \(status) invalidates the trial")
      }
    } catch {
      failure = String(describing: error)
      row["rejected_error"] = failure
    }
    do {
      try append(row)
      if complete && failure == nil { completeCount += 1 }
    } catch { if failure == nil { failure = "Frame journal failed: \(error)" } }
  }

  func recordPNG(_ data: Data) throws -> (String, String) {
    let digest = sha256Hex(data)
    let name = "frame-\(sha256Hex(data)).png"
    if images[digest] == nil {
      guard data.count <= 64 * 1024 * 1024 - storedBytes else {
        throw ProbeError("64 MiB unique-image budget exceeded")
      }
      let path = output.appendingPathComponent(name)
      try data.write(to: path, options: .withoutOverwriting)
      try FileManager.default.setAttributes([.posixPermissions: 0o600], ofItemAtPath: path.path)
      images[digest] = name
      storedBytes += data.count
    }
    return (name, digest)
  }

  func stream(_ stream: SCStream, didStopWithError error: Error) {
    recordStreamFailure(String(describing: error))
  }
  func recordStreamFailure(_ message: String) {
    queue.async {
      guard self.failure == nil, self.finalReport == nil else { return }
      self.failure = "Stream stopped: \(message)"
    }
  }

  func finish() -> [String: Any] {
    queue.sync {
      if let finalReport { return finalReport }
      timer?.cancel()
      timer = nil
      do {
        try journal.synchronize()
        try journal.close()
      } catch { closeFailure = String(describing: error) }
      let report: [String: Any] = [
        "collector_error": failure as Any? ?? NSNull(),
        "journal_close_error": closeFailure as Any? ?? NSNull(),
        "frames_received_within_deadline": frameCount,
        "callbacks_after_deadline_ignored": afterDeadline,
        "deadline_mach_ticks": String(deadline), "complete_frames": completeCount,
        "unique_images": images.count,
        "image_bytes": storedBytes, "marker_count": markerIDs.count,
        "timebase_numerator": timebase.numer,
        "timebase_denominator": timebase.denom, "eligible_latency_observations": 0,
        "latency_gate_passed": NSNull(),
        "pixel_mapping_calibrated": false, "display_visibility_calibrated": false,
        "clock_calibrated_for_latency": false,
        "frames_with_display_time_after_receipt": futureDisplayCount,
        "frames_outside_presentation_window": outsidePresentationWindow,
      ]
      finalReport = report
      return report
    }
  }
}
