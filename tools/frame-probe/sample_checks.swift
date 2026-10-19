// Offline synthetic CoreMedia samples and local storage faults only. No
// SCStream, window enumeration, privacy API, recorder capture or input is used.
import CoreMedia
import CoreVideo
import Foundation
import ImageIO
@preconcurrency import ScreenCaptureKit

private func syntheticSample(status: Int? = SCFrameStatus.complete.rawValue,
                             displayed: UInt64? = 123, image: Bool = true,
                             color: UInt8 = 30) throws -> CMSampleBuffer {
    var pixel: CVPixelBuffer?
    guard CVPixelBufferCreate(kCFAllocatorDefault, 4, 3, kCVPixelFormatType_32BGRA, nil, &pixel) == kCVReturnSuccess,
          let pixel else { throw ProbeError("Cannot allocate synthetic pixel buffer") }
    CVPixelBufferLockBaseAddress(pixel, [])
    if let base = CVPixelBufferGetBaseAddress(pixel) {
        memset(base, 0, CVPixelBufferGetDataSize(pixel))
        for y in 0..<3 {
            let row = base.advanced(by: y * CVPixelBufferGetBytesPerRow(pixel)).assumingMemoryBound(to: UInt8.self)
            for x in 0..<4 { row[4*x] = color; row[4*x+3] = 255 }
        }
    }
    CVPixelBufferUnlockBaseAddress(pixel, [])
    var format: CMVideoFormatDescription?
    guard CMVideoFormatDescriptionCreateForImageBuffer(allocator: kCFAllocatorDefault, imageBuffer: pixel,
                                                      formatDescriptionOut: &format) == noErr, let format else {
        throw ProbeError("Cannot describe synthetic pixel buffer")
    }
    var timing = CMSampleTimingInfo(duration: .invalid, presentationTimeStamp: CMTime(value: 3, timescale: 2),
                                   decodeTimeStamp: .invalid)
    var sample: CMSampleBuffer?
    let result = image ? CMSampleBufferCreateReadyWithImageBuffer(allocator: kCFAllocatorDefault,
        imageBuffer: pixel, formatDescription: format, sampleTiming: &timing, sampleBufferOut: &sample) :
        CMSampleBufferCreateReady(allocator: kCFAllocatorDefault, dataBuffer: nil, formatDescription: format,
            sampleCount: 1, sampleTimingEntryCount: 1, sampleTimingArray: &timing,
            sampleSizeEntryCount: 0, sampleSizeArray: nil, sampleBufferOut: &sample)
    guard result == noErr, let sample,
          let attachments = CMSampleBufferGetSampleAttachmentsArray(sample, createIfNecessary: true) else {
        throw ProbeError("Cannot construct synthetic sample (\(result))")
    }
    let row = unsafeBitCast(CFArrayGetValueAtIndex(attachments, 0), to: CFMutableDictionary.self)
    func put(_ key: SCStreamFrameInfo, _ value: NSNumber) {
        let text = key.rawValue as CFString
        CFDictionarySetValue(row, Unmanaged.passUnretained(text).toOpaque(), Unmanaged.passUnretained(value).toOpaque())
    }
    if let status { put(.status, NSNumber(value: status)) }
    if let displayed { put(.displayTime, NSNumber(value: displayed)) }
    return sample
}

