package workbench

func addProjectFileDetails(id string, schema map[string]any, contract *commandContractDetail) bool {
	saving := id == "project.save" || id == "project.save.start"
	loading := id == "project.load" || id == "project.load.start"
	if !saving && !loading {
		return false
	}
	path := inputProperty(schema, "path")
	path["maxLength"], path["x-max-utf8-bytes"] = 4096, 4096
	path["description"] = "Literal local filesystem path; nonempty paths must not be whitespace-only or contain NUL. Relative paths resolve with filepath.Abs in the host process working directory; ~ and environment variables are not expanded. Other leading/trailing spaces are preserved; no suffix is required. Save does not create parent directories. OS/filesystem limits can be stricter."
	schema["x-file-format"] = map[string]any{"format": DocumentFormat, "default_write_version": DocumentVersion, "kept_sweep_write_version": SweepDocumentVersion, "read_version_minimum": 1, "read_version_maximum": SweepDocumentVersion, "max_bytes": MaxProjectBytes, "legacy_max_bytes": MaxDocumentBytes, "legacy_versions": []int{1, 2, 3}, "models": "Embedded curated model dependencies and retained completed kept results; no model network fetch"}
	contract.Errors = append(contract.Errors,
		commandFailureDetail{"busy", "admission", "Shared file workers are exhausted or incompatible project work is active; canceling work keeps its slot"},
		commandFailureDetail{"conflict", "admission", "Supplied parameter/document generation differs"},
		commandFailureDetail{"file-error", "project_work", "Missing/inaccessible/nonregular/changed file, unsupported filesystem operation or IO failure"},
		commandFailureDetail{"invalid-document", "project_work", "Format/model/graph/settings/embedded result or file size invalid"},
		commandFailureDetail{"timeout", "project_work", "Cooperative 30-second deadline expires; a blocked OS call may return later"},
		commandFailureDetail{"canceled", "project_work", "Explicit cancellation before publication; captured state is not installed"},
		commandFailureDetail{"superseded", "project_work", "Newer incompatible document/source/work makes captured publication obsolete"})
	mode := "immediate queued admission, not completion; retry returns owned original admission"
	if id == "project.save" || id == "project.load" {
		mode = "waiting alias for the same admitted job; terminal error plus owned job snapshot on failure; owned retry joins original terminal outcome without repeating IO"
		contract.Errors = append(contract.Errors, commandFailureDetail{"canceled", "wait", "Caller disconnect/cancellation ends only the wait; admitted work continues"}, commandFailureDetail{"operation-unavailable", "wait", "Original admitted job has expired"})
	}
	contract.Completion = &commandCompletionDetail{Mode: mode, Observe: []string{"project.work.inspect", "project.inspect", "events.poll"}, Cancel: "project.work.cancel", CooperativeDeadlineSeconds: 30}
	contract.Effects = []string{"Preparation/validation/encoding/disk IO occur outside the editor mutex; progress is named phases/bytes, not a percentage", "Admission consumes the actor/retry key even if later work fails; after correcting a file use a new key for a new attempt. Retry the same canonical parameter content: omitted/null fields and different path spellings have distinct fingerprints. A canceled waiting request does not cancel admitted work", "No partial result is published. Runtime jobs are not stored in projects; source generation/result identity and optional artifact hash describe captured work"}
	if saving {
		path["type"], path["default"] = []string{"string", "null"}, ""
		path["description"] = path["description"].(string) + " Omitted/null/empty uses snapshot.document.path; without an association choose a destination first."
		contract.Preconditions = []string{"Syntactically valid explicit path or existing document association; drawing may be empty/incomplete/unsupported", "No active analysis/remaining sweep worker; no active shared project job except an Open, which Save supersedes even if Save later fails", "At most eight shared file workers, including canceled/superseded workers that have not returned"}
		contract.Effects = append(contract.Effects, "Capture graph/settings/startup/models, kept results, view/probes/pins/display/example and pause preference at admission. Later edits remain in the editor; later schematic/settings/pin/display/example/kept-result changes stay dirty. View/probe/pause-only changes retain the existing non-dirty policy; no history entry or physics restart", "Existing other destinations require overwrite=true; the associated file uses its saved disk hash unless overwrite=true. Recheck regular-file identity/content immediately before replacement even after explicit overwrite; new destinations are published exclusively", "Once publishing starts cancellation refuses. Publication can finish after document replacement/Close but cannot associate that artifact with the replacement; inspect associated_with_current_document and result_document_generation", "Success reports artifact_sha256 and updates only the captured saved baseline when still associated. Temp/recovery cleanup warnings are separate from successful publication; directory sync is best effort, not power-loss qualification")
		contract.Errors = append(contract.Errors, commandFailureDetail{"unavailable", "admission", "No explicit or associated destination"}, commandFailureDetail{"file-exists", "project_work", "Unapproved existing/newly created destination"}, commandFailureDetail{"file-conflict", "project_work", "Associated disk baseline changed, or destination changed during preparation"})
		schema["x-phases"] = []string{"queued", "serializing", "checking-destination", "writing", "publishing", "cleanup", "recovery-cleanup", "completed"}
		schema["examples"] = []any{SaveParams{}, SaveParams{Path: "My circuit.elab"}}
	} else {
		path["minLength"] = 1
		contract.Preconditions = []string{"Required non-null nonempty local path; any existing project job including canceling must finish first", "Dirty schematic/preferences/kept results require discard_changes=true; that consent covers captured work, not later edits", "At most eight shared file workers, including workers not yet returned"}
		contract.Effects = append(contract.Effects, "Require a bounded regular file; refuse links/special files and strict format/model/data errors. Only a complete validated current capture atomically replaces the document", "Intervening content/settings/pins/display/example, recording/sweep membership/sequence, source association/disk baseline, input mode or actuation change supersedes Open. Selection/pan are allowed; success restores the file's saved view/preferences", "Success advances document generation, associates the opened file, restores saved pause/kept results and starts fresh physics rather than resuming saved live readings. File completion need not mean numerical readiness; use simulation.wait and inspect results", "Changed graph/settings/preferences add an admitting-actor undo entry; Undo preserves the opened file association and marks restored older content dirty. Failure/cancel preserves the existing document, physical state, save point and history except job/admission bookkeeping")
		contract.Errors = append(contract.Errors, commandFailureDetail{"unsaved-changes", "admission", "Current dirty document lacks explicit discard"})
		schema["x-phases"] = []string{"queued", "reading", "validating", "completed"}
		schema["examples"] = []any{LoadParams{Path: "My circuit.elab"}}
	}
	return true
}
