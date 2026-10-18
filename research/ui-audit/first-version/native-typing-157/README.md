# Native typing reproduction · /157

October 18, 2026. **Input qualification fails:** two of six automated Cursor A entries do not produce the requested value on the unchanged /157 native executable. An immediate 8 ms attempt is refused and retains 6 ms. A later requested 14 ms draft visibly remains 12 ms before Enter, then retains 12 ms after Enter. Four attempts match the packaged agent. This is reproducible input-delivery evidence, with cause still unestablished; it is not a product or SDK repair.

The original isolated host is PID 10736. Public commands seed a paused 5 V /10 kΩ /1 µF RC charge, 21 retained samples through 20 ms and [a saved reference](<RC reference.elab>). CUA performs actual native clicks, Cmd+A, text input, Enter and observations. It uses fresh AX targets when the refusal changes indices. [Observer](observer-source.py), [states](native-session.json.gz), [UI transcription](ui-observation.txt).

| Attempt | Requested | Input sequence | Accepted result |
|---|---|---|---|
| 1 | 4 ms | Click/select/type/Enter in one call | 4 ms; agent parity |
| 2 | 6 ms | Click/select/type/Enter in one call | 6 ms; agent parity |
| 3 | 8 ms | Click/select/type/Enter in one call | Refused; prior 6 ms retained |
| 4 | 10 ms | Observe private text before separate Enter | 10 ms; agent parity |
| 5 | 12 ms | Observe private text before separate Enter | 12 ms; agent parity |
| 6 | 14 ms | Observe private text before separate Enter | Private text still 12 ms; accepted 12 ms |

One additional control, excluded from the six, observes focus after clicking before sending Cmd+A/type. It then observes private 14 ms before Enter; the committed result matches the independent packaged agent’s complete scope/transient state. [Control verifier](control-source.py). This suggests examining the asynchronous focus/text publication boundary, but the causal mechanism is not proven. The source review found existing native acknowledgment/stale-publication and input-ordering guards; those intentions cannot establish this missing behavior or justify an untested repair. Next is one bounded causal reproduction around that boundary, with exact request/error and native client/publication observations before changing code.

Every attempt preserves components, wires/nets, complete transient data, run/electrical IDs, pause/settings, document/history and saved bytes. No incorrect new cursor is accepted in the two failures; the previous accepted value remains visible. The raw observer’s `passed` means the observation/preservation protocol completed; [checkpoint](checkpoint.json) explicitly fails requested-input acceptance. Four successes and two failures are observations in this automated sample, not a statistical rate, hardware typing test or latency result.

[Release identity](source-release-identity.json) verifies 68 native source files, 343 runtime inputs and all 158 /157 manifest entries. No product source, SDK, release version, prior evidence or canonical app changed. Existing quality/runtime gates are reused. The original host closes normally, exit 0, with [PID absence verified](process-verification.json). No owner window, printer or OS setting is operated. Full typing/IME/focus, reader, visible-p95, other environments/distribution and owner/participant gates remain open. Owner walkthrough remains Not tried yet; P0 stays 17 implemented /10 partial.
