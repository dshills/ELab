package workbench

func addViewportCapabilityDetails(id string, schema map[string]any) bool {
	if id != "view.fit" && id != "view.viewport.set" {
		return false
	}
	c := commandContractDetail{Version: 1, Scope: "schematic-viewport", Preconditions: []string{"Ordinary revision/document guards and retry retention apply"}, Effects: []string{"Session presentation only; preserves graph, physics/run/time/charge, history and document dirty state", "snapshot.viewport exposes measured width_px/height_px, registered, following and optional fit_failure; no OS window resize is requested"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Nonfinite or out-of-range dimensions, missing required size, malformed fields or non-boolean follow_viewport"}, {"conflict", "admission", "Document or command revision changed before this report/navigation"}}, Completion: &commandCompletionDetail{Mode: "immediate shared view/state update; no background solver job", Observe: []string{"project.inspect", "events.poll"}}}
	if id == "view.fit" {
		inputProperty(schema, "follow_viewport")["default"] = false
		c.Effects = append(c.Effects, "Omitted/false follow_viewport preserves legacy one-shot fitting and disables following; omitted dimensions default 650 by 400", "True follows viewport size changes; omitted dimensions inherit registered size, otherwise use legacy defaults; supplied dimensions register the client canvas size", "Pan, zoom, object centering and centered navigation disable following, including accepted zero-distance navigation", "Example load/reset enables following; without a registered canvas its template view remains until the first report; saved file open/recovery and New preserve manual view mode")
		c.Errors = append(c.Errors, commandFailureDetail{"view-limit", "admission", "Whole schematic would require zoom below .01; command refuses without changing view/mode"})
	} else {
		c.Effects = append(c.Effects, "width_px and height_px are both required finite numbers within 64–100000; identical size does not recompute fit", "When following, size changes recompute complete schematic bounds using existing view.fit semantics; manual views keep their center/zoom", "If a valid reported size cannot fit above .01 zoom, accept the measured size, preserve the last view and expose fit_failure=view-limit; a later larger size retries", "Native publication is deferred past paint, coalesced while pending and guarded by captured revision/generation; stale reports retry from a later frame without overwriting navigation")
	}
	schema["x-command-contract"] = c
	return true
}
