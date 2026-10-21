# Items required to complete the first version

Updated October 21, 2026. This is the remaining-work checklist, based on the current repository and recorded results. It does not claim that pending checks have passed.

## What “first version” means

The existing [delivery plan](34-first-version-plan.md) defines the first deliverable as a **private macOS hobbyist alpha**: draw, simulate, edit, measure, undo, save/reopen and export, with the same operations available to agents. This checklist retains that scope. The original [PRD](07-product-requirements.md) defines a larger MVP; its additional requirements appear separately below and are not silently waived.

Current position:

- **Owner version: 0.172.0 /189.** Its initial canvas, parts list and toolbar are owner-confirmed visible. The four owner workflow tasks have not been accepted.
- **Current review candidate: 0.178.0 /195**, built from clean `8d7d646344cf8a98a9638494fa0fe4d5661e2e3a`. The private-alpha engineering reference workflows, declared native correction classes, minimum/working-size reference UI, paired native/agent results and package/offline checks pass within their recorded scope. The owner answered “Not tried yet” for /195; that is neither a workflow pass nor failure. It preserves /189 and is not promoted. /194 remains preserved with its original blank-inspector failure.
- **No new component family or analysis feature is needed to finish this cut.** MOSFETs, transient simulation, probes, cursors, persistence, exports and printing already exist. Shipped features still need honest behavior and appropriate validation.
- The owner authorized desktop use on October 21. The isolated /194 session completed targeted native input checks and exposed a blank-inspector transition blocker; /195 repairs that transition in the native app. The ordinary startup/clean-account gate remains explicitly deferred for the private alpha only.

**Current engineering handoff:** [remaining owner/release steps](68-first-version-engineering-handoff.md), [native walkthrough and final input evidence](../docs/validation/native-walkthrough-195/README.md). No reproduced engineering blocker remains in the declared private-alpha checks. The candidate remains sealed and unpromoted; owner task results and release designation remain open.

**Earlier October 21 native checkpoint:** the public-Vellum **0.177.0 /194** candidate passed representative actual recording/timing, Verify/reset and sweep corrections, agent interleaving, exact data preservation and fresh reopening. It also exposed a blank inspector after replacing a long recording document with the LED example. [Findings and repair scope](67-native-release-acceptance.md), [frozen native evidence](../docs/validation/native-acceptance-194/README.md). The fix is not in /194; [the /195 native/package check](../docs/validation/inspector-candidate-195/README.md) passes the repaired transition. At that checkpoint the physical workflows/layout were unaccepted; the later report above advances engineering evidence only. No owner acceptance or promotion has occurred.

The remaining critical path is: **owner walkthrough on /195 → resolve any reproduced blocker → designate one release with rollback.** Documentation preparation does not close an interaction or acceptance gate.

**Historical execution checkpoint, October 20:** [exact /193 headless qualification](../docs/validation/headless-release-193/README.md) now passes reference workflows, project/electrical/storage/capacity suites across three non-native adapters, viewer/position corrections, sixteen analysis cases and fresh saved-result/example reopens. Sandboxed external paths also pass development-file/IP-denial controls; five helper clients are explicitly outside that sandbox claim. All owned processes close normally. At that checkpoint all 670 Go files matched the completed quality gate; current source has the separately qualified 671-file gate. This advances the headless portions of items 3–8. Items 1–2, native adapter/input acceptance, owner feedback and promotion remain open; no native app was started.

**Dependency change, October 21:** the owner requested migration to public Vellum. Source builds now use [public base `e58d405` with explicit compatibility fixes](../third_party/vellum/README.md). The prior checkpoint remains evidence for unchanged /193 bytes, not for a package using the new SDK. Source quality and affected package/native qualification must account for this new pin; the affected native checks are now recorded above and owner acceptance remains open.

## Remaining private-alpha checklist

### 1. Accept the input fixes through actual native controls

**Status:** the declared representative native classes pass using unaffected /194 cases plus /195 actual numeric-index, Previous/Play, coupled interval, Verify sampling and DC boundary checks. [Evidence](../docs/validation/native-walkthrough-195/README.md). Every startup override, keyboard/IME path and binding is not claimed. **Responsible:** engineering; this scoped pass is complete.

Accepted evidence combines /195 with unaffected /194 actual-input cases and unchanged-control source checks. The following declared problem classes have actual typing/button/menu outcomes, including activation without Return:

