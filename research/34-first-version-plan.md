# First usable version: delivery plan

**Written October 17, 2026. Status: in progress; package 1 command/startup checks complete. Alpha release gates remain open.**

## Concrete next handoff

**Deliver one macOS private alpha for the owner to use.** It must support drawing an LED circuit from blank, live LED/RC/transistor simulation, value edits, probes/cursors, undo, save/reopen and schematic export. The external agent must perform the same reference workflows. This is the first-version cut line; additional devices and analysis features do not enter this milestone.

The implementation has spent too long accumulating improvements and evidence without promoting a build the owner can use. Delivery now has four remaining work units:

| Order | Remaining work | Evidence needed to leave this unit |
|---|---|---|
| 1 | Finish the two reproduced UI repairs: DC captions crowding source symbols, and stale accessibility bounds after scrolling. Preserve the current work; qualify the minimal framework patch before changing the pinned SDK. | Native reproduction passes after each fix; targeted regressions pass. Finish the core keyboard loop and minimum-window check. Record unsupported display/accessibility configurations as unqualified rather than extending the investigation indefinitely. |
| 2 | Run acceptance tasks A–D below through the external agent against the UI artifacts. | Saved connectivity/values and numerical/state results agree; invalid edits, undo, save/reopen and recovery preserve work. Produce a clearly labeled owner-review build at this boundary. |
| 3 | Qualify and package that source once. | Final quality gate, release-diff review, copied-app offline launch and file round-trip pass. Include the matching agent, examples, known issues and rollback instructions. Promote the qualified bundle to the canonical app path. |
| 4 | Hand over the app and a 15–20 minute task sheet. | Publish exact app/agent paths, version and hashes. Owner feedback becomes a bounded defect queue; waiting for feedback does not authorize more feature development. |

**Stop rules:** no general refactoring, new feature batches, optional schema campaigns or broad Vellum refresh. Reuse applicable evidence. Run focused checks for a repair, then one full gate on the final source; rerun only what a subsequent change invalidates. A blocker investigation gets one focused reproduction/repair pass before a checkpoint that states the unresolved failure, smallest next action and revised estimate. Do not silently turn a blocker into an open-ended framework project.

**Delivery decision:** data loss, incorrect or misleading simulation, broken core user/agent parity, and inability to complete A–D block the alpha. Wider environment, performance, accessibility and physical-print qualification gaps must be disclosed and remain open for the full MVP; they are not inferred passes. Any permitted limitation requiring owner review is presented with the concrete review build.

The existing **2–4 focused engineering day estimate is provisional, not a delivery commitment**. Re-estimate when unit 1 closes. The next progress report must identify which of these four units closed and link its artifact or failing reproduction; a test count alone is not delivery progress.

## Delivery summary

**The next deliverable is one installable macOS alpha, not another feature batch.** The existing LED, RC and transistor workflows are the release scope. New features and framework upgrades are frozen except for demonstrated blockers.

The immediate queue is:

1. Finish the native UI walkthrough. Save defaults, device/file decisions, rapid coordinate repeats, measurement access and post-dialog keyboard navigation are repaired or directly verified. The DC caption anchor and scrolled accessibility defect are now repaired. Next are the minimum-window/display boundary and the external-agent acceptance pass.
2. Run the four acceptance tasks below through the external agent and compare their saved circuits and results with the UI runs. Deliver a clearly labeled review build at this checkpoint.
3. Run the final qualification checks, then package and promote one app with its matching agent, example projects, known issues and rollback path.
4. Hand over a 15–20 minute owner walkthrough. Restrict subsequent alpha work to failures in that walkthrough.

Packages 1–6 below define the detailed exit criteria. Package 1 is complete; package 2 is the current constraint. The 2–4 day estimate below remains provisional until the native input/file issues are resolved. No completion date is established by that estimate. Each package exit must report what passed, what remains and any revised estimate; it must not silently expand the release scope.

## Decision and outcome

The next milestone is a **macOS hobbyist alpha that the owner can install, launch and use**, followed by the original PRD's broader MVP validation. Work is now organized around delivering that build. Additional components, analysis modes and standalone contract-documentation campaigns leave the immediate queue.

The alpha must let a maker draw a circuit, see simulation update, change a value, measure a waveform, recover from a mistake, and save/reopen the project. An external agent must complete the same loop through the shared command interface. The core remains offline and requires no account.

This is an intermediate delivery milestone, not a declaration that [PRD draft 0.3](07-product-requirements.md) has passed. Its accessibility, complete agent conformance, distribution and ten-person hobbyist gates remain requirements for MVP acceptance. This plan does not silently weaken those gates or mark their evidence complete.

