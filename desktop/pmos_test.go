package main

import (
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestPMOSBlankConstructionUIAgentParity(t *testing.T) {
	testMOSFETBlankConstruction(t, workbench.PMOSExampleGraph())
}
func TestPMOSExampleSharedLoadPulseAndProtection(t *testing.T) {
	testMOSFETExample(t, "pmos", "triode")
}
