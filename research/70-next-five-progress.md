# Next-five execution

October 21, 2026. Work follows [the ordered queue](69-next-five-engineering-items.md), without owner review or feature expansion. Current sealed /195 remains preserved; it does not contain the repair below.

## 1. Demanding simulation interaction

**Reproduced blocker: SAVE-ACTIVE-ANALYSIS-01.** In the exact /195 agent, a coupled flasher with 1 µF timing capacitors and a 10-second/10-ms-output Verify permits real in-flight pan, move and Undo, but Save refuses with `busy: Finish or cancel the active analysis before saving the project`. This blocks the declared save-during-work task. The earlier 100 nF variant reached the existing per-sample integration budget before a usable overlap and remains an unavailable overlap attempt, not a simulator correctness failure.

The repair removes only the blanket analysis admission refusal. Existing serialization already captures completed kept assets atomically under the editor mutex. Verify/sweep work continues independently, unfinished assets are omitted, and later completion remains dirty against the captured save baseline. File-job exclusivity, overwrite/conflict guards, publication, models, formats and solver budgets remain unchanged. Both waiting/start Save contracts and the Save documentation now state this behavior.

Three new deterministic regression cases fail on the original source: verification completion before or after file publication, and a pending sweep. They pass with the repair, preserving earlier recordings, valid selection, live physics, fresh reopening and later-result dirtiness. The old replacement/cancellation regression remains; its obsolete Save-refusal assertion is replaced by these preservation checks.

The repaired development agent passes seven observed real-running spans: pan, move, Undo, Save, value edit, Undo and Pause, followed by explicit Verify cancellation. Before/after observations retain the same running job with successful step progress. The electrical edit truthfully marks paused results stale; Restart restores a fresh result. The original file and 21-row recording remain exact; cancellation publishes no result rows. Owned clients finish joined shutdown and exit zero. This is backend interaction evidence, not visible latency. Affected native Save acceptance is reserved for the matching final candidate.

## 2. Existing user/agent coverage

The native-source inventory has no reported issues and all its source hashes match. Its 32 commands without a direct literal UI route classify as read-only observation, async/waiting counterparts or legacy wiring; that list is not a count of untested human actions. The existing dynamic probe/pin/trace/recording and file-decision tests supply scoped callback comparisons and will run in the full quality gate.

Fresh development-agent checks pass current catalog identity and pure discovery; both corrected Save contracts; dynamic capacitor/probe identities; incompatible units, missing/duplicate probes and unchanged refused state; stale command guards; idempotent edits; conflicting retry keys; and async Save job identity/file preservation on retry. Focused project-contract, disconnected-save-wait, probe/cursor and timing race tests pass. One harness attempt compared an untouched capacitor to an independently rounded decimal literal; the retained correction compares its actual original bits. No product or tolerance change followed.

This closes the selected queue scope. It is not exhaustive every-binding, event/platform or screen-reader acceptance. Native final-package confirmation still follows the source checkpoint.

## Remaining sequence

3. Run the core keyboard workflow on a separately owned matching native candidate, including correction/focus and Save/reopen. Reuse /195 reference physics and unaffected control evidence.
4. Qualify existing schematic SVG/PDF/PNG page bounds, margins, scale and tiling using local files and previews; do not submit a physical print.
5. Complete source/package qualification and designate the engineering release with owner review explicitly deferred. Preserve /189, canonical /120, /195 and all failed evidence. No public-distribution or clean-account pass is inferred.

Raw current-run artifacts are under `/private/tmp/elab-next-five-195`; their producers are under ignored `build/next-five-tools`. The final evidence archive/receipt will freeze them after the remaining checks. No new native app has yet been launched in this queue.
