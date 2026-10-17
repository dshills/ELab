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
)

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
	for _, mutation := range []string{"missing-case", "wrong-limits", "wrong-size", "wrong-clock", "wrong-engine", "changed-graph", "missing-edge", "changed-row", "changed-fine-row", "foreign-node", "changed-model", "changed-damping", "negative-kcl", "missing-boundary", "changed-boundary", "missing-reopen", "changed-reopen"} {
		t.Run(mutation, func(t *testing.T) {
			var changed CapacityResult
			if err := json.Unmarshal(data, &changed); err != nil {
				t.Fatal(err)
			}
			switch mutation {
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
