# All-discrete amplifier and loudspeaker

Available in the /277 electronics candidate; the /276 package is preserved. Choose **All-discrete
amplifier and loudspeaker** in Examples, or use
`example.load {"id":"class-ab-speaker"}`. The eight-BJT circuit retains the
original 1 nF compensation and 7.55 V biased capacitor startup. Its 10 mV peak,
10 Hz input drives the electrical speaker equivalent. This is not a zero-energy
power-on test, a manufacturer amplifier, an acoustic model or a power rating.

Six default traces show input, speaker terminal voltage/current, Q6/Q7 collector
currents and C1 voltage. The output sample is 100 µs. Compare both polarities,
change speaker resonance/coil resistance, or capture a run using Verify. Start
with a 0.1 s interval and 0.0001 s sample (the existing 1000-step bound). Use
smaller samples when investigating fast startup; requests can still reach the
unchanged work guard. Rendering, playback and WAV export remain explicit.

The previous source paired every capacitance with every inductance to limit
internal steps. The unrelated 1 nF compensation and speaker coil forced a
17.7 ns ceiling, exhausting work even after the fast mode had decayed. Larger
purely analog BJT/speaker networks now eliminate algebraic unknowns from the
local MNA Jacobian to find energy-scaled storage modes. Checked eigenvectors,
conditioning, inverse identity and backward residual provide a conservative
uncertainty radius. Imaginary frequencies and positive growth bound proposals;
negative real decay uses Radau's existing stiff damping and the full adaptive
error estimator. Midpoint/endpoint modes are checked as well. Failed extraction
or unresolved eigenbases retain the original conservative LC ceiling.

A norm-based initial guess resolves fast startup before growing the interval.
Later accepted proposals and checked endpoint ceilings carry forward privately; they are not physical state,
a persisted parameter or a new accuracy/budget setting. Every trial still uses
full/two-half Radau solves, existing voltage/current/storage error tolerances,
Newton and residual tests, source corners, node ranges, cancellation and the
512-attempt/100 ms guard. The attempt limit counts adaptive trials, as before;
modal analyses and Newton iterations have separate private diagnostic counters,
with every modal solve and decomposition charged to the same wall deadline. All other device families keep the original ceiling.
Gonum v0.17.0 supplies the small real-matrix eigen decomposition; its checksum
and BSD notices are pinned in the repository.

Users and agents share ordinary example, drawing, configuration, step/verify,
probe, recording, project and export commands. No new public command is required.
Captured models, undoable edits and saved graphs retain their original meaning.
The original resistor-loaded class-AB and powered-follower speaker examples are
preserved. Physical native-input acceptance remains separate from routed tests.

The complete two-polarity numerical/refinement test uses a deterministic test
clock while enforcing attempt counts and all electrical/error criteria. Separate
shared-service capture/reopen and ordinary/offscreen UI-step tests use the
production wall clock. Injected-expiry tests verify atomic refusal for all
accepted voltages, charge, readouts and private proposals. These test scopes
are recorded separately; deterministic-clock output is not wall-rate evidence.
