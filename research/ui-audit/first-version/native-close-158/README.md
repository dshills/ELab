# Native value edit and keyboard close · /158

October 18, 2026. The byte-identical packaged /158 executable accepts complete `2.2k` value input, applies once, protects unsaved work on Cmd+W, preserves it after Cancel, and saves it before normal keyboard-initiated close. A fresh packaged agent reopens the saved circuit with matching connectivity and fresh electrical results. This adds a bounded core-workflow check after the native text handoff repair; no product source or version changes.

The original isolated host is PID 28948. Public commands seed a 5 V /1 kΩ LED loop, save [the baseline](<Original baseline.elab>) and select R1. CUA supplies actual native typing, Apply, Cmd+W and decision-button input. Native SHA is `570b805b42a9f3ee01acb6fecd3bdea5420c57a0b38dd247d1c52689127d82b5`, matching /158; the matching agent SHA is `cea0fd702622b7ded25e4319cd27e69df9ad11c7a7d519a77c2f5347bf7ac042`. [Identity](source-release-identity.json), [observer/controller](observer-source.py), [journal](native-session.json.gz), [UI transcription](ui-observation.txt).

| Native action | Verified outcome |
|---|---|
| Type `2.2k` privately | Complete focused text; accepted graph, scope, physics, document/history and drawing remain unchanged. |
| Apply once | Canonical 2.2 kΩ, fresh 1.344737452917332 mA, one Change value history entry and dirty document; full graph/results/scope match independent packaged-agent commands. Prior save remains unchanged. |
| Cmd+W | Visible unsaved-change decision; accepted edited circuit and prior file remain unchanged. |
| Cancel | Returns to the edited circuit with dirty state, physics, scope, run/electrical identities and history intact; prior file remains unchanged. |
| Fresh Cmd+W, Save and continue | Completed user-owned Save precedes closed=true/clean document. Saved graph/settings/models except view match [independent agent output](<Agent expected.elab>). Original process exits normally, code 0. |
| Fresh agent Open | [Working.elab](Working.elab) reopens from a blank agent process with matching components, wires/nets and fresh complete electrical results. |

The saved file is 2,587 bytes, SHA `5bfc3604190a4b335ad0cc30ce127b3c0ab0645d1206e5bd8072de88d8481e3f`; captured completed `project.save` metadata reports actor `user` and the same hash/bytes. Save/close preserve the accepted graph/scope/physics, run/electrical identities, settings and history. The comparison excludes only saved `view`, since native layout and the headless viewport differ; no byte-identical full-file claim is made.

The separate `events.poll` observer captures a clean closed service snapshot. Its shutdown job is still **queued**, with `exit_ready:false`; this is semantic close evidence, not observation of every cleanup flag. The subsequent original process exits 0 and [PID absence is verified](process-verification.json). CUA’s immediate post-Save query returns “App quit”; both registered test-app inventory entries later report stopped. No `getApp` is called after closure, avoiding a new blank instance. No owner window/project, printer or OS setting is operated.

This qualifies one native ASCII field/Apply and saved-document keyboard close/Cancel/Save path. Graph/selection/baseline save are publicly seeded, not new native construction or Save As. Untitled/native picker, title-bar/Cmd+Q, errors/conflicts/concurrent/held/blur/crash, full reader/IME, visible latency, other environments/distribution/study and owner acceptance remain separate. Earlier /156 file/decision checks retain their recorded scope. Source `622d31b`, SDK `274f5cc`, all 68 native source hashes, 343 runtime inputs and 196 release-manifest files verify unchanged; earlier quality/runtime gates are reused. Owner remains Not tried yet; P0 stays 17 implemented /10 partial. [Checkpoint](checkpoint.json).
