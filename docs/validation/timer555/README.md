# 555 timer validation

[Model, pins, commands and integration contract](../../timer555.md).

Independent fixed-parameter RC reductions check astable first/steady phases,
repeated output edges at 1 ms and 5 ms output steps, and one-shot width. The
reference includes the declared input/off leakage and discharge resistance.
Maximum observed astable event error is 0.086 µs at 1 ms sampling and 2.21 µs at
5 ms sampling in the pinned fixture. One-shot width is 0.516387580872 s versus
0.516387603546 s analytically. The event bracket is bounded separately from
integration error; these are equation checks, not physical timing qualification.

The new tests also cover latch/reset/trigger priority, terminal Jacobians and
power/current balance, external control voltage, independent timers, live wiper
and switch preservation in the latch hold band, failure/cancellation rollback,
and continuity of capacitor voltage and inductor current at an output edge.
Strict model/result/array imports, captures, save/reopen, recovery inspection,
exact eight-pin array discovery and rotated pin geometry are covered. An
optional control bypass is omitted from the coil test so the unchanged global
LC ceiling is governed by the timing capacitor; a small bypass with a coil can
require shorter output sampling under the existing work bound.

Native-control tests draw a circuit from blank and compare agent results. All
three examples share menu loading, stepping, inspector readings, probe actions
and unsaved-work protection. Rotated symbol tests include pin-text geometry and
body hit testing. Command source inventory remains 166 public commands, 404
sites and 134 literal routes; three native aliases target `example.load`.

The complete root race suite passes: workbench 100.779 s, conformance 119.693 s,
socket transport 1.378 s. Its initial run caught an outdated test construction
of the pre-timer limit envelope; the corrected test and final full run pass.
Historical frozen fixtures and their other numeric limits were not rewritten.
The full desktop offscreen/race suite passes (493.853 s). Later symbol/caption/
selection presentation changes pass focused native/agent and rotation checks
(23.793 s) and final electrical/storage/caption checks. Lint and root/desktop
vet pass. These test durations are not visible-latency measurements.

The public stdio replay passes discovery, all three examples, independent
switching-time references, retained left/right edge frames, live wiper changes,
kept-record save/reopen and SVG/PNG/PDF exports. The preserved /205 release archive
is not replaced by this development candidate.

Native /235 demonstrated capacitor/output/LED switching, one-shot high and low
states, and the exact wiper control changing 50% to 80% at accepted time
21.850 s. The paused capacitor display stayed 2.743 V. Loading the one-shot
from that edited fixture correctly showed unsaved-work protection; only the
owned synthetic wiper edit was discarded. Native inspection found an inherited
“NPN scope” caption; the presentation follow-up uses “Live 555 scope” and a
555-specific experiment description. Focused scope/timer/browser/race tests
pass (25.259 s). Electrical equations and commands did not change.

Final candidate **0.185.1 /236** is built from clean source
`81cd748a71317b29d1d618c709b4c6bbafbae454`. Both native and agent binaries
record that source with `vcs.modified=false`; the ad hoc signature verifies
strictly. [Package manifest](build-result.json) records their hashes. This is a
local development candidate, not a promoted public release.

The [packaged stdio replay](packaged-stdio.json) passes all three examples,
edge capture, independent timing, live wiper preservation, kept-record
save/reopen and SVG/PNG/PDF export. Its largest observed reference error is
1.159 µs at 5 ms sampling. All three exported PNG schematics were inspected:
pins, power feed, timing network, labels and parts lists are readable.

The [native result](native-result.json) records the feature-specific /235
interactions and final /236 presentation check. The final window shows the
flasher operating with capacitor, output and LED-current traces, the corrected
“Live 555 scope” caption and “555 experiment” inspector. It is left running in
Auto, with U1 selected and the inherited Apollo theme. The earlier owned /235
session was closed normally. Other sessions and user projects were preserved.
This is not a new whole-distribution acceptance or latency qualification.

Final lint and source-inventory verification pass. The qualified /205 release
archive retains its original checksum and has not been promoted or replaced.