- [x] Invalid recorded index/time → correction to the previously accepted value → Previous/Next/Play. The correction and requested transport action must both take effect correctly.
- [x] Invalid Verify duration/sample step → correction to the accepted value → Verify or Use project startup. No old accepted value may silently replace rejected text.
- [x] Invalid DC sweep Start/Stop/Points → correction → Run. The accepted curve must use the visible corrected settings.
- [x] Invalid timing threshold or interval → correction → direction selection or paging. Invalid text must keep a relevant error; correction must recover on the first intended action.
- [x] Dependent thresholds/interval bounds that become valid together must commit together where supported by the repaired retained-entry flow. An individually invalid intermediate value must not produce a measurement using old settings.
- [x] An unfinished timing draft must not commit into a different probe selected by an agent. Fresh input for the new probe must remain usable. Exact /194 native/agent probe-ownership case passed; /195 does not change that control logic.
- [x] Recorded-position refusal must visibly explain the valid index/time range and units, including a one-row DC capture.

**Done when:** each declared native case has an observed outcome matching the public command result, with capture rows/metadata preserved and no unreported fallback, wrong target or blocked correction. Retain failures or unavailable observations rather than replacing them with passing attempts. The [prepared fixtures](../docs/validation/native-check-193-preparation/README.md) cover six proposed cases; they do **not** cover this entire list or constitute native execution.

### 2. Confirm the final candidate's ordinary UI remains usable

**Status:** the reference-control engineering pass is complete in both themes, at the enforced minimum and a larger working size. Native keyboard scrolling, menus, errors/correction, inspector and scope reachability are observed. [Evidence](../docs/validation/native-walkthrough-195/README.md). Owner first-use feedback and exhaustive accessibility/layout remain separate.

- [x] Observe a real visible window with schematic, parts list and toolbar; distinguish a failed capture from a user-visible display defect. /195 Light/Dark observations and restored-inspector screenshots pass.
- [x] Check essential controls in both themes at the declared minimum layout size and one normal working size. Check resized windows rather than assuming a large-window screenshot proves minimum-size behavior.
- [x] Confirm buttons and labels are complete, scroll areas behave normally, native menus are usable, and the inspector/scope controls remain reachable.
- [x] Check keyboard focus, visible errors and a corrected field after error-driven layout changes in the affected workflows.

**Done when:** no reproduced clipping, inaccessible core control or blank owner-visible workspace blocks a reference task. This is a focused acceptance pass, not another redesign. Complete screen-reader/IME coverage remains a broader MVP gate.

### 3. Complete the four reference workflows on the final candidate

**Status:** all four engineering reference workflows pass on the exact /195 bytes, combining the new native LED/RC/flasher/CSV walkthrough with the earlier /195 native Save/Open/SVG/cancellation checks. Six saved projects and final input captures match fresh packaged agents. [Evidence](../docs/validation/native-walkthrough-195/README.md). Actual owner task acceptance remains open in item 10.

- [x] **LED from blank:** place and wire source/resistor/LED/ground; observe about 2.917 mA at 5 V/1 kΩ with the pinned generic model; change to 2 kΩ and see lower current automatically; undo/redo; remove and repair a wire. Incomplete wiring must not present an old reading as current.
- [x] **RC experiment:** use 10 kΩ/1 µF, charge/discharge and Pause/Step/Resume/Restart; measure with voltage/current probes and cursors. The ideal-step reference is 3.160602794 V at 10 ms; use the appropriate reference for finite source ramps. Paused edits must display truthful stale/current state.
- [x] **Transistor experiment:** run the capacitor-timed NPN flasher; inspect both sides and measure timing; change the timing capacitors and observe slower flashing. Retain the pulse-driven NPN and existing NMOS/PMOS smoke checks and long-running pulse regression when their coverage is affected.
- [x] **Keep and share:** save a separate project, reopen it, inspect values/connectivity, export readable schematic and CSV data, and cancel an unsaved-change prompt without losing edits.

**Done when:** all four workflows finish without hidden setup, lost work, misleading results or a blocked essential action. Compare graph/state/numerical outcomes with the matching agent. Reuse unaffected earlier tests; repeat only what final repairs or package changes invalidate.

### 4. Qualify shared user/agent behavior for the release workflows

