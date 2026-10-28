package conformance

import "testing"

func TestAnalysisBoundariesInProcessAndSocket(t *testing.T) {
	var reference []AnalysisBoundaryResult
	for _, mode := range []struct {
		name    string
		factory Factory
	}{{"in-process", InProcess}, {"socket", ordinaryAgentFactory(t)}} {
		t.Run(mode.name, func(t *testing.T) {
			results, err := RunAnalysisBoundaries(t.Context(), mode.factory, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != 3 || len(results[0].Commands) == 0 {
				t.Fatal("missing boundary cases/actions")
			}
			if reference == nil {
				reference = results
			}
			if err := CompareAnalysisBoundaries(reference, results); err != nil {
				t.Fatal(err)
			}
			if CompareAnalysisBoundaries(results[:2], results) == nil {
				t.Fatal("missing case accepted")
			}
			changed := append([]AnalysisBoundaryResult{}, results...)
			changed[0].Commands = nil
			if CompareAnalysisBoundaries(results, changed) == nil {
				t.Fatal("missing actions accepted")
			}
			changed = append([]AnalysisBoundaryResult{}, results...)
			changed[0].NoiseRMS *= 1.001
			if CompareAnalysisBoundaries(results, changed) == nil {
				t.Fatal("changed noise accepted")
			}
			changed = append([]AnalysisBoundaryResult{}, results...)
			changed[0].Commands = append([]string{}, results[0].Commands...)
			changed[0].Commands[0] = "wrong"
			if CompareAnalysisBoundaries(results, changed) == nil {
				t.Fatal("changed action accepted")
			}
			if len(results) != 3 {
				t.Fatal("missing cases")
			}
			for i, count := range []int{1, 16, 62} {
				r := results[i]
				if r.Resistors != count || r.Parts != count+2 || r.CapturesReopened != 3 || !r.Edited || r.NoiseRMS <= 0 {
					t.Fatal("incomplete evidence", r)
				}
			}
		})
	}
}
