# Native control and command inventory

The [source ledger](13-native-command-inventory.json) records **49 production native Go files, 380 source sites and 159 public command IDs**. Literal calls, request fields, registry declarations and reviewed native aliases reference 129 public commands. The remaining 30 IDs are listed explicitly; many are discovery/inspection queries, legacy aliases, waiting variants or dynamically routed asynchronous actions. That list is not automatically a list of missing features.

**142 sites require binding review.** The other categories are 212 literal routes, nine declared native-wrapper/menu sites and 17 registry declarations. These are syntax sites, including overlapping factories and callbacks; they are not 380 unique visible controls or 129 runtime-qualified capabilities. No literal route names an unknown command.

The ledger covers action/button/menu factories, package-level native menus, registry tables, callback fields/assignments, time-field factories, custom canvases, event routers, direct request command fields and shared dispatch bridges. Each entry retains its source, line, enclosing function and expression. File hashes and the public ID set make changes reproducible and reviewable. Both platform Go files are included; smoke/tests, framework internals and Objective-C event delivery are excluded and remain separate evidence scopes.

The standard build creates `build/elab-ui-audit`. To check the current sources:

```sh
./build/elab-ui-audit --verify research/13-native-command-inventory.json
```

To review a change, create a candidate at a new path:

```sh
./build/elab-ui-audit --output build/native-inventory-candidate.json
```

The tool never overwrites an existing output. Compare the candidate's routes and source changes before updating the ledger; regeneration alone is not a review. Root tests verify ledger freshness and reject unknown literal routes. Deliberate fixture changes exercise dynamic callbacks, field assignments, menu routes, excluded smoke code and source digest changes.

## Defect found during review

Ordinary `screen.action` buttons previously captured targets/parameters from the rendered view but obtained a fresh mutation revision at activation. An old Apply button could overwrite an agent's newer resistance; an old Delete could act on an obsolete selection. Actions now capture the view's command revision and document generation. A single immutable context is shared while constructing the view, avoiding repeated full snapshot cloning per button.

The optional public request `expected_document_generation` guard checks document identity atomically alongside the required revision guard. It also rejects an action rendered after background-open admission but before document publication, when command revision alone remains unchanged. Successful retry keys still return their original retained response; this guard is context, not part of command intent. Existing clients without it remain supported.

Native regressions prove stale Apply/Delete refusal and normal fresh Apply. A controlled blocked-loader regression proves the publication-only case, and core tests cover optional/required guard behavior and retry preservation. [Release evidence](../docs/validation/native-inventory-validation.json) records the broader checks. This repair covers ordinary action-factory buttons; it does not establish every raw field, registry wrapper, native menu, pointer gesture or framework lifecycle path.

## Remaining qualification

Resolve the 142 dynamic/relay/custom sites against instantiated states and public parameters, and attach runtime evidence per user capability. Audit file Save/Discard/Cancel semantics, raw field commits/IME, full menu/keyboard/focus behavior, canvas gesture lifetimes and the framework's window lifecycle. Inspection/readout equivalence also needs explicit evidence where the UI consumes shared Go snapshots/functions directly. Full agent parity stays **partial** until those proofs exist; this inventory prevents source drift and makes the remaining work explicit.

The 0.46.0 lifecycle checkpoint explicitly inventories `Window.CloseRequest`, preserves active file operations/decisions on repeated close, and adds File Close / Cmd+W through `app.close`. See [close evidence and remaining OS/runtime limits](../docs/native-close.md).

The 0.47.0 field-context checkpoint binds assembled Change callbacks before native selector adaptation, preserving rendered revision/document guards. [Field semantics and remaining mounted-runtime limits](../docs/rendered-fields.md) qualify the specific repaired paths without treating every dynamic site as executed.

The 0.48.0 PDF-output checkpoint adds shared page settings and export routes; source references are still distinct from mounted runtime qualification. [PDF output and remaining printing scope](../docs/schematic-pdf.md).

The 0.49.0 PNG checkpoint adds one public image export route through the existing guarded picker/worker bridge; the shared Output page controls retain PDF settings. This ledger still measures source routes, not universal interaction coverage.

The 0.50.0 rectifier checkpoint adds a protected example alias through the registry and both Examples menus. The menu/value/scope paths are tested against an agent; source counts alone remain insufficient for full parity qualification.

The 0.52.0 annotation checkpoint changes the schematic mesh consumer without adding a semantic ingress: native rendering reads the same owned source lines available to agents and exports. The source hash was reviewed/refreshed; route and binding-review counts are unchanged.



