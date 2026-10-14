package conformance

import (
	"fmt"
	"math"

	"github.com/dshills/elab/workbench"
)

func checkNPNPulse(s *session, frozen frozenElectrical, out *ElectricalResult) error {
	c := circuits[2]
	c.parts = append([]part(nil), c.parts...)
	for i := range c.parts {
		if c.parts[i].id == "Drive" {
			c.parts[i].kind = "pulse-source"
		}
	}
	if err := buildElectrical(s, c); err != nil {
		return err
	}
	for _, step := range []float64{.001, .0005} {
		metadata, rows, _, err := electricalVerify(s, .2, step, []string{"v:Transistor.collector", "i:Transistor.collector", "v:Drive.positive"})
		if err != nil {
			return err
		}
		if metadata.Engine != workbench.NPNTransientEngineID || metadata.NPNModel == nil || metadata.Settings.HighSeconds != .05 || metadata.Settings.LowSeconds != .05 {
			return fmt.Errorf("NPN pulse engine/model/stimulus differs")
		}
		m := metadata.NPNModel
		if err = referenceModelMatches(frozen.NPNModel, m.ID, map[string]float64{"saturation_current_a": m.SaturationCurrent, "forward_beta": m.ForwardBeta, "reverse_beta": m.ReverseBeta, "temperature_c": m.Temperature}); err != nil {
			return err
		}
		for _, edge := range frozen.Ideal {
			at, err := refNumber(edge.Time)
			if err != nil {
				return err
			}
			if at > .2 {
				continue
			}
			source, err := refNumber(edge.Source)
			if err != nil {
				return err
			}
			ref := frozen.NPN[0]
			if source == 5 {
				ref = frozen.NPN[2]
			}
			found := false
			for _, row := range rows {
				if math.Abs(row.Time-at) > 1e-12 || len(row.Values) != 3 || math.Abs(row.Values[2]-source) > 1e-9 {
					continue
				}
				found = true
				label := fmt.Sprintf("NPN pulse step=%g t=%g input=%g", step, at, source)
				if err = out.frozenCheck(label+" VCE", row.Values[0], ref.Collector, .001, .01); err != nil {
					return err
				}
				if err = out.frozenCheck(label+" IC", row.Values[1], ref.IC, 1e-9, .01); err != nil {
					return err
				}
				break
			}
			if !found {
				return fmt.Errorf("NPN pulse missing frozen edge/sample t=%g input=%g", at, source)
			}
		}
	}
	return nil
}
