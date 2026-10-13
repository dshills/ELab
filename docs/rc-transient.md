# Capacitors and live RC transient simulation

Implemented October 13, 2026. The workbench now supports an ideal capacitor, an ideal pulse source, and a bounded first-order RC transient engine alongside the existing LED DC engine. The public catalog contains 35 commands; users and agents use the same dispatcher, settings, input events, trace data, history, and file format.

## Working workflow

Place one source, resistor, capacitor, and ground from blank, then wire source → resistor → capacitor → ground, with source negative grounded. A DC source begins charging immediately; Charge/Discharge change its live drive while retaining capacitor voltage. The RC example uses a 5 V ideal pulse, 10 kΩ, and 1 µF, with 50 ms high/low intervals. Auto repeats the cycle; manual Charge/Discharge override it until Auto is selected again.

The docked scope plots output voltage relative to ground in teal and source voltage in amber. Meters show signed current into the output, output voltage, and stored energy. Time and τ, run identity, paused/stale state, pending input, and retained time bounds are visible or available in the structured snapshot. The plot is a fixed rolling view; it exposes no private canvas pan/zoom behavior. Up to 512 sample/edge points and 64 input events are retained, with no display decimation. UI repaint pacing is separate from solver samples.

Pause freezes time. A manual Step or Resume applies any queued input and continues stored state. Restart resets time/voltage to the saved initial condition while keeping the current source mode. Electrical/model/settings edits start a fresh run, with a restart marker, or remain paused/stale until resumed. Moving/rotating, selection, and viewport navigation preserve time and energy. Broken/unsupported circuits stop advancement and mark the previous trace stale.

## Numerical method and limits

For each interval with constant input, the engine solves the exact ideal response:

`Vc(t + Δt) = Vin + (Vc(t) − Vin) exp(−Δt / (R C))`

It splits at actual pulse edges even between sample points. Current is `(Vin − Vc) / R`; energy is `0.5 C Vc²`. This directly integrates the supported linear first-order circuit; it is not a general SPICE/MNA engine. There is no capacitor ESR/leakage, finite source rise/fall time, arbitrary initial polarity, mixed LED/RC support, or multi-capacitor network support yet.

Capacitance is bounded to 1 pF–1 F; existing R and source ranges remain. Validated settings include sample step, high/low durations, initial voltage, and playback speed. Manual advancement is bounded to 1 simulated second / 1000 sample steps per command. Live playback is best-effort paced and caps wall-time catch-up rather than claiming hardware real time. The initial default playback is deliberately slowed to 0.02 simulated seconds per wall second.

Version 2 `.elab` files include transient settings and part model versions. Existing version 1 files load with default settings; an explicit Save upgrades them. Stored configuration participates in dirty state and undo/redo. Time, waveform history, input override, and solver memory are session state; reopening begins a fresh Auto run from saved initial voltage. This extends the [persistence slice](persistence.md).

## Validation

- Root/desktop race tests and `go vet` pass. Existing LED and persistence coverage still passes.
- Analytical 10 kΩ / 1 µF step response at 10 ms: **3.160602794142789 V**. First charging peak at 50 ms: **4.966310265004574 V**. First discharge minimum at 100 ms: **0.033462735346615 V**. Settled repeated-pulse extrema approach **0.033464255 / 4.966535745 V**.
- A refined sample step with edges between samples preserves the analytical result. Tests check current/energy identities, bounded traces, paused time, continued state on Resume, electrical restarts, unit parsing/bounds, settings history, detached trace snapshots, arbitrary IDs/reversed pin mapping, and version 1 migration.
- Actual UI callbacks construct the RC circuit from blank and match direct agent charge/discharge samples within floating-point precision.
- Native smoke now verifies both LED drawing/value editing and RC UI stepping to the 10 ms reference, including rendered scope pixels.
- A headless agent and an external desktop agent both construct from blank, check automatic edges and manual input continuity, preserve time across move/rotate, reject invalid capacitance, and save/reopen configuration.
- Native inspection confirms the complete voltage plot and controls. Charge while paused queues an input; Step raises voltage. Discharge plus Step lowers voltage with the same run ID and continuing time.

Evidence: [headless RC workflow/data](validation/headless-rc.json), [desktop RC workflow/data](validation/desktop-rc.json), [portable RC fixture](validation/rc-circuit.elab), and [native scope screenshot](validation/rc-transient-ui.jpg). `scripts/rc-demo.py` repeats these checks. The [agent API](agent-api.md) documents command parameters and sample fields.

The full MVP still needs general transient networks/models, transistor timing, selectable probes and cursors, recorded captures/CSV export, recovery, performance/accessibility qualification, and broader platform acceptance.

The subsequent [NPN switch slice](npn-switch.md) adds transistor DC operation; mixed RC/NPN transient circuits remain unsupported.