**Status:** scoped reference parity passes on /195, including exact graph, settings, results, probes/cursors and kept data. /194 native interleaving and /193 three-adapter contract/error/retry checks are reused for unchanged commands; the current inventory changes only the inspector source checksum. Full source checks pass. Exhaustive every-binding conformance remains a full-MVP obligation.

- [x] Pin the final app and its matching agent to the same source/version; exercise the affected workflows through the visible app endpoint and headless agent.
- [x] Compare components, connections, values, probes, measurements, result provenance, history and saved projects—not just successful command responses.
- [x] Confirm invalid requests, stale revisions, retries and user/agent interleaving preserve work. The repaired timing-probe ownership case is required.
- [x] Check discovery/help for changed behavior and preserve structured errors and inspectable job/close results.
- [x] Reuse the existing in-process/stdio/socket/native-socket conformance machinery for the release coverage invalidated by the changes. A standalone agent must not be mistaken for control of the owner's open app.

**Done when:** the final release's reference capabilities work through the public contract without a GUI-only step, and affected UI/agent outcomes agree. Exhaustive coverage of every shipped binding remains a separate full-MVP obligation; a command count is not proof of it.

### 5. Preserve project data, captured results and recovery behavior

**Status:** current native Save/Open/cancellation, six project reopens, original 1001-row preservation, one-row DC and corrected 105-row Verify capture pass. Prior /193 file/error contracts and native decision/recovery evidence are reused because persistence, recovery and command Go code are unchanged. [Native /156 decisions](ui-audit/first-version/native-decisions-156/README.md), [/161 recovery](ui-audit/first-version/native-recovery-161/README.md) and [/174 storage-error recovery](../docs/validation/native-recovery-error-174/README.md) keep their historical scope; no new crash, power-loss or owner-file test is claimed. Both new /195 native sessions close normally with joined workers and removed endpoints.

- [x] Save/reopen on the final candidate preserves graph, values, model identities, source/initial settings, selected probes and kept results as specified by the format.
- [x] Recorded rows, metadata and raw exports remain unchanged while position/timing/measurement settings change.
- [x] Cancel, Discard and Save-and-continue retain their documented ownership behavior; failed saves preserve the last valid manual file.
- [x] Existing malformed-file refusals and recovery protections still apply to affected paths. Reuse prior controlled recovery evidence unless a final change invalidates it.
- [x] Close a separately owned test session normally and observe cleanup readiness and process/endpoint disappearance. Refuse cleanup if unexpected owner work appears.

**Done when:** no reproduced work loss, silent overwrite, corrupted connectivity or capture mutation remains. Session-only Verify defaults and recorded position must not be falsely treated as persisted physical state. Simultaneous owner activity also means shared preference files cannot be assumed unchanged merely because a test child is isolated.

### 6. Resolve any numerical or simulation-state regression introduced by the repairs

**Status:** complete for the current repair scope. From /193 the only production Go change is the inspector viewport identity; no solver/model/sampling/state code changes. Current native RC observations and CSV match the independent RC law; both flasher endpoints/cursors match the packaged agent. Existing /193 electrical/capacity and MOSFET/pulse regressions retain their scope. No numerical tolerance was relaxed.

- [x] Confirm the final change set is limited to the qualified areas. If physics, models, sampling or state semantics change, rerun the corresponding independent numerical and timestep-refinement checks.
- [x] Keep the established LED, RC and NPN tolerances from the PRD; do not loosen bounds merely to get a passing build.
- [x] Confirm Running/Paused/Updating/Incomplete/Failed and recorded/live distinctions remain truthful through the affected edits, transport and measurement actions.
- [x] A failed solve or unavailable measurement must never appear as a current valid zero or stale result.

**Done when:** the final source/package has applicable numerical and state evidence, with no misleading result. Agreement between two clients of the same solver is parity evidence, not an independent numerical reference.

### 7. Finish source quality on the final implementation

**Status:** satisfied for the /195 implementation: all 671 Go files match the source that passed the full quality gate, including the new inspector-scroll regression. Reopen only if implementation changes. **Responsible:** engineering.

- [x] For any new repair, retain a focused failing reproduction and meaningful regression checks, then review the change for Go structure, error handling, concurrency and ownership.
- [x] Format/lint and run relevant focused tests during repair. Run `scripts/check.sh` once on the final changed source: PRD coverage check, lint/static analysis, Python tests, root and desktop race tests/vet, and the configured mounted offscreen tests.
- [x] Record source identity and test scope; ensure the candidate actually contains the qualified source.
- [x] Leave the worktree clean with changes committed. Preserve existing owner work, earlier candidates and original failed evidence.

