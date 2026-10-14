package workbench

import "context"

type networkAdvanceFunc func(*mixedState, *Transient, float64, Graph) (*mixedState, *Transient, error)

// liveNetworkSampleLocked returns with the service lock held, but performs the
// numerical work without it. The captured kernel, capacitor memory and trace
// are independent of published data, including the mutable stale flags.
// Publication needs the same run, applied state, input revision and trace.
// Presentation edits can proceed without invalidating an otherwise valid sample.
func (s *Service) liveNetworkSampleLocked(ctx context.Context, run uint64, end float64) (bool, error) {
	applied, trace, actuation := s.mixedState, s.transient, s.actuationRevision
	captured := cloneMixedState(applied)
	input := cloneTransient(trace)
	desired := s.graph.Clone()
	advance := s.networkAdvance
	s.mu.Unlock()
	next, samples, err := advance(captured, input, end, desired)
	s.mu.Lock()
	if ctx.Err() != nil || s.closed || s.paused || s.run != run || s.mixedState != applied || s.transient != trace || s.transient.Stale || s.actuationRevision != actuation {
		return false, nil
	}
	if err != nil {
		return true, err
	}
	s.mixedState, s.mixed, s.transient = next, next.result, samples
	s.revision++
	s.signalLocked("simulation")
	return true, nil
}
