// Local qualification instrumentation. No capture occurs in --help/--self-test.
import AppKit
import CoreImage
import CoreMedia
import CryptoKit
import Darwin
import Foundation
import ImageIO
@preconcurrency import ScreenCaptureKit
import UniformTypeIdentifiers

struct ProbeError: Error, CustomStringConvertible {
    let description: String
    init(_ description: String) { self.description = description }
}

struct Options {
    var pid: Int32
    var window: UInt32
    var executable: URL
    var digest: String
    var output: URL
    var seconds: Double

    static func parse(_ arguments: [String]) throws -> Options {
        guard arguments.first == "--capture" else {
            throw ProbeError("Capture needs explicit --capture and prior human authorization.")
        }
        let keys = ["--pid", "--window", "--executable", "--sha256", "--output", "--seconds"]
        var fields: [String: String] = [:]
        var index = 1
        while index < arguments.count {
            let key = arguments[index]
            guard keys.contains(key), fields[key] == nil, index + 1 < arguments.count else {
                throw ProbeError("Unknown, repeated or incomplete argument: \(key)")
            }
            fields[key] = arguments[index + 1]
            index += 2
        }
        guard let pid = Int32(fields["--pid"] ?? ""), pid > 0,
              let window = UInt32(fields["--window"] ?? ""), window > 0,
              let path = fields["--executable"], path.hasPrefix("/"),
              let output = fields["--output"], output.hasPrefix("/"),
              let digest = fields["--sha256"], digest.count == 64,
              digest.allSatisfy({ "0123456789abcdef".contains($0) }),
              let seconds = Double(fields["--seconds"] ?? "2"), seconds.isFinite,
              seconds >= 0.1, seconds <= 5 else {
            throw ProbeError("Need positive PID/window, absolute paths, lowercase SHA-256 and 0.1–5 seconds.")
        }
        return Options(pid: pid, window: window, executable: URL(fileURLWithPath: path),
                       digest: digest, output: URL(fileURLWithPath: output), seconds: seconds)
    }
}

func validMarker(_ text: String) -> Bool {
    !text.isEmpty && text.utf8.count <= 32 && text.utf8.allSatisfy {
        (48...57).contains($0) || (65...90).contains($0) || (97...122).contains($0) || $0 == 45 || $0 == 95
    }
}

func upperBoundMilliseconds(marker: UInt64, displayed: UInt64, numerator: UInt32, denominator: UInt32) -> Double? {
    guard marker > 0, displayed >= marker, numerator > 0, denominator > 0 else { return nil }
    return Double(displayed - marker) * Double(numerator) / Double(denominator) / 1_000_000
}

func sha256Hex(_ data: Data) -> String {
    SHA256.hash(data: data).map { String(format: "%02x", $0) }.joined()
}

func writeJSON(_ value: [String: Any], to url: URL) throws {
    let data = try JSONSerialization.data(withJSONObject: value, options: [.prettyPrinted, .sortedKeys])
    try data.write(to: url, options: .withoutOverwriting)
    try FileManager.default.setAttributes([.posixPermissions: 0o600], ofItemAtPath: url.path)
}

func createPrivateOutput(_ url: URL) throws {
    guard url.path.withCString({ mkdir($0, 0o700) }) == 0 else {
        throw ProbeError("Cannot create a new private output directory (errno \(errno)); existing data is never reused")
    }
}

// Foundation can abbreviate an existing /private/tmp to /tmp while keeping a
// missing child under /private/tmp. Resolve the longest existing POSIX ancestor
// instead, so aliases and symlinked parents have one physical prefix.
func physicalComponents(_ url: URL) -> [String]? {
    var path = url.path
    guard path.hasPrefix("/"), !path.split(separator: "/").contains(where: { $0 == "." || $0 == ".." }) else { return nil }
    var tail: [String] = []
    while true {
        if let resolved = realpath(path, nil) {
            defer { free(resolved) }
            return String(cString: resolved).split(separator: "/").map(String.init) + tail.reversed()
        }
        var attributes = stat()
        guard lstat(path, &attributes) != 0, errno == ENOENT else { return nil }
        let name = (path as NSString).lastPathComponent
        guard !name.isEmpty, name != "/" else { return nil }
        tail.append(name)
        path = (path as NSString).deletingLastPathComponent
    }
}

func permittedOutput(_ url: URL, roots: [URL]) -> Bool {
    guard let path = physicalComponents(url) else { return false }
    guard !path.contains(where: { $0.lowercased().hasSuffix(".app") }) else { return false }
    return roots.contains { root in
        guard let prefix = physicalComponents(root), !prefix.isEmpty else { return false }
        return path.count > prefix.count && Array(path.prefix(prefix.count)) == prefix
    }
}