The print-preview checkpoint adds guarded start/refresh/cancel/close controls and an exact-ID read-only image query. Native decoding uses the same public chunk command as an external agent; source mapping remains separate from mounted image/pointer/OS evidence. See [preview contract](../docs/print-preview.md).

The DC sweep checkpoint adds parameter/range/multi-probe controls, a separate parameter-axis mesh, exact measured-point cursor, stored curve selection/keep/export and navigation routes. Cursor events retain the painted operation/revision/document context. Source routes are reviewed against the shared dispatcher; runtime and physical input qualification remain separate. See [DC sweep contract](../docs/dc-sweeps.md).

The standalone sweep-import checkpoint adds one guarded picker route plus a waiting/background command pair. Native import/point/save behavior is compared with agent commands; imports retain source identity without claiming current-source freshness. See the [DC sweep contract](../docs/dc-sweeps.md).

Recovery setup controls now submit `recovery.configure.start`, with shared progress/cancellation and installation outcome. Existing waiting `recovery.configure` remains source-routed at startup; setup directory/scan/lease cleanup run outside the editor mutex. Site counts are unchanged, with one more public/routed ID; syntax counts remain separate from runtime proofs. See [setup contract](../docs/recovery.md#background-recovery-setup).

Native snapshot dismissal now routes to `recovery.dismiss.start`; its waiting compatibility ID remains available to agents. The source-routed set changes from the waiting ID to the immediate ID without changing site or routed-ID counts. The visible removal outcome uses shared project-work state. See [dismissal contract](../docs/recovery.md#background-recovery-dismissal).

The logarithmic-sweep checkpoint adds one shared spacing selector and its `sweep.settings.set` route. The mesh and cursor now share normalized display positions for either axis; source references remain distinct from mounted/physical-input evidence.

The captured-run comparison adds a reference selector, unit-time entry and shared statistics-display toggle through `recording.compare.set`. The pure `recording.compare` query remains available independently of the current viewer selection. New source routes and callback contexts were reviewed; complete physical interaction/accessibility qualification remains separate.

Threshold timing adds captured probe/level/direction/interval controls and bounded result-page navigation through `recording.timing.set`; the pure `recording.timing` query exposes the same bracket measurements. Context guards and the captured ID protect stale callbacks. Cached pages are derived session state, cleared on operation selection. These syntax routes do not establish complete physical-input/accessibility coverage.

Component-value hints read the shared numeric domains already exposed through `parts.list` and filtered command discovery. The Inspector source hash was refreshed; semantic route/site counts are unchanged. Unit-aware values and expected underflow refusals are covered separately by native callback and packaged smoke checks.

Local printing adds printer/copy controls and captured-page submit, work cancel and retained spool-job refresh/cancel routes. Native callbacks use the same shared command contracts as agents. Actual libcups is exercised against an isolated IPP fixture; no physical printing acceptance is implied. See [printing](../docs/printing.md).

Captured-reference overlays add a checkbox through the existing `recording.compare.set` command and read bounded captured traces from the shared recording view. Native scopes use shared axes and dashed reference geometry; source/callback mapping is separate from full physical-input/accessibility qualification. The narrow-interval axis/cursor repair is covered by a native geometry regression.

Scope pointers now retain prepared interval/run/capture/cursor/revision context before owner dispatch. Recorded pointer/time-field requests optionally pin the selected capture through the shared cursor command; drags bridge only their own successful writes. The conditional bridge remains a reviewed dynamic site, with native/public/GPU evidence scoped in [scope input context](../docs/scope-input-context.md); OS presentation and complete physical gesture qualification remain separate.

Native protected Close conditionally maps `app.close` to `app.close.start` in `requestReplace`, preserving the existing pending Save/Discard/Cancel identity. That dispatch-variable path remains a binding-review site; the literal-source list is not an inferred proof of the final alias target. Shared shutdown acknowledgement has two native literal ingress sites.

The 0.79.0 captured-verification guidance checkpoint adds a disclosure Change callback and its literal `diagnostics.details.set` route. The ledger now has 359 syntax sites: 202 literal routes and 131 binding-review sites, with other categories and 124 referenced public IDs unchanged. Native unit and owned packaged callback smoke qualify this specific disclosure; complete mounted keyboard/IME/accessibility and other dynamic paths remain separate. Failure explanations come from the same operation summaries agents inspect and do not invent current-object markers.

The 0.80.0 finite-pulse clock checkpoint adds frequency, period and high-phase duty callbacks through `source.pulse.timing.set`. Review found six new sites (three callbacks/factories and three literal ingress calls), 155 public IDs and 125 referenced IDs. Counts are now 365 sites, 205 literal routes and 134 binding-review sites. Each new field has an intervening-agent rejection test, and packaged waveform smoke invokes all three fields. These checks qualify the stated semantic paths without claiming complete physical typing/IME/focus or accessibility coverage.

The 0.81.0 shared ideal-square clock checkpoint adds `simulation.pulse.timing.set` and extracts three existing field callbacks into `pulseClockControls`. Reviewed routing chooses the finite or shared command from the captured timing kind and retains the command/document guard. The extraction replaces three finite literal ingress sites with two helper ingress sites: 364 total sites, 204 literal routes, 134 binding-review sites, 156 public IDs and 126 referenced IDs. Native callback tests and packaged smoke cover all three shared fields; a native/headless public blink workflow checks the new command. The source mapping does not establish every instantiated binding or physical input behavior.

The 0.82.0 exact waveform-entry checkpoint changes sine parameters and finite pulse levels to text commits through the existing public source commands. Review covers six concrete parameter fields, the shared guarded pulse commit relay, and the mechanical `timeParams` → `textFieldParams` rename in six other files. Consolidating pulse time/level commits removes one duplicate literal ingress: 363 sites /203 literal routes; 45 files, 156 public IDs, 126 referenced IDs and 134 binding-review sites remain. Tests cover exact/no-op/invalid/stale commits for each new field; packaged waveform smoke invokes them. Neither the relay inventory nor callback tests establish complete physical keyboard/IME/accessibility behavior.

The 0.83.0 tiled-output checkpoint adds `print.preview.page.set` and two native page-navigation routes, two settings Change callbacks/ingress calls and a shown-page PNG Activate/file ingress. Review covers 371 sites /208 literal routes /137 binding-review sites, 157 public IDs and 127 referenced IDs across 45 files. Native tests exercise the settings/navigation/file action; packaged smoke loads and navigates page images. Source/callback evidence does not establish all raw picker, focus, IME or accessibility behavior.

The 0.84.0 keyboard-construction checkpoint replaces per-part buttons with one Vellum ListBox and extracts placement/named-wire controls into two reviewed production files. The list commits the existing `drawing.tool.set`; search, coordinates, placement, terminal selection, connection and Escape retain their guarded public routes. Native focus continuation is presentation following successful user commands, with no new domain capability. The extraction adds one literal route and two binding-review sites: 47 files /374 sites /209 literal routes /139 binding-review sites; 157 public IDs and 127 source routes remain. Tests cover stale selection, empty matches, failed placement and focus order. A mounted keyboard-only LED construction/edit/save provides bounded physical-input evidence; full keyboard, native-dialog focus, IME, screen-reader and participant qualification stay open. See [workflow and observations](../docs/keyboard-construction.md).

The 0.85.0 interval-measurement checkpoint adds one native production file with a captured-probe selector and guarded start/end time fields. All commits use `recording.statistics.set`; readouts consume the same cached derived result returned by the pure `recording.statistics` query. Review covers 48 files /378 sites /159 IDs /128 source routes /141 binding-review sites /211 literal routes. Source counts do not establish physical typing, accessibility or broader input acceptance. See [electrical measurement contract](../docs/interval-measurements.md).

The 0.86.0 RC-network checkpoint adds the native-only `example.led-fade` menu/registry alias, explicitly mapped to `example.load` with `{id:"led-fade"}`. Its guarded replacement preserves the ID through Save/Discard/Cancel. Source review covers 49 files /380 sites /159 public IDs /129 source routes /142 binding-review sites /212 literal routes; the two menu sets containing a native alias now classify among nine declared wrapper/menu sites. No public command was added per template. Native/agent transient and protected replacement tests qualify this slice, not exhaustive runtime bindings. See [RC-network contract](../docs/rc-led-networks.md).

The 0.88.0 MOSFET checkpoint reviews five changed native source hashes: symbol drawing, generic example aliases, application/toolbar menus, region readouts and plot shortcuts. The source ledger still covers 49 files /380 sites /159 public IDs /129 referenced IDs, with 142 binding-review, 212 literal, nine wrapper/menu and 17 registry sites. `example.nmos` declares the same public `example.load` target as LED fade. The small alias loop captures one template per registration; native tests exercise both aliases and unsaved replacement. The new device uses existing part selection, placement, wiring, value, probe, simulation and output commands. This is source/binding test evidence; it does not certify every mounted input path.
