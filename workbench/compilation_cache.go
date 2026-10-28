package workbench

import (
	"crypto/sha256"
	"encoding/json"
	"maps"
	"slices"
)

// compilationCache retains one private compilation. Its owner must serialize
// access (Service uses Service.mu). Neither source graphs nor returned results
// share mutable storage with the entry. Numerical workers never own this store.
type compilationCache struct {
	valid      bool
	key        [32]byte
	compiled   Compiled
	status     string
	diagnostic *Diagnostic
}

func (s *Service) compileGraphLocked() (Compiled, string, *Diagnostic) {
	return s.compilation.compile(s.graph)
}

func (c *compilationCache) compile(graph Graph) (Compiled, string, *Diagnostic) {
	// Include all graph metadata: positions, labels, initial conditions, physical
	// models and nested blocks can appear in Compiled even without new topology.
	// JSON also distinguishes signed zero. Invalid/nonfinite inputs never alias
	// the previous entry through an ignored encoding error.
	data, err := json.Marshal(graph)
	if err != nil {
		compiled, status, diagnostic := graph.Compile()
		return cloneCompiled(compiled), status, cloneCompileDiagnostic(diagnostic)
	}
	key := sha256.Sum256(data)
	if !c.valid || c.key != key {
		compiled, status, diagnostic := graph.Compile()
		*c = compilationCache{valid: true, key: key, compiled: cloneCompiled(compiled),
			status: status, diagnostic: cloneCompileDiagnostic(diagnostic)}
	}
	return cloneCompiled(c.compiled), c.status, cloneCompileDiagnostic(c.diagnostic)
}

func cloneCompileDiagnostic(in *Diagnostic) *Diagnostic {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneCompiled(in Compiled) Compiled {
	out := in
	out.MixedGraph = cloneCompilationGraph(in.MixedGraph)
	parts := []Component{in.Source, in.Resistor, in.LED, in.Capacitor, in.Transistor, in.Input, in.BaseResistor}
	owned := cloneCompilationGraph(Graph{Components: parts}).Components
	out.Source, out.Resistor, out.LED, out.Capacitor = owned[0], owned[1], owned[2], owned[3]
	out.Transistor, out.Input, out.BaseResistor = owned[4], owned[5], owned[6]
	out.TerminalNet = maps.Clone(in.TerminalNet)
	out.Nets = slices.Clone(in.Nets)
	for i := range out.Nets {
		out.Nets[i].Terminals = slices.Clone(in.Nets[i].Terminals)
		out.Nets[i].Labels = slices.Clone(in.Nets[i].Labels)
	}
	return out
}

func cloneCompilationGraph(in Graph) Graph {
	out := in.Clone()
	// Graph.Clone's drawing API normalizes empty arrays. Preserve the compiler's
	// original nil/empty representation, including zero-valued shortcut parts.
	if in.Components == nil {
		out.Components = nil
	}
	if in.Wires == nil {
		out.Wires = nil
	}
	for i := range out.Components {
		if in.Components[i].Terminals == nil {
			out.Components[i].Terminals = nil
		}
	}
	return out
}
