# Native Delete / Undo / Redo · /153

October 18, 2026. This validation-only follow-up extends the [native focus/Right Arrow check](../native-keyboard-153/README.md). Product source, SDK, version, commands, models, project format and existing review packages are unchanged.

A separately identified harness uses the byte-identical 0.136.0 /153 executable, with appearance/recovery persistence disabled. Its original endpoint-backed host is PID 6872. The matching packaged headless agent first seeds a paused 5 V /1 kΩ LED example and produces an independent four-stage public-command reference. A second client seeds the native session identically, then only reads snapshots until all UI stages finish. CUA selects the live app by its unique bundle ID, observes a fresh screenshot, clicks R1 and sends Backspace, Cmd+Z, Cmd+Shift+Z, then Cmd+Z. Each input is followed by a fresh AX observation ([excerpts](ui-observation.txt)). This is automated native OS-routed input, not a human hardware-keyboard trial.

| Native input | Public reference | Components / wires | Reading state |
|---|---|---|---|
| Backspace | `object.delete` R1 | 3 /2 | Previous result stale |
| Cmd+Z | `edit.undo` | 4 /4 | Still stale while paused |
| Cmd+Shift+Z | `edit.redo` | 3 /2 | Still stale while paused |
| Cmd+Z | `edit.undo` | 4 /4 | Still stale while paused |

All four same-PID stages match every component/wire/net field, selection, drawing state, Undo/Redo labels, status, pause state and all LED result fields except independent run/electrical identities ([native journal](native-session.json), [agent reference](agent-reference.json)). The Delete history entry is owned by `user`; Undo exposes that same entry in Redo, and Redo reapplies it. Native command revisions 9, 11, 12 and 14 include automatic viewport observations and are not counts of key presses.

After deletion, the AX state explicitly shows “Ⅱ Paused · stale reading” and “Previous reading · stale,” alongside the 3-component/2-wire footer. Undo restores the graph, not freshness: paused electrical edits remain stale until Resume/Restart under the existing PRD state contract. The AX diff keeps these stale labels throughout the sequence, and every public snapshot has `result.stale=true`. The stored prior current is 0.0029167965774405296 A; it is not claimed as a fresh result for the incomplete circuit. Resume/Restart and fresh post-Undo solving are outside this native sequence.

The controller then saves the final restored graph through public commands ([project](history-restored.elab)) and closes the original native host normally, exit 0. The saved graph matches the final native/reference graph. All owned clients close and final process inventory has no matching harness native process ([verification](verification.json)). This is not a native Save dialog test. The [observer source](observer-source.py) retains a bounded lease, complete snapshots and atomic report publication; it issues no semantic mutations between seeding and the end of UI observation. No owner window or project is operated.

The [read-only source/release identity proof](source-release-identity.json) verifies all 137 existing release-manifest entries, 68 native source hashes and 343 runtime inputs against clean source `7400257` and the packaged agent. The executable checksum is `71dc5a02d876d7fd8937d2b99d18619b0dd79a5c736580e5d321b59b17d37d29`. No rebuild, version increment or promotion occurs. Existing full quality/copied-runtime gates are reused, not rerun; the new evidence is this native history path and its strict agent comparison.

Screenshots were inspected in the tool session but not exported. The durable UI observations are transcribed AX excerpts plus the same-process journal, exact public reference and saved project. This does not qualify native key autorepeat, held/interleaved/blur input, actual field typing/IME, complete VoiceOver, eligible visible p95, other hardware/environments, signing/distribution or the hobbyist study. Tool/poll durations are not event-to-pixel timings. Owner walkthrough remains “Not tried yet,” and P0 remains 17 implemented /10 partial. No OS setting or printer operation occurs. [Checkpoint](checkpoint.json).
