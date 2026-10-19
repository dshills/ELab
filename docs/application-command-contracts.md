# Application discovery contracts

Filtered `capabilities.list {"command":"..."}` now publishes version 1 `x-command-contract` for all 162 current public IDs. The last twenty annotations cover the following existing behaviors. This source change is not in the sealed /169 app/agent. The ordinary catalog, output schemas, protocol/schema versions, mutation classification and handlers are unchanged. Annotation presence is enforced for future commands; behavioral acceptance remains a separate obligation.

| Family | Published behavior |
|---|---|
| `capabilities.list`, `parts.list`, `examples.list` | Pure owned inventories; exact detailed filter, null/UTF-8-byte/unknown-ID errors, model/pin/default/domain and example guidance discovery |
| `events.poll` | Required same-process revision cursor; 0–5000 ms, omission/null 1000 ms, four pending waits; nonblocking versus timed-out idle, coalesced changed/closed snapshots, conflict/busy/canceled outcomes |
| `display.inspect` | Illustrative phase 0–1, omission/null zero, 1 wraps to zero; identified wire readings and cues without advancing physics |
| `display.preferences.set` | All six fields required; 0.01–30 V, conventional/electron direction, motion requires cues, reduced-motion behavior; persisted/dirty/undoable project preferences without an electrical restart |
| `diagnostics.details.set`, `appearance.theme.set` | Required visibility/theme choices, session presentation effects, project/history preservation and separately observed optional theme persistence |
| `appearance.configure`, `appearance.wait` | Explicit local path or empty disable; 4096-byte preference file, two active workers, captured/atomic persistence, global loading/saving/ready/error/disabled outcomes and five-second wait behavior |
| `example.load`, `example.current.reset`, seven legacy loaders | Shipped IDs and alias mapping, dirty/discard and generation/current-example guards; current reset preserves pause, loaders run fresh, file association/recordings are detached and template view/scope defaults restored |
| `circuit.connection.set` | Compatibility membership control for the `R1.2`/`D1.anode` wire only; required terminals, no-op behavior and ordinary topology/history rules. General drawings use `wire.connect`/`object.delete` |

Appearance persistence retains its existing global state instead of a per-operation ID. A successful configuration is admission, not disk completion. Inspect `snapshot.appearance` or use `appearance.wait`; a successful wait may contain `status:"error"` and an error string. Caller timeout does not cancel the store. Reconfigure/disable supersedes work cooperatively; workers occupy slots until they actually return. Disabling does not delete the old file or change theme, and a session theme choice after a persistence error does not overwrite the failed file. This documents the current lifecycle; it does not satisfy the broader requirement for complete asynchronous job identity.

Example aliases retain their existing meanings: `example.reset` loads LED, `example.npn-pulse` loads `blink`, and the other named loaders map to their returned `x-example-id`. `example.current.reset` restores the current recognized template and preserves pause. Save before a replacement when outgoing work must be retained; discard is explicit and does not overwrite the saved file. Undo is not a promise to restore a detached file association or discarded retained operations.

[Source and headless protocol proof](validation/remaining-contracts/README.md) includes original missing annotations, exact unchanged ordinary discovery, actual preference-file/error, example protection, connection and event behavior. The 162 annotations do not establish exhaustive accuracy of all command preconditions/defaults/units/errors, complete native binding coverage, reader/IME usability, release qualification or owner acceptance.
