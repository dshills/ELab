# Flip-flop and counter validation

[Electrical, wiring and agent contract](../../sequential-logic.md).

**Desktop follow-up completed:** the [complete offscreen desktop race suite](../desktop-suite/README.md)
subsequently passed all 423 top-level tests in 651.999 seconds with a 30-minute
runner timeout. This resolves the incomplete broad desktop check recorded
below. Simulation limits and application code were unchanged.

The feature adds D/JK/T flip-flops and synchronous binary/decade counters,
five examples, shared native/agent controls, stateful transient integration,
scope probes, recording validation, persistence and schematic symbols/exports.
The fixed educational model does not represent a particular IC or hardware
propagation/setup/hold behavior.

## Electrical and command checks

Truth-table tests cover both flip-flop states, every counter state, all JK
input combinations, enabled/disabled counting, modulo-16/modulo-10 wrap,
asynchronous reset priority, held/falling clock edges and no fabricated startup
edge. Other tests check power loss/restore, overvoltage refusal, clock deadband,
invalid sampled data refusal, analytic 15-pin Jacobians, KCL and passive power.

Live tests independently count recognized source edges and check output bits,
complements and loaded voltage/current balance. Common-clock D pipelines run
in both component orders and read preceding Q values; D-to-NQ feedback toggles.
Live enable/reset tests preserve time, hold state and avoid counting reset
release. RLC tests preserve capacitor voltage/coil current across event rows.
A 555 drives a T clock; a regression ensures its initially high settling
output establishes a baseline rather than causing a false initial toggle.
Failed/canceled work cannot leak trial memory.

Strict capture tests require model identity, complete unique device results,
all memory fields and exact array sizes; invalid stored states/model/guard
readings refuse import. Saved projects and kept recordings reopen, recovery
retains model metadata, and snapshots own their data. Required-input wiring,
sequential DC-sweep refusal, counter search and schema shape are checked.
SVG markup is parsed at every rotation; a pixel test checks actual closed
switch contact continuity in raster output. No renderer workaround was added
for apparent gaps seen in downscaled preview images.

Native tests draw a T fixture from blank using the same placement/wiring
commands as an agent, compare electrical results and reset actuation, load all
five examples with unsaved protection, and check rotated symbols/hit areas.
The source inventory remains 166 public commands /404 sites /134 literal
routes /82 production desktop sources. New menu aliases use `example.load`;
the ordinary API handles placement, wiring, switches, probes, simulation and IO.

The [packaged stdio replay](packaged-stdio.json) checks all five examples with
independent edge-sequence expectations, more than 800 sampled rows per example,
all 16/10 counter states, complementary Q outputs, enable/hold, reset,
conservation, kept captures, save/reopen and SVG/PNG/PDF hashes. The counter
capture uses 4 ms samples to stay within the existing 1,000-output-step bound;
the curated live examples retain their 2 ms defaults. The initial replay's
larger request correctly refused that bound; production limits were unchanged.
The [logic regression replay](logic-regression-stdio.json) also passes all
three combinational examples on /240, whose model code matches /241.

## Test runs and limits

- Full root `go test -race ./...` passes, including conformance (118.308 s)
  and workbench (122.972 s). This run preceded the final startup-baseline fix.
- After that fix, **all workbench race tests pass (128.613 s)**. The focused
  sequential/timer/gate race run also passes (24.224 s).
- Initial native sequential/symbol race tests pass (42.439 s). The later
  sequential plus mounted verification race run passes (93.007 s), including
  the verification group active at the broad suite timeout.
- The full offscreen desktop race run reached Go's default **10-minute suite
  timeout** while `TestMountedVerificationCorrectionToAcceptedTime` had been
  running seven seconds. Its stack shows active view/JSON work, not an identified
  deadlock. No earlier test failure is logged. The full suite is **not claimed
  green**; the complete timeout log is retained as `desktop-race.txt`.
- Lint (goimports/staticcheck), root and offscreen desktop vet, inventory
  verification and export checks pass. After the final coordinate-only change,
  all sequential tests pass again (1.666 s), followed by the final packaged
  replay and native layout checks.

These are equation, state-machine and workflow checks, not independent SPICE,
physical hardware timing or fresh distribution qualification. Numerical,
wall-clock and transport guards were not relaxed.

## Package and native observations

Final candidate **0.187.1 /241** was packaged from clean source
`f57308daf7aa05eb8eb8641878cf4baafa8208dc`; both binaries record
`vcs.modified=false` and strict ad hoc signature verification passes.
[The manifest](build-result.json) records hashes and preserved release status;
its launch flag describes packaging time. The subsequent
[native result](native-result.json) records actual launch and observations.
The qualified /205 release archive remains unchanged.

CUA checks of /240 observed the running binary counter, an enable-open hold
at 15 while time advanced from 2.934 to 3.774 s, and asynchronous reset to zero
at 4.604 s. The Simulate inspector showed 2 ms samples and explicit edge/reset/
restart guidance. Only the owned fixture's switch edits were discarded at its
normal protected Quit. Other app sessions and user projects were preserved.

Native inspection found the supply-ground caption crowding the clock waveform
annotation. /241 moves only the two supply/ground coordinates; final D and
counter screenshots show separation. All five final exported PNGs were also
visually inspected for complete symbols, wiring, captions and parts lists.
The final /241 app remains running the clean decade-counter example in Auto,
with U1 selected, Logic category visible and Apollo inherited. Its observed
count 1 at 4.110 s agrees with 21 recognized rising edges modulo 10.

`artifacts.json` hashes the raw supporting evidence copied at this checkpoint;
this narrative is intentionally outside that raw-file manifest.
