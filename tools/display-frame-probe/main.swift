import AppKit
import CoreMedia
import Darwin
import Foundation
@preconcurrency import ScreenCaptureKit

struct Options {
  let inspect: Bool
  let pid: Int32
  let window: UInt32
  let executable: URL
  let digest: String
  let display: UInt32?
  let crop: CGRect?
  let output: URL?
  let seconds: Double?

  static func parse(_ args: [String]) throws -> Options {
    guard let mode = args.first, ["--inspect-target", "--capture-display"].contains(mode) else {
      throw ProbeError("Use explicit --inspect-target or authorized --capture-display mode")
    }
    let inspect = mode == "--inspect-target"
    let keys =
      ["--pid", "--window", "--executable", "--sha256"]
      + (inspect ? [] : ["--display", "--crop", "--output", "--seconds"])
    var fields: [String: String] = [:]
    var index = 1
    while index < args.count {
      let key = args[index]
      guard keys.contains(key), fields[key] == nil, index + 1 < args.count else {
        throw ProbeError("Unknown/repeated/incomplete option \(key)")
      }
      fields[key] = args[index + 1]
      index += 2
    }
    guard fields.count == keys.count, let pid = Int32(fields["--pid"] ?? ""), pid > 0,
      let window = UInt32(fields["--window"] ?? ""), window > 0,
      let path = fields["--executable"], path.hasPrefix("/"),
      let digest = fields["--sha256"], digest.count == 64,
      digest.allSatisfy({ "0123456789abcdef".contains($0) })
    else {
      throw ProbeError("Need exact positive PID/window, absolute executable and lowercase SHA-256")
    }
    var display: UInt32?
    var crop: CGRect?
    var output: URL?
    var seconds: Double?
    if !inspect {
      guard let d = UInt32(fields["--display"] ?? ""), d > 0, let raw = fields["--crop"],
        let folder = fields["--output"], folder.hasPrefix("/"),
        let duration = Double(fields["--seconds"] ?? ""), duration.isFinite, duration >= 0.1,
        duration <= 5
      else {
        throw ProbeError(
          "Capture needs explicit display, crop, absolute fresh output and 0.1–5 seconds")
      }
      display = d
      crop = try parseRectangle(raw)
      output = URL(fileURLWithPath: folder)
      seconds = duration
    }
    return Options(
      inspect: inspect, pid: pid, window: window, executable: URL(fileURLWithPath: path),
      digest: digest,
      display: display, crop: crop, output: output, seconds: seconds)
  }
}

func displayMode(_ display: UInt32) -> [Int] {
  guard CGDisplayIsActive(display) != 0, let mode = CGDisplayCopyDisplayMode(display) else {
    return []
  }
  return [mode.width, mode.height, mode.pixelWidth, mode.pixelHeight]
}

