# Five electronics capabilities

October 27, 2026. Implemented in order with electrical reference tests, working
examples, shared user/AI commands, and complete Prism reviews through Anthropic
Claude Opus 5.5. No repository push, release upload or solver-limit increase. Review diffs were
sent to the explicitly authorized Anthropic provider.

| Item | Implementation commit | Evidence and review dispositions |
| --- | --- | --- |
| Class-AB plus speaker stepping | 361b77ba | [Transient](transient/README.md) |
| Capacitor/inductor losses | 92372827 | [Passives](passives/README.md) |
| Current sources and electronic loads | fa12ddfa | [Current devices](current-devices/README.md) |
| Configurable N/P JFETs | 89a367f7 | [JFETs](jfets/README.md) |
| Loop return ratio and captured step response | 9984b36c | [Analysis](analysis/README.md) |

Combined regression corrections are committed as 8976ae93. [Review findings,
reproductions and focused checks](regressions/README.md). Final application,
package and runtime receipts are recorded below.

The catalog has 58 component kinds, 209 shared commands and 78 examples. Loop
return ratio uses controlled-source injection in configured dynamic op-amps;
ordinary signal gain and all-discrete physical-wire loop injection are distinct.
Margins are sampled estimates, not hardware/Nyquist stability certificates.
Captured step response requires explicit initial/final references, tolerance and
hold; it measures observed data without inferring an unseen final steady state.

Routed/software tests do not qualify native OS queue or hardware-rate acceptance.
Original native checks 3/6/9/11, participant and clean-account gates remain open.
Old qualified downloads/candidates, owner projects and sessions remain preserved.

## Combined source checks

[Application matrix](qualification/application-report.json): the complete portable
race invocation passed every package except three relay/sensor time-guard cases.
All three passed together in an exact-source isolated race control (3.327 s).
This is aggregate coverage, not a falsely clean single invocation. Raw full and
isolated receipts are retained. All complete ordinary/offscreen desktop suites
passed (361/498 top-level; 885/1556 including subtests) before the narrow resistive
fallback; final focused controls pass both drivers after that correction. All vets,
lint/imports, 77 Python tests and PRD/source-inventory verification pass. No solver
limit changed. Native OS queue/hardware acceptance is separate.

The first portable attempt had blocked Unix sockets, obsolete test metadata and
a preview-test panic. The corrected full run completed all tests; the focused
controls close its three resource-time refusals. A mistakenly named inventory
script was a controller-only failed invocation, excluded from passing evidence;
the actual Go inventory verifier is used.

## Exact /277 package

[ELab 0.199.0 /277](../../../build/ELab-0.199.0-277-Electronics/ELab.app)
and matching agent are built from clean source `92d52f48d9fdf78f1c926a21183c58d4434de3e7`.
[Package identities](qualification/package.json), [summary](qualification/summary.json),
[archive and SHA-256](qualification/archive.json). The unsigned/ad-hoc candidate is
local; no release promotion, repository push or public upload occurred.

All 78 examples reopen paused and clean with exact graphs using the matching
packaged agent. All 134 old project/SVG files and all 67 old catalog entries
remain exact. Eleven new projects/SVGs extend the copied catalog.
[Example receipts](qualification/examples.json).

The exact matching headless agent and copied native service both complete all
five workflows: a 101-row all-discrete speaker capture; capacitor/inductor loss,
current/load and both JFET parameter edit/undo/redo/save/reopen; a CP capture;
a 201-point loop-return sweep preserving live state; and a 1001-row RC capture
with 3.912070926 ms observed settling and exact saved-capture recomputation.
[Headless](qualification/headless-checks.json),
[native](qualification/native-checks.json), [controller](qualification/new_workflows.py.txt).
The native receipt's agent hash identifies its sandbox forwarding wrapper; the
matching shipped agent is independently pinned by the package/runtime receipts.
These are shared-command runtime checks, not physical input or hardware proof.

The copied native smoke and all four conformance adapters pass under outbound IP
and development-file denial. A fixture-only denial preserves shared geometry
settings. All 102 external processes exit 0 normally. The separate owned feature
app also exits 0 and removes its endpoint. Owner sessions/projects/settings are
preserved. [Runtime](qualification/runtime-result.json),
[public suite](qualification/public-report.json.gz),
[owned close](qualification/native-final-process-state.json).

All 1169 frozen executable/test/dependency inputs and all 27 protected prior
package/SDK identities remain exact. The final 42,508,652-byte archive verifies
all 412 payload files, modes, OS extraction and strict ad-hoc seals. Gonum BSD
notices are present in both the app-bound notice bundle and outer Licenses.
The notice inventory remains an inherited rights-review record, not a new legal
clearance. [Seals](qualification/final-signatures.txt).

The initial workflow controller used `current` instead of the actual documented
`current_source` model metadata key. Its KeyError and initial fixtures are
preserved and excluded from passing evidence; the corrected controller completes
without a product change. Build output retains the existing duplicate `-lobjc`
linker warning. Physical native checks 3/6/9/11 remain open.

Readable Go build-info receipts trim trailing tabs emitted by `go version -m`;
the original exact output bytes are retained alongside them as `.txt.gz` files.
No executable, package or test input changed during that evidence formatting.
