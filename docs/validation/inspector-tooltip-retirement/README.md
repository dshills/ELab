# Inspector tooltip retirement qualification

October 25, 2026. See [implementation notes](../../../research/115-inspector-tooltip-retirement.md).
The elab4 compatibility SDK, reproduced failure, passing regressions, complete
application checks and clean-source /256 package/native receipts are recorded here.

The change closes only native TooltipStage popups originating in a retiring
managed subtree. It does not dispatch numeric Change, keyboard, mouse or other
ELab commands. The core popup-stack inspection lock is released before exact
stage deletion; no menu or retained sibling tooltip is dismissed. Reconciliation
moves surviving descendants before retiring obsolete ancestors.

The regression uses routed MouseMove and a controlled hover clock. Before the
fix, both visible bounds and arrow tooltips remain after removal. Afterward,
visible/pending cases, same-ID fresh owners and subsequent normal help pass.
The second regression verifies retained help/menu identities, same-ID owner
replacement and zero numeric Change callbacks.

The whole exported archive is checked against its manifest. The prepared SDK
verifies the ordered stage recipe, including new exact-input/output stage 111.
All 8,735 exported files are pinned; comparison reports only the new lifecycle
seam, binding, regression/recipe files and regenerated SDK documentation.
Previous /255 and qualified alpha hashes remain in preserved.json.

No ELab application Go source, public command, component model, project schema,
simulation limit or established electrical assertion is changed. The upstream
remains f9dee54, with compatibility source 2c349fc. Exact source/archive hashes
are in sdk.lock.json.

## Complete application checks

Lint (Goimports/Staticcheck), Python tooling, PRD ledger, root/desktop/rendered
race tests, ordinary/rendered desktop vet and native build all pass. No tests
were skipped or electrical assertions weakened. Exact commands, deadlines and
elapsed times are in quality-complete.json; compressed raw logs are retained.

| Suite | Top-level passed | Including subtests | Skips |
| --- | ---: | ---: | ---: |
| root-race.jsonl | 861 | 1940 | 0 |
| desktop-race.jsonl | 332 | 765 | 0 |
| mounted-race.jsonl | 453 | 1337 | 0 |

Framework checks cover full portable/ordinary-driver race, ordinary/tagged vet,
and focused rendered tooltip, lifecycle, menu, focus and number-field cases.
This is not a full rendered framework rerun. See framework-complete.json.

The six packaged example SVG/PNG exports and both WAV outputs match /255
byte for byte. PDF export also passes; PDF bytes are not compared because
metadata varies between exports. See replay-comparison.json.

## Completed package check

ELab **0.194.1 /256** is built from clean source
`fa733f1d74acb06f268030abfbfd38db5c122f57`, with its matching agent alongside it. Both build-info
receipts report that revision and `vcs.modified=false`. The unchanged SDK packager
records matching version, build, SDK release and pre-signing source-binary hash.
Strict ad-hoc signature verification passes. The qualified unsigned alpha remains
unchanged.

The packaged replay saves, fresh-reopens and exports all six latest electronics
examples as SVG/PNG/PDF. Both sound examples render and export matching WAV bytes
without autoplay. Artifact hashes and replay receipts are retained here.
The separately scoped native check is described in `native-notes.txt`; its UI
and state evidence are saved. The owned session was closed normally and its
endpoint is absent. Owner projects and existing applications were preserved.

## Native result and scope

Actual PWM bounds help was visible before an agent inspector-section change;
it cleared without pointer input. Fresh help appeared after restoring Inspect.
Loading the LED example with that popup visible also cleared it. Matching-agent
receipts show unchanged components, electrical revision and run during hover and
section changes. The owned native session closed normally. See native-notes.txt
and the before/after UI captures.

Separate existing observations remain: an offscreen retained numeric field can
position help at the window edge after scrolling, and some custom-mesh captures
are intermittent. This fix qualifies retired-inspector cleanup; it does not claim
to resolve those unrelated positioning/capture observations.
