package workbench

import "context"

type initialSolverFunc func(context.Context, kernelPlan) (*simulationKernel, error)

// Reuse the private numerical kernel so live startup and Verify share models,
// initial conditions and phase results. No clock or service data is touched here.
func solveInitial(ctx context.Context, plan kernelPlan) (*simulationKernel, error) {
	k, err := newSimulationKernel(ctx, plan)
	if err != nil {
		return nil, err
	}
	c, t := plan.compiled, k.trace
	t.SourceID = c.Source.ID
	t.TransistorID = c.Transistor.ID
	t.LEDID = c.LED.ID
	t.CapacitorID = c.Capacitor.ID
	t.OutputNet = c.Output
	if c.Analysis == NPNTransientAnalysis {
		t.SourceID = c.Input.ID
		t.SupplyID = c.Source.ID
		t.ResistorID = c.Resistor.ID
		t.BaseResistorID = c.BaseResistor.ID
		t.OutputNet = c.Collector
	}
	t.Events = []InputEvent{{Time: 0, Mode: "restart"}}
	return k, nil
}
func (s *Service) startInitialLocked(c Compiled) {
	if s.cancel != nil {
		s.cancel()
	}
	mode := s.inputMode
	if s.pendingInputMode != "" {
		mode = s.pendingInputMode
	}
	if mode != "auto" && mode != "high" && mode != "low" {
		mode = "auto"
	}
	plan := kernelPlan{compiled: c, settings: s.transientSettings, mode: mode, run: s.run, revision: s.electrical}
	ctx, cancel := context.WithTimeout(context.Background(), s.initialTimeout)
	s.cancel = cancel
	s.status = "updating"
	solve := s.initialSolver
	s.liveWorkers.Add(1)
	go func() {
		defer s.liveWorkers.Done()
		k, err := solve(ctx, plan)
		s.mu.Lock()
		defer s.mu.Unlock()
		if ctx.Err() == context.Canceled || s.closed || s.run != plan.run || s.electrical != plan.revision {
			return
		}
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		s.cancel = nil
		cancel()
		if err != nil {
			s.status = "error"
			code, id := "mixed-initialize-failed", ""
			if c.Analysis == NPNTransientAnalysis {
				code, id = "solve-failed", c.Transistor.ID
			}
			s.diagnostics = []Diagnostic{{code, err.Error(), id}}
		} else {
			s.result = nil
			s.npn = k.npn
			s.npnHigh = k.high
			s.npnLow = k.low
			s.transient = k.trace
			s.mixedState = k.mixed
			s.mixed = nil
			if k.mixed != nil {
				s.mixed = k.mixed.result
			}
			if c.Analysis == NetworkDCAnalysis {
				s.transient = nil
			}
			s.inputMode = mode
			s.pendingInputMode = ""
			if s.paused {
				s.status = "paused"
			} else {
				s.status = "running"
				if IsTransientAnalysis(c.Analysis) {
					s.launchTransientLocked(c)
				}
			}
		}
		s.revision++
		s.signalLocked("simulation", "diagnostics", "jobs")
	}()
}
