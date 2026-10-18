# Native unsaved-change choices · /156

October 18, 2026. The actual macOS Open panel and visible ELab unsaved-change controls complete Cancel, Discard and Save and continue on disposable LED projects. Native picker Cancel also preserves the circuit. This extends the [clean Save As/Open check](../native-files-156/README.md) without changing product source, SDK, app bytes or version.

The final original endpoint-backed host is PID 76608, using the byte-identical /156 executable, recovery/appearance persistence disabled and the exact packaged agent. The public agent seeds `Working.elab` at 1 kΩ, prepares a separate `Target.elab` at 2 kΩ and changes the working circuit to an unsaved 2.2 kΩ. CUA drives every picker and unsaved choice, with AX observed after selection and each action. The [transcription](ui-observation.txt) and [compressed same-process journal](native-session.json.gz) retain the outcomes.

| Stage | Document generation | R1 Ω | Fresh current mA | Dirty |
|---|---|---|---|---|
| dirty-before-cancel | 2 | 2200 | 1.344737453 | True |
| unsaved-cancel | 2 | 2200 | 1.344737453 | True |
| picker-cancel | 2 | 2200 | 1.344737453 | True |
| discard-loaded | 3 | 2000 | 1.476723549 | False |
| dirty-before-save | 3 | 2200 | 1.344737453 | True |
| save-then-loaded | 4 | 1000 | 2.916796577 | False |

Cancel returns to the same dirty working circuit. Components, wires, nets, selected object, pause/settings, scope, full LED result, run/electrical identities, history entries, document and status remain unchanged. Native picker Cancel preserves these same fields. Automatic viewport and observation revisions are outside this domain comparison. Both on-disk files remain byte-identical to their initial copies.

Discard loads the selected 2 kΩ target through user-owned `project.load`, advances document generation 2→3, clears dirty state and computes a fresh result. Neither file changes. Graph, net, selection, pause/settings, scope, history labels and all result fields match an independent exact packaged-agent discard/load sequence, except independent run/electrical identities.

After the visible Discard result, the public agent changes the loaded target to an unsaved 2.2 kΩ. Native Open selects the original working file; one Save and continue click writes the outgoing target, then loads the working file at generation 4 with fresh 1 kΩ results. [Target.elab](Target.elab) contains the complete edited graph; [Working.elab](Working.elab) remains byte-identical to its original manual save. Every saved document field except viewport matches [the independent public-agent save](Agent-preserved.elab). The final native state matches the agent save/load sequence under the same semantic comparison. A [fresh packaged-agent process](fresh-file-reference.json.gz) separately reopens both resulting files and matches their graph, nets, settings and fresh results. This establishes file preservation, not independent model accuracy.

Two earlier observation runs remain explicitly failed in [observer-failures](observer-failures). PID 75610 stopped after Cancel because the harness incorrectly requested `scope_probes` from a snapshot. PID 76224 stopped after Cancel because an absent optional `redo_entry` was directly indexed. Both owned hosts terminated with -15 during cleanup. These are observer schema failures, not product failures or passing complete runs. The corrected [controller](observer-source.py) compares the actual snapshot `scope` and checks optional-field presence plus value; it passed the retained second-run before/after snapshots before the final run began. It does not suppress an actual missing-versus-present mismatch. All three scripts, reports and host logs are preserved.

The final original native host closed normally through public `app.close`, exit 0. [Process verification](process-verification.json) found no matching final or earlier owned host. The [identity check](source-release-identity.json) verifies clean source `22a84d5`, 68 native source hashes, 343 runtime inputs and all 135 release-manifest entries. Native SHA-256 is `24e98f76934fbc2956474ef0db5bd10ed7d71ef6adf1758039f6222110d56cdf`; agent SHA-256 is `0b0d62a5b34156f04cb4ef298f6a8671af8b7e20e9fa1133c4a8cb79a32de5dd`.

This is one automated native saved-document decision sequence. Untitled Save and continue, save failure/cancel, external-file conflicts, held/concurrent choices, other replacement/close intents and crash recovery retain their separate evidence scopes. No screenshot was exported; no human hardware-keyboard, full reader/IME, visible p95, other environment/distribution/study or owner acceptance is inferred. Existing full quality/copied-runtime gates are reused, not rerun. No product rebuild, version increment, promotion, owner project/window, printer or OS setting change occurs. Owner walkthrough remains “Not tried yet”; P0 stays 17 implemented /10 partial. [Checkpoint](checkpoint.json).
