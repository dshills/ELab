# Internal review: side panels and Simulate tools

Reviewed the shared state/admission path, mounted layout, native chrome seam,
focus and scroll ownership, retained callbacks, existing-control reachability,
agent discovery and file/session boundaries. Review was internal; no external
provider or Prism was used.

## Implementation decisions

- Panel visibility and preferred logical widths live in the existing service,
  with `workspace.panels.set` using the normal command/document guards and retry
  identities. Both flags and both widths are required, avoiding ambiguous partial
  updates. Width bounds are identical in admission, detailed discovery and common
  schemas. A rejected request is atomic.
- `workspace.simulation.set` selects one existing task. Navigation neither
  launches/cancels work nor changes inspector visibility, accepted tool settings,
  results, recordings, electrical state or circuit history. The existing command
  catalog is retained; these are two additional presentation commands.
- Both panels remain visible initially. Choices survive document replacement
  within the session but are excluded from files and app-launch persistence.
  Preferred widths survive collapse and temporary window-size constraints.
- Nested managed SplitPanes reuse the pinned runtime's input/accessibility
  behavior. Layout reserves drawing space, and at a narrow window either edge
  can grow its panel by reducing the other visible panel to its minimum. Ordinary
  layout allocation does not write back to the service. A user resize can update
  both displayed widths in one guarded command.
- Hidden panels/forms are unmounted. Header controls remain available to reopen
  them. Closing transfers focus to the header; opening continues into the panel.
  The Simulate chooser remains outside the single managed form scroller. Its
  document/section/tool identity prevents a long form's offset from blanking the
  next shorter tool. Existing individual long forms still scroll.
- All inspector forms retain their IDs, callbacks and shared commands. Live
  sampling/base-drive controls stay under Live controls, independent of circuit
  details. The live Scope/Spectrum dock remains separate from recorded FFT tools.
  Print and sweep views keep contextual left controls and right task navigation.
- The app-local native style adapter suppresses redundant outer-host scrollbars
  only. Real managed Scroll nodes and the center workspace retain normal
  overflow. No framework cache, sibling checkout or dependency was changed.

## Corrections found during implementation and review

- Strict retained geometry exposed an incorrect divider-width assumption. The
  conversion now includes the pinned backend's complete 18-unit grip and margins.
- At narrow widths a fixed keyboard step exceeded the permitted split range.
  The step is bounded by that range. Drawing space and secondary-toolbar overflow
  are checked after constrained resizes, rather than inferred from constants.
- Initial grouping put LC sampling/base drive with circuit details. Existing
  focus/settlement tests exposed that regression; those controls now remain live.
- Native checks exposed redundant horizontal bars on private panel hosts and
  then on chooser navigation. Scoped overflow styling and the existing inspector
  field polish correct these without disabling real wheel/scrollbar operation.
- Existing fixtures that assumed all Simulate forms were simultaneously mounted
  now navigate the actual chooser. Their electrical, parser, settlement, timing
  and focus assertions remain. A union-of-controls test checks reachability of
  existing active forms across all tool choices.
- New tests cover stale held panel/chooser callbacks and an actual queued divider
  callback across an intervening public agent edit. They preserve the pinned
  runtime's existing contracts; no speculative SDK change was introduced.
- Full regression caught ordinary command discovery growing to 65,967 bytes,
  beyond its unchanged 65,536-byte budget. Sine/pulse input definitions now use
  the existing local schema registry instead of repeating nested waveform
  objects in the catalog. Input quantity/time grammar and detailed contracts
  remain; numeric output waveform types are separate. Existing schema-budget,
  reference-resolution, waveform and numeric-output tests cover this correction.
- The candidate includes the workspace-layout guide used by the updated agent
  documentation, so its panel/tool reference is available beside the app.

## Evidence boundaries

Retained tests use real managed panels, forms and dividers; the full-workspace
manual fixture substitutes only unsupported canvas-extension paint. These prove
driver routing/layout, not hardware-rate OS queues. Native CUA screenshots show
collapse/reopen, both panel edges, tool selection, wheel scrolling and an actual
scrollbar drag in owned disposable sessions. A highlighted menu item was committed
with Return before recording its result. Dependent AX toggles were observed and
settled individually rather than treated as a single successful batch.

The final native state had Parts width 229 and Inspector width 363 logical units,
Recorded runs selected and the owned RC circuit paused. Document state, electrical
revision and run ID matched the pre-interaction control. An initial controller
assertion guessed a narrower width interval and failed; the measured state was
retained instead of changing the product or reporting an exact unobserved width.
All owned native sessions closed normally. No owner project/session was used.

Full regression results and matching clean-source candidate checks are recorded
in the adjacent validation README. The matching MCP bridge exercises the same
new commands, preserves all old catalog IDs, refuses stale/invalid requests and
captures RC voltage with both panels hidden. Exact candidate native smoke and
all 87 packaged example graph checks pass. Its initial disabled-recovery smoke
invocation was a fixture configuration error, corrected without product edits.
Four historical native
queue/hardware checks, participant acceptance, clean-account installation and
public signing retain their existing separate status. This change does not
qualify those checks or replace the preserved /285 release.
