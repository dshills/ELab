// Offline filesystem/marker checks. No window, sample buffer, privacy API,
// stream or screen capture is created or queried here.
import Foundation

func journalFaultChecks() throws -> [String] {
    let temporary = FileManager.default.temporaryDirectory.appendingPathComponent("elab-frame-journal-checks-\(UUID().uuidString)")
    try createPrivateOutput(temporary)
    defer { try? FileManager.default.removeItem(at: temporary) }
    var cases: [String] = []
    let original = Data("existing calibration evidence\n".utf8)

    func folder(_ name: String) throws -> URL {
        let url = temporary.appendingPathComponent(name)
        try createPrivateOutput(url)
        return url
    }
    func expectRefusal(_ directory: URL) throws {
        var rejected = false
        do {
            let unexpected = try Collector(output: directory, window: 0)
            _ = unexpected.finish()
        } catch { rejected = true }
        guard rejected else { throw ProbeError("Existing frame-journal destination was admitted: \(directory.lastPathComponent)") }
    }
    func waitForFile(_ url: URL) throws {
        let deadline = DispatchTime.now().uptimeNanoseconds + 1_000_000_000
        while !FileManager.default.fileExists(atPath: url.path) {
            guard DispatchTime.now().uptimeNanoseconds < deadline else { throw ProbeError("Offline marker check timed out") }
            Thread.sleep(forTimeInterval: 0.001)
        }
    }
    func require(_ condition: Bool, _ message: String) throws {
        guard condition else { throw ProbeError(message) }
    }

    do {
        let directory = try folder("existing-file")
        let path = directory.appendingPathComponent("frames.jsonl")
        try original.write(to: path, options: .withoutOverwriting)
        let before = try FileManager.default.attributesOfItem(atPath: path.path)[.posixPermissions] as? NSNumber
        try expectRefusal(directory)
        try require(try Data(contentsOf: path) == original, "Existing journal bytes changed")
        let after = try FileManager.default.attributesOfItem(atPath: path.path)[.posixPermissions] as? NSNumber
        try require(before == after, "Existing journal permissions changed")
        cases.append("existing-journal-refused-with-bytes-and-mode-preserved")
    }
    do {
        let directory = try folder("existing-directory")
        let path = directory.appendingPathComponent("frames.jsonl")
        try createPrivateOutput(path)
        try expectRefusal(directory)
        try require(try FileManager.default.contentsOfDirectory(atPath: path.path).isEmpty, "Directory journal destination changed")
        cases.append("directory-journal-destination-refused")
    }
    for dangling in [false, true] {
        let directory = try folder(dangling ? "dangling-link" : "existing-link")
        let target = temporary.appendingPathComponent(dangling ? "missing-canary" : "canary")
        if !dangling { try original.write(to: target, options: .withoutOverwriting) }
        let path = directory.appendingPathComponent("frames.jsonl")
        try FileManager.default.createSymbolicLink(at: path, withDestinationURL: target)
        try expectRefusal(directory)
        try require(try FileManager.default.destinationOfSymbolicLink(atPath: path.path) == target.path, "Journal symbolic link changed")
        if dangling {
            try require(!FileManager.default.fileExists(atPath: target.path), "Dangling journal target was created")
        } else {
            try require(try Data(contentsOf: target) == original, "Journal symlink target changed")
        }
        cases.append(dangling ? "dangling-journal-link-refused" : "journal-link-refused-with-target-preserved")
    }
    do {
        let directory = try folder("normal-close")
        let collector = try Collector(output: directory, window: 0)
        let path = directory.appendingPathComponent("frames.jsonl")
        let mode = try FileManager.default.attributesOfItem(atPath: path.path)[.posixPermissions] as? NSNumber
        try require(mode?.intValue == 0o600, "New journal permissions differ from 0600")
        let report = collector.finish()
        let repeated = collector.finish()
        try require(report["collector_error"] is NSNull && report["frames_received"] as? Int == 0, "Empty offline journal reports failure/frames")
        try require(NSDictionary(dictionary: report).isEqual(to: repeated), "Repeated finalization changes the report")
        try require(try Data(contentsOf: path).isEmpty, "Empty journal contains invented evidence")
        cases.append("private-new-journal-and-idempotent-finalization")
    }
    do {
        let directory = try folder("active-collision")
        let collector = try Collector(output: directory, window: 0)
        defer { _ = collector.finish() }
        try Data("active_marker".utf8).write(to: directory.appendingPathComponent("marker.request"), options: .withoutOverwriting)
        collector.startMarkers()
        try waitForFile(directory.appendingPathComponent("marker-active_marker.json"))
        // Wait for the marker callback, but leave this collector and its timer
        // active while the second constructor tries to open the same journal.
        collector.queue.sync { }
        let path = directory.appendingPathComponent("frames.jsonl")
        let before = try Data(contentsOf: path)
        try require(!before.isEmpty, "Offline marker evidence was not retained")
        try expectRefusal(directory)
        try require(try Data(contentsOf: path) == before, "Second collector truncated completed marker evidence")
        let report = collector.finish()
        try require(report["marker_count"] as? Int == 1, "Active collector lost its marker count")
        cases.append("second-collector-refused-with-marker-evidence-preserved")
    }
    do {
        let directory = try folder("acknowledgment-collision")
        let collector = try Collector(output: directory, window: 0)
        defer { _ = collector.finish() }
        let acknowledgment = directory.appendingPathComponent("marker-occupied.json")
        try original.write(to: acknowledgment, options: .withoutOverwriting)
        let journal = directory.appendingPathComponent("frames.jsonl")
        try Data("occupied".utf8).write(to: directory.appendingPathComponent("marker.request"), options: .withoutOverwriting)
        collector.startMarkers()
        let deadline = DispatchTime.now().uptimeNanoseconds + 1_000_000_000
        while try Data(contentsOf: journal).isEmpty {
            guard DispatchTime.now().uptimeNanoseconds < deadline else { throw ProbeError("Offline marker refusal check timed out") }
            Thread.sleep(forTimeInterval: 0.001)
        }
        let report = collector.finish()
        try require(try Data(contentsOf: acknowledgment) == original, "Existing marker acknowledgment changed")
        try require((report["collector_error"] as? String)?.contains("Marker journal/acknowledgment failed") == true, "Marker write failure was lost")
        try require(report["marker_count"] as? Int == 0 && report["eligible_latency_observations"] as? Int == 0, "Failed acknowledgment was counted as accepted")
        cases.append("acknowledgment-write-failure-retained-with-prior-file-preserved")
    }
    return cases
}
