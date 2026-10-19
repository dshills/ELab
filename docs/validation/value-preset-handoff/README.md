# Value-preset field handoff repair · October 19, 2026

**Source repair validated; clean replacement remains next.** In /165’s production bindings, type unfinished text in the selected resistor/capacitor value field and click a preset. The field commits its draft, but the preset retains the old command revision and refuses; the circuit value remains unchanged. All four shortcuts reproduce this failure through the pinned retained widgets and actual event router. This is a core value-editing defect, not new feature work.

[Initial reproduction](before.json) retains four failures with original production source snapshots. [Final-test replay](final-test-original.json) runs the valid final tests against those same original files through a Go overlay; all four still fail. The worktree source is never replaced. Original files are byte-identical to clean `0e75875`. Initial fixture development errors (history-field name and incompatible minimum dimensions) were corrected before the retained reproduction. The first repaired run also exposed a fixture’s decimal-literal expectation for `100uF`; the existing parser retains different binary bits. Final comparisons use the separate public agent’s exact stored components, preserving that existing numeric behavior rather than relaxing a product tolerance.

The [repair](../../../desktop/value_apply.go) gives presets the existing guarded Apply settlement path. Only accepted changes from this field, object and document may continue its original action context. A preset keeps its explicit requested value while the field finishes; Apply continues to use the settled draft. Agent/other-field changes, replacement, failed settlement and duplicate completion retain their guards. Native control IDs, labels and the shared `component.set` command are unchanged.

Validation on final source:

- [Mounted regression](../../../desktop/value_presets_offscreen_test.go): four typed-draft/preset clicks match independent in-process agent values exactly, preserve wiring and undo with one circuit history entry. A held preset refuses a later agent edit; a fresh click succeeds.
- [Deferred regression](../../../desktop/value_presets_test.go): explicit 2 kΩ survives a deferred 9 kΩ draft and duplicate completion. Eight cases cover agent-before/after, selection/document replacement, another user field, a rejected draft and completion/admission errors. Existing Apply and correction tests also pass.
- [Full quality](quality.json): `./scripts/check.sh` exits 0 in **207.671 s**, including formatting/imports, Staticcheck, audit freshness, 54 Python checks, root/native race tests and vet, and the retained-widget family already included by the standard runner.
- [Development native smoke](native-smoke.json): separate modified-source executable completes the existing general smoke in **42.681 s**, exits 0 and is absent. This checks integration; it is not physical OS preset input or a clean review qualification.

[Inventory review](inventory-review.json) preserves all command/alias/route sets: **69 native sources, 393 sites, 162 public IDs, 132 routed IDs**. Apply/preset factory calls are explicitly recorded as requiring binding review; labels do not invent routes. The 167 dynamic/review-category sites are not an untested-binding count. [Preservation](preservation.json) verifies all 788 /165 manifest entries, its receipt/ZIP and the canonical /120 native binary. No sealed package was changed or new release created.

[Checkpoint](checkpoint.json) hashes final source and retained evidence. Logs/build info are losslessly compressed; producer scripts are included. The SDK, models, project format and public API remain unchanged. The current packaged /165 has this newly reproduced first-click defect; source passing does not close replacement qualification, general typing/IME/accessibility, visible latency, owner walkthrough or other PRD gates. Owner status remains “Not tried yet”; no promotion.

Read-only check:

```sh
python3 -B research/ui-audit/first-version/verify-value-preset-handoff.py
```
