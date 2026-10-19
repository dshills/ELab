// Qualification only: CoreMedia/Mach clock consistency. No AppKit,
// ScreenCaptureKit, window, privacy, image capture or input API is used.
import CoreMedia
import Darwin
import Foundation

struct ClockCheckError: Error, CustomStringConvertible {
    let description: String
}

@main
struct Main {
    static func main() {
        do {
            let args = Array(CommandLine.arguments.dropFirst())
            if args.isEmpty || args == ["--help"] {
                print("frame-probe-clock --check (host/Mach clocks only; no capture or input)")
                return
            }
            guard args == ["--check"] else { throw ClockCheckError(description: "Only --check or --help is accepted") }
            var timebase = mach_timebase_info_data_t()
            guard mach_timebase_info(&timebase) == KERN_SUCCESS, timebase.numer > 0, timebase.denom > 0 else {
                throw ClockCheckError(description: "Mach timebase unavailable")
            }
            let clock = CMClockGetHostTimeClock()
            var observations: [[String: Any]] = []
            var unbracketed = 0
            for _ in 0..<200 {
                let before = mach_absolute_time()
                let hostTime = CMClockGetTime(clock)
                let ticks = CMClockConvertHostTimeToSystemUnits(hostTime)
                let after = mach_absolute_time()
                // Compare in the host clock's own rational representation.
                // Converting nanoseconds back to 125/3 ns ticks can floor a
                // second time and move a valid sample one tick backwards.
                let beforeHost = CMClockMakeHostTimeFromSystemUnits(before)
                let afterHost = CMClockMakeHostTimeFromSystemUnits(after)
                let contained = CMTimeCompare(beforeHost, hostTime) <= 0 && CMTimeCompare(hostTime, afterHost) <= 0
                if !contained { unbracketed += 1 }
                observations.append(["before_mach_ticks": String(before), "host_mach_ticks": String(ticks),
                                     "after_mach_ticks": String(after), "host_seconds": CMTimeGetSeconds(hostTime),
                                     "host_value": String(hostTime.value), "host_timescale": hostTime.timescale,
                                     "before_host_value": String(beforeHost.value), "before_host_timescale": beforeHost.timescale,
                                     "after_host_value": String(afterHost.value), "after_host_timescale": afterHost.timescale,
                                     "inside_host_time_bracket": contained,
                                     "roundtrip_ticks_inside_mach_bracket": before <= ticks && ticks <= after])
            }
            var conversions: [[String: Any]] = []
            for ticks: UInt64 in [0, 1, 123_456_789, mach_absolute_time()] {
                let host = CMClockMakeHostTimeFromSystemUnits(ticks)
                let restored = CMClockConvertHostTimeToSystemUnits(host)
                let difference = restored >= ticks ? restored - ticks : ticks - restored
                let seconds = CMTimeGetSeconds(host)
                let ratioSeconds = Double(ticks) * Double(timebase.numer) / Double(timebase.denom) / 1e9
                let delta = abs(seconds - ratioSeconds)
                guard difference <= 1, seconds.isFinite, delta < 1e-8 else {
                    throw ClockCheckError(description: "Host/Mach conversion differs beyond one tick or 10 ns")
                }
                conversions.append(["mach_ticks": String(ticks), "restored_mach_ticks": String(restored),
                                    "host_seconds": seconds, "timebase_seconds": ratioSeconds,
                                    "absolute_difference_seconds": delta])
            }
            let report: [String: Any] = ["passed": unbracketed == 0,
                "scope": "CoreMedia host-clock/Mach consistency on this execution only",
                "timebase_numerator": timebase.numer, "timebase_denominator": timebase.denom,
                "observations": observations, "unbracketed_observations": unbracketed, "conversions": conversions,
                "capture_executed": false, "input_operated": false,
                "physical_display_timing_calibrated": false, "pixel_mapping_calibrated": false,
                "display_visibility_validated": false, "eligible_latency_observations": 0,
                "latency_gate_passed": NSNull()]
            var data = try JSONSerialization.data(withJSONObject: report, options: [.prettyPrinted, .sortedKeys])
            data.append(10)
            try FileHandle.standardOutput.write(contentsOf: data)
            guard unbracketed == 0 else {
                throw ClockCheckError(description: "CoreMedia host time outside \(unbracketed) of 200 surrounding host-time brackets")
            }
        } catch {
            FileHandle.standardError.write(Data("frame-probe-clock: \(error)\n".utf8))
            exit(1)
        }
    }
}
