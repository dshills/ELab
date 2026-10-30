# Experimental Native UI validation

Branch: `exp-ui`. Base: `f2a6cd10`. No merge or push.

The rebuilt native workspace follows the four supplied images in `exp-design`.
It preserves real simulation, documents and shared commands. The draft assumes a
browser and several capabilities that the existing application does not have;
[implementation notes](../../../exp-design/implementation-notes.md) distinguish
those deliberate adaptations from implemented UI work. This is experimental UI
qualification, not a new public release or acceptance of every draft requirement.

## Package

`build/ELab-0.204.0-302-Native-Experiment/ELab.app`

Executable SHA-256:
`3702379a8b8fd7411df9aee36cc7e7b199bac1914d63bda8ca8f2092bde1eab7`

The app has a verified strict ad-hoc seal. It is not Developer ID signed or
notarized; distribution audit remains pending. Earlier experimental candidates
and all previously qualified release/SDK packages remain preserved.

## What changed

- Full-height Parts and Inspector rails, centered document toolbar and a
  schematic/instrument split. Existing drawer resize/collapse commands remain.
- Floating tool and zoom controls, compact sampling notices and actual accepted
  probe readings on the sheet. No mock data or electrical graph changes.
- Consistent native typography, neutral surfaces, segmented navigation and
  compact inspector forms across Light, Dark, System and Apollo.
- Connect by name and Sweeps popovers; local document Save/Save As and exports.
- Scope/Spectrum/Sweep organization with advanced editors in bounded rows and
  stable button slots. Inspector scrolling stays vertical; compact chrome yields
  title space before hiding any command.

The source inventory retains the same 229 public commands, 63 alias targets,
189 command names with source routes and 40 without source routes. This is source
coverage evidence, not a claim of complete runtime route coverage.

## Test evidence

See [verification.json](verification.json) for exact local log paths and hashes.
All Go runs used the pinned offline SDK, `GOWORK=off`, readonly modules and the
project-local cache. Race runs used `GOMAXPROCS=4` and `-p=1`. No simulation limit
was relaxed, including when earlier concurrent exploratory runs exceeded a budget.

| Scope | Result |
| --- | --- |
| Complete ordinary desktop race suite | 411 top-level tests pass; 403.407 seconds |
| Broad mounted/rendered suite | 159 pass, two initial failures; both repaired and re-run successfully below |
| Instrument layout/editing followup | Six top-level tests pass across their theme/example cases |
| Rendered closeout | 34 top-level tests pass, including spectrum, scope, named wiring and popup layout |
| Compact followup | Four top-level tests pass; final compact action followup also passes |
| Final inspector followup | One caption-dependent fixture needed a stable-ID update; targeted disclosure paint/expansion followup is separate |
| Portable root race run | Solver/workbench, conformance, transport and other packages pass; the initially stale source-ledger test passes after reviewed regeneration |
| Python tooling | 104 tests pass |
| Static checks | Imports/staticcheck, ordinary/rendered vet and final source-inventory race followups pass |
| PRD audit | Coverage validation passes: 27 requirements, 17 implemented and 10 partial; not product acceptance |

The complete ordinary run precedes the last compact-only sizing/scroll refinements.
Relevant mounted tests and native checks cover those refinements; no unchanged
solver or framework full gate was repeated merely to generate another receipt.
Earlier failed logs are retained locally and are not represented as green runs.

### Repaired defects and review

Internal review covered command origins, dirty-document replacement, popover
scene ownership, pointer routing, resize allocation, disabled/selected control
geometry and data purity. No external code-review service was used.

1. Reproduced rounded primary-action clipping, then fixed its outer slot and
   verified both painted rounded ends across themes.
2. Reproduced a queued Fit overwriting an intervening agent pan. Fit now captures
   the normal command/document guard; the actual owner-queue regression passes.
3. Reproduced native overlay click-through: Zoom during placement added a resistor.
   Both the AppKit bridge and Core fallback now exclude actual overlay bounds.
   Native retest changed zoom/tool with 17 components, 16 wires and a clean document.
4. Retired pre-modal prepared pointer frames by epoch and forced redraw on local
   popover transitions. Software bridge tests pass; this is not OS queue acceptance.
5. Replaced toolbar overflow of spectrum text editors with bounded rows. Real
   mounted frequency edits, focus-loss handoffs, peak selection and live updates pass.
6. Fixed native popup selectors that searched the main scene for popup controls.
   Popovers retain their own managed scene routing; terminal options open and select.
7. Reproduced a blank Sweeps viewport despite accessible descriptors. A bounded
   scroll allocation now paints the fields; mounted paint allocation and native
   operation both pass. A real 12-point AC sweep completed without changing the
   live source's 400 Hz setting.
8. Corrected dual Scope/Sweep selection, invisible Fit text, quick examples that
   referenced nonexistent menu aliases, selected-state contrast and compact
   horizontal overflow. All six quick examples load through `example.load`.
9. Preserved intentional selector own-choice continuation while refusing an agent
   revision. The former one-shot native-menu fixture now tests the actual popover
   callback route; mounted named-wiring tests independently exercise routed input.

The SDK and sibling Vellum checkout were not edited. App-local adapters handle
floating layout and native pointer exclusion; simulation and document semantics
stay in the existing workbench/service layer. Bundled glyph coverage is cached
immutably behind a synchronized rasterizer, and plot/pill rendering is pure.

## Native and visual evidence

[Design QA](../../../design-qa.md) records the normalization, comparison history
and explicit platform/state differences. Full comparisons contain the supplied
reference on the left and the implementation on the right:

- [Simulate/Spectrum](simulate-spectrum-comparison.png), final /301 full-size capture.
- [Wire/Connect](wire-connect-comparison.png), /299 capture of the unchanged popup layout.
- [Document/Examples](document-examples-comparison.png), /299 with the native menu verified open.
- [Sweeps](sweeps-comparison.png), /299 with a completed real 12-point AC result.
- [Toolbar detail](toolbar-detail.png) and [inspector detail](inspector-detail-comparison.png).
- [Compact Dark with both drawers and three instrument tabs](dark-compact-drawers.png), final /302.

Native checks used documented CUA APIs and the public local agent service only.
Fixtures used `--no-recovery --no-appearance`, bundled examples and in-memory
edits. Every owned fixture was closed normally; its original window geometry was
restored. No owner document was saved, no clipboard capture was needed and no
security setting changed. A stale desktop-tool element and local revision
conflicts were refreshed explicitly, not treated as product failures.

These checks do not close historical native acceptance items 3, 6, 9 or 11
(Save-panel first responder, controlled hardware-rate grip and two parser-origin
checks), nor establish universal accessibility/IME acceptance or latency. No new
claim about those unrelated gates is made by this experiment.
