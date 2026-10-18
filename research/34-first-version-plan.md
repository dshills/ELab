# First usable version: delivery plan

Updated October 18, 2026. Owner: implementation agent; owner acceptance: project owner.

## Outcome and scope

Deliver a **private macOS hobbyist alpha** that lets the owner draw a circuit, watch it simulate, edit values, measure a waveform, undo a mistake, save/reopen and export a schematic. An external agent must perform the same reference workflows through public commands. The core runs offline without an account.

This is the first-version cut line. The original [PRD](07-product-requirements.md) remains authoritative for the broader MVP, including accessibility, full agent conformance and an actual ten-person hobbyist study. A private review build does not close those gates.

| In the first version | Out of the release queue |
|---|---|
| Place, wire, move, rotate, delete; edit values with units | New devices/models and analysis modes |
| Live LED, RC, NPN and existing MOSFET workflows; Pause/Step/Resume/Restart | PCB layout and general SPICE compatibility |
| Voltage/current probes and scope cursors | Cloud accounts, collaboration and provider integration |
| Undo/redo, Save/Save As/Open, unsaved decisions and crash recovery | Windows/Linux ports |
| Schematic SVG/PDF/PNG, scope CSV, existing print preview/printing | Broad Vellum upgrades or cosmetic redesigns |
| Existing themes, native menus, readable and reachable core controls | General refactoring and optional schema campaigns |
| Matching external agent, examples, notices and rollback instructions | Additional feature batches while feedback is pending |

The wider implemented library stays available. Freeze Vellum, models, project format and feature scope unless a reproduced release blocker requires a change.

## Current position

- The editor, simulation, measurements, history, persistence, recovery and schematic output exist. NPN, NMOS and PMOS are implemented; more component breadth is unnecessary for this milestone.
- UI reference tasks, minimum-window checks, reference user/agent parity and isolated copied-runtime checks established a private review boundary. [Evidence](ui-audit/first-version/README.md), [review walkthrough](35-first-version-review.md).
- Slider-context review **0.127.0 /144** is packaged with its matching agent and examples. The canonical app remains **0.105.0 /120**; earlier review folders/windows are preserved. Packaging is not promotion.
- **Clock blocker closed for the review package:** 0.125.0 /142 is built cleanly from `f87b71f`. Its exact copied runtime passes smoke, four adapters, reference parity and pulse/Pause/Resume/save checks through 3.41 s. The source also has native interaction evidence through 40.9 s. The defective /141 is retained for rollback/history, not promoted. [Replacement receipt](ui-audit/first-version/corrected-review-checkpoint.json).
- The full-PRD [audit](11-prd-implementation-audit.md) records 17 implemented /10 partial P0 requirements. These are evidence categories, not a completion percentage.

- **Held-slider blocker closed in /144:** the clean SDK repair and exact copied app/agent pass source quality, retained-widget regression, native smoke/four-adapter conformance, reference parity and live pulse/Pause/Resume/save checks. [Replacement receipt](ui-audit/first-version/slider-review-checkpoint.json).

- **Rendered-button blocker closed in /143:** the clean replacement incorporates the verified guard repair and passes exact copied-runtime smoke/four-adapter conformance, reference parity and pulse/Pause/Resume/save acceptance. [Handoff receipt](ui-audit/first-version/guarded-review-checkpoint.json).

## Ordered work and finish criteria

| Order | Deliverable | Exit condition | Status |
|---|---|---|---|
| 1 | Stabilize startup/examples and core UI | Repeated catalog loads work; reference tasks use real controls; essential controls are reachable in both themes at the declared minimum size. | Private review boundary passed. Wider display/accessibility/input evidence remains open. |
| 2 | Verify user/agent reference parity | Same components, connectivity, values and results; invalid edits, undo, files and crash recovery preserve work. | Reference pass completed; exhaustive mounted-action conformance remains open. |
| 3 | Package the clock repair | Build one clean replacement app/agent from the repair commit. Exact copied runtime passes native smoke, public adapters, reference parity and the previously failing live pulse case. Record version/hashes and retain rollback. | Complete and retained in review /144: clean source, exact copied runtime and reference acceptance pass. |
| 4 | Hand over the replacement | Explicit app path, matching agent, LED/RC/flasher projects, notices, known limits and the existing 15–20 minute walkthrough. | Corrected app/agent/examples and task sheet delivered at the linked path. Owner window preserved; feedback pending. |
| 5 | Resolve walkthrough failures and promote | No reproduced work loss, misleading result, blocked reference task or core parity defect. Complete supported-alpha gates below, or obtain explicit owner review of permitted private-alpha limitations. Promote once with rollback. | Owner feedback and remaining qualification pending. Waiting does not authorize feature expansion. |
| 6 | Validate the broader MVP | Close original PRD accessibility, agent coverage, distribution, performance and actual hobbyist-study gates; correct observed failures. | Subsequent milestone, not claimed complete by the private alpha. |

