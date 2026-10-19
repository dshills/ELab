# Recorded timing correction · feedback/layout handoff

The clean **0.147.0 /164** candidate fixes ordinary private threshold/direction changes, but correction after an invalid entry still refuses the direction. Its error disappears while related fields settle, the resulting canvas resize publishes a new viewport revision, and the direction command retains the previous revision. **/164 remains unqualified.** A focused source repair now keeps this same field's feedback visible until the pending action finishes. Fixed physical replay and a new clean replacement remain required.

## Exact candidate observations

[Build identity](candidate/build-result.json) pins clean `eef34ce`, native/agent hashes and version /164. The [native identity](native-original/identity.json) records exact candidate bytes in the isolated disposable bundle; public commands seed one pulse-NPN recording. This is not an intact-bundle, from-blank, reader or general typing qualification.

The [full observer report](native-original/report.json.gz), [UI conclusion](native-original/ui-conclusion.json), [producer](native-original/observe.py) and input requests preserve ten stages:

- Private `3 V`, then first-menu AX Rising, matches the packaged agent at accepted 3 V/Rising. Private `4 V`, then first-menu coordinate-opened Falling, matches at 4 V/Falling. Each uses exactly two accepted timing commands. [AX result](native-original/AX-3V-rising.jpg), [pointer result](native-original/pointer-4V-falling.jpg).
- Invalid `bad` and a direction choice preserve accepted 4 V/Falling and show a refusal. [Refusal](native-original/invalid-direction-refused.jpg). Its own error-row resize changes viewport revision; the timing measurement remains unchanged.
- Correcting privately to `3.5 V` preserves accepted 4 V/Falling until choosing Rising. That choice commits 3.5 V but leaves Falling selected, with “expected command revision 16; current command revision is 17.” [Correction result](native-original/corrected-3.5V-rising.jpg). The independent agent accepts 3.5 V/Rising. The native report remains false; successful earlier cases do not qualify this failure.

Graph, physics, document and history stay unchanged during measurements. Exact original capture metadata/pages and all 521 saved rows survive; native/agent saved comparisons exclude only view and independent capture provenance. [Native save](<native-original/Native timing.elab.gz>), [agent save](<native-original/Agent timing.elab.gz>). Fresh-process reopening is not claimed here. [Cleanup](native-original/cleanup.json) confirms original PID 85110 closes normally at 0, is absent, and the stopped disposable bundle is restored from all three checked backups. The owner's app/projects are untouched.

The [candidate manifest](candidate/CANDIDATE_MANIFEST.json) preserves /164's four app/agent files as unqualified; it is not a release receipt. The runtime/four-adapter qualification was not started after this failure. Prior source checks retain their earlier scope.

## Source repair and checks

The final [original-source overlay regression](original-final-regression.txt.gz) fails both correction/agent subcases against the original /164 action implementation: the accepted corrected field hides the still-needed refusal. The fixture uses the real viewport publisher to continue the refusal through its initial error-row layout, then defers the remaining field completions. It does not simulate a passing physical layout. [Original implementation](source-original/recorded_timing_actions.go.txt), [overlay](original-overlay.json).

The [repair](../../../desktop/recorded_timing_actions.go) tags the originating refusal with its structured-help identity. A pending same-recording/document action may hold only that exact current refusal, including its own accepted viewport continuation. Only the action's own accepted field responses advance the held feedback revision. Completion releases it before the final direction/page command; new field errors, settlement failures and intervening agent edits retain their existing refusals. No stale request is advanced to a later actor's state.

[Focused race tests](focused-final.txt) pass in **8.773 s**, covering correction after real refusal-layout publication, agent intervention, distinct unrelated-request help and replacement of held feedback by a settlement error, plus existing direction/page/menu/mounted/viewport guards. [Full source quality](quality.json), [output](quality.txt), pass in **197.947 s**; [development native smoke](native-smoke.json), [output](native-smoke.txt), pass in **40.193 s**. Full quality includes formatting/imports/Staticcheck, audit freshness, 51 Python tests, root/native race/vet, mounted families and tagged vet. General native smoke does not establish fixed physical correction or clean replacement integration.

The [inventory review](inventory-review.json) changes only the action implementation's production hash and line positions. Every non-line site, command/alias/route set and the **69 files /391 sites /162 public IDs /132 routed IDs** remain unchanged. [Source identity](source-identity.json) pins the current action and tests. Models, SDK, project format and public commands are unchanged.

[Preservation](preservation.json) verifies all 24 previously qualified reviews, the original unqualified /147 and failed /164 candidates, canonical /120 and the SDK lock. The [earlier direction-repair evidence](prior-source-proof.json) remains byte-for-byte intact; its current-source verifier is historical after this new action change. Raw outputs, metadata, duplicate ledgers and synthetic saved projects are losslessly compressed without trimming.

Owner walkthrough remains Not tried yet. The next deliverable is a clean replacement with this correction, the exact native private-entry/refusal/correction replay, copied-runtime/reference/agent integration and an updated handoff. Broader reader/IME, visible p95, clean environment, exhaustive conformance and participant gates remain open. No candidate is promoted.

Verify without launching an application:

```sh
python3 -B docs/validation/recorded-timing-feedback/verify.py
```
