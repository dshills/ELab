# Project command discovery

Users and agents share the existing project dispatcher. Request `capabilities.list {"command":"project.save.start"}` for an expanded input schema and a versioned `x-command-contract`. The ordinary catalog and local registry stay unchanged; detailed annotations are generated only on demand.

Eight commands now describe their actual state requirements and effects:

| Command | Contract |
|---|---|
| `project.inspect` | Owned current snapshot, with no file read, calculation, wait or mutation |
| `project.clear` | Protected New, new document generation, detached file and retained session IDs/history |
| `project.save.start` | Immediate queued admission of an immutable captured save |
| `project.save` | Waiting adapter for that same background work |
| `project.load.start` | Immediate queued admission; complete validated replacement only while its captured context remains current |
| `project.load` | Waiting adapter for that same background open |
| `project.work.inspect` | Owned metadata for an exact retained job, including older terminal outcomes |
| `project.work.cancel` | Exact active job cancellation before publication; observed completion remains separate |

Each schema includes preconditions, effects, errors with admission/work/wait phases, and completion/observation guidance. File commands also describe literal local paths, versions, byte limits, named phases, cooperative deadlines and cancellation. `x-project-work` exposes the shared eight-worker/eight-retained-job limits and status inventory. These are shared limits across file, export, import and recovery work.

## Important client behavior

Save captures committed data at admission. Later edits stay in the editor and can remain dirty after success. An existing associated file is protected by its disk hash; explicit overwrite still rechecks destination identity/content before publication. Once publication begins, cancellation refuses. A save may finish its artifact after New or Close without attaching it to the replacement document. Historical association records describe completion time.

An admitted failure consumes its retry key. After repairing a missing or malformed file, use a new key for a new open attempt. Retrying the same waiting request returns its original outcome; retrying a start command returns its original admission. Omitted/null parameters and distinct path spellings have distinct canonical fingerprints. Request disconnection ends only the wait.

Paths resolve relative to the host working directory. ELab does not expand `~` or environment variables, impose a suffix, trim valid surrounding spaces or create save parents. The limit is 4,096 UTF-8 bytes; operating-system limits may be stricter. Optional Save path accepts omitted/null/empty as the associated path. Optional overwrite/discard booleans accept omitted/null as false. Load path remains required and non-null. Detailed schemas explicitly describe those existing decoder semantics.

New detaches the file but preserves lifetime component/wire ID reservations and the retry cache. Prefer automatic unused IDs when building again within the same session. Undo can restore drawing/settings/preferences, but does not restore the old file association or cleared kept results.

See [background Open](project-work.md), [captured Save](project-save.md) and [agent envelope rules](agent-api.md) for the complete workflows.

## Validation

`elab-conformance --project-contract-suite --artifacts NEW_DIRECTORY` adds a real owned-file workflow to the ordinary three-adapter comparison. Add `--native-binary /absolute/path/to/ELab` to include a fresh packaged native socket session. It compares all eight detailed schemas and normalized public calls, and retains fourteen observed snapshots, the admitted save document, historical job and three observed refusal codes and the actual destination contents after the outside-writer refusal. Each adapter captures an RC project/kept recording, edits after Save admission, protects an external writer, explicitly overwrites, refuses a terminal cancellation and malformed Open, detaches with New, and reopens fresh/paused through both command forms.

The checker rejects missing contracts, incorrect bounds/deadlines/observers/errors, omitted evidence, changed captured content/outcomes/time and adapter schema differences. Artifact paths are normalized for call comparison. Exact inspection/cancellation IDs must target their observed save job before comparison maps them to roles; native startup recovery legitimately shifts counters. Original IDs remain in the report; numerical state, history, kept results and persisted content are checked for preservation. Native viewport/layout publications are not required to be byte-identical. Controlled race tests separately block publication, prove cancellation boundaries and owned result copies, check exact deadline admission, validate literal/null paths, retain IDs after New and exercise failed-open retry after file repair. This does not establish every event ordering, filesystem, mounted input/accessibility or release gate.
