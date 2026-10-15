# Native control and command inventory

The [source ledger](13-native-command-inventory.json) records **38 production native Go files, 313 source sites and 133 public command IDs**. Literal calls, request fields, registry declarations and reviewed native aliases reference 110 public commands. The remaining 23 IDs are listed explicitly; many are discovery/inspection queries, legacy aliases, waiting variants or dynamically routed asynchronous actions. That list is not automatically a list of missing features.

**115 sites require binding review.** The other categories are 172 literal routes, seven declared native-wrapper sites, 17 registry declarations and two menu sets. These are syntax sites, including overlapping factories and callbacks; they are not 313 unique visible controls or 110 runtime-qualified capabilities. No literal route names an unknown command.

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

Resolve the 115 dynamic/relay/custom sites against instantiated states and public parameters, and attach runtime evidence per user capability. Audit file Save/Discard/Cancel semantics, raw field commits/IME, full menu/keyboard/focus behavior, canvas gesture lifetimes and the framework's window lifecycle. Inspection/readout equivalence also needs explicit evidence where the UI consumes shared Go snapshots/functions directly. Full agent parity stays **partial** until those proofs exist; this inventory prevents source drift and makes the remaining work explicit.

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