**Time-box result:** replacement packaging/qualification completed in this focused release session; the corrected build and walkthrough are ready. Next is owner acceptance and the remaining qualification table below. Any new blocker gets a bounded reproduction/repair session and revised estimate if unresolved. Owner acceptance and broader MVP dates depend on actual access and feedback; do not invent dates for those gates.

## Acceptance tasks

Run these through native controls and the external agent with the same pinned components/models. Retain saved projects and compare connectivity, values and numerical/state outcomes. Reuse existing independent numerical references.

| Task | Observable success |
|---|---|
| A · LED from blank | Place 5 V source, 1 kΩ resistor, LED and ground; wire the loop. Read about 2.9168 mA. Edit to 2 kΩ and see lower current automatically. Undo/redo; disconnect and repair a wire. Incomplete drawings never present old readings as current. |
| B · RC charge/discharge | Build 10 kΩ /1 µF with 0–5 V pulse. Observe voltage; pause/step/resume and measure with cursors. Ideal-step reference: 3.160602794 V at 10 ms within PRD tolerance; use the matching reference for finite ramps. Paused edits have truthful stale/current state. |
| C · Transistor experiment | Load/edit the capacitor-timed NPN flasher, inspect both sides and measure its period. Verify pulse-driven NPN against existing independent references. Smoke-test NMOS/PMOS examples. Include the repaired long-running pulsed capacity regression. |
| D · Keep and share | Save, close/reopen, export readable schematic and scope data. Exercise Cancel/Discard/Save on replacement. Recover a controlled interrupted session without overwriting its last manual save. |

Scripts alone do not establish native focus, typing, dialog or pointer behavior. Native and agent reference evidence already exists; repeat what the repair/package invalidates rather than rebuilding every archive.

## Release-blocker list

| Issue | Severity and closure |
|---|---|
| Retained text field can carry an old private draft into another document/object | **Blocks promotion; source repair verified, replacement package pending.** Driver-routed keyboard tests reproduce both schedules; field removal/remount alone does not cover this case. Public IDs/commands stay unchanged in the proposed text-state identity repair. |
| Held potentiometer drag can overwrite an agent edit or replacement after repaint | **Closed in 0.127.0 /144.** Old SDK fails both interleavings; pinned repair passes normal/repaint/refusal controls, full source quality and clean exact copied-runtime acceptance. [Receipt](ui-audit/first-version/slider-review-checkpoint.json). Actual OS concurrent/focus-loss qualification remains separate. |
| Retained simulation button can act on an agent-replaced experiment | **Closed in 0.126.0 /143.** Tests refuse stale controls and preserve fresh/live-sample behavior; full quality, actual fresh native focus, clean replacement and exact-runtime acceptance pass. [Receipt](../docs/validation/rendered-simulation-buttons/checkpoint.json). |
| Live pulse clock drifts into a sub-minimum integration gap | **Closed in 0.125.0 /142.** Exact copied runtime crosses the old failure, nonzero Resume/save checks and reference parity pass. [Before/after evidence](../docs/validation/live-clock-grid/README.md). |

The rendered simulation-button guard defect is closed in /143. The held-slider overwrite is closed in /144; /143 remains immutable and affected. The owner walkthrough and the broader qualification table remain open. A 64-part/100-node dense topology reaches the existing 100 ms integration budget before a staged sample; graph and committed physics survive. Disclose this solver limit, and do not equate capacity limits with guaranteed convergence. The repaired 60-net topology does not qualify every difficult circuit.

New work loss, corrupted connectivity, misleading simulation, core user/agent mismatch or inability to complete A–D enters this list and blocks promotion. Broader evidence gaps below remain open requirements; they are not inferred passes or silently waived blockers.

## Qualification still required

