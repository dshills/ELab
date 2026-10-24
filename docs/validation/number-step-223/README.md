# Decimal number steps /223

[Receipt](report.json), [frozen inputs](source-inputs.json),
[raw evidence](raw.tar.gz), [member hashes](raw-manifest.json).

The /222 native playback control displays 0.031999998 after typing 0.03,
clicking plus and pressing Up. Existing binary addition has accumulated into a
different float32, so lowering text precision would hide a stored-value mismatch.
Two framework and two real live/recorded playback regressions reproduce it.
The initial live test's conflicting isolated minimum width is a retained harness
failure, corrected to fit the production maximum before the genuine baseline.

Managed steps now add the displayed value/increment as exact decimals and round
once to the existing native float32. The accepted native value and round-trip
text agree. The public command types remain unchanged; supplied high-precision
values and old saved values are not shortened or migrated. Seven driver cases
cover repeated positive/negative steps, off-grid values, page steps, both wraps
and a tight finite boundary. Both production playback cases pass with single
shared-command delivery and unchanged circuit topology.

Focused draft/wheel/precision checks and complete framework/application gates
pass. Every one of 8,439 SDK members matches its manifest; only the factory line,
helper, test and generated provenance change. Dependency, notices and bootstrap
bytes are unchanged. All 8,437 previous SDK cache members, native /222 bytes and
the /205 download are preserved. Source checks do not qualify a fresh native
candidate; changed-candidate native acceptance follows separately.
