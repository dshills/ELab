# Resistive sensor validation

ELab 0.189.0 /246 adds NTC thermistors and light-dependent resistors, per-component
sliders and exact numeric input, and the Automatic night-light / Thermistor relay
thermostat examples. The [component contract](../../resistive-sensors.md) specifies
pin names, reference resistance, equations, environment domains, limitations,
probes, persistence and staged command behavior.

The final app and agent were built from clean committed source
`5b0b903adcab5f540c5d2aa13df06d1458eb9009`; binary hashes, strict ad-hoc signature
verification and preserved-release status are in
[the build receipt](final-build-final-result.json). The pinned Vellum SDK was
unchanged. This is a local development candidate; the qualified /205 unsigned
private-alpha archive is preserved.

## Electrical and command coverage

Focused race tests cover high-precision NTC resistance reference values, LDR
power-law/dark/bright behavior, Ohm/Kirchhoff and power conservation, analytical
switched-RC response, comparator hysteresis and relay-coil continuity. Failure,
cancellation, supersession and return-to-committed-value cases retain accepted
physics and history. Ownership, strict capture validation, duplicate, startup
batch, undo/redo, save/load/recovery and fixed-environment DC sweep are covered.

The two controls call `sensor.temperature.set` / `sensor.light.set` through the
same guarded dispatcher as agents. Native callback tests compare results to
an independent service using those public commands and reject a retained native
field after an intervening agent edit. Unchanged float32 display values preserve
more precise environment settings. The native source inventory contains 170
public commands, 410 command sites, 138 literal routes and 83 production desktop
sources; this inventory is source evidence, separate from runtime checks.

The final packaged [stdio replay](final-stdio.json), reproducible with
`scripts/sensors-demo.py`, passes six light states and four temperature states.
It checks both hysteresis directions, accepted time/run and coil-current
continuity, model resistance, comparator/LED/relay state and conservation. Each
example is independently verified, saved with a kept capture, reopened fresh and
exported as SVG, PNG and PDF. [Export hashes](export-manifest.json) identify the
local files. Both PNG schematics were visually inspected, including labels,
connections and applied environment settings in their parts lists.

## Native interaction

The [native receipt](native-checks.json) and [screenshots/accessibility observations](ui)
record physical mouse dragging and keyboard entry in the final /246 Apollo app:
darkness and full brightness toggle the night-light; exact 1 lux gives the
expected 50.1187 kΩ and LED-on state; 20 °C /30 °C yield the expected thermistor
resistances and energized/released relay states. Scope traces show coil decay
and flyback on release. These are owned disposable fixtures, closed normally.

The first native round found a reference caption crowding a ground caption,
a base-emitter resistor wire routing over the resistor, and dense sensor
readings. /246 moves the caption, provides a separate ground, separates readings,
shortens help and snaps temperature dragging to half-degree steps. The numeric
field and agent API continue to accept precise temperatures. Final native
screenshots confirm the repairs.

## Regression history

An initial root run found the ordinary discovery catalog at 65,762 bytes,
above the existing 65,536-byte budget, and a batch-count test expecting 17 rather
than 19 supported child commands. Reusing registered parameter schemas keeps
discovery below its original limit; the batch contract assertion now includes
both new commands. The five failed discovery assertions and sensor cases pass
in [the focused rerun](discovery-sensor-race.txt).

Two subsequent broad race runs encountered existing flasher samples exceeding
the production 100 ms wall-clock budget. One included concurrent build/lint
work; an isolated retry passed all workbench tests but still hit a conformance
flasher sample. The [complete root suite passes](root-race-bounded.txt) with
`GOMAXPROCS=2` for the test process. The full workbench pass in that run is cached
from the immediately preceding same-source isolated 147.991-second run.
The first mounted run also found a stale browser test expecting four Passives.
The correct category now has six; the assertion was updated in `99d506fb`
(test-only), passed three focused race repetitions, and the complete mounted
suite was restarted. The partial failing run is retained separately. The
packaged production sources are unchanged by this test correction.

No simulation wall-clock/attempt limit, accuracy tolerance or packaged app
runtime setting was changed. Desktop tests also use `GOMAXPROCS=2`, run each
suite alone and retain the existing 30-minute test-runner timeout.

The complete standard desktop race suite passes 315 top-level tests /736
including subtests in 541.659 seconds. The complete mounted
control race suite passes 431 top-level tests /1280 including subtests
in 811.74 seconds. Neither suite fails or skips tests.
Goimports/staticcheck, root and both desktop vets, 67 Python tests, the existing
27-requirement coverage audit and current native inventory verification pass.
See [report.json](report.json), the [standard log](desktop-race-final.jsonl),
and the [mounted-control log](desktop-offscreen-race-final.jsonl). No user review
is pending.

The models are generic educational approximations. They do not include
self-heating, room/heater thermal feedback, optical/thermal response delay,
manufacturer tolerances or physical ratings.
