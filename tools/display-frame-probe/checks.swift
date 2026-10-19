// Synthetic pixels and filesystem/geometry faults only. No UI, SCStream,
// access preflight or capture call occurs in this entry point.
import CoreMedia
import CoreVideo
import Darwin
import Foundation
import ImageIO
@preconcurrency import ScreenCaptureKit

func fixturePlan(scale: Double = 1) throws -> CapturePlan {
  let display = CGRect(x: 100, y: 200, width: 800, height: 600)
  let window = CGRect(x: 110, y: 210, width: 4, height: 3)
  return try CapturePlan(
    displayID: 7, display: display, scDisplay: display, window: window,
    scWindow: window, scale: scale, backingScale: scale,
    requested: CGRect(x: 10, y: 10, width: 4, height: 3))
}

func testRequire(_ condition: Bool, _ message: String) throws {
  if !condition { throw ProbeError(message) }
}
func mustRefuse(_ name: String, _ body: () throws -> Void) throws {
  do { try body() } catch { return }
  throw ProbeError("Invalid fixture admitted: \(name)")
}

func testSample(
  status: Int? = SCFrameStatus.complete.rawValue, displayed: UInt64? = 200,
  scale: Double? = 1, contentScale: Double? = 1, width: Int = 4, height: Int = 3,
  content: CGRect? = CGRect(x: 0, y: 0, width: 4, height: 3),
  screen: CGRect? = CGRect(x: 110, y: 210, width: 4, height: 3), image: Bool = true
) throws -> CMSampleBuffer {
  var pixel: CVPixelBuffer?
  guard
    CVPixelBufferCreate(kCFAllocatorDefault, width, height, kCVPixelFormatType_32BGRA, nil, &pixel)
      == kCVReturnSuccess,
    let pixel
  else { throw ProbeError("Cannot allocate synthetic pixels") }
  CVPixelBufferLockBaseAddress(pixel, [])
  if let base = CVPixelBufferGetBaseAddress(pixel) {
    memset(base, 0, CVPixelBufferGetDataSize(pixel))
    for y in 0..<height {
      let row = base.advanced(by: y * CVPixelBufferGetBytesPerRow(pixel)).assumingMemoryBound(
        to: UInt8.self)
      for x in 0..<width {
        row[4 * x] = 70
        row[4 * x + 3] = 255
      }
    }
  }
  CVPixelBufferUnlockBaseAddress(pixel, [])
  var description: CMVideoFormatDescription?
  guard
    CMVideoFormatDescriptionCreateForImageBuffer(
      allocator: kCFAllocatorDefault, imageBuffer: pixel, formatDescriptionOut: &description)
      == noErr,
    let description
  else { throw ProbeError("Cannot describe synthetic pixels") }
  var timing = CMSampleTimingInfo(
    duration: .invalid, presentationTimeStamp: CMTime(value: 1, timescale: 60),
    decodeTimeStamp: .invalid)
  var sample: CMSampleBuffer?
  let result =
    image
    ? CMSampleBufferCreateReadyWithImageBuffer(
      allocator: kCFAllocatorDefault, imageBuffer: pixel, formatDescription: description,
      sampleTiming: &timing, sampleBufferOut: &sample)
    : CMSampleBufferCreateReady(
      allocator: kCFAllocatorDefault, dataBuffer: nil, formatDescription: description,
      sampleCount: 1, sampleTimingEntryCount: 1, sampleTimingArray: &timing,
      sampleSizeEntryCount: 0, sampleSizeArray: nil, sampleBufferOut: &sample)
  guard result == noErr, let sample,
    let attachments = CMSampleBufferGetSampleAttachmentsArray(sample, createIfNecessary: true)
  else {
    throw ProbeError("Cannot construct synthetic metadata")
  }
  let row = unsafeBitCast(CFArrayGetValueAtIndex(attachments, 0), to: CFMutableDictionary.self)
  func put(_ key: SCStreamFrameInfo, _ value: AnyObject) {
    let text = key.rawValue as CFString
    CFDictionarySetValue(
      row, Unmanaged.passUnretained(text).toOpaque(), Unmanaged.passUnretained(value).toOpaque())
  }
  if let status { put(.status, NSNumber(value: status)) }
  if let displayed { put(.displayTime, NSNumber(value: displayed)) }
  if let scale { put(.scaleFactor, NSNumber(value: scale)) }
  if let contentScale { put(.contentScale, NSNumber(value: contentScale)) }
  if let content { put(.contentRect, content.dictionaryRepresentation) }
  if let screen { put(.screenRect, screen.dictionaryRepresentation) }
  return sample
}

