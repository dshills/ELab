# Running slider and keyboard focus qualification

October 18, 2026. **No new defect reproduced.** Current production source and pinned SDK are identical to the qualified 0.127.0 /144 package. No version or bundle changes were made.

The consumer test uses the driver's real offscreen retained widgets and pointer/keyboard routing. The fixture supplies ELab's production potentiometer descriptor and context binding, plus a harmless second button to observe keyboard focus moving. A small shared fixture now handles bounded timing, input, frames and cleanup; it does not substitute the production Change callback.

- Tab moves keyboard focus from a held slider to the button without committing the unfinished drag. Releasing commits once. Replacing the document after Tab makes release refuse with the entire domain snapshot unchanged.
- The RC service uses its **real live clock**, independently of the driver's manual gesture clock. Samples advance while the pointer is held, without changing source/history. A solver-publication repaint does not invalidate the original command context.
- Release commits one user history entry and one wiper event, preserves the run/electrical identity, and continues to later accepted output. Left/right event frames retain capacitor voltage within 1e-10 V.
- The pre-event and subsequent charging match independently derived Thevenin RC equations within 2 mV. Public dispatcher commands with actor `agent`, replayed at the observed event time, produce identical components/wires and matching time/voltage within 1e-12 s /2 mV. This replay is in-process; it is not a new external-process transport qualification.
- Pause followed by one shared user Undo restores the complete original graph/history tip, stays paused, and marks retained physics stale. It does not present old solver memory as current.

[Race output](race.txt) records three complete runs: the original four interruption controls, two keyboard-focus cases and the running RC case. In the repeated run, accepted wiper time was 4 ms, event voltage 1.995280651 V, and the later replay voltage 4.626952142 V. The test asserts both the independent equation and native/agent agreement; these are functional/numerical observations, not visible latency samples.

The [full quality gate](quality.txt) passes audit freshness, lint/imports/Staticcheck, 43 Python tests, root/native race and vet, all mounted-slider offscreen race tests, and tagged vet. Default native results were reused from a valid Go test cache because its source did not change; the changed offscreen tests executed afresh. `scripts/check.sh` now selects all `TestMountedSlider` tests so these checks remain in the gate.

Actual OS window deactivation is outside the manual adapter's input domain. Physical running/interleaved pointer behavior, window blur, IME, complete VoiceOver, visible p95 and exhaustive mounted actions remain open. Owner walkthrough is still pending. This evidence narrows EDIT-03/SIM-02 gaps without changing their P0 status or declaring MVP completion.
