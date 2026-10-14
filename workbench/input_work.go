package workbench

import (
	"context"
	"fmt"
	"time"
)

type InputWork struct {
	ID                 string       `json:"id"`
	Command            string       `json:"command"`
	Status             string       `json:"status"`
	Actor              string       `json:"actor"`
	RunID              uint64       `json:"run_id"`
	ElectricalRevision uint64       `json:"electrical_revision"`
	ActuationRevision  uint64       `json:"actuation_revision"`
	CommandRevision    uint64       `json:"command_revision"`
	Time               float64      `json:"time_s"`
	Mode               string       `json:"requested_mode,omitempty"`
	Wiper              *WiperParams `json:"requested_wiper,omitempty"`
	Error              *Failure     `json:"error,omitempty"`
}

func cloneInputWork(in *InputWork) *InputWork {
	if in == nil {
		return nil
	}
	out := *in
	if in.Wiper != nil {
		v := *in.Wiper
		out.Wiper = &v
	}
	if in.Error != nil {
		v := *in.Error
		out.Error = &v
	}
	return &out
}

type inputSolveFunc func(context.Context, *mixedState, *Transient, Graph, string, string) (*mixedState, *Transient, error)
type inputJob struct {
	work          InputWork
	ctx           context.Context
	cancel        context.CancelFunc
	state         *mixedState
	trace         *Transient
	originalState *mixedState
	originalTrace *Transient
	graph         Graph
	event         string
	reason        string
	mode, pending string
}

func solveInputWork(ctx context.Context, state *mixedState, trace *Transient, graph Graph, mode, event string) (*mixedState, *Transient, error) {
	if mode != "" {
		return overrideMixedContext(ctx, state, trace, mode)
	}
	physical := graphWithAppliedSwitches(graph, state.net.c.MixedGraph)
	c, _, d := physical.Compile()
	if d != nil {
		return nil, nil, fmt.Errorf("%s", d.Message)
	}
	net := newMixedNetwork(c)
	next, err := net.solveContext(ctx, state.x, state.capV, 0, trace, trace.Sample.Time)
	if err != nil {
		return nil, nil, err
	}
	appendTrace(trace, trace.Sample)
	trace.Sample = mixedSample(c, next.result, net.sourceVoltage(c.Source, trace, trace.Sample.Time))
	appendTrace(trace, trace.Sample)
	appendInputEvent(trace, InputEvent{Time: trace.Sample.Time, Mode: event})
	return next, trace, nil
}
func (s *Service) activeInputWorkLocked() bool {
	return s.inputJob != nil && operationActive(s.inputJob.work.Status)
}
func (s *Service) startInputWorkLocked(mode string, wiper *WiperParams, graph Graph, event string) error {
	if s.mixedState == nil || s.transient == nil || s.transient.Stale {
		return &actionError{"unavailable", "wait for a fresh network result before applying an input"}
	}
	s.cancelStepLocked("superseded")
	s.cancelInputWorkLocked("superseded")
	s.actuationRevision++
	s.inputSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	command := "simulation.input.set"
	if wiper != nil {
		command = "potentiometer.wiper.set"
	}
	job := &inputJob{work: InputWork{ID: fmt.Sprintf("input-%d", s.inputSerial), Command: command, Status: "queued", RunID: s.run, ElectricalRevision: s.electrical, ActuationRevision: s.actuationRevision, CommandRevision: s.commandRevision + 1, Time: s.transient.Sample.Time, Mode: mode, Wiper: wiper}, ctx: ctx, cancel: cancel, state: cloneMixedState(s.mixedState), trace: cloneTransient(s.transient), originalState: s.mixedState, originalTrace: s.transient, graph: graph.Clone(), event: event, mode: s.transient.Mode, pending: s.transient.PendingMode}
	s.inputJob = job
	solve := s.inputSolve
	s.liveWorkers.Add(1)
	go s.runInputWork(job, solve)
	return nil
}
func (s *Service) cancelInputWorkLocked(reason string) {
	if s.activeInputWorkLocked() {
		if s.inputJob.reason == "" {
			s.inputJob.reason = reason
		}
		s.inputJob.work.Status = "canceling"
		s.inputJob.cancel()
	}
}
func (s *Service) inputCurrentLocked(job *inputJob) bool {
	return s.inputJob == job && !s.closed && s.run == job.work.RunID && s.electrical == job.work.ElectricalRevision && s.actuationRevision == job.work.ActuationRevision && s.mixedState == job.originalState && s.transient == job.originalTrace && !s.transient.Stale && s.transient.Mode == job.mode && s.transient.PendingMode == job.pending
}
func (s *Service) runInputWork(job *inputJob, solve inputSolveFunc) {
	defer s.liveWorkers.Done()
	s.mu.Lock()
	if job.ctx.Err() == nil {
		job.work.Status = "running"
		s.revision++
		s.signalLocked()
	}
	s.mu.Unlock()
	next, trace, err := solve(job.ctx, job.state, job.trace, job.graph, job.work.Mode, job.event)
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.inputCurrentLocked(job)
	if !current && job.reason == "" {
		job.reason = "superseded"
	}
	if job.reason != "" {
		job.work.Status = job.reason
		job.work.Error = &Failure{Code: job.reason, Message: "Input work discarded; committed graph, time and charge were preserved"}
	} else if err != nil || job.ctx.Err() != nil {
		if err == nil {
			err = job.ctx.Err()
		}
		job.work.Status = "failed"
		job.work.Error = &Failure{Code: "solve-failed", Message: err.Error()}
	} else {
		if job.work.Wiper != nil {
			before := s.historyLocked()
			for i, p := range s.graph.Components {
				if p.ID == job.work.Wiper.ID {
					v := job.work.Wiper.Position
					s.graph.Components[i].WiperPosition = &v
				}
			}
			before.Actor = job.work.Actor
			before.Name = "Adjust wiper"
			before.CommandRevision = job.work.CommandRevision
			s.undo = append(s.undo, before)
			s.redo = nil
			if len(s.undo) > 100 {
				s.undo = append([]historyEntry(nil), s.undo[len(s.undo)-100:]...)
			}
		}
		s.mixedState, s.mixed, s.transient = next, next.result, trace
		// A verification captured while this work was pending used the previous
		// committed inputs. Distinguish admission from successful publication.
		s.actuationRevision++
		if job.work.Mode != "" {
			s.inputMode = job.work.Mode
			s.pendingInputMode = ""
		}
		job.work.Status = "completed"
	}
	job.cancel()
	if s.inputJob == job {
		s.revision++
		s.signalLocked()
	}
}

func (s *Service) inputWorkLocked() *InputWork {
	if s.inputJob == nil {
		return nil
	}
	return cloneInputWork(&s.inputJob.work)
}