func ownedWindow(_ window: UInt32, pid: Int32) -> CGRect? {
  guard
    let rows = CGWindowListCopyWindowInfo(
      [.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID) as? [[String: Any]]
  else { return nil }
  let matches = rows.filter { ($0[kCGWindowNumber as String] as? NSNumber)?.uint32Value == window }
  guard matches.count == 1, let row = matches.first,
    (row[kCGWindowOwnerPID as String] as? NSNumber)?.int32Value == pid,
    (row[kCGWindowLayer as String] as? NSNumber)?.intValue == 0
  else { return nil }
  return metadataRect(row[kCGWindowBounds as String])
}

@MainActor func requireIdentity(_ o: Options) throws {
  guard let app = NSRunningApplication(processIdentifier: o.pid), !app.isTerminated, !app.isHidden,
    app.bundleIdentifier == "com.dshills.elab",
    app.executableURL?.resolvingSymlinksInPath() == o.executable.resolvingSymlinksInPath(),
    sha256Hex(try Data(contentsOf: o.executable)) == o.digest
  else { throw ProbeError("Exact owned ELab process identity unavailable") }
}

@MainActor func discover(_ o: Options) async throws -> (CapturePlan, SCDisplay, [Int]) {
  try requireIdentity(o)
  guard CGPreflightScreenCaptureAccess() else {
    throw ProbeError("Screen Recording access unavailable; no permission request made")
  }
  _ = NSApplication.shared
  let content = try await SCShareableContent.excludingDesktopWindows(
    true, onScreenWindowsOnly: true)
  let windows = content.windows.filter {
    $0.windowID == o.window && $0.owningApplication?.processID == o.pid && $0.isOnScreen
  }
  guard windows.count == 1, let window = windows.first,
    let cgWindow = ownedWindow(o.window, pid: o.pid)
  else {
    throw ProbeError("Unique owned on-screen window identity unavailable")
  }
  let displays = content.displays.filter { $0.frame.contains(window.frame) }
  guard displays.count == 1, let display = displays.first,
    o.display == nil || o.display == display.displayID
  else {
    throw ProbeError("Target must lie entirely within exactly one explicit display")
  }
  let screens = NSScreen.screens.filter {
    ($0.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")] as? NSNumber)?.uint32Value
      == display.displayID
  }
  guard screens.count == 1, let screen = screens.first else {
    throw ProbeError("Independent display backing scale unavailable")
  }
  let filter = SCContentFilter(display: display, excludingWindows: [])
  let plan = try CapturePlan(
    displayID: display.displayID, display: CGDisplayBounds(display.displayID),
    scDisplay: display.frame,
    window: cgWindow, scWindow: window.frame, scale: Double(filter.pointPixelScale),
    backingScale: screen.backingScaleFactor, requested: o.crop)
  let mode = displayMode(display.displayID)
  guard mode.count == 4, display.width == Int(display.frame.width),
    display.height == Int(display.frame.height)
  else {
    throw ProbeError("Display mode or point dimensions disagree")
  }
  return (plan, display, mode)
}

@main struct Main {
  @MainActor static func main() async {
    var output: URL?
    var observer: DisplayCollector?
    var stream: SCStream?
    var startRequested = false
    var captureStarted = false
    var needsStop = false
    do {
      let args = Array(CommandLine.arguments.dropFirst())
      if args == ["--self-test"] {
        try offlineChecks()
        return
      }
      if args == ["--preflight"] {
        print(
          "{\"screen_access_previously_granted\":\(CGPreflightScreenCaptureAccess()),\"permission_requested\":false,\"capture_executed\":false}"
        )
        return
      }
      if args.isEmpty || args == ["--help"] {
        print(
          "display-frame-probe --self-test | --preflight | --inspect-target --pid PID --window ID --executable /path/ELab --sha256 HASH"
        )
        print(
          "Authorized capture: --capture-display [identity options] --display ID --crop x,y,width,height --output /new/temp/directory --seconds 0.1...5"
        )
        print(
          "Explicit verified window crop only; no audio, input operation, OS permission request or automatic latency acceptance."
        )
        return
      }
      let options = try Options.parse(args)
      if let path = options.output {
        guard
          permittedOutput(
            path,
            roots: [URL(fileURLWithPath: "/private/tmp"), FileManager.default.temporaryDirectory])
        else {
          throw ProbeError(
            "Output must be a new private temporary directory outside every app bundle")
        }
      }
      let (plan, display, mode) = try await discover(options)
      if options.inspect {
        let result: [String: Any] = [
          "capture_plan": plan.json, "display_mode": mode,
          "capture_executed": false, "permission_requested": false, "pid": options.pid,
          "window_id": options.window,
        ]
        print(
          String(
            decoding: try JSONSerialization.data(withJSONObject: result, options: [.sortedKeys]),
            as: UTF8.self))
        return
      }
      guard let directory = options.output, let seconds = options.seconds,
        options.crop == plan.source, options.display == plan.displayID
      else {
        throw ProbeError("Missing explicit verified crop; uncropped display fallback is forbidden")
      }
      let filter = SCContentFilter(display: display, excludingWindows: [])
      let config = SCStreamConfiguration()
      config.sourceRect = plan.source
      config.width = plan.width
      config.height = plan.height
      config.destinationRect = CGRect(x: 0, y: 0, width: plan.width, height: plan.height)
      config.scalesToFit = false
      config.preservesAspectRatio = true
      config.minimumFrameInterval = CMTime(value: 1, timescale: 60)
      config.queueDepth = 3
      config.pixelFormat = kCVPixelFormatType_32BGRA
      config.showsCursor = false
      config.capturesAudio = false
      if #available(macOS 15.0, *) { config.captureMicrophone = false }
      // No exclusion masks: an overlapping window must remain visible.
      if #available(macOS 14.2, *) { filter.includeMenuBar = true }
      guard config.sourceRect == plan.source, config.sourceRect.width > 0,
        config.sourceRect.height > 0
      else {
        throw ProbeError("Source crop not retained by stream configuration")
      }
      try createPrivateOutput(directory)
      output = directory
      let start = mach_absolute_time()
      let pid = options.pid
      let window = options.window
      let collector = try DisplayCollector(
        output: directory, plan: plan, start: start, seconds: seconds,
        verifyReceipt: {
          try plan.validateReceipt(
            window: ownedWindow(window, pid: pid), owner: pid, expectedPID: pid,
            display: CGDisplayBounds(plan.displayID), mode: displayMode(plan.displayID),
            expectedMode: mode)
        })
      observer = collector
      let settings: [String: Any] = [
        "capture_plan": plan.json, "display_mode": mode, "seconds": seconds,
        "start_mach_ticks": String(start), "deadline_mach_ticks": String(collector.deadline),
        "pid": pid, "window_id": window, "target_executable_sha256": options.digest,
        "tool_executable_sha256": sha256Hex(
          try Data(contentsOf: URL(fileURLWithPath: CommandLine.arguments[0]))),
        "filter": "display-excluding-no-windows", "minimum_frame_interval": "1/60",
        "queue_depth": 3,
        "audio": false, "microphone": false, "cursor": false, "permission_requested": false,
      ]
      try writeJSON(settings, to: directory.appendingPathComponent("configuration.json"))
      let capture = SCStream(filter: filter, configuration: config, delegate: collector)
      stream = capture
      try capture.addStreamOutput(collector, type: .screen, sampleHandlerQueue: collector.queue)
      startRequested = true
      needsStop = true
      try await capture.startCapture()
      captureStarted = true
      collector.startMarkers()
      let now = mach_absolute_time()
      if now < collector.deadline {
        var ratio = mach_timebase_info_data_t()
        _ = mach_timebase_info(&ratio)
        let remaining =
          Double(collector.deadline - now) * Double(ratio.numer) / Double(ratio.denom)
          / 1_000_000_000
        try await Task.sleep(for: .seconds(remaining))
      }
      try await capture.stopCapture()
      needsStop = false
      try requireIdentity(options)
      let (after, _, afterMode) = try await discover(options)
      guard after.json as NSDictionary == plan.json as NSDictionary, afterMode == mode else {
        throw ProbeError("Postcapture geometry/identity changed")
      }
      var report = collector.finish()
      report["configuration_sha256"] = sha256Hex(
        try Data(contentsOf: directory.appendingPathComponent("configuration.json")))
      report["capture_executed"] = true
      report["geometry_rechecked_after_stop"] = true
      let reason =
        (report["collector_error"] as? String) ?? (report["journal_close_error"] as? String)
        ?? ((report["complete_frames"] as? Int ?? 0) > 0
          ? nil : "No complete validated frame was received")
      report["command_error"] = reason as Any? ?? NSNull()
      report["capture_start_requested"] = startRequested
      report["capture_started"] = captureStarted
      try writeJSON(report, to: directory.appendingPathComponent("report.json"))
      guard reason == nil else {
        throw ProbeError("Capture invalid; inspect retained metadata/report")
      }
      print(
        "{\"capture_executed\":true,\"eligible_latency_observations\":0,\"latency_gate_passed\":null}"
      )
    } catch {
      var stopError: String?
      if needsStop, let stream {
        do { try await stream.stopCapture() } catch { stopError = String(describing: error) }
      }
      var report =
        observer?.finish() ?? ["eligible_latency_observations": 0, "latency_gate_passed": NSNull()]
      report["command_error"] = String(describing: error)
      report["stop_error"] = stopError as Any? ?? NSNull()
      report["capture_start_requested"] = startRequested
      report["capture_started"] = captureStarted
      if let output,
        !FileManager.default.fileExists(atPath: output.appendingPathComponent("report.json").path)
      {
        try? writeJSON(report, to: output.appendingPathComponent("report.json"))
      }
      FileHandle.standardError.write(Data(("display-frame-probe: \(error)\n").utf8))
      exit(1)
    }
  }
}
