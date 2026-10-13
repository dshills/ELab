# Next ten hobbyist workflow improvements

**Requested and implemented October 13, 2026.** Following selectable probes and scope cursors, the product owner asked for approximately ten additional project items while away. These are bounded additions selected from the existing PRD's draw–measure–recover–save workflow. They do not complete all MVP gates.

| Item | Implemented behavior | Shared command / state | PRD |
|---|---|---|---|
| 1. Component search | Case-insensitive component tray search across name, kind, units, pins, and model | `parts.search.set`; `matching_parts` | EDIT-01, ACCESS-01 |
| 2. Component duplication | Unconnected copy with fresh ID, same value, orientation, and capacitor startup override; optional exact position | `component.duplicate` | Editing boundary, EDIT-03 |
| 3. Named undo/redo | Next undo/redo action names in the UI and snapshot, including transactions | `edit.undo` / `edit.redo`; `undo_name` / `redo_name` | EDIT-03, AGENT-05 |
| 4. Atomic edit batches | Up to 100 schematic edits commit once or leave the entire drawing and ID reservations unchanged; one named undo | `edit.batch`; native two-capacitor presets use it | AGENT-05 |
| 5. Direct plotting | Selected part offers Plot voltage/current; adds trace or replaces the last at six | `scope.probes.set` | SCOPE-01 |
| 6. Fixed scope interval | Follow rolling history, choose last 200 ms, or enter start/end times | `scope.window.set`; `scope.window` / validity | SCOPE-02 |
| 7. Interval statistics | Min/max, sample mean, and sample RMS per selected trace over precisely the displayed retained interval | `scope.statistics` | SCOPE-02 |
| 8. Saved probes | Save captures selected probes; reopen restores them while starting a fresh run | `project.save` / `project.load`; `.elab` v3 | FILE-01, primary workflow |
| 9. Waveform CSV | Selected fresh interval with SI signal headers, timestamps, run/electrical revision, engine and sample resolution | `export.scope.csv` | FILE-03 |
| 10. Schematic SVG | Portable complete-sheet rendering with readable part/pin/value labels and the same routes as the native canvas; works headlessly | `export.schematic.svg` | FILE-03, AGENT-03 |

Six selected traces extend the PRD's proposed four-trace boundary so both collectors, both LED currents and both capacitor voltages can be inspected together. The retained 512-sample bound and 1 MiB response limit remain tested.

All native semantic controls invoke the same service as agents. Component search, plotting, intervals and cursor changes preserve solver state. Probe selection is captured by explicit Save without itself marking the drawing dirty, like saved viewport navigation; cursors/windows are session state. Electrical edits restart or stale the experiment according to existing pause rules. Batches cannot include simulation, files, nested batches, undo, or arbitrary code. They share revision conflicts and retry deduplication.

Version 3 files retain embedded generic model identities, settings, and graph plus `scope_probes`. Version 1/2 files still load. Unknown versions, invalid probe IDs, duplicates, or malformed values fail before replacing the project. Earlier app releases refuse newer documents clearly.

CSV export refuses stale/expired windows or empty traces; it retains duplicate edge timestamps. Mean/RMS are statistics of recorded samples, including edge samples, rather than time-weighted integrals. Neither cursors nor statistics invent sub-sample accuracy. A fixed window that rolls out of memory becomes explicitly unavailable. SVG includes symbols, pins, values, orientation, startup overrides, and model/revision metadata. It is a schematic artifact, with no PCB/physical device identity implied.

Exports atomically publish to regular files, require explicit overwrite for existing destinations, refuse the current schematic path, and leave graph/dirty/solver state unchanged. Native file pickers and agents use identical export parameters. SVG wire routes are shared with native rendering and cached with immutable returned copies.

## Validation and artifacts

- Root and desktop race tests and vet: `./scripts/check.sh`.
- Native GPU/UI smoke: `./build/elab --smoke`, covering probes, cursor lines, search, atomic paired edits, direct plotting and window statistics.
- Shared-command demonstrations: `scripts/probes-demo.py` and `scripts/workflow-demo.py` through headless stdio and the packaged desktop's private socket.
- [Headless workflow report](../docs/validation/headless-workflow.json), [desktop workflow report](../docs/validation/desktop-workflow.json), [probe timing report](../docs/validation/desktop-probes.json).
- [Probed flasher project](../docs/validation/workflow-artifacts/flasher-probed.elab), [interval CSV](../docs/validation/workflow-artifacts/flasher-interval.csv), [schematic SVG](../docs/validation/workflow-artifacts/flasher-schematic.svg).
- Regression cases exercise batch rollback including ID reservations, single undo, conflicts, duplication, inclusive decimal time boundaries, rolling expiration, save/reopen, v2 compatibility, CSV units/identity, SVG XML/pins, export conflicts and link refusals. Native callbacks/pointer gestures and export adapters are compared with agents.

Final packaged build: **0.9.0 / 23**, private local bundle at `build/ELab.app`. The [final validation record](../docs/validation/workflow-validation.json) records binary identity, numerical equivalence and the paused inspection state. Prior application bundle and user files were preserved.

## Remaining priorities

The next significant gaps are explicit junction/net-label primitives and broader topologies, generic diode/switch/potentiometer coverage with electrical fixtures, bounded Verify operations and recorded results, autosave/recovery, long-running job/events/cancellation, and capacity/latency/accessibility/clean-machine distribution acceptance. Participant usability tests remain necessary. The current generic device assumptions and bounded solver limits remain in effect; no full MVP or professional signoff claim is made.
