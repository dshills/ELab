package main

import (
	"fmt"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func noiseReadouts(r *workbench.SweepResult) []ui.Node {
	if r == nil || r.Noise == nil {
		return nil
	}
	if !workbench.NoiseSummaryValid(r) {
		return []ui.Node{quiet("noise-unavailable", "Noise result is incomplete")}
	}
	n := r.Noise
	if n == nil {
		return nil
	}
	nodes := []ui.Node{section("noise-title", "Integrated noise"), quiet("noise-model", n.Model), quiet("noise-band", fmt.Sprintf("%.5g…%.5g Hz · sampled linear PSD integral", r.Start, r.Stop))}
	for j, p := range r.Probes {
		input := "Input referred: unavailable (zero transfer in band)"
		if n.InputDefined[j] {
			input = fmt.Sprintf("Input referred %.5g µV RMS", n.InputRMS[j]*1e6)
		}
		nodes = append(nodes, quiet(fmt.Sprintf("noise-rms-%d", j), fmt.Sprintf("%s\nOutput %.5g µV RMS\n%s", p.Label, n.OutputRMS[j]*1e6, input)))
		values := []string{}
		for _, c := range n.Contributors {
			values = append(values, fmt.Sprintf("%s · %.4g µV RMS", c.ID, c.RMS[j]*1e6))
		}
		nodes = append(nodes, quiet(fmt.Sprintf("noise-contributors-%d", j), strings.Join(values, "\n")))
	}
	return append(nodes, quiet("noise-help", "Declared uncorrelated white sources only. No 1/f, excess/avalanche noise, correlations or manufacturer fit. Refine the frequency grid to check RMS."))
}
