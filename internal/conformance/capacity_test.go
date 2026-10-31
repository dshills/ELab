package conformance

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/dshills/elab/workbench"
)

func TestCapacityLimitEnvelopeAcceptsOnlyExactHistoricalOrCurrentFields(t *testing.T) {
	current := workbench.Limits()
	if !capacityLimitsCompatible(current) {
		t.Fatal("current limits refused")
	}
	oldFETs := current
	oldFETs.MOSFETs, oldFETs.PMOS, oldFETs.CombinedMOSFETs, oldFETs.CombinedFETs = 2, 2, 2, 2
	if !capacityLimitsCompatible(oldFETs) {
		t.Fatal("pre-bridge envelope refused")
	}
	for _, hybrid := range []func(*workbench.CircuitLimits){
		func(l *workbench.CircuitLimits) { l.CombinedMOSFETs = 2 },
		func(l *workbench.CircuitLimits) { l.MOSFETs, l.PMOS = 2, 2 },
		func(l *workbench.CircuitLimits) { l.JFETs, l.CombinedFETs = 0, 0 },
	} {
		invalid := current
		hybrid(&invalid)
		if capacityLimitsCompatible(invalid) {
			t.Fatal("accepted an unpublished mixed four-FET envelope", invalid)
		}
	}
	preExpansion := oldFETs
	preExpansion.Transistors, preExpansion.PNP, preExpansion.CombinedBipolar = 2, 2, 2
	if !capacityLimitsCompatible(preExpansion) {
		t.Fatal("pre-expansion limits refused")
	}
	legacy := preExpansion
	legacy.Timers555 = 0
	legacy.PNP, legacy.CombinedBipolar = 0, 0
	if !capacityLimitsCompatible(legacy) {
		t.Fatal("historical limits refused")
	}
	previous := preExpansion
	previous.Timers555 = 0
	if !capacityLimitsCompatible(previous) {
		t.Fatal("pre-timer limits refused")
	}
	for _, mutate := range []func(*workbench.CircuitLimits){
		func(l *workbench.CircuitLimits) { l.Timers555 = 99 },
		func(l *workbench.CircuitLimits) { l.PNP = 99 },
		func(l *workbench.CircuitLimits) { l.CombinedBipolar = 99 },
		func(l *workbench.CircuitLimits) { l.ElectricalNodes = 99 },
		func(l *workbench.CircuitLimits) { l.Transistors = 99 },
	} {
		for _, value := range []workbench.CircuitLimits{current, preExpansion, previous, legacy} {
			mutate(&value)
			if capacityLimitsCompatible(value) {
				t.Fatal("changed capacity limits accepted", value)
			}
		}
	}
	for _, field := range []string{"timer", "pnp", "combined", "old-npn", "old-pnp", "old-combined"} {
		invalid := current
		switch field {
		case "timer":
			invalid.Timers555 = 0
		case "pnp":
			invalid.PNP = 0
		case "combined":
			invalid.CombinedBipolar = 0
		case "old-npn":
			invalid.Transistors = 2
		case "old-pnp":
			invalid.PNP = 2
		case "old-combined":
			invalid.CombinedBipolar = 2
		}
		if capacityLimitsCompatible(invalid) {
			t.Fatal("unpublished hybrid capacity accepted", field)
		}
	}
}