**Planning estimate:** 2–4 focused engineering days to an owner-testable alpha, subject to the native interaction pass. This is an estimate, not measured remaining effort. Re-estimate after package 2 using actual failures. A framework blocker gets a concrete report, reproduction and revised delivery estimate; it does not trigger another open-ended feature expansion.

## Where we actually are

- The editor, live DC/transient simulation, probes/cursors, undo, projects, recovery and schematic output already exist. NPN, NMOS and PMOS support is implemented. There is enough component breadth for this milestone.
- The [implementation audit](11-prd-implementation-audit.md) records **17 implemented / 10 partial P0 requirements**. Those counts describe evidence against the full PRD, not a release completion percentage.
- The latest qualified private candidate is `build/ELab-0.121.0-138-vellum-refresh-candidate.app`; the canonical `build/ELab.app` remains at 0.105.0 / 120. Many engineering checkpoints have not become the app the owner normally opens.
- The [current Vellum integration](../docs/vellum-refresh.md) is pinned and tested. Freeze it for this milestone unless a reproducible release blocker requires a fix.
- Automated and numerical evidence is substantial. Actual pointer/keyboard behavior, readable layouts, visible latency and installation experience still need a focused release pass. Source inspection and GPU readback alone do not establish those outcomes.

The delivery problem is now completing and packaging the existing workflow. More inventory and annotations alone will not establish that the application is pleasant to use.

## Frozen alpha scope

| Area | Required behavior |
|---|---|
| Start and draw | Launch to a usable blank sheet; find/place components, connect terminals, select/move/rotate/delete and edit values with units. Open an example without hidden setup. |
| Live simulation | Automatic updates for valid supported circuits; clear incomplete/failed/paused/stale states; Pause, Step, Resume and Restart behave consistently. Invalid edits preserve the drawing and never display old readings as current. |
| Observe and measure | Read LED current and capacitor voltage; select voltage/current probes; use scope cursors and timing deltas. |
| Keep work | Undo/redo; Save, Save As, Open; unsaved-change decisions; recovery after a controlled process crash. Reopening restores the project and begins with truthful simulation state. |
| Output | Export a readable schematic SVG/PDF/PNG and measurement CSV through existing shared commands. Keep print preview/printing available; do not claim physical-printer qualification without an actual test. |
| Appearance | Existing themes, native menus and predictable scrolling; essential controls remain readable and reachable at the declared minimum window size. |
| Agents | Discover capabilities and execute the same construction, edit, simulation, measurement, recovery and file actions through the public interface. Preserve existing commands and structured outcomes. No GUI automation needed for the reference workflows. |
| Delivery | One clearly identified, double-clickable app plus a documented external-agent entry point, example files, known limitations and rollback build. |

The existing wider library and advanced tools remain available, but adding features to them is outside this release queue. Any defect that loses work, corrupts connectivity, misrepresents a result or breaks command parity is a blocker regardless of feature area.

**Deferred development:** new devices/models, PCB layout, general SPICE compatibility/import, cloud/provider integration, collaboration, Windows/Linux ports, more analysis modes and cosmetic redesigns unrelated to a demonstrated usability failure. Completing all optional on-demand schema annotations is not the alpha's finish line; actual command behavior and parity are.

## Ordered work packages

I own implementation, review, automated checks, direct UI inspection and packaging. The owner supplies the final short usability walkthrough. Fresh-account/hardware tests and participant sessions require actual access or people; they are never replaced with inferred passes.