| Gate | Existing evidence and remaining work |
|---|---|
| Quality/correctness | The [October 18 audit freshness guard](../docs/validation/prd-review-20261018/checkpoint.json) now rejects stale saved status reports; all 27 P0 ID/status pairs remain unchanged. Current /144 source passes `scripts/check.sh`: audit freshness, lint/Staticcheck, 43 Python tests, root/native race/vet, the new offscreen held-slider race regression and tagged vet. Review touched concurrency/error/state ownership; no general restructuring. Exact replacement smoke, public-runtime and acceptance checks also pass. |
| Layout/input | Both themes and app windows at 1280×800 and 1440×900 inspected. Enforced floor 1040×720 mounted units; observed native frame 1053×762 pixels. Core keyboard loop works. One [mounted paused pot drag/Undo](ui-audit/first-version/potentiometer-gesture/checkpoint.json) matches agent physics/history; the held-agent-edit/replacement case now has a failing/passing retained-widget reproduction and isolated SDK repair; [deterministic running/Tab-focus checks](ui-audit/first-version/slider-running-focus/checkpoint.json) now verify live charge continuity, one commit/Undo and public agent replay. A [native unfinished-field replacement](ui-audit/first-version/native-value-replacement/checkpoint.json) removes old text and preserves the replacement on fresh Apply; attempted app-switch/minimize did not establish deactivation. Actual OS concurrent/running/window-blur gestures remain unqualified. Actual OS display modes, monitor transitions, hardware, IME and fast human typing remain unqualified. |
| Visible responsiveness | PRD targets: p95 edit acknowledgment 100 ms and first updated sample 500 ms. Need at least 20 eligible native observations per LED, RC and resistor-loaded NPN reference; flasher is extra stress. Include navigation/cancel/save/undo under difficult solves and cold routing. Backend capacity timings and functional UI runs do **not** establish event-to-pixel timing. Current capture method yields no eligible visible p95 cohort. Report unavailable measurement rather than an inferred pass. |
| Files/recovery | Reference round-trip, failed saves, replacement decisions and controlled crash recovery have evidence. Retain it where unchanged; rerun package-specific file paths. Last manual saves must survive. |
| Agent surface | Reference graph/numerical/state parity passes. The [October 18 native RC control check](ui-audit/first-version/live-control-parity/checkpoint.json) also matches Charge/Discharge/Step/Restart to the external agent and independent RC law. Preserve capabilities, structured outcomes and retry/conflict/cancel behavior. Inventory drift and remaining mounted binding evidence remain explicit; reference parity is not exhaustive conformance. |
| Accessibility | Native Cocoa inspector selector checks pass, but external snapshots can omit visible controls. Complete VoiceOver acceptance is open. Prepared OS-settings test awaits actual permission; a posted question is not approval. |
| Packaging/distribution | Prior copied app/agent pass offline with repository/SDK/toolchain/cache reads and outbound IP denied. Replacement bytes repeat this gate successfully; version/hash are in the replacement receipt. Clean account, other environments, signing/notarization and public distribution remain unqualified. |
| Printing | Native submission of one LED page to the Brother queue completed as job 161. Owner confirmed October 18: “Printed clearly and completely.” Readability/completeness of this page are confirmed; measured margins/media/scale, other settings and device cancellation remain open. Do not print more merely to fill evidence. |
| Acceptance | Owner replied October 18: “Not tried yet” for the four walkthrough tasks; acceptance remains pending. The original ten-person study with corrections remains a broader MVP gate. |

Integrity, simulation truthfulness and core parity cannot be waived. Broader environment/accessibility/timing gaps may be disclosed for a private owner-test alpha; explicit review is required before supported-alpha promotion. Full-MVP requirements remain open until established.

## Rules that keep delivery finite

1. Work in the order above. A new feature is not progress toward this release.
2. A blocker gets one focused reproduction/repair pass, then a checkpoint and revised estimate if unresolved. No open-ended framework investigation.
3. Run focused checks while fixing, then one full gate on final source. Repeat only checks invalidated by changes or failures.
4. Build a new candidate for concrete fixes; do not increment versions for instrumentation or documentation alone.
5. Report the closed gate or actual failing reproduction at each checkpoint. Test totals alone are not delivery progress.
6. Waiting for owner workflow feedback, settings permission or participant access is not authorization to expand scope. Continue only independent release work.
7. Preserve prior app folders and the owner's open work. Promote only a qualified candidate with rollback.

Detailed history remains in the [UI evidence index](ui-audit/first-version/README.md), [consolidated review receipt](ui-audit/first-version/consolidated-review-checkpoint.json), [capacity benchmark](ui-audit/first-version/capacity-responsiveness-checkpoint.json), [clock repair receipt](ui-audit/first-version/live-clock-grid-checkpoint.json) and Git history. This plan governs upcoming work ahead of the historical continuation backlog.
