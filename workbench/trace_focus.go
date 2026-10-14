package workbench

type TraceFocus struct {
	ProbeID     string   `json:"probe_id"`
	Objects     []string `json:"object_ids"`
	Terminals   []string `json:"terminal_refs"`
	Nets        []string `json:"net_ids"`
	Unavailable bool     `json:"unavailable"`
	Reason      string   `json:"reason,omitempty"`
}

func cloneTraceFocus(in *TraceFocus) *TraceFocus {
	if in == nil {
		return nil
	}
	out := *in
	out.Objects = append([]string{}, in.Objects...)
	out.Terminals = append([]string{}, in.Terminals...)
	out.Nets = append([]string{}, in.Nets...)
	return &out
}
func (s *Service) traceFocusLocked(c Compiled) *TraceFocus {
	if s.focusProbe == "" {
		return nil
	}
	out := &TraceFocus{ProbeID: s.focusProbe, Objects: []string{}, Terminals: []string{}, Nets: []string{}}
	if s.recording.viewing {
		out.Unavailable = true
		out.Reason = "Recorded plot is separate from the current live schematic"
		return out
	}
	for _, p := range probeInventory(s.graph) {
		if p.ID != s.focusProbe {
			continue
		}
		for _, ref := range []string{p.Positive, p.Reference} {
			part, _, ok := s.graph.Terminal(ref)
			if !ok {
				continue
			}
			out.Objects = appendUnique(out.Objects, part.ID)
			out.Terminals = appendUnique(out.Terminals, ref)
			if node := c.TerminalNet[ref]; node != "" {
				out.Nets = appendUnique(out.Nets, node)
			}
		}
		if len(out.Objects) == 0 {
			out.Unavailable = true
			out.Reason = "Probe has no current schematic source"
		}
		return out
	}
	out.Unavailable = true
	out.Reason = "Probe source was removed; undo or select another trace"
	return out
}
func appendUnique(values []string, value string) []string {
	for _, old := range values {
		if old == value {
			return values
		}
	}
	return append(values, value)
}
func (s *Service) focusTraceLocked(p IDParams) error {
	if p.ID == "" {
		s.focusProbe = ""
		return nil
	}
	c, _, _ := s.graph.Compile()
	ids := s.scopeProbes
	if !s.scopeConfigured {
		ids = defaultProbes(c)
	}
	available := false
	for _, probe := range probeInventory(s.graph) {
		if probe.ID == p.ID {
			available = true
			break
		}
	}
	for _, id := range ids {
		if id == p.ID && available {
			s.focusProbe = p.ID
			return nil
		}
	}
	return &actionError{"not-found", "Choose a selected live probe to inspect"}
}
