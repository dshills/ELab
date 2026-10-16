# Device identity in mixed-network state captions

The live network scope's state caption describes one trace-state source. That source follows the existing rule: first NPN when present, otherwise first MOSFET, otherwise the generic `network` state. The caption now takes its device ID from the same shared `MixedRegion` policy that chooses the trace state. Selecting another device does not retarget it.

This fixes a 0.102 regression in circuits containing both device families. Q1 could be in saturation while MX was off, yet the caption said `MX saturation`. It now says `Q1 saturation`. The separate whole-network MOSFET card still reports its explicitly named first MOSFET, such as `MX REGION: cutoff`; it is a different reading. The selected inspector and individual probes let users inspect either device directly.

External agents retain the same commands and snapshot fields. `project.inspect` exposes the accepted `transient.sample.state`, `mixed.transistors` with each `transistor_id/state`, and `mixed.mosfets` with each `id/state`. Use those identities when interpreting a multi-device circuit. The native scope uses that accepted sample's state and the shared source rule; its text does not treat a pending requested input as a completed result. For example, a paused Low request remains pending until the next accepted Step, then the caption follows the new accepted NPN state.

No public command, model, numerical equation, work limit or file shape changes. Native blank construction and agents use the existing placement/wiring/selection/step commands. The regression constructs a pulse-driven NPN LED plus a separately grounded-gate NMOS, checks distinct saturated/off states, changes selection, and advances across the falling pulse edge. Current-run owned baseline/candidate/headless workflows have identical physical results. Native screenshots show the corrected ID and physical Low→Step acceptance at 1 ms.

[Before/after review](../research/ui-audit/mixed-region/README.md), [validation receipt](validation/mixed-region-validation.json). This is a bounded identity repair; broad layout, input/accessibility, latency and release qualification remain separate gates.
