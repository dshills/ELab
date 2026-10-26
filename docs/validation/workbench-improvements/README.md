# Everyday workbench engineering checks

October 26, 2026. Implements the five ordered priorities in
[research/119](../../../research/119-effortless-workbench.md). The existing
53 component kinds, 62 examples, electrical models and solver limits are
unchanged. Seven shared semantic commands bring discovery to 194; no existing
command is removed. [User/agent workflow guide](../../workbench-workflows.md).

## Implemented behavior

1. **Drawing.** Snapped component ghosts, valid/invalid destination feedback,
   dashed pending routes and nearest unambiguous terminal acquisition. Crossings
   remain separate. W/V/H choose Wire/Select/Pan; R rotates the selected part,
   with text editing left local. Existing moves, rotation and connection Undo
   retain their commands. Read-only `drawing.preview` exposes the same plan,
   geometry, admission result and commit guards to agents.
2. **Understanding.** Selected-part explanations combine current/stale/unavailable
   accepted readings, connectivity, relevant existing diagnostics and published
   educational model limits. `component.explain` returns the same facts. No
   unknown solver failure is given an invented physical cause. Cached template
   differences and reused selected-object inspection avoid duplicate view work.
3. **Measurement.** Plot reveals and focuses the live scope, keeping every other
   trace. Full scope refuses a seventh slot with actionable shared guidance.
   Recorded view returns to live without losing its captured data. Existing
   cursor/window/recorded commands and numerical behavior remain intact.
4. **Experiments.** Active example questions, suggested edits, original expected
   observations and model limits are available in Simulate as well as Document.
   Exact component/wire/settings comparisons track edits and Undo. Expectations
   are explicitly about the original template, not fabricated edited results.
5. **Collaboration.** Atomic batch previews and session proposals show supplied
   purpose, actor and actual before/after differences, including nested device
   parameters. Existing affected objects are highlighted. Shared Apply/Refresh/
   Dismiss controls refuse stale sources and old proposal IDs. An actual apply
   is one ordinary named Undo operation; document replacement clears proposals.

## Complete source checks

The final run passes lint/imports, all vets, 74 Python tests, PRD checks and build.
Race suites pass **880 portable /347 desktop /484 rendered** top-level tests
(**1,963 /847 /1,518** with subtests), with no skipped tests. Three root packages
have no tests and are reported separately. Six unchanged portable packages use
their exact passing cache from the prior complete final-source run; both final
desktop variants are fresh. [Counts](test-counts.json),
[steps and source freeze](quality-report.json),
[1,371 frozen inputs](quality-frozen-inputs.json).

The [native source inventory review](inventory-review-final.json) records 99
sources, 460 sites and 158 literal routes. This is static mapping, not dynamic
every-binding/native queue acceptance. Existing SDK elab9 bytes are unchanged;
its previous complete qualification is carried rather than repeated. Thirteen
prior release/SDK identities, including qualified /270, are
[preserved](preservation-before.json).

Focused regressions cover all 53 ghost kinds, closest/tied pins, preview purity,
guarded commit/Undo, real accepted switch-state isolation, proposal lifetime ID
retirement, retry copies, cross-actor application, recorded/live source handling,
full-trace refusal and experiment comparisons. Actual retained-driver mouse
down/up applies proposals across Light/Dark/System/Apollo. Routed evidence does
not establish native OS queue or hardware-rate behavior.

## Visual evidence

[Light proposal](visual/light-proposal.png),
[Apollo proposal](visual/apollo-proposal.png),
[placement ghost](visual/light-placement.png),
[wire preview](visual/apollo-wire.png),
[occupied position](visual/apollo-occupied.png).

Proposal captures use the retained driver's GPU scene, with actual mounted
control geometry and input. Canvas review images rasterize the actual production
mesh triangles in a review-only software fixture; they are not native screenshots.
All twelve final canvas scenes match the previously inspected scenes byte for
byte. [Visual scope/identities](visual-identity.json),
[review-only renderer](preview-render.go.txt).

## Retained investigations

- Ordinary discovery first exceeds its existing 64 KiB limit. Shared output
  schema definitions reduce it to **64,969 bytes**; no transport limit is raised.
- Duplicated command-revision fields in derived explanations break failed-file
  operation comparisons despite unchanged project data. Those unnecessary fields
  are removed; accepted reading run/electrical/time provenance remains intact.
- Initial concurrent/controller work and isolated runs encounter explicit 100 ms
  integration-budget errors. An isolated unchanged /270 source control passes;
  matched current runs vary. Duplicate inspection work is removed and unchanged
  template diffs are cached. The complete final portable race run passes with
  the original models, precision, 512-attempt and 100 ms limits. This does not
  prove one cause for all earlier timing failures or universal performance.