| Order | Deliverable | Exit condition | Status |
|---|---|---|---|
| 1. Stabilize examples and entry points | Repair reproducible startup/reset defects; reconcile the example catalog, demo and instructions. | Every current example loads; repeated load/reset gives a valid fresh run; dirty-project and stale-document guards work; blank startup is usable. Targeted regression tests accompany fixes. | Complete for command/startup behavior; direct control usability follows in package 2 |
| 2. Complete the real UI pass | Walk the four reference tasks below using actual controls. Fix clipping, menu behavior, scrolling, focus, value commits, wiring and scope interactions that obstruct them. | All tasks completed at 1280×800 and 1440×900 display settings, with the visible window size recorded; inspect both themes. Define/enforce the minimum usable window size from evidence. No essential button is clipped or unreachable; keyboard completion of the core loop works. | In progress; [one issue list](ui-audit/first-version/README.md), RC readability/cursor/compact controls repaired; remaining walkthrough and input/file issues next |
| 3. Close correctness and parity blockers | Repeat the same tasks through an external agent, including invalid edits, interrupted jobs, save/reopen and recovery; check native action-to-command mapping. | Matching connectivity, values and numerical/state outcomes; no GUI-only decision in the reference loop; no known data-loss, stale-result or parity defect. Unresolved wider audit evidence is explicitly listed. | Planned |
| 4. Qualify the release candidate | Build from the frozen SDK/source; run the final quality, numerical, native smoke and responsiveness checks. | Gates below pass or have an explicit owner-reviewed alpha limitation where allowed. Core correctness/data ownership gates cannot be waived. One candidate identity is recorded. | Planned |
| 5. Package and promote | Produce the owner bundle, agent executable/launch instructions, starter projects and concise release notes. Update stale SDK/package commands in the main README. | Launch the copied bundle outside the checkout without development-file access or network; save/reopen there; verify app and agent packaging. Preserve the previous canonical bundle, then promote the qualified candidate to `build/ELab.app`. Publish exact paths/version/hash. | Planned |
| 6. Owner acceptance and bounded fixes | Provide the build and a 15–20 minute task sheet; fix reproducible failures in that loop. | Owner can complete the loop and receives a known-issues list. Record actual feedback; make a new candidate only for concrete fixes. Then schedule the remaining full-MVP gates. | Planned |

The **first reviewable candidate is delivered after packages 1–3**, with remaining qualification clearly labeled. Packages 4–5 produce the supported local alpha. Package 6 is user acceptance, not a reason to keep adding features while awaiting feedback.

### First concrete defect: repaired

The legacy LED load command, `example.reset`, was absent from the dispatcher's forced-restart list. A regression reproduced the failure: reloading an unchanged, paused LED template advanced the document generation but left the run/electrical identities unchanged and cleared its results. The shared loader mapping now drives the restart decision. All 18 catalog reloads and seven legacy reloads pass; the public demo now includes the two op-amp examples. See the [checkpoint and evidence](../docs/validation/first-version-examples/README.md).

## Acceptance tasks

Use the same component definitions and pinned models for UI and agent runs. Retain the resulting `.elab` projects, measured values and a short issue/evidence table. Reuse existing independent numerical references rather than generating another redundant reference campaign.

| Task | User action and observable success |
|---|---|
| A. LED from blank | Place 5 V source, 1 kΩ resistor, LED and ground; wire the loop. Read approximately 2.9168 mA under the pinned model. Change to 2 kΩ and see lower current automatically. Undo/redo; disconnect a wire, identify the problem and repair it. |
| B. RC charge/discharge | Build 10 kΩ / 1 µF with a 0–5 V pulse. Plot capacitor voltage, pause/step/resume, place cursors and measure timing. For the ideal-step reference, check 3.160602794 V at 10 ms within the PRD tolerance. Use the matching reference for finite ramps. Edit while paused and verify stale/current status. |
| C. Transistor experiment | Load and edit the capacitor-timed NPN LED flasher; observe both sides, select collector/LED probes and measure its period. Check pulse-driven NPN switching against existing independent references. Load NMOS and PMOS examples as regression smoke. |
| D. Keep and share | Save, close and reopen the edited project. Export schematic and scope data; inspect output readability. Exercise Cancel/Discard/Save on replacement. Recover a controlled interrupted session without overwriting its last manual save. |

Run A–D through the native UI and the public external client. Inspect actual input/focus/dialog paths in the UI pass; command scripts alone do not count as UI completion. All current templates must load even though only these jobs receive the full task walkthrough.

## Candidate release checklist

All boxes start unchecked. Prior evidence may be reused only when its source/build and scope remain applicable; mark new measurements with their actual configuration.

