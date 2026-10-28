package workbench

import "strings"

func addBlockCapabilityDetails(id string, schema map[string]any) bool {
	if !strings.HasPrefix(id, "block.") {
		return false
	}
	bounded := func(path string, min, max int) {
		if p := inputProperty(schema, path); p != nil {
			p["minLength"] = min
			p["maxLength"] = max
		}
	}
	bounded("id", 1, 24)
	bounded("name", 1, 64)
	bounded("parameter", 1, 32)
	if p := inputProperty(schema, "members"); p != nil {
		p["minItems"] = 1
		p["maxItems"] = MaxBlockMembers
		p["uniqueItems"] = true
	}
	if p := inputProperty(schema, "ports"); p != nil {
		p["minItems"] = 1
		p["maxItems"] = MaxBlockPorts
	}
	if p := inputProperty(schema, "parameters"); p != nil {
		p["maxItems"] = MaxBlockMembers
	}
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "single-level-circuit-blocks", Preconditions: []string{"Standard actor/revision/document/retry guards apply to mutations; choose IDs from snapshot components", "Block create/paste ID is unused ASCII ValidID, 1–24 characters; members are 1–32 existing unowned ordinary parts, excluding circuit blocks and net labels; pins are 1–16 unique named member terminals; published parameters are unique numeric-valued members", "Every external connection crosses a declared named pin; all physical members and wires count against existing capacities"}, Effects: []string{"Single-level named boundaries and visible projections, not a separate electrical model. Ordinary primitive identities, models, storage and constraints remain authoritative; grouping/release/view/move do not restart physics", "Create, paste, value, move, delete and release participate in ordinary document history; values preserve the target kind's existing units/ranges", "Copy owns an independent in-app clipboard template, survives project replacement and is not an OS clipboard change or saved clipboard. Reopen a project containing a block, Copy, then paste into another drawing. Paste assigns independent IDs/parameters, preserves internal wires and excludes external wires", "Show internals exposes ordinary parts; named pins always remain the external electrical boundary. Release removes only the boundary. Deleting a whole block removes its members/wires; deleting a member first requires Release", "Instances and visual expansion persist in the existing project graph. Named block-pin voltage probes resolve to the exact existing internal net; there is no aggregate block current or hidden capacity"}, Errors: []commandFailureDetail{{Code: "invalid-params", Phase: "admission", When: "Malformed/nonfinite/overlapping/wrong-target/undeclared-crossing parameters preserve graph/history/clipboard"}}, Completion: &commandCompletionDetail{Mode: "immediate graph/session edit; ordinary initialization after an electrical edit is observed separately", Observe: []string{"project.inspect", "simulation.wait", "block.inspect"}}}
	return true
}
