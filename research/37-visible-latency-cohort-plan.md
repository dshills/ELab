# Visible responsiveness: fixed collection protocol

October 19, 2026. Status: declared before collection; **zero cohort trials run**.
The [measurement decision](36-visible-latency-method.md) and original PRD thresholds
remain authoritative. The [causal composition calibration](../docs/validation/display-causal-input-173/README.md)
now demonstrates the complete baseline → acknowledged marker → CUA Apply →
committed/stale → fresh-result sequence. Proceed with the bounded software-frame
characterization below; do not expand calibration into an open-ended instrumentation
project. The calibration trial is excluded from all cohort percentiles.

## Fixed build, method and order

Use signed native **0.156.0 /173**, SHA
`ddd0e0cfdfad56c25c9d2d4d0882a1bb79095197101c9e79a844626bfaaa7a1a`,
matching agent SHA
`134f9863e2885e384035e30550adfc5fe0bea1bdee9f3d3cd42feaf802bb7de5`,
SDK `b22478cf8cec8dda1ccdcd93d6ff6b62e2cc11de`, and the unchanged display
recorder SHA
`a8cec4e8f9d16b2b5ad6b751f5657b537c5ed6ea956e7695403168b445bb923f`.
Before collection, record actual OS/build/device/display metadata. Expected host
is Mac16,11, macOS 26.6.2 /25G83, display 3 at 5120×1440/native 1× and owned
window `(2034,303,1053,802)`. A configuration mismatch stops admission for review;
do not silently merge different builds or modes into one percentile.

There are exactly **60 scheduled trials**, in this order: `led-01` through
`led-20`, then `rc-01` through `rc-20`, then `npn-01` through `npn-20`. Every
scheduled ID must have a retained outcome, including unavailable/inconclusive
attempts. Preparation mistakes and failed recordings are retained against that
ID; do not replace them with a favorable retry. Any later collection is a new,
separately declared cohort with its own reason and all original results preserved.
The [machine-readable declaration](37-visible-latency-cohort-plan.json) fixes the
60 IDs, edit parameters, binary/source hashes, display mode and percentile rules.

The reference electrical topologies are those in
[fixtures.go](../internal/conformance/fixtures.go),
SHA `6d1af00dea6c7dc786c5eb6c57147c689f71d43a064c4b2d9fb11dae2234e909`:

| Reference | Baseline | Single UI edit |
|---|---|---|
| LED | 5 V, generic red LED, `Limit` = 1 kΩ, grounded return | `Limit`: 1 kΩ → 2 kΩ |
| RC | Ideal 0–5 V pulse, `Limit` = 10 kΩ, `Storage` = 1 µF, grounded return | `Storage`: 1 µF → 2 µF |
| Resistor-loaded NPN | 5 V collector supply, 1 kΩ collector load, 10 kΩ base resistor, grounded emitter, generic Is=1e-14/Bf=100 model; prepare `Drive` at 0 V | `Drive`: 0 V → 5 V |

Use the existing default pulse timing and record it numerically with the RC
baseline. No LED is inserted into the NPN collector load. Seed/reset these exact
graphs through public commands before each trial, resume the live simulation,
select the edited part and wait for a fresh baseline. Record the reference state
and independently computed expected electrical behavior before input. RC is a
running transient, not a paused edit that intentionally waits for a restart.

## Each scheduled trial

1. Verify the owned PID/window/executable and unchanged crop/mode. Read public
   baseline state. Type the target value privately through CUA and inspect fresh
   AX state; public accepted parameters must still equal the baseline.
2. Arm a **three-second**, fixed-crop, local-only recording with the existing
   ≤300 callbacks/≤64 MiB unique image limits. The CUA controller signals start,
   waits for a complete baseline frame, requests a unique marker and reads its
   complete acknowledgment before clicking Apply exactly once. No model/tool
   round trip occurs between acknowledgment and click.
3. Let the recorder finish its fixed interval, irrespective of how quickly the
   result appears. Read final public state; pause RC only after recording. Save
   graph/result/scope/diagnostics, all images/journals, input events and cleanup
   outcome. Do not perform another electrical edit inside that trial's recording.
4. Inspect every complete frame's relevant regions. Identify the **first visible
   committed parameter value**, independently of whether the solver has finished,
   and the **first fresh valid result** for that new value. A draft, pressed button,
   old reading, stale label or unrelated animation cannot establish the fresh result.
   A fast solve need not expose a captured stale frame. Retain manual original-image
   annotations and exact region hashes; moving transient readings must not be
   classified solely by a static current/voltage template.
5. Check the new graph/model and public physics against the matching packaged
   agent and independent reference laws/conservation. Require one electrical
   revision/run change. For RC preserve actual sample times, source mode/timing
   and the new 20 ms time constant. Do not compare independently running sessions
   at different times as if their numerical samples were simultaneous.

Recorder errors, changed identity/geometry, missing image/acknowledgment, unknown
image state, inconsistent timestamps, nonfresh/invalid physics or ambiguous
causality make that observation unavailable. Preserve the raw evidence and reason.
Idle callbacks and differing callback counts alone do not prove missing OS frames.

## Analysis and decision

Report all 20 scheduled outcomes per reference and both response criteria. Compute
marker → first committed-value frame and marker → first fresh-valid-result frame
using exact Mach timebase conversion. The marker precedes the CUA input and
accepted edit, so these are conservative **software composition-frame bounds**,
including controller overhead. Raw recorder eligibility flags stay unchanged;
external analysis must explicitly state which observations support this method.

Use the nearest-rank convention: `p95 = sorted_values[ceil(0.95*n)-1]`; for 20
available bounds this is the 19th ordered value. Do not report a complete-cohort
p95 from a successful subset. If any scheduled measurement is unavailable, report
the missing count/reasons and only clearly labeled descriptive available values.
Keep slow and inconclusive values in all distributions.

Compare the full-cohort bounds with the original **100 ms acknowledgment** and
**500 ms first updated sample** targets. A conservative bound below a target can
support that target only within the validated software-frame method. A bound
above it is inconclusive about product latency because controller overhead is
included; do not call it a product failure or weaken the target. Do not equate
these timestamps with physical scanout or claim an exact OS-input/accepted-edit
interval. Any PRD gate conclusion must state this measurement scope and its limits.

Do not change the app, solver limits, devices, SDK or recording duration in the
middle of collection. If a product defect requires repair, preserve the cohort
and declare a new build/cohort rather than mixing pre- and post-repair values.
Navigation/cancel/save/undo under difficult solves, cold routing, human typing/IME,
accessibility, owner walkthrough and the hobbyist study remain separate work.