func TestCapacityRCReferenceMatchesIndependentDecimalEigenvalues(t *testing.T) {
	data, err := os.ReadFile("testdata/capacity-rc-reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var frozen struct {
		Version     int      `json:"version"`
		Method      string   `json:"method"`
		Capacitance string   `json:"capacitance_f"`
		Resistance  []string `json:"resistance_ohm"`
		Rows        []struct {
			Time    string    `json:"time_s"`
			Voltage [4]string `json:"voltage_v"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(data, &frozen); err != nil {
		t.Fatal(err)
	}
	if frozen.Version != 1 || len(frozen.Rows) != 12 || frozen.Capacitance != "1e-7" || !reflect.DeepEqual(frozen.Resistance, []string{"14000", "14000", "14000", "13000"}) || !strings.Contains(frozen.Method, "80-decimal-digit Jacobi") {
		t.Fatal("missing independently generated reference coverage")
	}
	wantTimes := []float64{0, .000001, .0001, .001, .005, .01, .025, .05, .1, .125, .15, .2}
	for k, row := range frozen.Rows {
		at, err := strconv.ParseFloat(row.Time, 64)
		if err != nil || at != wantTimes[k] {
			t.Fatal(err)
		}
		actual := capacityRCReference(at)
		for i, text := range row.Voltage {
			want, err := strconv.ParseFloat(text, 64)
			if err != nil || !finiteConformance(want) || !finiteConformance(actual[i]) || math.Abs(actual[i]-want) > 2e-12 {
				t.Fatal("Go matrix exponential differs from independent spectral reference", at, i, actual[i], want, err)
			}
		}
	}
}

func TestCapacityPublicReportRecomputesAndRejectsCorruption(t *testing.T) {
	file, err := os.Open("testdata/capacity-public-reference.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	var reference CapacityResult
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	if err := CompareCapacity(reference, reference); err != nil {
		t.Fatal("genuine public capacity evidence refused", err)
	}
	for _, mutation := range []string{"changed-check-reference", "changed-check-error", "changed-check-tolerance", "missing-case", "wrong-limits", "wrong-size", "wrong-clock", "wrong-engine", "changed-graph", "missing-edge", "changed-row", "changed-fine-row", "foreign-node", "changed-model", "changed-damping", "negative-kcl", "missing-boundary", "changed-boundary", "missing-reopen", "changed-reopen"} {
		t.Run(mutation, func(t *testing.T) {
			var changed CapacityResult
			if err := json.Unmarshal(data, &changed); err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "changed-check-reference":
				changed.Checks[0].Reference += changed.Checks[0].Tolerance / 100
				changed.Checks[0].AbsoluteError = math.Abs(changed.Checks[0].Actual - changed.Checks[0].Reference)
			case "changed-check-error":
				changed.Checks[0].AbsoluteError = math.Nextafter(changed.Checks[0].AbsoluteError, 1)
			case "changed-check-tolerance":
				changed.Checks[0].Tolerance *= 1.01
			case "missing-case":
				changed.Cases = changed.Cases[:2]
			case "wrong-limits":
				changed.Limits.ElectricalNodes = 99
			case "wrong-size":
				changed.Cases[0].Nodes = 99
			case "wrong-clock":
				changed.Cases[0].Settings.HighSeconds = .05
			case "wrong-engine":
				changed.Cases[0].Engine = "elab.network-mna-be.v1"
			case "changed-graph":
				entry := &changed.Cases[0]
				entry.Graph.Components[2].Value *= 2
				raw, _ := json.Marshal(entry.Graph)
				entry.GraphSHA256 = fmt.Sprintf("%x", sha256.Sum256(raw))
			case "missing-edge":
				entry := &changed.Cases[0]
				entry.Rows = append(entry.Rows[:100], entry.Rows[101:]...)
			case "changed-row":
				changed.Cases[2].Rows[10].Values[0] += .1
			case "changed-fine-row":
				changed.Cases[2].RefinedRows[10].Values[2] += .1
			case "foreign-node":
				nodes := changed.Cases[0].Observations[0].Nodes
				for key, value := range nodes {
					delete(nodes, key)
					nodes["foreign"] = value
					break
				}
			case "changed-model":
				changed.Cases[2].Observations[0].MOSFETs[0].Model.Threshold += 1
			case "changed-damping":
				changed.Cases[2].Observations[1].NumericalDamping = 1e-6
			case "negative-kcl":
				changed.Cases[2].Observations[1].KCLResidual = -.01
			case "missing-boundary":
				changed.Cases[0].Boundary = CapacityBoundary{}
			case "changed-boundary":
				changed.Cases[0].Boundary.RejectedStepPreservedState = false
			case "missing-reopen":
				changed.Cases[0].Reopens = changed.Cases[0].Reopens[:1]
			case "changed-reopen":
				changed.Cases[0].Reopens[1].RecordingByteExact = false
			}
			if err := CompareCapacity(reference, changed); err == nil {
				t.Fatal("corrupted capacity evidence accepted")
			}
			if err := CompareCapacity(changed, changed); err == nil {
				t.Fatal("identical corruption accepted as its own reference")
			}
		})
	}
}

func TestCapacityHistoricalFETDiscoveryAllowsOnlyPairedAbsence(t *testing.T) {
	for _, base := range []workbench.CircuitLimits{workbench.Limits(), func() workbench.CircuitLimits {
		v := workbench.Limits()
		v.Transistors, v.PNP, v.CombinedBipolar = 2, 2, 2
		return v
	}()} {
		old := base
		old.MOSFETs, old.PMOS, old.CombinedMOSFETs = 2, 2, 2
		old.JFETs, old.CombinedFETs = 0, 0
		if !capacityLimitsCompatible(old) {
			t.Fatal("pre-JFET discovery envelope refused", old)
		}
		for _, change := range []func(*workbench.CircuitLimits){func(v *workbench.CircuitLimits) { v.JFETs = 0 }, func(v *workbench.CircuitLimits) { v.CombinedFETs = 0 }, func(v *workbench.CircuitLimits) { v.JFETs = 1 }, func(v *workbench.CircuitLimits) { v.CombinedFETs = 3 }} {
			bad := base
			change(&bad)
			if capacityLimitsCompatible(bad) {
				t.Fatal("mixed FET discovery envelope admitted", bad)
			}
		}
	}
}
