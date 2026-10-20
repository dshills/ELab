# FLOATING-INPUT-01: source voltage in network transient samples

**A 5 V floating DC source could report 4.524 V or 0.476 V as `input_v` after
one 1 ms step.** The scalar used the positive node relative to ground, while
the actual source difference and differential probe correctly remained 5 V.
After Power off it could even report a nonzero negative input. Original /176
reproductions remain in `public/original/`; no package is modified by this fix.

The source repair uses the solved positive-minus-negative voltage when reporting
an accepted network sample or switch edge. Reading the solved state also keeps
the accepted left/right pulse sides; recalculating the configured waveform at an
edge would lose that distinction. Initial and manual-override sample creation
already used the correct source stimulus. No solver equation, integration budget,
model, command, schema, native binding or saved format changes.

The new regression covers four shifted RC circuits: ground at capacitor pin 1
or pin 2, each with/without a series switch. Through shared service commands it
checks 20 ms charging against the independent RC exponential, every retained
scalar against the 5 V differential probe, queued switch application, Power off,
unchanged accepted history and run/document/history identity. The final test
fails against both original production files via a Go overlay, then passes in
the focused race suite on corrected source. Existing grounded waveforms and
mixed-network cases remain in that suite and the full quality gate.

The final default quality gate passes in **359.596 s**: Go format/import checks,
Staticcheck, all 65 Python tests, root/desktop race tests and vet, plus registered
mounted offscreen checks. All 651 application Go source files are unchanged
through that gate. This wall duration is a check-run record, not a performance
acceptance result.

The same four cases are built from blank using public `edit.batch`, ordinary
simulation/switch/probe commands and waiting `app.close`, first against /176 and
then the development agent. All eight owned headless processes close normally:
**46 requests before and 46 after**, with no protocol errors. Ten paired Step
snapshots have identical graph, solver results, scope, document identity, history
and transient fields except `input_v`. The final runs have zero scalar/probe
mismatches; originals retain the actual mismatches. No new native window or
physical input/latency/owner acceptance is inferred.

`report.json` pins the source and artifacts; `quality.json` and compressed logs
retain the full quality result. `sources/` includes original/final production
files, the final regression and the exact public producer as archived text.
The runtime profile keeps repository/SDK/toolchain/cache/shared geometry access
and outbound IP denied for the copied agents. All fixture data is synthetic and
private. Existing canonical /120, review /176, ZIP and SDK are preserved.

Run `python3 -B docs/validation/floating-source-input/verify.py` for read-only
integrity, raw-protocol/analytic replay, before/after field comparisons and
source/artifact pins. It starts no process or UI and performs no network IO.
Current-source pins intentionally stop matching after later legitimate changes;
historical raw evidence remains scoped to these inputs.

This is a **source-qualified repair**. A clean replacement package and its
copied native/agent checks remain the next release step; /176 remains affected.
Owner walkthrough and broader first-version/MVP gates remain open.
