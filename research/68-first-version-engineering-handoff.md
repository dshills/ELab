# First-version engineering handoff

October 21, 2026. **ELab 0.178.0 /195 has passed the private-alpha engineering reference checks. Owner testing and release designation remain.** No feature work or further dependency upgrade is needed for this cut.

Use the existing [review app](../build/ELab-0.178.0-195-candidate/ELab.app), matching [agent](../build/ELab-0.178.0-195-candidate/elab-agent) and [four-task walkthrough](35-first-version-review.md). The owner’s latest answer for this exact candidate is **“Not tried yet.”** /189 is preserved; /195 is not promoted.

## Completed engineering work

| Area | Evidence and disposition |
|---|---|
| Native correction cases | /194 duration/reset/sweep/threshold/probe-interleaving cases plus /195 numeric-index, Previous/Play, coupled interval, Verify sampling and one-row DC recovery pass. [Combined walkthrough](../docs/validation/native-walkthrough-195/README.md). |
| Reference UI | Both themes at the enforced minimum and a larger working size; keyboard scrolling, selected-object/value recovery, scope expansion and menus remain usable. This is the reference-control pass, not exhaustive accessibility acceptance. |
| LED drawing | Built from blank through native placement/wiring, edited, undone/redone, disconnected and repaired. 2.917 mA at 1 kΩ; 1.477 mA at 2 kΩ. Incomplete wiring labels the previous value stale. |
| RC | Native charge/discharge and Pause/Step/Resume/Restart, voltage/current probes and cursors. 3.160602794 V at 10 ms; 1.162720790 V after another 10 ms of discharge. All 40 CSV rows match trace data and the applicable independent RC law. |
| Flasher | Both sides measured. Doubling timing capacitors approximately doubles the period from 488 to 976 ms. Matching agents reproduce both endpoint states and cursors. |
| Files/parity | /195 native Save/Open/SVG/cancellation and CSV, six fresh project reopens, plus a seventh fresh counterpart for final index/sampling checks. Graphs, values, models, probes, results and kept recordings agree. Prior /193 file/adapter contracts and native decisions/recovery evidence are reused where unchanged. |
| Inspector blocker | Replacement documents receive their own scroll viewport. The original failing mounted regression, full source gate and native repair check are retained. Ordinary edits preserve scroll. |
| Quality/package/offline | All 671 Go hashes match the full quality gate. Exact copied app/agent works with source/SDK/toolchain/cache access and outbound IP denied. Existing signature/ZIP/extraction/example checks remain valid; package bytes/modes are preserved. All owned native and counterpart sessions close normally. |

The [detailed report](../docs/validation/native-walkthrough-195/report.json) distinguishes native input, public fixture setup/saves, independent numerical references and same-engine parity. Earlier [native/package](../docs/validation/inspector-candidate-195/README.md) and [input](../docs/validation/native-acceptance-194/README.md) evidence retain their original scope. No new binary was built for this follow-up.

## Remaining first-version work, in order

1. Obtain owner results for the four /195 walkthrough tasks. “Not tried yet” does not pass or fail a task. Do not repeat reminders or substitute engineering tests for acceptance.
2. Diagnose any reproduced blocked task, lost work, misleading result or wrong target. Minor framework wording in numeric-validation feedback remains; correction and transport recovery pass. No reproduced engineering blocker remains in the declared private-alpha checks.
3. After owner review, designate one app/agent pair, reconcile release notes/status with accepted evidence, and preserve /189 and rollback. The candidate and its packaging notes remain sealed until that decision; passing tests do not promote it.

Ordinary default-startup/clean-account qualification is explicitly deferred for this private alpha. Tested native environment: macOS 26.6.2 arm64, local ad-hoc signature, no notarization/public-distribution qualification. Full PRD accessibility, exhaustive command coverage, performance, hobbyist study, printing settings and distribution remain separately listed in the [checklist](66-first-version-completion-checklist.md).