// All mutable recorder state is confined to queue, including delegate errors,
// marker acknowledgments and finalization. Framework callbacks enqueue onto it.
final class Collector: NSObject, SCStreamOutput, SCStreamDelegate, @unchecked Sendable {
    let queue = DispatchQueue(label: "elab.frame-probe.samples")
    private let output: URL
    private let journal: FileHandle
    private let context = CIContext(options: [.cacheIntermediates: false])
    private let window: UInt32
    private var timebase = mach_timebase_info_data_t()
    private var timer: DispatchSourceTimer?
    private var failure: String?
    private var frameCount = 0
    private var completeCount = 0
    private var storedBytes = 0
    private var images: [String: String] = [:]
    private var markerIDs = Set<String>()
    private var latestMarker: UInt64 = 0
    private var finalReport: [String: Any]?

    init(output: URL, window: UInt32) throws {
        self.output = output
        self.window = window
        let file = output.appendingPathComponent("frames.jsonl")
        let descriptor = file.path.withCString {
            Darwin.open($0, O_WRONLY | O_CREAT | O_EXCL | O_CLOEXEC, mode_t(0o600))
        }
        guard descriptor >= 0 else {
            throw ProbeError("Cannot create a new frame journal (errno \(errno)); existing evidence is never overwritten")
        }
        journal = FileHandle(fileDescriptor: descriptor, closeOnDealloc: true)
        super.init()
        guard mach_timebase_info(&timebase) == KERN_SUCCESS, timebase.denom != 0 else {
            throw ProbeError("Cannot establish Mach timebase")
        }
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

    private func append(_ row: [String: Any]) throws {
        var data = try JSONSerialization.data(withJSONObject: row, options: [.sortedKeys])
        data.append(10)
        try journal.write(contentsOf: data)
    }

    private func readMarker() {
        guard failure == nil, finalReport == nil else { return }
        let request = output.appendingPathComponent("marker.request")
        guard let attributes = try? FileManager.default.attributesOfItem(atPath: request.path),
              attributes[.type] as? FileAttributeType == .typeRegular,
              let size = attributes[.size] as? NSNumber, size.intValue <= 32,
              let data = try? Data(contentsOf: request), let id = String(data: data, encoding: .utf8),
              validMarker(id), !markerIDs.contains(id) else { return }
        do {
            let ticks = mach_absolute_time()
            let row: [String: Any] = ["event": "marker", "id": id, "mach_ticks": String(ticks),
                                      "frame_count_before": frameCount]
            try append(row)
            try journal.synchronize()
            try writeJSON(row, to: output.appendingPathComponent("marker-\(id).json"))
            markerIDs.insert(id)
            latestMarker = ticks
        } catch { failure = "Marker journal/acknowledgment failed: \(error)" }
    }

    // This is a receipt-time metadata check, not proof that the window was
    // unobscured at the earlier display timestamp. Never use it as a pass gate.
    private func targetInOnscreenStack() -> Bool {
        guard let rows = CGWindowListCopyWindowInfo([.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID)
                as? [[String: Any]] else { return false }
        return rows.contains { ($0[kCGWindowNumber as String] as? NSNumber)?.uint32Value == window }
    }

    func stream(_ stream: SCStream, didOutputSampleBuffer sampleBuffer: CMSampleBuffer, of type: SCStreamOutputType) {
        guard type == .screen, failure == nil, finalReport == nil else { return }
        do {
            guard frameCount < 300 else { throw ProbeError("Frame budget exceeded; capture is incomplete") }
            frameCount += 1
            let received = mach_absolute_time()
            let attachment = (CMSampleBufferGetSampleAttachmentsArray(sampleBuffer, createIfNecessary: false)
                              as? [[SCStreamFrameInfo: Any]])?.first
            let status = (attachment?[.status] as? NSNumber)?.intValue
            let displayed = (attachment?[.displayTime] as? NSNumber)?.uint64Value
            var row: [String: Any] = ["event": "frame", "index": frameCount,
                "received_mach_ticks": String(received), "status": status as Any? ?? NSNull(),
                "display_mach_ticks": displayed.map(String.init) as Any? ?? NSNull(),
                "target_in_onscreen_stack_at_receipt": targetInOnscreenStack(),
                "eligible_for_latency": false]
            if let scale = attachment?[.scaleFactor] as? NSNumber { row["scale_factor"] = scale }
            if let rect = attachment?[.contentRect] { row["content_rect_description"] = String(describing: rect) }
            let presentation = CMTimeGetSeconds(CMSampleBufferGetPresentationTimeStamp(sampleBuffer))
            if presentation.isFinite { row["presentation_seconds"] = presentation }
            if let displayed, let upper = upperBoundMilliseconds(marker: latestMarker, displayed: displayed,
                                                                 numerator: timebase.numer, denominator: timebase.denom) {
                row["since_acknowledged_marker_ms"] = upper
            }
            if status == SCFrameStatus.complete.rawValue, let displayed, displayed > 0,
               let buffer = CMSampleBufferGetImageBuffer(sampleBuffer) {
                completeCount += 1
                let image = CIImage(cvPixelBuffer: buffer)
                guard let cg = context.createCGImage(image, from: image.extent) else { throw ProbeError("Cannot encode frame") }
                let data = NSMutableData()
                guard let writer = CGImageDestinationCreateWithData(data, UTType.png.identifier as CFString, 1, nil) else {
                    throw ProbeError("Cannot create PNG encoder")
                }
                CGImageDestinationAddImage(writer, cg, nil)
                guard CGImageDestinationFinalize(writer) else { throw ProbeError("Cannot finish PNG") }
                let bytes = data as Data
                let digest = sha256Hex(bytes)
                let name = "frame-\(digest).png"
                if images[digest] == nil {
                    guard storedBytes + bytes.count <= 64 * 1024 * 1024 else {
                        throw ProbeError("64 MiB image budget exceeded; capture is incomplete")
                    }
                    try bytes.write(to: output.appendingPathComponent(name), options: .withoutOverwriting)
                    try FileManager.default.setAttributes([.posixPermissions: 0o600],
                                                           ofItemAtPath: output.appendingPathComponent(name).path)
                    storedBytes += bytes.count
                    images[digest] = name
                }
                row["png"] = name
                row["png_sha256"] = digest
            }
            try append(row)
        } catch { failure = String(describing: error) }
    }

    func stream(_ stream: SCStream, didStopWithError error: Error) {
        let message = String(describing: error)
        queue.async { self.failure = "Stream stopped: \(message)" }
    }

    func finish() -> [String: Any] {
        queue.sync {
            if let finalReport { return finalReport }
            timer?.cancel()
            timer = nil
            do { try journal.synchronize(); try journal.close() } catch { failure = String(describing: error) }
            let report: [String: Any] = ["collector_error": failure as Any? ?? NSNull(), "frames_received": frameCount,
                    "complete_frames": completeCount, "unique_images": images.count,
                    "image_bytes": storedBytes, "marker_count": markerIDs.count,
                    "timebase_numerator": timebase.numer, "timebase_denominator": timebase.denom,
                    "eligible_latency_observations": 0, "latency_gate_passed": NSNull()]
            finalReport = report
            return report
        }
    }
}

func selfTest() throws {
    let valid = ["--capture", "--pid", "7", "--window", "9", "--executable", "/tmp/ELab",
                 "--sha256", String(repeating: "a", count: 64), "--output", "/tmp/fresh", "--seconds", "2"]
    _ = try Options.parse(valid)
    for invalid in [Array(valid.dropFirst()), valid + ["--pid", "8"], valid + ["--unknown", "x"],
                    ["--capture", "--pid", "0"], Array(valid.dropLast()) + ["nan"],
                    Array(valid.dropLast()) + ["6"]] {
        do { _ = try Options.parse(invalid); throw ProbeError("Invalid capture arguments accepted") }
        catch let error as ProbeError where error.description == "Invalid capture arguments accepted" { throw error }
        catch { }
    }
    guard validMarker("trial_01"), !validMarker("../escape"), !validMarker(""),
          !validMarker(String(repeating: "x", count: 33)), !validMarker("line\n"),
          upperBoundMilliseconds(marker: 0, displayed: 100, numerator: 1, denominator: 1) == nil,
          upperBoundMilliseconds(marker: 100, displayed: 99, numerator: 1, denominator: 1) == nil,
          upperBoundMilliseconds(marker: 10, displayed: 20, numerator: 1, denominator: 0) == nil,
          upperBoundMilliseconds(marker: 10, displayed: 24_000_010, numerator: 125, denominator: 3) == 1000 else {
        throw ProbeError("Marker or timebase checks failed")
    }
    let allowed = [URL(fileURLWithPath: "/private/tmp")]
    guard permittedOutput(URL(fileURLWithPath: "/private/tmp/clip"), roots: allowed),
          !permittedOutput(URL(fileURLWithPath: "/private/tmp-elsewhere/clip"), roots: allowed),
          !permittedOutput(URL(fileURLWithPath: "/private/tmp/Test.app/clip"), roots: allowed),
          !permittedOutput(URL(fileURLWithPath: "/Applications/ELab.app/clip"), roots: allowed) else {
        throw ProbeError("Temporary output/app-bundle exclusion checks failed")
    }
    let temporary = FileManager.default.temporaryDirectory.appendingPathComponent("elab-frame-probe-unit-\(UUID().uuidString)")
    try createPrivateOutput(temporary)
    defer { try? FileManager.default.removeItem(at: temporary) }
    var reused = false
    do { try createPrivateOutput(temporary) } catch { reused = true }
    guard reused else { throw ProbeError("Existing output directory was accepted") }
    let pretendApp = temporary.appendingPathComponent("Private.app")
    try createPrivateOutput(pretendApp)
    let alias = temporary.appendingPathComponent("alias")
    try FileManager.default.createSymbolicLink(at: alias, withDestinationURL: pretendApp)
    guard !permittedOutput(alias.appendingPathComponent("clip"), roots: [temporary]) else {
        throw ProbeError("Symlinked application output was accepted")
    }
    let ack = temporary.appendingPathComponent("marker-trial_01.json")
    try writeJSON(["id": "trial_01", "mach_ticks": "18446744073709551615"], to: ack)
    let original = try Data(contentsOf: ack)
    var refused = false
    do { try writeJSON(["id": "replacement"], to: ack) } catch { refused = true }
    guard refused, try Data(contentsOf: ack) == original,
          let decoded = try JSONSerialization.jsonObject(with: original) as? [String: String],
          decoded["mach_ticks"] == String(UInt64.max) else { throw ProbeError("Exclusive acknowledgment or tick precision check failed") }
    let journalCases = try journalFaultChecks()
    let result: [String: Any] = ["self_tests_passed": true, "capture_executed": false,
                                "eligible_latency_observations": 0, "journal_fault_cases": journalCases]
    print(String(decoding: try JSONSerialization.data(withJSONObject: result, options: [.sortedKeys]), as: UTF8.self))
}

@main
struct Main {
    @MainActor static func main() async {
        var ownedOutput: URL?
        var collector: Collector?
        var stream: SCStream?
        var started = false
        do {
            let arguments = Array(CommandLine.arguments.dropFirst())
            if arguments == ["--self-test"] { try selfTest(); return }
            if arguments == ["--preflight"] {
                // Read existing access only; never request a privacy grant or
                // enumerate windows as part of this readiness check.
                let access = CGPreflightScreenCaptureAccess()
                print("{\"screen_access_previously_granted\":\(access),\"permission_requested\":false,\"capture_executed\":false}")
                return
            }
            if arguments.first == "--target-windows" {
                guard arguments.count == 7, arguments[1] == "--pid",
                      arguments[3] == "--executable", arguments[5] == "--sha256",
                      let pid = Int32(arguments[2]), pid > 0,
                      arguments[4].hasPrefix("/"), arguments[6].count == 64,
                      arguments[6].allSatisfy({ "0123456789abcdef".contains($0) }) else {
                    throw ProbeError("Use --target-windows --pid PID --executable /path/ELab --sha256 HASH")
                }
                let executable = URL(fileURLWithPath: arguments[4])
                guard let application = NSRunningApplication(processIdentifier: pid), !application.isTerminated,
                      application.bundleIdentifier == "com.dshills.elab",
                      application.executableURL?.resolvingSymlinksInPath() == executable.resolvingSymlinksInPath(),
                      sha256Hex(try Data(contentsOf: executable)) == arguments[6] else {
                    throw ProbeError("Exact target process/executable identity not established")
                }
                guard CGPreflightScreenCaptureAccess() else {
                    throw ProbeError("Screen Recording access unavailable; no permission request was made")
                }
                let content = try await SCShareableContent.excludingDesktopWindows(true, onScreenWindowsOnly: true)
                let windows = content.windows.filter { $0.owningApplication?.processID == pid && $0.isOnScreen }
                let rows = windows.map { ["window_id": $0.windowID, "owner_pid": UInt32(pid)] }
                let result: [String: Any] = ["target_windows": rows, "capture_executed": false,
                                             "permission_requested": false]
                print(String(decoding: try JSONSerialization.data(withJSONObject: result, options: [.sortedKeys]), as: UTF8.self))
                return
            }
            if arguments.isEmpty || arguments == ["--help"] {
                print("frame-probe --self-test | --preflight | --capture --pid PID --window ID --executable /path/ELab --sha256 HASH --output /new/path [--seconds 0.1...5]")
                print("frame-probe --target-windows --pid PID --executable /path/ELab --sha256 HASH (owned-window IDs only)")
                print("Capture requires prior human authorization; this tool never requests OS permission or operates input. No latency decision is made.")
                return
            }
            let options = try Options.parse(arguments)
            guard permittedOutput(options.output, roots: [URL(fileURLWithPath: "/private/tmp"), FileManager.default.temporaryDirectory]) else {
                throw ProbeError("Output must be a new temporary subdirectory outside every app bundle")
            }
            guard CGPreflightScreenCaptureAccess() else {
                throw ProbeError("Screen Recording access unavailable. No permission request or settings change was made.")
            }
            guard let application = NSRunningApplication(processIdentifier: options.pid), !application.isTerminated,
                  application.bundleIdentifier == "com.dshills.elab",
                  application.executableURL?.resolvingSymlinksInPath() == options.executable.resolvingSymlinksInPath(),
                  sha256Hex(try Data(contentsOf: options.executable)) == options.digest else {
                throw ProbeError("Exact target process/executable identity not established")
            }
            let content = try await SCShareableContent.excludingDesktopWindows(true, onScreenWindowsOnly: true)
            guard let window = content.windows.first(where: { $0.windowID == options.window && $0.owningApplication?.processID == options.pid }),
                  window.isOnScreen else { throw ProbeError("Exact target window unavailable") }
            // The window filter consults the Window Server display connection.
            // A command-line process must initialize AppKit before creating it.
            _ = NSApplication.shared
            let filter = SCContentFilter(desktopIndependentWindow: window)
            let configuration = SCStreamConfiguration()
            configuration.width = Int(ceil(filter.contentRect.width * CGFloat(filter.pointPixelScale)))
            configuration.height = Int(ceil(filter.contentRect.height * CGFloat(filter.pointPixelScale)))
            guard configuration.width > 0, configuration.width <= 4096,
                  configuration.height > 0, configuration.height <= 4096 else { throw ProbeError("Window exceeds native-pixel bounds") }
            configuration.minimumFrameInterval = CMTime(value: 1, timescale: 60)
            configuration.queueDepth = 3
            configuration.showsCursor = false
            configuration.capturesAudio = false
            if #available(macOS 15.0, *) { configuration.captureMicrophone = false }
            if #available(macOS 14.2, *) { configuration.includeChildWindows = false }
            configuration.pixelFormat = kCVPixelFormatType_32BGRA
            configuration.ignoreShadowsSingleWindow = true
            try createPrivateOutput(options.output)
            ownedOutput = options.output
            let observer = try Collector(output: options.output, window: options.window)
            collector = observer
            let capture = SCStream(filter: filter, configuration: configuration, delegate: observer)
            stream = capture
            try capture.addStreamOutput(observer, type: .screen, sampleHandlerQueue: observer.queue)
            try await capture.startCapture()
            started = true
            observer.startMarkers()
            try await Task.sleep(for: .seconds(options.seconds))
            try await capture.stopCapture()
            started = false
            var report = observer.finish()
            report["capture_executed"] = true
            report["pid"] = options.pid
            report["window_id"] = options.window
            report["executable_sha256"] = options.digest
            report["filter"] = "desktopIndependentWindow"
            report["unobscured_display_visibility_validated"] = false
            let detectedError = (report["collector_error"] as? String)
                ?? ((report["complete_frames"] as? Int ?? 0) > 0 ? nil : "No complete timestamped image was received")
            report["command_error"] = detectedError as Any? ?? NSNull()
            try writeJSON(report, to: options.output.appendingPathComponent("report.json"))
            guard detectedError == nil else { throw ProbeError("Capture incomplete; inspect local report") }
            print("{\"capture_executed\":true,\"eligible_latency_observations\":0,\"latency_gate_passed\":null}")
        } catch {
            let failure = String(describing: error)
            var stopFailure: String?
            if started, let stream {
                do { try await stream.stopCapture() } catch { stopFailure = String(describing: error) }
            }
            var report = collector?.finish() ?? ["eligible_latency_observations": 0, "latency_gate_passed": NSNull()]
            report["command_error"] = failure
            report["cleanup_error"] = stopFailure as Any? ?? NSNull()
            report["capture_started"] = started
            if let ownedOutput, !FileManager.default.fileExists(atPath: ownedOutput.appendingPathComponent("report.json").path) {
                try? writeJSON(report, to: ownedOutput.appendingPathComponent("report.json"))
            }
            FileHandle.standardError.write(Data(("frame-probe: \(failure)\n").utf8))
            exit(1)
        }
    }
}
