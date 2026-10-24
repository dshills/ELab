# RC-driven MOSFET references and active verification

October 23, 2026. Existing electronics validation using the exact /217 packaged
headless agent; no graphical launch, new component or new model. [Evidence](../docs/validation/gate-common-mode-217/README.md).

Seven fixtures exercise an NMOS gate charged through 1 kΩ into 1 µF, starting at
1.25 V gate-to-source, with 5 V drive and a 1 kΩ drain load. Six vary the source's
DC reference through −5, 0 and +5 V and reverse the capacitor terminals without
changing its physical startup voltage. The seventh replaces the gate resistor
with fifty series 20 Ω resistors: 56 simulated parts and 55 electrical nodes,
within the existing supported network limits. Each uses three DC supplies.

An independent reference uses `Vgs(t) = 5 − 3.75 exp(−t/1 ms)`, the pinned
educational MOSFET channel/body-diode equations and a scalar drain load line.
The reference imports no ELab code and computes drain behavior from analytic
gate voltage, not measured gate samples. It checks capacitor voltage/current,
gate-to-source voltage, drain-to-source voltage/current and source reference.
This validates implementation against the declared educational model; it does
not establish agreement with physical devices or a separate SPICE simulator.

All 287 rows and 1,722 scalar comparisons pass the declared voltage tolerance
of max(1 mV, 1%) and current tolerance of max(10 nA, 0.5%). The largest gate
voltage error is about 33.03 µV, drain voltage 1.842 µV, drain current 1.842 nA
and capacitor current 33.03 nA. These are observed maxima for these fixtures,
not universal error bounds. Reversed capacitor readings carry the expected
sign; physical gate/drain results remain consistent across DC references.
Verification preserves the live physical state and input files; its completed
capture intentionally adds recording metadata and marks recordings dirty.

The larger network also runs an ordinary 1,000-step Verify. Pan, Save, Undo and
Pause complete with operation status running observed before and after each:
approximately 2.28, 15.90, 2.24 and 2.21 ms including transport/serialization.
The cancellation request takes about 2.47 ms; the operation becomes canceled,
publishes no partial rows and preserves live charge/time. Input metadata remains
inspectable with zero samples, as required. A fresh agent reopens the file saved
before Undo and confirms its graph/view, moved component and empty recordings.
All owned processes finish normal shutdown with workers joined.

Cancellation precedes the first published progress update. Running status around
commands does not prove exact solver overlap, and these point measurements do
not qualify a percentile, physical input latency or the whole capacity envelope.
Existing source tests with an intentionally blocked kernel provide separate
concurrency evidence. Initial harness mistakes about required version-4 fields,
ready status, layout, expected recording metadata and canceled input metadata
are preserved separately from the passing observations.

The qualified /205 download and /217 candidate remain preserved. /217's native
acceptance is still separate; these headless results do not pass that gate.
