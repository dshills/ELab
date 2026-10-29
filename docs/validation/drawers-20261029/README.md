# Docked drawer correction — October 29, 2026

The owner clarified that Parts and Inspector should be drawers, and chose moving
the canvas aside. This corrects the earlier toolbar-based sidebar presentation.
Each open drawer now has a visible Close control at its edge; closing leaves a
labelled reopen tab. Dragging an inside divider outward can snap the drawer
closed while retaining its width. Home closes Parts; End closes Inspector.
Focus returns to the persistent edge tab. The former document-toolbar copies
are removed, and controls remain above scrolling forms.

## Tests and review

The [frozen final run](report.json) passes all nine steps: new/related drawer
race checks, full ordinary desktop and rendered suites, lint, both desktop vets,
104 Python tests, maintained command inventory and PRD audit. Input hashes are
exact throughout. Earlier zero/one split-bound and intrinsic-header-width
failures remain retained alongside passing corrections. No test/budget was
weakened, and no SDK cache was edited.

Tests cover edge-positioned controls and reopening in four themes; both-sided
pointer and keyboard collapse, focus and retained widths; continuous multi-move
drags after the slide-time threshold; actual agent-interleaved stale closing
gestures and independent fresh input; and existing minimum-window drawing tools.
The old pre-change multi-move control passed too: no speculative gesture or SDK
contract repair was made. Retained fixtures replace only unsupported canvas
paint, not the managed panels or input routes. They do not prove hardware-rate
OS queues.

The native /286 toolbar toggles closed/reopened panels in pointer checks and the
original owner layout was restored. The revised native preview visibly shows
direct edge Close controls and reopen tabs. Its model changed independently
while on screen, so it is not claimed as a controlled electrical or interaction
acceptance trial. Preserve intervening owner work in that preview; it must not
be discarded for cleanup. It subsequently exited normally without a controller
discard command; its [normal exit](preview-normal-exit.json) is recorded. Existing
owner sessions were not closed by this work.

[Internal review](internal-review.md) covers decisions, corrections, focus,
guards and native limitations. All non-desktop executable/test/dependency inputs
are [byte-identical](unchanged-service-sdk.json) to the preceding checks. The
same 228 commands remain; no backend, component, model, solver limit or dependency
changed. [Fifty prior package/SDK identities](preservation.json), including /286,
remain exact. Historical native/human/install/signing gates are unchanged.

The [matching clean-source candidate](candidate.json) is **0.202.1 /287**, source
`2fa981d9e7f937f9f1c2dcaaba634c13ba783a5b`, at
`build/ELab-0.202.1-287-Drawers/`. Exact candidate native GPU/descriptor-command
smoke passes and exits normally. Matching [MCP checks](packaged-mcp.json) retain
all 228 commands, refuse stale/invalid requests and capture the analytic RC
voltage with both drawers closed. All 87 packaged example graphs and the strict
ad-hoc integrity seal pass. Both owned bridge and native smoke exit normally.
Fifty prior identities are rechecked after integration; all remain exact.

This is not a new full release qualification; the preserved qualified release
remains /285. No push, upload, external review or security change was performed.
