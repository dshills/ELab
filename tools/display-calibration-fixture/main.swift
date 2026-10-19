// Owned static visual fixture for the explicitly authorized calibration method.
// No capture, global input listener, event injection or permission request.
import AppKit
import Darwin
import Foundation

struct Target: Decodable {
  struct Bounds: Decodable {
    let x: Double, y: Double, width: Double, height: Double
    var rect: CGRect { CGRect(x: x, y: y, width: width, height: height) }
  }
  let pid: Int32, window: UInt32, executable: String, sha256: String, bounds: Bounds
}

func cgBounds(window: UInt32, pid: Int32) -> CGRect? {
  guard
    let rows = CGWindowListCopyWindowInfo(
      [.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID) as? [[String: Any]]
  else { return nil }
  let matches = rows.filter {
    ($0[kCGWindowNumber as String] as? NSNumber)?.uint32Value == window
      && ($0[kCGWindowOwnerPID as String] as? NSNumber)?.int32Value == pid
  }
  guard matches.count == 1, let bounds = matches[0][kCGWindowBounds as String] as? NSDictionary
  else { return nil }
  return CGRect(dictionaryRepresentation: bounds)
}

@MainActor final class PatchPanel: NSPanel {
  override var canBecomeKey: Bool { false }
  override var canBecomeMain: Bool { false }
}

@MainActor final class PatchView: NSView {
  let patch: Patch
  private(set) var drawCount = 0
  private(set) var lastDrawFinished: UInt64 = 0
  init(_ patch: Patch) {
    self.patch = patch
    super.init(frame: CGRect(origin: .zero, size: patch.rect.size))
  }
  required init?(coder: NSCoder) { fatalError("No archived fixture views") }
  override var isFlipped: Bool { true }
  override var isOpaque: Bool { true }
  override func draw(_ dirtyRect: NSRect) {
    guard let context = NSGraphicsContext.current?.cgContext else { return }
    context.setShouldAntialias(false)
    for (r, line) in patch.bits.enumerated() {
      for (c, bit) in line.enumerated() {
        context.setFillColor(CGColor(gray: bit == "1" ? 1 : 0, alpha: 1))
        context.fill(
          CGRect(x: c * patch.cell, y: r * patch.cell, width: patch.cell, height: patch.cell))
      }
    }
    drawCount += 1
    lastDrawFinished = mach_absolute_time()
  }
}

@MainActor final class Fixture {
  let target: Target, output: URL, deadline: UInt64
  var panels: [PatchPanel] = [], views: [PatchView] = []
  var ticks = 0, ready = false, finished = false
  var failure: String?
  var timer: Timer?
  init(target: Target, output: URL, seconds: Double) throws {
    self.target = target
    self.output = output
    var ratio = mach_timebase_info_data_t()
    guard mach_timebase_info(&ratio) == KERN_SUCCESS, ratio.numer > 0, ratio.denom > 0 else {
      throw ProbeError("No valid Mach clock")
    }
    let span = seconds * 1_000_000_000 * Double(ratio.denom) / Double(ratio.numer)
    guard span.isFinite, span >= 1, span < Double(UInt64.max) else {
      throw ProbeError("Fixture lifetime overflow")
    }
    let (end, overflow) = mach_absolute_time().addingReportingOverflow(UInt64(span.rounded(.down)))
    guard !overflow else { throw ProbeError("Fixture deadline overflow") }
    deadline = end
    var initialized = false
    defer {
      if !initialized {
        for panel in panels {
          panel.orderOut(nil)
          panel.close()
        }
      }
    }
    try requireTarget(verifyBinary: true)
    let primary = CGDisplayBounds(CGMainDisplayID()).height
    for patch in try calibrationPatches(target.bounds.rect) {
      let frame = try appKitRect(patch.rect, primaryHeight: primary)
      let panel = PatchPanel(
        contentRect: frame, styleMask: [.borderless, .nonactivatingPanel], backing: .buffered,
        defer: false)
      panel.title = "ELab calibration " + patch.id
      panel.hasShadow = false
      panel.isOpaque = true
      panel.alphaValue = 1
      panel.level = .floating
      panel.hidesOnDeactivate = false
      panel.ignoresMouseEvents = true
      panel.animationBehavior = .none
      panel.isRestorable = false
      panel.contentMinSize = NSSize(width: 1, height: 1)
      let view = PatchView(patch)
      panel.contentView = view
      panel.setFrame(frame, display: true)
      panels.append(panel)
      views.append(view)
      panel.orderFrontRegardless()
      view.needsDisplay = true
      view.displayIfNeeded()
    }
    initialized = true
  }
  func requireTarget(verifyBinary: Bool = false) throws {
    guard target.pid > 0, target.window > 0, target.executable.hasPrefix("/"),
      target.sha256.count == 64,
      let app = NSRunningApplication(processIdentifier: target.pid)
    else { throw ProbeError("Exact frozen owned target process unavailable") }
    let observedBounds = cgBounds(window: target.window, pid: target.pid)
    let observedPath = app.executableURL?.resolvingSymlinksInPath()
    guard !app.isTerminated, !app.isHidden, app.bundleIdentifier == "com.dshills.elab",
      observedPath == URL(fileURLWithPath: target.executable).resolvingSymlinksInPath(),
      observedBounds == target.bounds.rect
    else {
      throw ProbeError(
        "Exact target guard: terminated=\(app.isTerminated), hidden=\(app.isHidden), bundle=\(app.bundleIdentifier ?? "nil"), executable=\(observedPath?.path ?? "nil"), bounds=\(String(describing: observedBounds))"
      )
    }
    if verifyBinary {
      guard
        sha256Hex(try Data(contentsOf: URL(fileURLWithPath: target.executable))) == target.sha256
      else { throw ProbeError("Owned executable hash changed") }
    }
  }
  func close(_ error: String?) {
    guard !finished else { return }
    finished = true
    failure = error
    if failure == nil {
      do { try requireTarget(verifyBinary: true) } catch { failure = String(describing: error) }
    }
    timer?.invalidate()
    timer = nil
    let ids = panels.map { $0.windowNumber }
    for panel in panels {
      panel.orderOut(nil)
      panel.close()
    }
    do {
      try writeJSON(
        [
          "closed": true, "error": failure as Any? ?? NSNull(), "helper_pid": getpid(),
          "owned_window_ids": ids, "closed_mach_ticks": String(mach_absolute_time()),
          "capture_executed": false, "input_injected": false,
        ], to: output.appendingPathComponent("closed.json"))
    } catch {
      failure = failure ?? "Fixture close report failed: \(error)"
      FileHandle.standardError.write(Data("fixture close report failed\n".utf8))
    }
    // CLI fixture owns no user document. Finish after its own panels/report;
    // no controller signal or synthetic input is needed to wake AppKit's loop.
    Darwin.exit(failure == nil ? 0 : 1)
  }
  func tick() {
    guard !finished else { return }
    do {
      try requireTarget()
      for (panel, view) in zip(panels, views) {
        guard let window = UInt32(exactly: panel.windowNumber), window > 0,
          cgBounds(window: window, pid: getpid()) == view.patch.rect
        else {
          throw ProbeError("Owned patch geometry differs")
        }
      }
      ticks += 1
      // Static-fixture preparation only: two fixed 50 ms owner-loop checks.
      // This is neither a timing measurement nor a selected favorable frame.
      if !ready && ticks >= 2 && views.allSatisfy({ $0.drawCount > 0 }) {
        let patches = zip(panels, views).map { panel, view in
          [
            "id": view.patch.id, "window_id": panel.windowNumber,
            "global_points": rectangle(view.patch.rect), "pattern_bits": view.patch.bits,
            "cell_points": view.patch.cell, "draw_count": view.drawCount,
            "alpha": panel.alphaValue, "level": panel.level.rawValue,
            "last_draw_finished_mach_ticks": String(view.lastDrawFinished),
          ] as [String: Any]
        }
        try writeJSON(
          [
            "helper_pid": getpid(), "ready_mach_ticks": String(mach_absolute_time()),
            "deadline_mach_ticks": String(deadline),
            "target_pid": target.pid, "target_window": target.window,
            "target_global_points": rectangle(target.bounds.rect), "patches": patches,
            "capture_executed": false, "input_injected": false,
          ], to: output.appendingPathComponent("ready.json"))
        ready = true
      }
      let request = output.appendingPathComponent("stop.request")
      if FileManager.default.fileExists(atPath: request.path) {
        let info = try FileManager.default.attributesOfItem(atPath: request.path)
        guard info[.type] as? FileAttributeType == .typeRegular,
          let bytes = info[.size] as? NSNumber, bytes.intValue <= 512,
          let stop = try JSONSerialization.jsonObject(with: Data(contentsOf: request))
            as? [String: String],
          let value = stop["not_before_mach_ticks"], let notBefore = UInt64(value),
          notBefore <= deadline
        else {
          throw ProbeError("Invalid owned fixture stop request")
        }
        if mach_absolute_time() >= notBefore { close(nil) }
      }
      if mach_absolute_time() >= deadline { close(nil) }
    } catch { close(String(describing: error)) }
  }
  func run() {
    timer = Timer.scheduledTimer(withTimeInterval: 0.05, repeats: true) { [weak self] _ in
      MainActor.assumeIsolated { self?.tick() }
    }
  }
}

@main struct Main {
  @MainActor static func main() {
    do {
      let args = Array(CommandLine.arguments.dropFirst())
      if args == ["--self-test"] {
        try fixtureOfflineChecks()
        return
      }
      if args == ["--help"] || args.isEmpty {
        print(
          "Static owned calibration patches: --show --target /owned-target.json --output /new/temp/path --seconds 1...30. --self-test creates no UI. No recording or input injection."
        )
        return
      }
      guard args.count == 7, args[0] == "--show", args[1] == "--target", args[3] == "--output",
        args[5] == "--seconds",
        args[2].hasPrefix("/"), args[4].hasPrefix("/"), let seconds = Double(args[6]),
        seconds.isFinite, seconds >= 1, seconds <= 30
      else {
        throw ProbeError("Explicit target, fresh output and 1–30 second fixture lifetime required")
      }
      let target = try JSONDecoder().decode(
        Target.self, from: Data(contentsOf: URL(fileURLWithPath: args[2])))
      _ = try calibrationPatches(target.bounds.rect)
      let output = URL(fileURLWithPath: args[4])
      guard
        permittedOutput(
          output,
          roots: [URL(fileURLWithPath: "/private/tmp"), FileManager.default.temporaryDirectory])
      else { throw ProbeError("Fixture output must be a private temporary directory") }
      let app = NSApplication.shared
      app.setActivationPolicy(.accessory)
      try createPrivateOutput(output)
      let fixture = try Fixture(target: target, output: output, seconds: seconds)
      fixture.run()
      app.run()
      if !fixture.finished { fixture.close("Fixture run loop ended before cleanup") }
      if fixture.failure != nil { exit(1) }
    } catch {
      FileHandle.standardError.write(Data("calibration-fixture: \(error)\n".utf8))
      exit(1)
    }
  }
}