func offlineChecks() throws {
  var cases: [String] = []
  let display = CGRect(x: -800, y: -600, width: 800, height: 600)
  let window = CGRect(x: -790, y: -580, width: 100, height: 80)
  let plan = try CapturePlan(
    displayID: 1, display: display, scDisplay: display, window: window, scWindow: window,
    scale: 2, backingScale: 2, requested: CGRect(x: 10, y: 20, width: 100, height: 80))
  try testRequire(
    plan.width == 200 && plan.height == 160 && plan.source.origin == CGPoint(x: 10, y: 20),
    "Negative display origin/native scale mapping failed")
  cases.append("negative-display-origin-and-2x-native-pixel-transform")
  for text in [
    "", "0,0,0,3", "0,0,-1,3", "0,0,1,-3", "nan,0,1,3", "0,inf,1,3", "0,0,1.5,3", "0,0,1,3,4",
    "0,0,,3",
  ] {
    try mustRefuse("crop:" + text) { _ = try parseRectangle(text) }
  }
  cases.append("empty-nonfinite-negative-fractional-and-malformed-crops-refused")
  let fixture = try fixturePlan()
  for (name, d, w, scw, scale, backing, crop) in [
    (
      "moved-sc-window", fixture.display, fixture.window, fixture.window.offsetBy(dx: 1, dy: 0),
      1.0, 1.0, fixture.source
    ),
    (
      "outside", fixture.display, fixture.window.offsetBy(dx: -20, dy: 0),
      fixture.window.offsetBy(dx: -20, dy: 0), 1.0, 1.0, fixture.source
    ),
    (
      "off-display-bottom", fixture.display, CGRect(x: 110, y: 799, width: 4, height: 3),
      CGRect(x: 110, y: 799, width: 4, height: 3), 1.0, 1.0, fixture.source
    ),
    (
      "scale-disagreement", fixture.display, fixture.window, fixture.window, 2.0, 1.0,
      fixture.source
    ),
    ("scale-zero", fixture.display, fixture.window, fixture.window, 0.0, 0.0, fixture.source),
    (
      "scale-infinite", fixture.display, fixture.window, fixture.window, Double.infinity,
      Double.infinity, fixture.source
    ),
    (
      "wrong-inside-crop", fixture.display, fixture.window, fixture.window, 1.0, 1.0,
      CGRect(x: 10, y: 10, width: 3, height: 3)
    ),
    (
      "wrong-origin", fixture.display, fixture.window, fixture.window, 1.0, 1.0,
      fixture.source.offsetBy(dx: 1, dy: 0)
    ),
    (
      "oversized", CGRect(x: 0, y: 0, width: 10000, height: 10000),
      CGRect(x: 0, y: 0, width: 5000, height: 3), CGRect(x: 0, y: 0, width: 5000, height: 3), 1.0,
      1.0, CGRect(x: 0, y: 0, width: 5000, height: 3)
    ),
  ] {
    try mustRefuse(name) {
      _ = try CapturePlan(
        displayID: 7, display: d, scDisplay: d, window: w, scWindow: scw, scale: scale,
        backingScale: backing, requested: crop)
    }
    cases.append(name + "-refused-before-capture")
  }
  for (name, w, owner, d, mode) in [
    ("hidden", Optional<CGRect>.none, Int32?(9), fixture.display, [1]),
    ("moved", Optional(fixture.window.offsetBy(dx: 1, dy: 0)), Int32?(9), fixture.display, [1]),
    ("owner", Optional(fixture.window), Int32?(10), fixture.display, [1]),
    ("display", Optional(fixture.window), Int32?(9), fixture.display.offsetBy(dx: 1, dy: 0), [1]),
    ("mode", Optional(fixture.window), Int32?(9), fixture.display, [2]),
  ] {
    try mustRefuse(name) {
      try fixture.validateReceipt(
        window: w, owner: owner, expectedPID: 9, display: d, mode: mode, expectedMode: [1])
    }
    cases.append(name + "-receipt-invalidates-frozen-crop")
  }
  let valid = [
    "--capture-display", "--pid", "7", "--window", "9", "--executable", "/tmp/ELab", "--sha256",
    String(repeating: "a", count: 64), "--display", "1", "--crop", "0,0,4,3", "--output",
    "/private/tmp/fresh", "--seconds", "1",
  ]
  _ = try Options.parse(valid)
  for bad in [
    Array(valid.dropFirst()), valid + ["--pid", "8"], Array(valid.dropLast()) + ["nan"],
    Array(valid.dropLast()) + ["5.1"], Array(valid.dropLast(2)), Array(valid.prefix(11)),
  ] {
    try mustRefuse("options") { _ = try Options.parse(bad) }
  }
  cases.append("explicit-mode-display-crop-duration-options-required")
  try mustRefuse("overflow") {
    _ = try captureDeadline(start: UInt64.max, seconds: 1, numerator: 1, denominator: 1)
  }
  try mustRefuse("zero-timebase") {
    _ = try captureDeadline(start: 1, seconds: 1, numerator: 0, denominator: 1)
  }
  cases.append("deadline-overflow-and-invalid-timebase-refused")
  let temp = FileManager.default.temporaryDirectory.appendingPathComponent(
    "elab-display-probe-checks-\(UUID().uuidString)")
  try createPrivateOutput(temp)
  defer { try? FileManager.default.removeItem(at: temp) }
  func collector(_ name: String, verify: @escaping @Sendable () throws -> Void = {}) throws -> (
    URL, DisplayCollector
  ) {
    let p = temp.appendingPathComponent(name)
    try createPrivateOutput(p)
    return (
      p,
      try DisplayCollector(output: p, plan: fixture, start: 100, seconds: 1, verifyReceipt: verify)
    )
  }
  func rows(_ p: URL) throws -> [[String: Any]] {
    try Data(contentsOf: p.appendingPathComponent("frames.jsonl")).split(separator: 10).map {
      try JSONSerialization.jsonObject(with: Data($0)) as! [String: Any]
    }
  }
  let badSamples: [(String, CMSampleBuffer)] = [
    ("missing-status", try testSample(status: nil)),
    ("missing-time", try testSample(displayed: nil)),
    ("zero-time", try testSample(displayed: 0)), ("future-time", try testSample(displayed: 400)),
    ("missing-scale", try testSample(scale: nil)),
    ("nan-scale", try testSample(scale: Double.nan)),
    ("infinite-content-scale", try testSample(contentScale: Double.infinity)),
    (
      "nonfinite-screen", try testSample(screen: CGRect(x: Double.nan, y: 210, width: 4, height: 3))
    ), ("scale-change", try testSample(scale: 2)),
    ("rescale", try testSample(contentScale: 0.5)),
    ("missing-content-scale", try testSample(contentScale: nil)),
    ("missing-content", try testSample(content: nil)),
    ("wrong-content-origin", try testSample(content: CGRect(x: 1, y: 0, width: 4, height: 3))),
    ("missing-screen", try testSample(screen: nil)),
    ("wrong-screen-origin", try testSample(screen: CGRect(x: 111, y: 210, width: 4, height: 3))),
    ("wrong-width", try testSample(width: 5)), ("missing-image", try testSample(image: false)),
    ("blank", try testSample(status: SCFrameStatus.blank.rawValue)),
    ("suspended", try testSample(status: SCFrameStatus.suspended.rawValue)),
  ]
  for (name, sample) in badSamples {
    let (path, c) = try collector(name)
    c.queue.sync { c.recordSample(sample, received: 300) }
    c.recordStreamFailure("secondary error")
    let report = c.finish()
    let journal = try rows(path)
    try testRequire(
      report["collector_error"] is String && report["unique_images"] as? Int == 0
        && journal.count == 1 && journal[0]["rejected_error"] is String,
      "Bad sample lost its cause or stored pixels: \(name)")
    try testRequire(
      (report["collector_error"] as? String)?.contains("secondary") == false,
      "Secondary stream error replaced the first sample cause")
    cases.append(name + "-metadata-retained-without-pixels")
  }
  do {
    let (path, c) = try collector(
      "receipt-rejection", verify: { throw ProbeError("synthetic geometry changed") })
    let sample = try testSample()
    c.queue.sync { c.recordSample(sample, received: 300) }
    let r = c.finish()
    try testRequire(
      r["collector_error"] as? String == "synthetic geometry changed"
        && r["unique_images"] as? Int == 0 && (try rows(path)).count == 1,
      "Changed geometry reached encoder")
    cases.append("receipt-failure-journaled-before-image-encoding")
  }
  do {
    let (path, c) = try collector("clock-regression")
    let first = try testSample(displayed: 200)
    let second = try testSample(displayed: 199)
    c.queue.sync {
      c.recordSample(first, received: 300)
      c.recordSample(second, received: 301)
    }
    let r = c.finish()
    let journal = try rows(path)
    try testRequire(
      r["complete_frames"] as? Int == 1 && r["unique_images"] as? Int == 1 && journal.count == 2
        && journal[1]["rejected_error"] is String, "Regressing clock reached image storage")
    cases.append("regressing-display-clock-invalidates-with-earlier-image-preserved")
  }
  do {
    let (path, c) = try collector("first-stream-error")
    c.recordStreamFailure("first")
    c.recordStreamFailure("second")
    let sample = try testSample()
    c.queue.sync { c.recordSample(sample, received: 300) }
    let r = c.finish()
    try testRequire(
      r["collector_error"] as? String == "Stream stopped: first" && (try rows(path)).isEmpty,
      "First stream failure or subsequent-frame refusal lost")
    cases.append("first-stream-error-retained-and-subsequent-frames-refused")
  }
  var png: Data?
  do {
    let (path, c) = try collector("valid-dedup")
    let sample = try testSample()
    c.queue.sync {
      c.recordSample(sample, received: 300)
      c.recordSample(sample, received: 301)
    }
    let r = c.finish()
    let journal = try rows(path)
    try testRequire(
      r["collector_error"] is NSNull && r["complete_frames"] as? Int == 2
        && r["unique_images"] as? Int == 1 && journal.count == 2,
      "Valid synthetic image/dedup failed")
    let p = path.appendingPathComponent(journal[0]["png"] as! String)
    let data = try Data(contentsOf: p)
    png = data
    guard let source = CGImageSourceCreateWithData(data as CFData, nil),
      let decoded = CGImageSourceCreateImageAtIndex(source, 0, nil)
    else { throw ProbeError("Synthetic PNG cannot decode") }
    try testRequire(
      decoded.width == 4 && decoded.height == 3
        && sha256Hex(data) == journal[0]["png_sha256"] as? String,
      "Native synthetic pixel dimensions/digest changed")
    try testRequire(
      (try FileManager.default.attributesOfItem(atPath: p.path)[.posixPermissions] as? NSNumber)?
        .intValue == 0o600, "Image permissions changed")
    let before = try Data(contentsOf: path.appendingPathComponent("frames.jsonl"))
    c.queue.sync { c.recordSample(sample, received: 302) }
    c.recordStreamFailure("late")
    try testRequire(
      NSDictionary(dictionary: r).isEqual(to: c.finish())
        && (try Data(contentsOf: path.appendingPathComponent("frames.jsonl"))) == before,
      "Late callbacks changed finalized evidence")
    cases.append("native-synthetic-pixels-decode-deduplicate-and-stay-ineligible")
    cases.append("late-frame-stream-and-finish-callbacks-preserve-evidence")
  }
  do {
    let (path, c) = try collector("collision")
    guard let png else { throw ProbeError("No PNG fixture") }
    let target = path.appendingPathComponent("frame-\(sha256Hex(png)).png")
    let sentinel = Data("prior bytes".utf8)
    try sentinel.write(to: target, options: .withoutOverwriting)
    let sample = try testSample()
    c.queue.sync { c.recordSample(sample, received: 300) }
    let r = c.finish()
    try testRequire(
      r["collector_error"] is String && r["unique_images"] as? Int == 0
        && (try Data(contentsOf: target)) == sentinel, "Image collision changed evidence")
    cases.append("existing-PNG-refused-with-original-bytes-preserved")
  }
  do {
    let (_, c) = try collector("deadline")
    let sample = try testSample()
    c.queue.sync { c.recordSample(sample, received: c.deadline + 1) }
    let r = c.finish()
    try testRequire(
      r["frames_received_within_deadline"] as? Int == 0 && r["unique_images"] as? Int == 0
        && r["callbacks_after_deadline_ignored"] as? Int == 1,
      "Late recording inspected/stored pixels")
    cases.append("callbacks-after-fixed-deadline-ignored-before-metadata-or-pixels")
  }
  do {
    let (path, c) = try collector("frame-limit")
    let sample = try testSample(status: SCFrameStatus.idle.rawValue)
    c.queue.sync { for _ in 0..<301 { c.recordSample(sample, received: 300) } }
    let r = c.finish()
    try testRequire(
      (r["collector_error"] as? String)?.contains("300-callback") == true
        && (try rows(path)).count == 300, "Callback budget changed")
    cases.append("301st-callback-refused-with-300-rows-preserved")
  }
  do {
    let (_, c) = try collector("storage-limit")
    let payload = Data(repeating: 42, count: 64 * 1024 * 1024)
    let first = try c.queue.sync { try c.recordPNG(payload) }
    let duplicate = try c.queue.sync { try c.recordPNG(payload) }
    try testRequire(first == duplicate, "Duplicate consumed storage budget")
    try mustRefuse("byte-limit") { _ = try c.queue.sync { try c.recordPNG(Data([1])) } }
    let r = c.finish()
    try testRequire(
      r["image_bytes"] as? Int == payload.count && r["unique_images"] as? Int == 1,
      "Storage accounting failed")
    cases.append("64-MiB-unique-byte-limit-duplicates-and-overflow")
  }
  let journal = try journalFaultChecks()
  let result: [String: Any] = [
    "self_tests_passed": true, "capture_executed": false, "ui_or_privacy_api_called": false,
    "eligible_latency_observations": 0, "cases": cases, "journal_cases": journal,
  ]
  print(
    String(
      decoding: try JSONSerialization.data(withJSONObject: result, options: [.sortedKeys]),
      as: UTF8.self))
}
