# Protected background shutdown

Native Close keeps the existing Save/Discard/Cancel decision. Dirty work refuses by default. Save completes before admission; Discard requires the explicit shared flag. The request also preserves its document-generation guard so an old choice cannot close an agent's replacement project.

After admission, `closed:true` freezes source mutations immediately. Recovery finalization and worker joining run in a separate worker, outside the editor mutex. The native window shows a short closing status, with editing/file actions disabled, until domain resources are ready to exit. Held recovery serialization no longer prevents snapshots or job inspection. Native startup errors return through deferred cleanup instead of calling `log.Fatal` inside the resource-owning function.

| Command | Behavior |
| --- | --- |
| `app.close.start` | Immediate protected close admission; normal mutation revision/key and optional `expected_document_generation`/`discard_changes` parameters. |
| `app.close` | Same job, waiting until finalization and worker joining return. Successful response can contain a failed shutdown outcome; inspect it. |
| `app.close.inspect {id}` | Pure owned metadata for the one retained process-local shutdown job. No IO, revision change or new work. |
| `app.close.acknowledge {id}` | Exact failed terminal job only. Allows exit, retains failure/errors and does not retry cleanup. Normal mutation guards/key still apply. |

`snapshot.shutdown` exposes job ID, original actor, input/command revisions, document generation, captured recovery directory, discard/checkpoint policy, status/phase, `cancelable:false`, `recovery_finalized`, `workers_joined`, `exit_ready`, separate `recovery_error`/`appearance_error` and a structured error. Metadata and cached admissions own their pointer data. An immediate-start retry returns original admission; inspect the ID for current state. A waiting retry with its original key joins that same job. A different close request after admission refuses `closed`.

The phases are queued, finalizing recovery, joining workers and completed/failed. Finalization serializes with earlier recovery writes using the store lock, while the editor lock remains free. Discard retires only the current document's owned recovery copies. Internal `Service.Close()` checkpoints dirty work when recovery is enabled; explicit saved files and other documents/sessions are preserved. Once the store is closed, a delayed capture cannot recreate discarded data. No recovery configuration is a successful no-op for the recovery finalization step.

Joining waits for actual return of admitted background work, including DC solves, Verify, live initialization/integration, preparation/file/print/read work, appearance persistence, recovery loops and recorded playback. Adds/admission are guarded by the editor mutex and refuse after close. The shutdown worker is outside that group so it cannot join itself. An uncooperative solver or blocked OS call can keep the phase waiting; the API does not forcibly interrupt it or report completion early. Appearance persistence gets its existing final flush; already committed file/spool work retains its own publication/acknowledgement semantics.

Cleanup or preference persistence failure sets failed, records its actual cause and keeps `exit_ready:false`. The native window displays the cause and **Close window anyway**, using the same acknowledgement command an agent can send. A later window-close request also acknowledges that visible failed outcome. Acknowledgement does not remove unexpected files, retry filesystem calls or change a failed outcome to success. Ordinary domain mutations remain refused, including attempts to open another native file picker.

Admission is irreversible: workers are canceled and source editing is frozen, so there is no cancel/resume command afterward. Cancel the unsaved-work decision before admission to keep working. Caller cancellation/deadline affects only a waiting request; accepted shutdown continues. `simulation.wait` retains its existing immediate closed-state behavior and does not wait for shutdown resources. Job queries/events remain available while the host exists; event `closed` also denotes frozen admission. Use `app.close` for a final outcome response, because a fast successful start can end the host before subsequent inspection.

Native and headless socket hosts wait for exit readiness before normal automatic teardown. Stdio keeps serving a failed outcome until acknowledgement; EOF or process signals can still end a host, and internal `Service.Close()` joins outstanding work. Domain exit readiness precedes platform/window/process teardown; it does not certify the OS lifecycle, UI callback queue, physical input, display completion or a general latency bound.

After `app.close.start`, the Python example client's
`wait_for_shutdown_outcome(snapshot['shutdown'])` inspects that exact job ID at
most once per 50 ms by default, with a 15-second cooperative observation budget.
It returns completed **or failed** metadata; failed does not mean exit-ready.
It never acknowledges, retries cleanup, mutates the circuit or closes the client.
Optional cancellation stops only observation. The budget and cancellation are
checked between synchronous RPCs; callers must separately bound transport IO.
Timeout or transport loss is not evidence that cleanup finished. Use waiting
`app.close` when a final response is required, since a successful host can exit
before a later inspection. Keep the explicit acknowledgement response after
reviewing an actual failure; another poll may race host exit.

`events.poll` is not a blocking shutdown wait once the service is closed: it
returns the current closed snapshot immediately, even if nothing changed and
exit is not ready. A custom observer that continues after close needs a positive
client-side backoff, cancellation and a deadline. A failed outcome should be
presented for review, without automatic acknowledgement. The `/176` UI collector's
over-polling remains preserved in its [original evidence](validation/shutdown-layout-review-176/README.md).

The public workflow accepts `--agent-binary PATH` to select a matching packaged
agent or forwarding wrapper; omission keeps the existing `build/elab-agent`
default. `--connect` still requires a designated fresh owned session.

[Client-wait validation](validation/shutdown-client-wait-176/README.md) records
controlled pacing/cancellation/deadline tests and public success/failure runs
against the unchanged `/176` agent. This is developer tooling; no new app
release or native interaction is implied.

The [public workflow](../scripts/shutdown-demo.py) runs success/failure scenarios through headless or an owned native endpoint. It proves dirty/stale refusal purity, exact captured identity, preservation of current source/physics/history and saved bytes, current-document recovery retirement with another document's copy retained, and explicit failure acknowledgement without retry/removal of a nonempty owned fault fixture. Controlled tests hold the recovery mutex and an uncooperative DC solve, verify wait disconnect/retry and joining, inspect cloned metadata, test native buttons/window policy and prevent new pickers. A stdio pipe regression proves failed shutdown remains inspectable until its acknowledgement reply is delivered. Native/GPU smoke closes its owned session only after shared readiness. Complete platform/resource, physical-input/accessibility and visible-latency acceptance remain separate gates.
