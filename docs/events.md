# Coalesced agent observation

`events.poll {"after_revision":123,"timeout_ms":1000}` observes the same Service used by native controls and agents. No mutation key or expected revision is needed. The cursor must come from this process's `project.inspect` or a prior poll. A larger cursor returns `conflict`; reconnect to another process requires a fresh inspection.

The result includes `after_revision`, `revision`, `command_revision`, `changed`, `timed_out`, `closed`, `coalesced:true`, and `publications`. A changed or closed result includes an independent complete `snapshot`. An idle result omits `snapshot`; it must not be treated as an empty circuit or replace a client's command guard. Topic records contain the most recent publication revision per category, in a fixed bounded order.

This is latest-state observation, not a lossless event log. Multiple samples/actions may be coalesced; topics are coarse hints, and the complete snapshot is authoritative. Fixed topics cover state, project, editor, view, simulation, diagnostics, jobs, measurements, recordings, recovery, preferences, lifecycle, and library. Successful theme commands publish preferences. Numerical/progress/recovery workers signal the same observer channel.

`timeout_ms` defaults to 1000 when omitted/null, accepts 0–5000, and zero queries immediately. At most four waiting polls are active; a fifth returns `busy`. Waiting releases the editor mutex. Cancellation releases its slot, and close wakes observers even if their cursor already equals the final revision. Requests on one stdio/socket stream are sequential; keep a separate observer connection if issuing controls while waiting.

Discovery advertises the contract under `capabilities.list.events` and local `urn:elab:events:1`, with a local snapshot reference instead of repeated full schemas. Existing request/response/catalog limits remain unchanged. Tests cover idle/timeout, future cursors, cancellation/quota, close, independent cloned state, reconnect/coalescing, physical/job publication, and socket/stdio parity.
