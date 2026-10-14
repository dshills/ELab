# DC parameter sweeps

A sweep calculates independent operating points over a selected DC bias, resistance or potentiometer wiper range. Use **DC sweep** in the drawing toolbar to open the settings rail, enter unit-aware endpoints and 2–201 points, add up to six readings, then **Run DC sweep**. The curve viewer plots one reading against the parameter. Click a measured point for exact stored readings; its cursor does not interpolate. **Back to circuit** restores the drawing and live panels. Live acquisition continues independently.

Each sweep captures its circuit, curated model parameters, configured switch state, accepted wiper values and source settings. Computation runs outside the editor lock. Later edits do not change captured results and make their source association unavailable. Only a complete successful curve is published. Cancellation, failure and timeout publish no partial point array. Progress is staged work rather than committed measurements. Eight curves may be kept; excluded completed curves can be evicted explicitly to make room.

## DC semantics

Capacitors are open circuits, with zero DC current and solved terminal voltage; startup charge is ignored. Sine sources contribute their offset, finite pulses their low level, and ideal pulses zero. A swept source instead uses the requested DC bias at each point, including −12–12 V. Frequency/phase, pulse timing, paused held inputs and pending actuation do not participate. Switches use their configured desired state, and potentiometers use the accepted graph position.

This follows the operating-point distinction described by the [ngspice DC documentation](https://ngspice.sourceforge.io/docs/ngspice-manual.pdf): DC transfer curves use steady operating points with capacitors open. ELab uses its own existing bounded MNA/Newton device models; it does not run ngspice or claim SPICE compatibility. No charge/time advancement, warm-start path following, physical ratings, temperature sweep, nested/log sweep or enumeration of multiple equilibria is implied.

All points begin from a separate solver state. Generic grounded-emitter NPN, diode, LED, resistor, switch and potentiometer limitations remain in force. Floating nodes exposed by opening capacitors may make the DC system singular; the entire sweep then fails clearly. A solved-system check also rejects zero-residual but underdetermined cases rather than assigning arbitrary node voltages. Strictly distinct representable axis values are required. Finite solver results and residuals remain subject to the existing tolerances, so sweeps do not imply arbitrary precision.

## Shared commands

| Command | Behavior |
| --- | --- |
| `sweep.settings.set` | Complete `component_id`, `parameter` (`voltage/resistance/wiper`), `start/stop` text, `points`, optional `probe_ids`. Can configure a partially drawn graph; running requires a complete supported circuit. |
| `sweep.start` | Optional `settings` override; otherwise uses next-sweep settings. Returns immediate admission. One worker, cooperative 30-second wall budget. |
| `project.inspect`, `events.poll` | Owned `dc_sweep` settings, targets, probes, operation summaries, selected complete result and view/cursor state. Publications include `sweeps`/`jobs`. |
| `sweep.inspect` | Exact retained ID; returns captured source/model/settings metadata and complete results, or failure metadata. |
| `sweep.cancel` | Exact active ID; requests cancellation. |
| `sweep.select`, `sweep.visibility.set` | Choose a retained curve or show/hide the sweep workspace/settings; preserve live simulation. |
| `sweep.trace.select` | Choose one captured probe for plotting with its own units. |
| `sweep.cursor.set` | Exact zero-based `index` or null to clear; optional `operation_id` refuses a changed selection. Native gestures also carry rendered command/document guards. The macOS GPU pointer bridge binds each packet to its painted pane, geometry and capture token; retired/changed frames are rejected. |
| `sweep.keep.set` | Include/exclude a completed curve from project saves/recovery. |
| `sweep.export` / `export.start` | Atomic complete JSON or SI CSV output; explicit ID/path/format and overwrite guards. Captured exports never substitute current editor data. |

Values are signed SI. Native current plots scale amperes to milliamperes for display; data stays in amperes. Linear endpoints are inclusive and descending ranges are supported. Resistance is 10–100000 Ω and wiper position 0–1. Endpoint strings are at most 128 bytes. Normal snapshots contain only the selected bounded result, not duplicated captured graphs for every operation. `simulation.wait` retains its existing semantics; inspect/poll sweep status to await completion.

## Saving and evidence

Kept sweep data and its source/model/settings travel in a version-5 `.elab` project. Ordinary projects without kept sweeps continue to write version 4; versions 1–4 remain readable. Older ELab builds refuse version 5 rather than discarding curves. Opening restores the selected captured curve and its point/trace/view state with fresh live physics. Saving may wait only after active analysis is finished or canceled; background-open publication cannot erase a sweep started after its admission. Recovery uses the same strict data validation and recording membership.

Standalone JSON is an archival/analysis export; importing it into the recorded time viewer is not supported. Reopen the `.elab` project to restore curves. Saved/imported numeric data is structurally/model validated, not certified by replaying every solve.

Regression tests cover analytic loaded dividers, existing independent LED/NPN kernels, capacitor-open and source-bias semantics, signed ranges, nonrepresentable axes, cold solver state, cancellation/close joining, immutable capture/results, retention limits, cursor guards, complete exports, strict version/model/point/extrema validation and saved restoration. Native tests cover controls/agent parity, multi-reading choices, plotted parameter axes, measured cursor selection, file export and stale contexts. The [public demo](../scripts/dc-sweep-demo.py) exercises three circuit families on stdio and native sockets. Release/runtime evidence is recorded separately.