**Done when:** final changed source passes required checks and review, and no unresolved regression remains. Documentation-only edits do not require another full Go test campaign.

### 8. Complete final-package integration and offline operation

**Status:** exact /195 copied native/agent reference jobs pass with repository/SDK/toolchain/cache/shared-geometry access and outbound IP denied. Signature, 124-file ZIP/extraction, notices and extracted-agent examples pass; all current package bytes/modes remain unchanged. Controllers run outside the child sandbox. Ordinary/extracted native default startup and clean-account installation are not claimed.

- [x] Exercise the affected final native/agent integration from the exact candidate or extracted package, with version/hash recorded.
- [x] Complete any invalidated copied-runtime/offline checks: no dependency on the source checkout, Vellum checkout, Go toolchain/cache, account or outbound service for core use.
- [x] Confirm app/agent identity, editable examples, notices, version, checksum, rollback and known-limit documentation match the selected review candidate. Release designation remains open.
- [x] If bytes change, repeat affected signature/archive/extraction and example save/reopen checks. /195 archive/extraction and example reopens pass; this does not establish native extracted startup.

**Done when:** the exact distributed private-alpha bytes have the required integration coverage and can perform the reference jobs locally. Do not create another package merely for new documentation or instrumentation; /195 exists for the reproduced inspector repair.

### 9. Make the startup and distribution boundary explicit

**Status:** the private-alpha boundary is documented in the sealed package, walkthrough and [current handoff](68-first-version-engineering-handoff.md). The owner-deferred ordinary default-startup/clean-account check is a deferral, not a test pass. macOS 26.6.2 arm64 and local ad-hoc signing are the tested envelope; public distribution remains separate.

- [x] Carry the existing deferral into the final private-alpha notes. Do not report a test pass or broaden that deferral to other gates.
- [x] State the tested macOS/hardware envelope, local ad-hoc signature and lack of public-distribution qualification.
- [x] Keep ordinary no-argument startup, clean installation, Finder/quarantine/Gatekeeper and signing/notarization on the full-MVP/public-distribution checklist.

**Done when:** the private release's limitations are accurate and reviewed. A disposable account/VM and appropriate signing/distribution setup are needed before broader claims. Do not alter shared owner settings to manufacture a clean-start test.

### 10. Obtain actual owner workflow results

**Status:** owner results are pending. /189 initial display was confirmed; the owner answered [“Not tried yet” for /195](ui-audit/first-version/owner-feedback-195.json). No four-task result is inferred. The engineering-qualified candidate and walkthrough are available without replacing /189.

- [ ] Record success, confusion or failure for the four tasks in the [owner walkthrough](35-first-version-review.md).
- [x] Identify the exact build for each observation. /189 feedback remains valuable, but cannot prove acceptance of fixes in the /195 review candidate.
- [x] After native qualification, offer the combined candidate for the affected owner retest without replacing or closing existing work unexpectedly.
- [x] Resolve reproduced work-loss, wrong-result, wrong-target and blocked core-task defects; record any minor remaining limitation clearly.

**Done when:** the owner can complete the core loop on the selected release, or an explicit, accurately scoped private-alpha limitation is recorded where permitted. Data integrity, simulation truthfulness and core user/agent capability parity cannot be waived.

### 11. Close the blocker list and designate one release

**Status:** engineering blocker dispositions and remaining steps are recorded in the [handoff](68-first-version-engineering-handoff.md). No reproduced blocking reference-task defect remains in this scope. Release designation/promotion depends on actual owner review; the candidate remains unpromoted.

- [x] Give each open recording/analysis/timing blocker a final disposition tied to the accepted build and actual evidence. Source-only or headless-only success must not be labeled native acceptance.
- [x] Review any remaining issue against the four workflows. Do not postpone a work-loss or misleading-result defect under a cosmetic limitation.
- [ ] Select one app/agent pair as the first-version release, update the handoff/version/limits consistently, and preserve the prior owner alpha and rollback path.
- [x] Make the concrete release available for owner review before promotion; do not overwrite the canonical or owner app merely because a candidate exists.

**Done when:** there is one unambiguous release folder and receipt, with known limits, walkthrough, examples and rollback. The current candidate archive alone is not that decision.

