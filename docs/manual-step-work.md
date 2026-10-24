# Background manual Step

Native **Step +1 sample** calls `simulation.step.start {duration_s}`. Admission returns a queued ID immediately; the shared worker computes from independent captured graph, phase solutions, capacitor memory and trace state. The last committed live readings remain paused while work is staged. The scope shows staged sample progress and **Cancel step**; numerical errors appear separately in the Inspector.

Only a fully successful job publishes its entire requested advance, including pending input consumption, switch events, capacitor voltages and retained trace. Failure, cancellation or supersession publishes none of that staged physical state. Geometry, selection, viewport and ordinary presentation undo survive successful publication. Pan, save, undo and cancel are available while the worker computes.

## Shared commands and state

| Command/query | Behavior |
|---|---|
| `simulation.step.start {duration_s}` | Immediate queued admission; one active job; one second / 1,000 output samples maximum |
| `project.inspect` | Latest session `step_work`: ID, status, actor, originating run/electrical/actuation/command revisions, start time, duration, staged sample counts/progress and optional error |
| `simulation.work.cancel {id}` | Cancel exactly the displayed active ID; an old ID cannot cancel newer work |
| `simulation.wait {}` | Wait for DC or current manual work, at most five seconds; inspect terminal `step_work` and its error; timeout leaves admitted work running |
| `simulation.step {duration_s}` | Compatibility command: use the same worker and wait for a terminal response; successful response contains completed physical state |

Statuses are queued, running, canceling, completed, failed, canceled and superseded. Progress represents staged computation, not committed time. Notifications are coalesced at roughly 50 ms, with initial and terminal publication. Final counts/progress remain consistent even on cancellation. Job IDs are never reused during a process; only the latest manual job is shown. A prior-run job is labeled Previous in the scope. No manual job is a kept recording or schematic history action.

The planned count uses the same absolute-time output targets as the worker,
including its existing final-tail snapping below 1 ps. Completed work therefore
reaches its declared total even when a duration is just above a sample boundary
or starts at a nonzero time. The duration, numerical equations, output targets
and 1,000-sample admission limit are unchanged; Verify retains its separate
strict final-sample policy.

Electrical/model/settings edits, switch/wiper/input changes, Restart, Resume and project replacement supersede the captured work. Pause and Close cancel it. Resume proceeds from the last committed sample. The worker checks cancellation between output samples and after numerical work; existing bounded kernel work remains in effect. A 30-second wall budget bounds the job. Network/NPN pulse startup is now background work. Some algebraic input/file/recovery work remains synchronous and is a separate work item.

## Waiting, retries and ownership

Immediate-start retry keys preserve their original queued admission snapshot. Query current state separately. The waiting compatibility command finalizes its cached response once before delivering a terminal result; its retries preserve that independently copied terminal snapshot/error and never recompute the advance. Accepted terminal failures retain the admitted retry identity. Admission refusals still leave state unchanged and are not cached.

Canceling the caller's wait or losing its connection does not cancel admitted work. Re-inspect or reconnect with the same original key. Prefer immediate start for a client that needs to query/cancel over its own sequential stdio connection. Both forms share command preconditions, actor attribution, the same numerical worker and physical publication guards. Background completion advances session revision without adding another admitted command or overwriting the latest actor.

Saving during work captures the committed graph/startup preferences and kept records. It does not persist staged charge, job metadata or a running operation. Fresh process reopen starts physical time at zero and has no manual job. Same-process prior-work metadata may remain visible by originating run ID.

## Evidence

Controlled blocked-kernel race regressions exercise navigation, save, presentation undo, busy/stale-ID refusal, exact cancellation, restart/resume/project/input/wiper supersession, request-wait cancellation and successful reconnect retry. A failure after an already staged sample leaves the entire physical trace/state unchanged. Caller mutation of returned errors cannot corrupt retry outcomes. Independent snapshots own the mutable physical fields; compiled networks and retained frame data are immutable.

Native tests prove immediate shared admission and matching completed RC data with agent requests. GPU/UI smoke exercises Step callbacks and scope rendering; smoke/test helpers wait explicitly only to inspect their final outputs. This does not establish visible-latency or screen-reader acceptance.

`scripts/step-work-demo.py` starts a 1,000-sample flasher advance, observes staged progress with unchanged committed readings, pans/saves/moves/undoes, then cancels all staged samples. A second advance completes atomically and matches independent Verify capacitor readings. The waiting command remains compatible, and a saved experiment reopens paused/clean with its recording and fresh physics. Use `--connect` only for an owned endpoint; the demo replaces that project. Use `--output REPORT --artifacts DIRECTORY` to retain evidence.

The complete PRD acceptance matrix remains [partial](../research/11-prd-implementation-audit.md): topology/performance qualification, file responsiveness, mounted gestures, visible latency, accessibility, clean installation and participant gates are still open.

Fresh network input/wiper operations use [background input work](input-work.md); manual Step refuses while that work is active.