- The legacy UI workflow assumes silent last-trace eviction. Its updated real
  selector flow explicitly removes that trace before Plot; the agent uses the
  corresponding public remove/plot commands. Preservation/refusal is separately
  tested, rather than weakening the new slot contract.
- A real accepted switch named `Preview` can lend its live position to a new
  default ghost. The successful reproduction is
  [before3](investigations/ghost-origin-before3.txt); neutral ghost rendering fixes
  it. Earlier fixture attempts incorrectly request transient stepping of DC
  runs; those setup errors are retained separately and excluded from the claim.
- A review-only renderer initially has a `.go` filename under build/, entering
  root package discovery. It is retained as `.go.txt` and compiled only through
  an explicit overlay. No product lint/test exclusion is added. Early artifact
  path/gofmt setup errors run no intended formatting/capture and are excluded.
- The count parser initially treats packages without tests as skipped tests.
  Final counts retain those package entries separately; actual skipped tests
  remain zero.

Failed/superseded records are in [investigations](investigations/). The final
quality recipe and raw race receipts are retained alongside this report.

## Qualification boundary

Clean candidate/native runtime identities and public workflow receipts are added
after packaging. This source report alone does not qualify a distribution.
Original native checks 3/6/9/11 remain open. New hover/shortcut hardware behavior,
participant ease-of-use, screen-reader/IME, separate-account installation and
public signing/notarization are not established by these source/driver checks.
No participant sessions are invented; the existing study packet stays unmeasured.
The heartbeat stays paused. No owner project writes, SDK/sibling checkout edits,
security weakening, push or upload are part of this batch.

## Packaged self-test adjustment

The initial clean /271 candidate correctly refuses the old smoke test's attempt
to plot a seventh trace. Its [failed sequence and preserved identities](investigations/native271/README.md)
remain recorded. The harness now explicitly clears its sixth visible selector
before Plot. No ordinary product command or behavior changes after source
checkpoint `699a475e`.

All **1,370 other frozen executable/test/dependency inputs** stay byte-identical;
the [narrow follow-up](smoke-harness/report.json) passes fresh ordinary/rendered
race workflow regressions, lint/imports, both desktop vets and build. Prior full
source gates are carried for those exact inputs, not relabeled a fresh complete
run after the self-test-only change. Final /272 native smoke and runtime evidence
will qualify the adjusted self-test itself.

## Runnable candidate /272

**ELab 0.196.0 /272** is available at
[the candidate app](../../../build/ELab-0.196.0-272-Workbench/ELab.app), with its
matching agent, all 62 existing projects and the workflow guide. Clean binary
source is `5fd9087344c12626880313c1d86a4508c8765ebc`; final documentation is a
later checkpoint. [Candidate identities](candidate272/272-package.json),
[summary](candidate272/summary.json),
[final strict ad-hoc seals](candidate272/final-signatures.txt).

The exact copied app passes its full native smoke in **41.130 seconds**. Both
that test and all four public conformance adapters run with development-file
and outbound IP denial; an additional fixture-only policy denies writes to the
owner's shared framework geometry folder. All **102 external conformance
processes exit 0 normally**. The copied bundle, agent and controller remain
exact. This testing restriction changes no platform security setting.
[Runtime receipt](candidate272/runtime-report.json.gz),
[public adapters](candidate272/public-report.json.gz),
[process cleanup](candidate272/external-process-exits.json.gz).

The new drawing/explanation/plot/experiment/proposal walkthrough passes through
both the exact copied native service and the matching headless agent. It checks
pure preview, a 2.916796577 mA LED reference, accepted explanation, plot reveal,
atomic diff, stale refusal, refreshed apply, one-operation Undo, template diff
and dismissal. The owned native feature fixture closes normally with exit 0 and
its endpoint removed. These are public-command workflows, not physical pointer/
keyboard acceptance. [Native walkthrough](candidate272/native-feature-new-workflows.json),
[headless walkthrough](candidate272/272-public-headless.json).

All **62 packaged examples reopen paused and clean with exact graphs** using
the matching agent; all 62 agents exit 0. All **126 example files** match the
qualified /270 ZIP. The thirteen prior qualified release/SDK identities and the
initial /271 candidate bytes remain exact.
[Examples](candidate272/272-examples.json),
[preservation](candidate272/preservation-after.json).

This is a runnable development candidate, not a promotion of the qualified
release or a new 30-minute endurance qualification. Source tests, retained-driver
captures and software geometry do not close original native checks 3/6/9/11,
hardware hover/shortcut acceptance, participant ease-of-use, IME/screen-reader,
separate-account installation or public signing/notarization. Those limits remain
as above. No owned test window remains open; no owner project, SDK cache or
sibling Vellum source was edited. No push/upload or heartbeat restart occurred.
