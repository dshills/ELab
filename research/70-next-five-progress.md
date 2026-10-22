# Next-five execution

October 21, 2026. Work follows [the ordered queue](69-next-five-engineering-items.md), without owner review or feature expansion. Current sealed /195 remains preserved; it does not contain the repair below.

## 1. Demanding simulation interaction

**Reproduced blocker: SAVE-ACTIVE-ANALYSIS-01.** In the exact /195 agent, a coupled flasher with 1 µF timing capacitors and a 10-second/10-ms-output Verify permits real in-flight pan, move and Undo, but Save refuses with `busy: Finish or cancel the active analysis before saving the project`. This blocks the declared save-during-work task. The earlier 100 nF variant reached the existing per-sample integration budget before a usable overlap and remains an unavailable overlap attempt, not a simulator correctness failure.

The repair removes only the blanket analysis admission refusal. Existing serialization already captures completed kept assets atomically under the editor mutex. Verify/sweep work continues independently, unfinished assets are omitted, and later completion remains dirty against the captured save baseline. File-job exclusivity, overwrite/conflict guards, publication, models, formats and solver budgets remain unchanged. Both waiting/start Save contracts and the Save documentation now state this behavior.

Three new deterministic regression cases fail on the original source: verification completion before or after file publication, and a pending sweep. They pass with the repair, preserving earlier recordings, valid selection, live physics, fresh reopening and later-result dirtiness. The old replacement/cancellation regression remains; its obsolete Save-refusal assertion is replaced by these preservation checks.

The repaired development agent passes seven observed real-running spans: pan, move, Undo, Save, value edit, Undo and Pause, followed by explicit Verify cancellation. Before/after observations retain the same running job with successful step progress. The electrical edit truthfully marks paused results stale; Restart restores a fresh result. The original file and 21-row recording remain exact; cancellation publishes no result rows. Owned clients finish joined shutdown and exit zero. This is backend interaction evidence, not visible latency. Matching /196 native Command-S acceptance now passes while the same Verify advances from 50% to 57%; subsequent completion is truthfully dirty and was saved separately.

## 2. Existing user/agent coverage

The native-source inventory has no reported issues and all its source hashes match. Its 32 commands without a direct literal UI route classify as read-only observation, async/waiting counterparts or legacy wiring; that list is not a count of untested human actions. The existing dynamic probe/pin/trace/recording and file-decision tests supply scoped callback comparisons and will run in the full quality gate.

Fresh development-agent checks pass current catalog identity and pure discovery; both corrected Save contracts; dynamic capacitor/probe identities; incompatible units, missing/duplicate probes and unchanged refused state; stale command guards; idempotent edits; conflicting retry keys; and async Save job identity/file preservation on retry. Focused project-contract, disconnected-save-wait, probe/cursor and timing race tests pass. One harness attempt compared an untouched capacitor to an independently rounded decimal literal; the retained correction compares its actual original bits. No product or tolerance change followed.

This closes the selected queue scope. It is not exhaustive every-binding, event/platform or screen-reader acceptance. Native final-package confirmation still follows the source checkpoint.

## 3. Core keyboard workflow

/196 blank LED placement, four named-terminal wires, invalid-resistance refusal and correction, native Save/Open pass using keyboard input. Readings match 2.917 mA at 1 kΩ and 1.477 mA at 2 kΩ. RC voltage/current probe choices and cursor times 1/10 ms commit on Tab and yield Δt 9 ms and 3.161 V at 10 ms. A control timeout/unintended Save panel was dismissed without writing. Invalid cursor Return resets the rejected draft with generic refusal; valid correction works. This is scoped core-keyboard evidence, not exhaustive draft-retention/IME/accessibility acceptance.

## 4. Schematic output

Native page controls match public A4/12 mm settings. Existing SVG/PDF/PNG commands produce complete fitted Letter/A4 drawings and all 12 fixed-scale tile pages. Page size, bounds, margin pixels, scale/overlap footers and preview/direct PNG byte matching pass. Independent parser/Poppler renders were visually inspected. Invalid settings/page identities refuse without replacing work. No physical printer job was submitted.

## 5. First engineering release

**Completed October 22:** [0.179.0 /196 first private engineering alpha](71-first-engineering-release.md). Full quality matches 672 source hashes; exact native/agent references, three-adapter project/electrical regressions, signature, extraction, three example reopens and final joined shutdown pass. No new feature or dependency upgrade. Owner review remains deferred; no full PRD/public-distribution or clean-account pass is inferred.

[The frozen receipt and raw evidence](../docs/validation/engineering-release-196/README.md) retain original failures, corrected harness attempts, the interrupted first /196 session and its disposition. The separate final owned native session supplies normal shutdown proof. /189, canonical /120, /195, locked dependencies and old candidates are preserved. No further item in this bounded queue remains.