## Additional requirements before calling it the complete PRD MVP

These remain part of the original product request. They are not prerequisites added by this checklist, and private-alpha delivery must not be reported as satisfying them.

| Remaining area | Required completion evidence |
|---|---|
| Complete user/agent coverage | Every shipped capability is discoverable and callable, with accurate schemas, units, defaults, preconditions, results and errors. Complete conformance covers dynamic bindings, state transitions, atomic failures, revisions, retries, undo, cancellation, reconnect, dialogs and teardown. UI automation is not the agent interface. Built-in and external clients use the same public contract; bundling a conversational AI provider is not required. |
| Keyboard, focus and accessibility | Complete the actual pilot-platform keyboard and screen-reader workflows, navigation, properties, diagnostics, plots and file decisions. Cover remaining focus/blur, fast typing, IME and interruption cases. AX labels or screenshots alone do not establish screen-reader support. |
| Performance and difficult solves | Establish the PRD's visible p95 acknowledgment target of 100 ms separately from the 500 ms first-valid-update target on the three reference circuits; retain existing warm-trial limitations. Validate the reference 200 ms Verify runs within two seconds and navigation/cancel/save/undo during a deliberately difficult solve. No repeated latency campaign without a concrete remaining measurement need and declared method. |
| Capacity and complete state contract | Validate the supported 50-component/100-node/four-trace envelope, boundaries, bounded history, cancellation and retained edit/save access. Finish remaining physical gesture/state-transition scenarios; a large simple fixture does not prove arbitrary convergence. Publish the actual supported envelope. |
| Clean setup and project portability | Reproducibly bootstrap the locked SDK/dependencies on a clean supported development environment. Perform ordinary default startup and supported offline installation in a separate account/VM; reopen self-contained projects without fetching dependencies. The private-alpha deferral does not pass this gate. |
| Distribution | Qualify intended installation/quarantine/Gatekeeper behavior and the appropriate signing/notarization/distribution process before public distribution. A local ad-hoc resource seal does not establish those properties. |
| Ten-person hobbyist study | Run five newcomers and five experienced hobbyists using the prepared study packet. Meet or explicitly revise with evidence the original gates: 8/10 overall and 4/5 newcomers build the LED in five minutes unassisted; 8/10 explain the resistance experiment; 8/10 plot a waveform and enter a cursor; 8/10 recover the deliberate mistakes; 9/10 distinguish result states; 8/10 identify a model limit and save/reopen. Investigate every observed invalid result presented as valid. Implement the resulting prioritized corrections. |
| Printing/output scope | Preserve readable schematic/CSV exports and the already confirmed complete LED print. Qualify measured margins/media/scale, tiling, other settings and cancellation before claiming those paths/devices supported. No additional physical print is required solely to add evidence. |
| Formal MVP acceptance | Reconcile all 27 P0 requirements, all original quality targets, the electrical/workflow suite and Gates C/D against final-build evidence. The current audit is 17 implemented/10 partial; those counts describe recorded engineering coverage, not product acceptance. |

## Delivery rules

1. Finish the native input fixes and reference workflows first. Do not add new devices, PCB work, cloud features or a broad Vellum upgrade to this release queue.
2. A reproduced defect gets one focused repair/checkpoint before deciding the next action. Avoid open-ended capture/renderer investigation without an actual defect.
3. Preserve successful unaffected evidence. Repeat checks only when a change, failure or unresolved coverage gap justifies them.
4. Keep private-alpha acceptance, full-MVP acceptance and public-distribution readiness distinct. Never infer one from another.
5. There is no evidence-backed completion date yet: native availability, outcomes, owner feedback and broader environment/participant access determine the remaining work. Estimate again after the native candidate pass and owner results; a large test count is not a schedule.

Sources: [delivery plan](34-first-version-plan.md), [current candidate receipt](ui-audit/first-version/timing-candidate-193.json), [candidate evidence](../docs/validation/timing-candidate-193/README.md), [prepared native fixtures](../docs/validation/native-check-193-preparation/README.md), [owner observation](ui-audit/first-version/owner-feedback-189.json), [scoped environment deferral](ui-audit/first-version/private-alpha-owner-decisions.json), and [original PRD](07-product-requirements.md). The older [gate reconciliation](41-release-gate-reconciliation.md) supplies historical scopes; its /174-era “next action” is not the current queue.
