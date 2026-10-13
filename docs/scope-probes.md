# Selectable scope probes and timing cursors

The scope records supported terminal voltages and branch currents during RC, pulse NPN, and mixed RC/NPN runs. Choose up to six traces in **Scope probes** in the right inspector. Flasher starts with both collector voltages, both LED currents, C1 voltage, and source voltage. Change a selector to inspect C2 voltage/current, either transistor's base or emitter current, or any other supported pin/branch. **Hidden** removes that channel. Every selection uses `scope.probes.set`.

Selection immediately fills a newly chosen trace from retained history. It preserves simulation time, capacitor memory, run ID, electrical revision, undo history, and document dirty state. Recording covers all supported probes even while hidden; only selected trace arrays are exported. Histories remain bounded to 512 samples, including duplicate-time samples around ideal source edges. Probe selection is presentation state and explicit Save captures it in version-3 `.elab` files without marking the drawing dirty solely for a selection change. Reopen restores probes and starts fresh simulation. Cursors and fixed windows are session-only. New/examples and older files restore defaults. Electrical restarts keep valid probe choices and clear cursors; geometry edits preserve everything. Removing a component removes its channels from the visible scope. A fixed window plots only its requested retained interval and reports min/max/sample mean/sample RMS per selected trace; expired windows report no statistics. See the [next-ten implementation record](../research/09-next-ten-implementation.md).

## Read the scope

Voltage traces use the left axis in volts; current traces use the right axis in milliamps. Both axes include negative values and scale to retained selected data. The legend and inspector identify each trace's color and direction. Numeric agent data always uses SI units (V, A, s, Hz).

- `v:Q1.collector` means that terminal relative to circuit ground. Every terminal has a voltage probe, including grounded pins.
- `v:C1` means capacitor pin 1 minus pin 2. All two-terminal parts also offer a voltage across the part, in library pin order.
- `i:C1` and `i:R1` are positive from pin 1 to pin 2. Reversed wiring changes the measured sign correctly.
- `i:D1` is positive from anode to cathode.
- `i:Q1.base` and `i:Q1.collector` enter the transistor. `i:Q1.emitter` leaves it.
- `i:V1` is current delivered out of the source's positive pin. Negative indicates absorption.

This is an observation of the existing solver, with its existing ideal-device and topology limits. Current probes add no component, loading, or invented current on a branch. An unsupported or incomplete circuit does not create new readings. Stale histories draw in gray and cannot produce valid cursor measurements.

## Measure timing

Pause the run to keep the rolling window still. Choose **A** or **B**, then click or drag in the plot to place the selected cursor; the ms number fields provide exact requested times. The scope draws both cursor lines and shows their actual recorded times, signed **Δt = B − A**, and **1 / |Δt|** in Hz. The inspector shows A, B, and B − A for every selected channel. Clear each cursor with its × button.

Cursors snap to the nearest retained sample. Equal-distance ties and duplicate timestamps choose the latest sample, so a cursor exactly on an ideal edge reads its right-hand state. The API distinguishes `requested_s` from the snapped `time_s`. There is no interpolation or sub-sample precision claim; choose a smaller sample interval for finer timing. Equal cursor times have Δt = 0 and no reciprocal frequency. Hz is the reciprocal of the chosen interval, not automatic period recognition: choose equivalent edges on the same branch to measure a period, or opposite sides to measure phase separation.

If a requested cursor time rolls outside the retained window, it stays identified as invalid with `reason: "outside retained history"`; timing/readings disappear until it is repositioned. A stale run similarly refuses measurement. No prior-run samples are joined into a measurement.

## Shared commands

Read `project.inspect.scope` to discover `available` probe descriptors (`id`, `label`, `unit`, `positive`, `reference`), selected IDs, selected trace points, cursor validity, and computed measurements. All mutations retain ordinary expected-revision and retry-key admission. Discover typed input/output schemas through `capabilities.list`.

| Command | Parameters | Behavior |
|---|---|---|
| `scope.probes.set` | `{"ids":["v:Q1.collector","v:Q2.collector","i:D1","i:D2"]}` | Replace selection with 0–6 unique IDs from available; unknown/duplicate IDs refuse |
| `scope.cursor.select` | `{"cursor":"a"}` | Select a/b for subsequent plot gestures |
| `scope.cursor.set` | `{"cursor":"a","time_s":0.245}` | Place a/b inside fresh retained history; return snapped sample and measurements |
| `scope.cursor.clear` | `{"cursor":"b"}` | Clear a/b and paired measurements |

`snapshot.scope.traces[].points` contains `time_s` / `value` in the descriptor's SI unit. `delta_s` appears only when both cursors are valid; `interval_hz` additionally requires nonzero Δt. `readings` stays empty until both cursors are valid. Each reading has `probe_id`, `a`, `b`, and `delta` in that probe's SI unit. `run_id` and `stale` identify trace provenance. Existing primary `transient.samples` fields remain compatible. Private full-probe solver frames do not enter JSON or discovery schemas. Responses are bounded to 1 MiB, including discovery.

## Reproduce the validation

```sh
./scripts/check.sh
./scripts/build.sh
./build/elab --smoke
python3 scripts/probes-demo.py --output docs/validation/headless-probes.json
# Development session only: this replaces its schematic with the flasher.
python3 scripts/probes-demo.py --connect build/probes.endpoint.json \
  --output docs/validation/desktop-probes.json
```

Checks cover newly selected history, both flasher phases, individual device/node results, negative RC discharge current, right-hand pulse-edge measurements, rolling expiration, stale readings, zero separation, restart, clean-document presentation edits, mutation refusals, retry snapshot isolation, and native callback/pointer equivalence to agents. Native smoke checks real UI controls and GPU traces/cursor strokes. The default 10 µF flasher's successive LED-current rising samples measure approximately 0.485 s at 5 ms sample resolution; this validates scope measurements against the application's solver, not physical hardware.
