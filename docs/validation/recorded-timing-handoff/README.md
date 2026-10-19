# Recorded threshold paging preserves field edits

**Source repair passes; clean replacement qualification remains next.** In /162, typing a valid threshold and clicking Next arrivals accepts the field, then refuses paging at the old command revision. Typing an invalid threshold can instead advance the old page and clear the validation error. The final mounted tests reproduce both failures against the original production source and pass after repair.

The paging action now waits for the four threshold/interval text fields. Only consecutive accepted responses from those fields, for the same recording and document, can continue its original context. Parameters come from those accepted responses, rather than a later agent snapshot. Agent edits, another field, recording selection and document replacement still invalidate the action. A related failed field keeps its error and blocks paging until correction. Previous/Next derive their page from the accepted settings; no more arrivals produces explicit feedback.

The same existing `recording.timing.set` command performs all changes. Commands, numerical models, project format and Vellum SDK remain unchanged. This is modified-source evidence based on `dd3dfd1`; the preserved /162 package does not contain the repair. No app has been promoted.

## Evidence

- [Original source with final mounted tests](original-final-regression.txt): valid `3 V` commits revision 6→7 but leaves page offset 0 with a conflict; invalid text advances to offset 8 and clears its error. The [original overlay](original-overlay/overlay.json) replaces only the two original production files, excludes the added repair file and supplies the fixture helper. Original bytes match /162's native inventory. This is an isolated Go source test, not an intact app-binary or physical OS-input test.
- [Final focused race checks](focused-complete.txt), [mounted tests](../../../desktop/recorded_timing_offscreen_test.go), [deferred/refusal tests](../../../desktop/recorded_timing_actions_test.go). A separate in-process agent supplies the normal two-command result. Tests preserve live graph, physics, document and history; they check four-field settlement, duplicate callbacks, agent-before/after, unrelated field, capture/document replacement, invalid input, settlement admission/completion failure, Previous, no-more-arrivals and first-click correction.
- [Full source quality](quality.json), [raw output](quality.txt): **187.546 s**, formatting/imports/Staticcheck, current audit and inventory, 51 Python tests, root/native race/vet, mounted cases and tagged vet. Unchanged backend suites reuse valid Go cache entries; native tests take 142.673 s and mounted tests 20.234 s. Existing duplicate Objective-C library warnings are retained. The new mounted family is included in `scripts/check.sh`. [Development native smoke](native-smoke.json) also passes in 38.968 s; [output](native-smoke.txt) and exact [modified build metadata](native-build-info.txt.gz) are retained. This is general runtime regression evidence, not physical timing-field input.
- [Patch](source-repair.patch.gz), [source identity](source-identity.json), [inventory review](inventory-review.json), [preservation](preservation.json). Source inventory changes 68→69 production files and 393→392 sites; all 162 public IDs, 132 routed IDs, aliases and issue sets remain unchanged. Dynamic site classification is not a count of untested bindings or a conformance percentage.
- [Fixture corrections](fixture-corrections.txt) distinguish the initial path/layout/hidden-control/helper/omitted-field errors from product failures. The early [valid-input reproduction](before-reproduction.txt) and the partial repair's [invalid-input failure](mounted-all-fields.txt) remain retained; the final original-source replay establishes both shipped-source failures independently.

## Limits and next step

The mounted test extracts production fields/buttons into a small layout and drives the real retained Cogent controls with a manual router. Deferred native settlement also has explicit completion/error tests. This does not establish physical Cocoa typing/menu/IME behavior, full-layout reachability, visible latency or exhaustive agent conformance. Probe/direction selectors and other forms retain their separate scopes. No new saved-capture/recovery or physical-print qualification is claimed.

All 23 sealed reviews and canonical /120 remain byte-for-byte intact. Source/native inventory intentionally differs from /162 now; its prior clean-runtime evidence remains historical. Owner remains **Not tried yet**, P0 stays 17 implemented /10 partial. Build one clean replacement, then repeat copied native/public integration and the timing command sequence before promotion.

Recheck retained evidence without launching an application:

```sh
python3 -B docs/validation/recorded-timing-handoff/verify.py
```