- [ ] **Workflow:** A–D complete in UI and external client; saved artifacts agree semantically; invalid edits and undo preserve ownership and correctness.
- [ ] **Layout/input:** both themes and the two display settings inspected; native menus, scrollbars, dialogs, focus, value entry, wiring and cursors usable; minimum window size recorded. Core keyboard loop completed. Actual VoiceOver findings recorded separately; complete screen-reader acceptance is still a full-MVP gate if not established for the alpha.
- [ ] **Quality:** `scripts/check.sh`, numerical reference/conformance suite and packaged native smoke pass on the final source; review the release diff for error handling, races, state ownership and accidental dependency changes. Add tests for repaired behavior, not documentation-only changes.
- [ ] **Visible responsiveness:** record hardware/OS/build and at least 20 repetitions per reference circuit for edit acknowledgment and first updated sample. Compare p95 with the PRD's 100 ms / 500 ms targets. Measure navigation/cancel/save/undo under difficult solves and cold routing. Never substitute backend timing for visible timing; report a miss and its impact before delivery.
- [ ] **Files/recovery:** round-trip the reference projects and retained results; exercise save failure and controlled process-crash restore using existing fault tooling; last manual saves remain intact.
- [ ] **Agent surface:** catalog/route inventory has no new drift; every visible action has a public command/query/structured decision; reference workflows and retry/conflict/cancel behavior pass. Record remaining mounted-runtime evidence gaps without claiming exhaustive conformance.
- [ ] **Packaging:** the copied app launches offline outside the checkout with repository/SDK/toolchain access denied; the documented agent launch works; version/hash and rollback path recorded. Clean-account installation remains explicitly unqualified until actually performed.
- [ ] **Handoff:** main README points to the current build; release notes state supported environment, model limits, known issues and the short owner task sheet. No unqualified claim of public distribution, complete accessibility, physical printer support or PRD completion.

A failed integrity, simulation-truthfulness or core parity check blocks promotion. A native layout/input failure that prevents A–D also blocks it. Broader environment, accessibility or timing evidence gaps may be disclosed for a private owner-test alpha; they stay open in the full MVP audit and require explicit review before broader release. Missing evidence is never a pass.

## Boundaries that keep this finite

1. Maintain one release-blocker list with reproduction, severity, owner and closure evidence. Package 2 creates it from actual use; no speculative redesign backlog is a blocker.
2. Work through the six packages in order. Reopen an earlier package only for a reproduced regression. Update this plan at each package exit with completed work, blockers and revised estimate.
3. Run targeted checks during repairs, then one complete final gate. Repeat a broad suite only when changes or failures justify it. Avoid accumulating another independent validation archive or version for each small annotation.
4. Freeze Vellum, models and file format during this release unless a blocker requires a change. Review Go boundaries and concurrency as part of touched-code review; defer general restructuring.
5. When the checklist passes, package and hand off. Do not replace delivery with another set of unrelated improvements. After handoff, continue only the bounded fix queue and prepare the full-MVP validation work already required by the PRD.

## Execution checkpoints

