# Feedback return ratio and captured step response

## Dedicated loop analysis

Open `feedback-loop`, choose Simulate, and run its default **U1 Loop return ratio**
target. The ordinary **VIN AC frequency** target remains the signal transfer
Vout/Vin. Users and agents use the existing `sweep.settings.set` / `sweep.start`,
inspection, cursor, keep, JSON/CSV export and project commands. The new target is
`component_id: U1`, `parameter: loop-frequency`; omit probe_ids or use exactly
`["loop:U1"]`. Frequency must ascend within 0.01 Hz–1 GHz, with 2–201 points.
The existing 30-second cancellable worker, source/replacement guards and complete
result publication apply. No live schematic, physics state or bias is edited.

This first loop method exposes the differential-to-compensation control of an
op-amp with optional dynamics enabled. It solves the closed-circuit DC bias,
then opens only that controlled current increment in the small-signal Jacobian.
Input/output loading, rails, compensation storage and other paths stay in place.
Let Aclosed=Aopen+u*vᵀ, where vᵀ extracts V+input−V−input. The returned differential
voltage from a one-volt equivalent control injection is −T, with
T=vᵀ*Aopen⁻¹*u. The characteristic factor is 1+T. This is a declared controlled-
source return-ratio method; it does not pretend an ordinary signal Bode curve is
loop gain or silently disconnect a loaded physical feedback wire.

Only configured dynamic op-amps advertise this injection target. Memoryless
op-amps, discrete transistor loops, disabled/saturated controls and singular
breaks are not silently substituted. Other supported analog devices can remain
in the surrounding small-signal network; event/history-dependent circuits refuse.
All independent voltage sources are AC grounded. Captured models, source, bias
settings, method, complex responses and margin estimates remain inspectable.

The plot gives return-ratio gain and phase. The inspector lists every resolved
unity-gain crossing with signed phase margin, and every odd-180° phase crossing
with gain margin. Estimates interpolate dB/phase over log frequency. Missing
crossings have an unavailable reason, never an invented infinite margin. Coarse
sampling can miss crossings; refine and expand the band. Other loops stay closed;
open-loop RHP poles/Nyquist winding are not counted. Margins alone certify neither
multiloop nor nonlinear nor physical hardware stability. The example's finite
output resistance and CL create an additional pole: increasing CL reduces margin.

## Overshoot and observed settling

Open `step-response` and Verify **20 ms /20 µs** with `v:C1`. In its recording,
choose Step response → C1, set initial reference **0 V**, final reference **5 V**,
band **2%**, and required hold **2 ms**. The ideal 1kΩ/1µF circuit has τ=1 ms and
reaches the 2% band at −τ ln(0.02) ≈3.912 ms, with no overshoot. The example obeys
the unchanged 1000-output-step Verify limit.

`recording.step-response.set` uses the same full settings as the inspector;
`recording.step-response` is a pure query of any retained completed capture:

```json
{"id":"verify-1","probe_id":"v:C1","start_s":0,"end_s":0.02,
 "initial_value":0,"final_value":5,"tolerance_fraction":0.02,
 "minimum_hold_s":0.002}
```

References are explicit SI values in the captured probe's unit, within ±1e6 and
distinct. The UI defaults the final reference to **1**, labels it, and does not
infer steady state from the last sample. Tolerance is 0.0001–0.5 of step amplitude;
hold must be positive and fit the covered interval. Start is the caller's step
onset; this tool does not detect or validate a physical input step automatically.
Time inputs accept the existing seconds/unit text syntax; stored output is numeric.

Saved adjacent rows are reconstructed linearly, clipped to the requested interval;
duplicate-time rows retain instantaneous left/right limits and overshoot peaks.
The direction-normalized peak beyond the supplied final reference gives overshoot
for rising or falling steps. Settling delay is the last arrival into the final
band, only when it then stays in band for the requested hold through end_s.
Unsettled, short-hold or uncovered captures have explicit reasons/errors and no
fabricated settling number. No extrapolation, hidden-peak prediction or future
steady-state guarantee is made. Result identity, source engine, settings and
method accompany it. Plot reduction does not affect the exact saved-row analysis.

Set is session-only and guards the exact selected recording; empty probe with
all-zero settings clears it. Selection/reopen clears the cached view. Saved
recordings reopen and reproduce a query with the same explicit references;
measurements do not modify live charge/time, graph, history or project contents.
