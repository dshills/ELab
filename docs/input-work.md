# Background input and wiper work

A fresh general-network transient applies `potentiometer.wiper.set` and running `simulation.input.set` through a private background worker. Resume also uses this worker when it consumes a queued network input. Admission returns immediately with `snapshot.input_work`; it does not publish a requested wiper into the schematic, history or physical readings. DC wipers use ordinary schematic editing and background initialization. Exact RC and specialized NPN input helpers retain their existing inexpensive behavior.

The Inspector shows requested mode/wiper, status, captured time and an exact-ID cancel action. The native slider shows the requested position while work is pending; saved data and readings continue to describe the committed circuit. Users and agents share the same commands and state.

## Agent sequence

1. Submit the command with an observed mutation precondition and retry key.
2. Read the returned `input_work.id`, `status`, `actor`, captured run/revisions/time and requested input.
3. Poll `project.inspect` or call `simulation.wait`. Wait is bounded to five seconds; inspect again if it times out.
4. Require `status: completed` before a dependent manual Step. `failed`, `canceled` and `superseded` preserve committed graph/time/charge and carry an error.

The states are `queued`, `running`, `canceling`, `completed`, `failed`, `canceled` and `superseded`. Only the latest input work is retained. Retrying successful admission returns the independently copied original queued response; it does not launch another calculation. Runtime work is omitted from saved projects and recovery. Reopening starts fresh physical state at time zero.

## Atomic publication and concurrency

The worker owns independent graph, capacitor memory, solution vectors and trace storage. It computes outside the service mutex, with a five-second cancellation context and unchanged solver/model/work limits. Publication requires the same run, electrical/actuation revision, committed state and trace identities, mode/pending input, and a current uncanceled job.

Live time waits while work is active, without accumulating catch-up time. Navigation, geometry, presentation and Save remain usable. Save captures the committed circuit. Successful wiper publication merges just the wiper into the current graph, preserving intervening layout/preferences, and adds one undo entry attributed to the admitting actor and command revision. Its history position follows commit order; its revision records original command admission. Admission and successful publication each advance actuation revision, so a Verify captured while work is pending cannot falsely remain associated after publication. Electrical revision, run, capacitor charge and simulated time are preserved.

Pause/Close cancel work. Electrical edits, restart, undo/redo, project replacement, switch changes and newer inputs supersede it. Returning a pending wiper to its committed position supersedes that wiper request. `simulation.work.cancel {id}` cancels the exact current input or manual-step job; old/unknown IDs refuse. Manual Step refuses with `busy` until input work settles. Cancellation of a Resume-applied input leaves its previous queued mode intact.

## Validation and remaining limits

Controlled race tests block the actual input worker and prove responsive navigation/save, atomic cancellation/failure, stale failure rejection, presentation merge, one actor-owned undo, retry ownership, queued Resume cancellation, frozen live time and verification provenance. The public [headless workflow](validation/headless-input-work.json) and packaged workflow exercise queued admission, successful wiper/input completion and fresh saved reopening. Actual native GPU smoke executes the transient wiper control and renders its completed status/readings.

Run `python3 scripts/input-work-demo.py --output REPORT --artifacts DIRECTORY`. `--connect` replaces only a designated owned development session. Backend admission timing is not input-to-pixel latency. Some file/recovery work remains synchronous; mounted-control focus, full accessibility, performance/reference consolidation, distribution and participant gates remain open.