- **October 17, package 1:** repaired LED reload, reconciled catalog/demo/guide, passed all 25 focused reload cases, full `scripts/check.sh`, and the public 18-template/seven-alias demo in fresh headless and native sessions. Both native test host and forwarding client exited normally. Canonical app remains unchanged. This closes command/startup behavior; physical input, layout, visible latency and owner acceptance remain open. Next action is package 2's actual UI walkthrough, not more schema annotations.
- **October 17, package 2 started:** built LED and RC from blank with actual controls; checked live value edits, undo/redo, wire-fault repair, native save/reopen, 10 ms RC stepping/cursors and both themes in two app-window sizes. Repaired misleading unset cursor fields and retested zero entry/clear in the rebuilt UI. [Evidence and five-item issue list](ui-audit/first-version/README.md). Next repair is compact schematic readability. Display configurations, remaining tasks and complete input/accessibility qualification remain open; the delivery estimate will be reassessed at package 2 exit.
- **October 17, package 2 readability repair:** replaced routing search envelopes/generic part margins with actual paths and shared enlarged-caption bounds. Rechecked RC framing in both themes and both observed window sizes; native/headless views match exactly. Fixed Theme clipping with both agent history labels and removed the compact tool-row scrollbar while retaining its controls. Full quality gate passes after a reviewed source-ledger refresh; command/site mappings are unchanged. [Updated issue list and screenshots](ui-audit/first-version/README.md). Remaining work is the transistor/device walkthrough plus the listed input, filename and accessibility investigations; package 2 is not yet closed.
- **October 17, package 2 flasher checkpoint:** observed alternating LEDs, inspected Q1/Q2 and measured a 485 ms cursor interval through native controls. A paused capacitor edit correctly marked prior results stale; Cancel preserved the edit and Undo restored the saved value. The first Save in a fresh process also reproduced the duplicated extension, narrowing UI-3 beyond panel reuse. [Evidence](ui-audit/first-version/README.md#flasher-walkthrough-checkpoint). NMOS was opened, but its full walkthrough and PMOS remain unqualified. The owned review host was saved privately and closed; no canonical app was promoted. The next implementation action is the Save filename repair, followed by the remaining UI tasks.
- **October 17, package 2 Save repair:** corrected macOS default filename handling without updating Vellum. Actual first/repeated Save, Save As and SVG/PDF/PNG defaults have one extension; Save As preserves every source-project JSON field and the exported PNG is readable. Focused race checks and the complete quality gate pass. [Evidence](ui-audit/first-version/README.md#save-filename-repair-and-export-checkpoint). Rapid replacement failures also occurred in AppKit's filename field, while settled search entry passed all five comparisons; ordinary typing/accessibility remain qualification work. Next are device/probe interactions, keyboard construction and the remaining file-decision/CSV paths. Package 2 remains open.
- **October 17, package 2 device/keyboard/file pass:** native NMOS/PMOS High/Low switch the LED in the expected opposite directions. Save-and-continue preserves NMOS before loading PMOS; Discard replaces the disposable PMOS copy with RC. Native CSV matches every point in its 241-row public scope observation. Keyboard construction, named wiring, Resume and a 1 kΩ → 2 kΩ edit work, and a separately constructed external-agent circuit matches the resulting graph and electrical result exactly. [Evidence and limitations](ui-audit/first-version/README.md#device-file-and-keyboard-workflow-checkpoint). Repeated coordinate arrows exposed a revision-refusal/input issue; measurement controls are buried in the inspector, and compact device overview text remains small. These demonstrated issues define the next repair queue, alongside scrolled AX publication. Complete keyboard history/save/reopen, display configurations and release qualification remain open; package 2 is not closed.
- **October 17, package 2 input/measurement repairs:** retained field callbacks now continue from their own accepted command responses, while stale opposite-axis/agent/document contexts remain refused. Native 20 Up /20 Down /20 Up produces all 60 intended coordinate increments with no revision error. CSV is beside the scope trace selector; probe controls precede general settings, and native current-probe selection reaches the shared scope. The relocated export's 177 rows match the public samples. [Evidence](ui-audit/first-version/README.md#coordinate-continuation-and-measurement-access-repair). Focused race tests and the full quality gate pass. Next are compact device overview readability and scrolled AX behavior, then keyboard history/file navigation. Package 2 remains open; no SDK/model/format change or app promotion occurred.
- **October 17, package 2 canvas/file-focus repairs:** reclaimed the empty status row, increasing compact flasher drawing height by 50 pixels and Fit from 25.16% to 33.28%, with exact native/headless parity. Inspected actual 1280×800/1440×900 app windows in both themes; OS display modes remain unqualified. Repaired a new native file-focus defect using an asynchronous, window-guarded application continuation. Keyboard Cancel/Save/Open and Undo/Redo now work; the source project stays unchanged. [Evidence](ui-audit/first-version/README.md#canvas-space-and-keyboard-file-focus-repair). Full final quality gate passes. Scrolled AX editing still fails until a repaint, narrowing its investigation; enlarged DC captions still crowd the source symbol. Those two concrete repairs and minimum-window/display qualification remain before package 2 exit. No SDK/model/format change or app promotion occurred.

- **October 17, package 2 source-caption repair:** drawing and shared Fit now use a scale-aware DC-source caption anchor. Compact/large and keyboard-rotated native observations pass; run/results/wires remain unchanged and native/headless Fit agrees. Clearance and containment regressions plus the complete quality gate pass. [Checkpoint](ui-audit/first-version/README.md#dc-source-caption-repair). Scrolled accessibility is being tested in an isolated framework copy; no new SDK pin or canonical promotion is implied. Minimum-window/release gates remain open.

- **October 17, package 2 scrolled AX repair:** reproduced missing publication after child-only Scroll paint and pinned the minimal clean private SDK `bc6b456` (exactly four changed source members). Direct native Cursor A editing now succeeds after scrolling without a theme repaint, with a valid 10 ms cursor confirmed publicly. Framework race/vet, consumer full quality, final targeted checks and actual native smoke pass; the smoke assertion was corrected to distinguish a measured canvas resize from a refused circuit edit. [Checkpoint](ui-audit/first-version/README.md#scrolled-accessibility-repair). Both demonstrated UI repairs are closed. Next: delimit the supported minimum-window/display evidence, then run the agent reference pass and deliver the review build. No canonical app was promoted.

## After the alpha

The full MVP still requires the remaining PRD audit closures: complete mounted interaction/agent coverage, pilot-platform accessibility, visible performance and difficult-workload qualification, clean supported installation/distribution evidence, and the actual ten-person formative study with corrections. Signing/notarization and physical printer testing are tracked according to the distribution and printing promises being made. These are explicit subsequent gates, not claims made by the private alpha.

The original PRD and historical audit remain authoritative about product requirements and completed evidence. **This document takes precedence over the historical continuation queue for the order of upcoming work.**
