# Inspect components, terminals and nets

Select a component to see all supported terminal voltages, differential voltage and branch currents in **Live measurements**. Click a terminal dot in Select mode to inspect its voltage relative to ground and highlight its actual connected wire routes and terminals. A terminal gesture does not move the part or pan the sheet. Selecting a wire shows its net voltage; ideal-wire current is not inferred from device currents. A disconnected drawing wire reports that it is not electrically connected rather than inventing a voltage.

The Inspector also provides named terminal, net and voltage/current selectors. These use ordinary semantic controls, supplying a non-pointer selection path. Each reading has signed units, a named voltage reference or current direction, accepted-result provenance and live/stale/unavailable status. **Plot** adds that exact probe, replacing the last channel if six are already shown. **Pin** keeps it on the sheet using the existing shared pin command. DC pin controls are available in the general Inspector as well as transient circuits. Plot requires a transient history; pins can be created before an accepted result exists and remain explicitly unavailable until one does.

DC inspection is labeled as a steady-state calculation. Transient inspection identifies the accepted sample time. A valid zero, including ground voltage, appears only after an accepted result; absent data has no numeric value. A paused electrical edit retains previous results as stale. Deleting an explicitly inspected source reports unavailable; undo can restore it. Moving a part or choosing a measurement preserves physical state. Recorded scope view leaves these explicitly **live** readings unchanged.

## Shared commands

| Command | Parameters | Behavior |
|---|---|---|
| `measurement.select` | `{"kind":"terminal","id":"C1.1"}` | Focus a current terminal without changing edit selection or physics |
| `measurement.select` | `{"kind":"net","id":"net:C1.1"}` | Capture the net's stable terminal anchor and follow its current group |
| `measurement.select` | `{"kind":"object","id":"R1"}` or `{"kind":"probe","id":"i:C1"}` | Inspect a current component/wire or exact voltage/current probe |
| `measurement.select` | `{"kind":"","id":""}` | Clear explicit measurement focus and return to the editing selection |
| `measurement.inspect` | `{}` | Read the current inspection, without changing any editor/actor state |
| `measurement.inspect` | `{"target":{"kind":"probe","id":"i:Q1.emitter"}}` | Query a current source without changing selection |

Selection uses normal revision guards and retry keys; it preserves time, charge, run/electrical revision, graph, history and dirty flags. Read-only queries require no mutation guards. Accepted kinds are `object`, `terminal`, `net` and `probe`; unknown kinds, absent sources, extra fields and incomplete nested targets refuse. Object IDs, terminal refs, net IDs and probe descriptors are discovered from the snapshot. `component.select` clears explicit measurement focus and automatically supplies the selected object's inspection.

`snapshot.measurement_inspection` and the query result have identical structure: optional `target` (`kind`, current `id`, optional stable `anchor`), `result_kind` (`dc`, `transient`, `none`), `available`, optional `reason`, `object_ids`, `terminal_refs`, `wire_ids`, current `nets`, and `readings`. Each reading contains a full probe descriptor, optional SI `value`, accepted `time_s`, `run_id`, `electrical_revision`, `state`, optional `reason`, and shared `display` / `reference_display` text. `available` identifies an inspectable source association; individual readings can still be unavailable or stale. An empty selection has no target and no fabricated values. Arrays/value pointers in snapshots and retry outcomes are isolated from callers.

## Stable net identity

Net IDs use a canonical physical terminal and can change when wiring changes. At selection, the service captures the terminal encoded by the current net ID. It follows that terminal's current group when a net is merged, renamed or split, exposing the new current net ID and members. It does not reuse an old net ID to select an unrelated group. If the anchor disappears, the selection becomes unavailable until undo or explicit reselection; another surviving member is not silently substituted. Current association metadata and previous accepted reading provenance stay distinct when a topology edit makes readings stale.

Inspection focus is session state. New/load/example/recovery replacement clears explicit focus; reopening retains saved pin preferences and kept records while starting fresh physics. It does not restore a trusted reading or an old net selection from disk. Interaction with a captured recorded schematic remains future work.

## Validation

```sh
./scripts/check.sh
./scripts/build.sh
./build/elab --smoke
python3 scripts/measurement-inspection-demo.py \
  --output docs/validation/headless-measurement-inspection.json \
  --artifacts docs/validation/measurement-inspection-artifacts
# Only an owned development session: this replaces its circuit.
python3 scripts/measurement-inspection-demo.py \
  --connect build/measurement-inspection.endpoint.json \
  --output docs/validation/desktop-measurement-inspection.json \
  --artifacts docs/validation/desktop-measurement-inspection-artifacts
```

Core checks cover the LED resistor's Ohm/reference equations, accepted ground zero versus absent data, exact RC voltage at 10 ms, signed discharge/NPN emitter conventions, object/terminal/net/wire/probe selection, read-only query invariants, renamed/split/deleted net anchors, malformed requests, retry isolation and recorded-view separation. Native tests cover DC pin availability, terminal gestures, named net/probe selectors and shared plotting/pinning. Built and packaged smoke rebuilds resources before checking actual GPU wire/terminal highlight pixels. The initial smoke setup assumed an unpaused LED example; it was corrected to resume/settle explicitly because examples preserve the existing pause state. Complete keyboard focus, screen-reader, visible-latency and participant qualification remain separate gates.
