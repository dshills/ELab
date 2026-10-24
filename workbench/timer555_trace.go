package workbench

import "reflect"

// Coincident timer/source/output boundaries can request the same accepted
// frame repeatedly. Retain all distinct edge sides, but do not invent duplicate
// samples from repeated bookkeeping in the new hybrid engine. Legacy engines
// keep their exact historical frame ordering.
func appendNetworkTrace(t *Transient, p TraceSample) {
	if (t.Engine == Timer555EngineID || t.Engine == SequentialEngineID || t.Engine == ControlEngineID) && len(t.Samples) > 0 && t.Samples[len(t.Samples)-1].Time == p.Time && reflect.DeepEqual(t.Samples[len(t.Samples)-1], p) {
		return
	}
	appendTrace(t, p)
}