func sampleFaultChecks() throws -> [String] {
    let temporary = FileManager.default.temporaryDirectory.appendingPathComponent("elab-frame-sample-checks-\(UUID().uuidString)")
    try createPrivateOutput(temporary)
    defer { try? FileManager.default.removeItem(at: temporary) }
    var cases: [String] = []
    var encodedFixture: Data?
    func folder(_ name: String) throws -> URL {
        let path = temporary.appendingPathComponent(name)
        try createPrivateOutput(path)
        return path
    }
    func require(_ condition: Bool, _ message: String) throws {
        guard condition else { throw ProbeError(message) }
    }
    func rows(_ directory: URL) throws -> [[String: Any]] {
        let data = try Data(contentsOf: directory.appendingPathComponent("frames.jsonl"))
        return try data.split(separator: 10).map {
            guard let row = try JSONSerialization.jsonObject(with: Data($0)) as? [String: Any] else {
                throw ProbeError("Malformed synthetic frame row")
            }
            return row
        }
    }
    func requireIneligible(_ report: [String: Any]) throws {
        try require(report["eligible_latency_observations"] as? Int == 0 && report["latency_gate_passed"] is NSNull,
                    "Synthetic data became eligible latency evidence")
    }
    do {
        let path = try folder("statuses")
        let collector = try Collector(output: path, window: 0)
        for status in [SCFrameStatus.idle, .blank, .suspended, .started, .stopped] {
            let sample = try syntheticSample(status: status.rawValue)
            collector.queue.sync { collector.recordSample(sample, targetOnscreen: { false }) }
        }
        let report = collector.finish()
        let journal = try rows(path)
        try require(report["frames_received"] as? Int == 5 && report["complete_frames"] as? Int == 0 &&
                    report["unique_images"] as? Int == 0, "Noncomplete statuses were lost or encoded")
        try require(journal.count == 5 && journal.allSatisfy { $0["png"] == nil && $0["eligible_for_latency"] as? Bool == false },
                    "Noncomplete frame rows were omitted or admitted")
        try requireIneligible(report)
        cases.append("idle-blank-suspended-started-stopped-journaled-without-images")
    }
    do {
        let path = try folder("missing-metadata")
        let collector = try Collector(output: path, window: 0)
        let samples = [try syntheticSample(status: nil), try syntheticSample(displayed: nil),
                       try syntheticSample(displayed: 0), try syntheticSample(image: false)]
        for sample in samples { collector.queue.sync { collector.recordSample(sample, targetOnscreen: { false }) } }
        let report = collector.finish()
        try require(report["frames_received"] as? Int == 4 && report["complete_frames"] as? Int == 0 &&
                    report["unique_images"] as? Int == 0 && (try rows(path)).count == 4,
                    "Missing metadata/image invented a complete image or dropped its row")
        try requireIneligible(report)
        cases.append("missing-status-display-time-zero-time-and-image-remain-incomplete")
    }
    do {
        let path = try folder("complete-dedup")
        let collector = try Collector(output: path, window: 0)
        let first = try syntheticSample()
        let changed = try syntheticSample(displayed: 456, color: 200)
        collector.queue.sync {
            collector.recordSample(first, targetOnscreen: { false })
            collector.recordSample(first, targetOnscreen: { true })
            collector.recordSample(changed, targetOnscreen: { true })
        }
        let report = collector.finish()
        let journal = try rows(path)
        try require(report["collector_error"] is NSNull && report["complete_frames"] as? Int == 3 &&
                    report["unique_images"] as? Int == 2 && journal.count == 3, "Complete image accounting failed: \(report), rows \(journal.count)")
        try require(journal[0]["png_sha256"] as? String == journal[1]["png_sha256"] as? String &&
                    journal[1]["png_sha256"] as? String != journal[2]["png_sha256"] as? String,
                    "PNG deduplication merged changed pixels or lost repeated frames")
        for row in journal {
            let name = row["png"] as! String
            let file = path.appendingPathComponent(name)
            let data = try Data(contentsOf: file)
            try require(sha256Hex(data) == row["png_sha256"] as? String, "Stored image digest differs")
            let mode = try FileManager.default.attributesOfItem(atPath: file.path)[.posixPermissions] as? NSNumber
            try require(mode?.intValue == 0o600, "Stored image permissions differ from 0600")
            guard let image = CGImageSourceCreateWithData(data as CFData, nil),
                  let decoded = CGImageSourceCreateImageAtIndex(image, 0, nil) else { throw ProbeError("Synthetic PNG cannot decode") }
            try require(decoded.width == 4 && decoded.height == 3, "Synthetic PNG dimensions changed")
            try require(row["eligible_for_latency"] as? Bool == false, "Complete synthetic frame became eligible")
        }
        encodedFixture = try Data(contentsOf: path.appendingPathComponent(journal[0]["png"] as! String))
        try require(journal[0]["target_in_onscreen_stack_at_receipt"] as? Bool == false &&
                    journal[1]["target_in_onscreen_stack_at_receipt"] as? Bool == true,
                    "Explicit synthetic receipt metadata was lost")
        try requireIneligible(report)
        let before = try Data(contentsOf: path.appendingPathComponent("frames.jsonl"))
        collector.queue.sync { collector.recordSample(changed, targetOnscreen: { true }) }
        collector.recordStreamFailure("late synthetic callback")
        let repeated = collector.finish()
        try require(NSDictionary(dictionary: report).isEqual(to: repeated) &&
                    (try Data(contentsOf: path.appendingPathComponent("frames.jsonl"))) == before,
                    "Late callback changed finalized evidence")
        cases.append("complete-images-decode-deduplicate-and-keep-each-ineligible-frame")
        cases.append("sample-and-stream-callbacks-after-finalization-do-not-change-evidence")
    }
    do {
        let path = try folder("image-collision")
        let collector = try Collector(output: path, window: 0)
        guard let encodedFixture else { throw ProbeError("Missing synthetic PNG fixture") }
        let destination = path.appendingPathComponent("frame-\(sha256Hex(encodedFixture)).png")
        let sentinel = Data("existing image evidence".utf8)
        try sentinel.write(to: destination, options: .withoutOverwriting)
        let sample = try syntheticSample()
        collector.queue.sync { collector.recordSample(sample, targetOnscreen: { false }) }
        let report = collector.finish()
        try require(report["collector_error"] is String && report["unique_images"] as? Int == 0 &&
                    (try Data(contentsOf: destination)) == sentinel && (try rows(path)).isEmpty,
                    "Image collision overwrote prior evidence or counted a refused image")
        try requireIneligible(report)
        cases.append("existing-image-refused-with-prior-bytes-preserved-and-no-success-row")
    }
    do {
        let path = try folder("frame-limit")
        let collector = try Collector(output: path, window: 0)
        let sample = try syntheticSample(status: SCFrameStatus.idle.rawValue)
        collector.queue.sync {
            for _ in 0..<301 { collector.recordSample(sample, targetOnscreen: { false }) }
        }
        collector.recordStreamFailure("secondary stream stop")
        let report = collector.finish()
        try require(report["frames_received"] as? Int == 300 && (try rows(path)).count == 300 &&
                    (report["collector_error"] as? String)?.contains("Frame budget exceeded") == true,
                    "Frame limit was bypassed or its first failure replaced")
        try requireIneligible(report)
        cases.append("301st-callback-refused-at-production-300-frame-limit-first-error-retained")
    }
    do {
        let path = try folder("image-limit")
        let collector = try Collector(output: path, window: 0)
        // Synthetic encoded-byte payload: tests the real 64 MiB storage
        // boundary without pretending these bytes are a captured/valid PNG.
        let payload = Data(repeating: 42, count: 64 * 1024 * 1024)
        let first = try collector.queue.sync { try collector.recordPNG(payload) }
        let duplicate = try collector.queue.sync { try collector.recordPNG(payload) }
        try require(first == duplicate, "At-limit duplicate consumed the image budget")
        let sample = try syntheticSample()
        collector.queue.sync { collector.recordSample(sample, targetOnscreen: { false }) }
        let report = collector.finish()
        try require(report["image_bytes"] as? Int == payload.count && report["unique_images"] as? Int == 1 &&
                    (report["collector_error"] as? String)?.contains("64 MiB image budget exceeded") == true,
                    "Production image budget was bypassed")
        try require(try Data(contentsOf: path.appendingPathComponent(first.0)) == payload && (try rows(path)).isEmpty,
                    "Budget failure changed prior bytes or invented an image row")
        try requireIneligible(report)
        cases.append("64-MiB-unique-byte-limit-keeps-duplicates-and-refuses-new-image")
    }
    do {
        let path = try folder("stream-stop")
        let collector = try Collector(output: path, window: 0)
        collector.recordStreamFailure("first synthetic stop")
        collector.recordStreamFailure("second synthetic stop")
        let sample = try syntheticSample()
        collector.queue.sync { collector.recordSample(sample, targetOnscreen: { false }) }
        let report = collector.finish()
        try require(report["collector_error"] as? String == "Stream stopped: first synthetic stop" &&
                    report["frames_received"] as? Int == 0 && (try rows(path)).isEmpty,
                    "Stream failure was overwritten or later samples admitted")
        try requireIneligible(report)
        cases.append("first-stream-error-stops-admission-and-survives-secondary-error")
    }
    return cases
}
