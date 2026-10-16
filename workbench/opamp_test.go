package workbench

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"testing"
)

type opAmpReference struct {
	Model           string `json:"model"`
	GeneratorSHA256 string `json:"generator_sha256"`
	Cases           []struct {
		Name    string        `json:"name"`
		Volts   opAmpVoltages `json:"volts"`
		Current [5]float64    `json:"current_a"`
		Target  float64       `json:"target_v"`
		Enable  float64       `json:"enable"`
	} `json:"cases"`
	Amplifiers []struct {
		Kind   string  `json:"kind"`
		Vin    float64 `json:"vin_v"`
		Load   float64 `json:"load_ohm"`
		High   float64 `json:"high_v"`
		Low    float64 `json:"low_v"`
		Output float64 `json:"output_v"`
	} `json:"amplifiers"`
}

func opAmpReferences(t *testing.T) opAmpReference {
	t.Helper()
	data, err := os.ReadFile("testdata/opamp-model-reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var reference opAmpReference
	if err := json.Unmarshal(data, &reference); err != nil {
		t.Fatal(err)
	}
	generator, err := os.ReadFile("../scripts/opamp-reference.py")
	if err != nil {
		t.Fatal(err)
	}
	if reference.Model != OpAmpModelID || fmt.Sprintf("%x", sha256.Sum256(generator)) != reference.GeneratorSHA256 {
		t.Fatal("reference does not identify the chosen model/generator")
	}
	return reference
}

func TestOpAmpCurrentLawAgainstFrozenDecimalReferences(t *testing.T) {
	for _, reference := range opAmpReferences(t).Cases {
		t.Run(reference.Name, func(t *testing.T) {
			got := opAmpCurrents(reference.Volts)
			if !finite(got.target) || !finite(got.enable) || math.Abs(got.target-reference.Target) > 1e-10 || math.Abs(got.enable-reference.Enable) > 1e-15 {
				t.Fatal("target/supply enable differs from 90-digit reference", got, reference)
			}
			for row := range got.jacobian {
				for _, value := range got.jacobian[row] {
					if !finite(value) {
						t.Fatal("nonfinite analytic stamp at finite terminal voltages", reference.Name)
					}
				}
			}
			for pin := range got.current {
				if !finite(got.current[pin]) || math.Abs(got.current[pin]-reference.Current[pin]) > 1e-11 {
					t.Fatal("terminal current differs from 90-digit reference", pin, got.current, reference.Current)
				}
			}
		})
	}
}

func TestOpAmpAnalyticJacobianMatchesIndependentPerturbations(t *testing.T) {
	for _, reference := range opAmpReferences(t).Cases {
		if reference.Name == "tiny-span" || reference.Name == "subnormal-supply" {
			continue // Finite differences cannot resolve these tiny supply scales.
		}
		t.Run(reference.Name, func(t *testing.T) {
			got := opAmpCurrents(reference.Volts)
			for column := range opAmpTerminalCount {
				h := 1e-6
				if column < opAmpOutput {
					h = 1e-9 // Resolve the high-gain differential input transition.
				}
				if reference.Name == "soft-start" || reference.Name == "unpowered" {
					h = 1e-11
				}
				plus, minus := reference.Volts, reference.Volts
				plus[column] += h
				minus[column] -= h
				a, b := opAmpCurrents(plus), opAmpCurrents(minus)
				for row := range opAmpTerminalCount {
					measured := (a.current[row] - b.current[row]) / (2 * h)
					exact := got.jacobian[row][column]
					if !finite(exact) || math.Abs(exact-measured) > 1e-10+3e-6*math.Max(math.Abs(exact), math.Abs(measured)) {
						t.Fatalf("pin %d /voltage %d: analytic %.12g, perturbation %.12g", row, column, exact, measured)
					}
				}
			}
		})
	}
}

func TestOpAmpConservesTerminalChargeAndDissipatesPower(t *testing.T) {
	generator := rand.New(rand.NewPCG(42, 91))
	for trial := range 1000 {
		var volts opAmpVoltages
		for pin := range volts {
			volts[pin] = generator.Float64()*60 - 30
		}
		result := opAmpCurrents(volts)
		currentSum, power, scale := 0., 0., 0.
		for pin, current := range result.current {
			currentSum += current
			power += volts[pin] * current
			scale = math.Max(scale, math.Abs(current))
		}
		if !finite(power) || power < -1e-14 || math.Abs(currentSum) > 1e-14*math.Max(scale, 1e-12) {
			t.Fatal("device creates charge/power", trial, volts, result, currentSum, power)
		}
		if math.Abs(result.activeOutput) > OpAmpDeviceModel().ActiveCurrentLimit*result.enable || result.enable < 0 || result.enable > 1 {
			t.Fatal("active output exceeds its declared supply-dependent bound", result)
		}
		for column := range opAmpTerminalCount {
			sum, scale := 0., 0.
			for row := range opAmpTerminalCount {
				sum += result.jacobian[row][column]
				scale = math.Max(scale, math.Abs(result.jacobian[row][column]))
			}
			if math.Abs(sum) > 1e-14*math.Max(scale, 1e-12) {
				t.Fatal("Jacobian violates terminal-current conservation", trial, column, sum)
			}
		}
	}
}

func TestOpAmpHasNoHiddenGroundReference(t *testing.T) {
	volts := opAmpVoltages{.000003, 0, 1.2, 5, -5}
	before := opAmpCurrents(volts)
	for pin := range volts {
		volts[pin] += 12
	}
	after := opAmpCurrents(volts)
	for pin := range before.current {
		if math.Abs(before.current[pin]-after.current[pin]) > 1e-11 {
			t.Fatal("common voltage translation changed terminal current", pin, before.current, after.current)
		}
	}
	if math.Abs(after.target-before.target-12) > 1e-10 {
		t.Fatal("target does not follow translated supplies")
	}
}

func TestOpAmpClosedLoopAmplifiersAgainstIndependentScalarKCL(t *testing.T) {
	for _, reference := range opAmpReferences(t).Amplifiers {
		t.Run(fmt.Sprintf("%s/%g/%g/%g", reference.Kind, reference.Vin, reference.High, reference.Load), func(t *testing.T) {
			// Eliminate the passive feedback node independently; solve only the
			// output's KCL. No network/Newton implementation is used here.
			residual := func(output float64) float64 {
				volts := opAmpVoltages{reference.Vin, output, output, reference.High, reference.Low}
				if reference.Kind == "follower" {
					device := opAmpCurrents(volts)
					return device.current[opAmpOutput] + device.current[opAmpInverting] + output/reference.Load
				}
				if reference.Kind == "noninverting" {
					volts[opAmpInverting] = (output/90000 + reference.Vin/1e7) / (1./90000 + 1./10000 + 1e-7)
					return opAmpCurrents(volts).current[opAmpOutput] + output/reference.Load + (output-volts[opAmpInverting])/90000
				}
				volts[opAmpNoninverting] = 0
				volts[opAmpInverting] = (reference.Vin/10000 + output/20000) / (1./10000 + 1./20000 + 1e-7)
				return opAmpCurrents(volts).current[opAmpOutput] + output/reference.Load + (output-volts[opAmpInverting])/20000
			}
			low, high := -30., 30.
			for range 100 {
				mid := .5 * (low + high)
				if residual(mid) > 0 {
					high = mid
				} else {
					low = mid
				}
			}
			output := .5 * (low + high)
			if math.Abs(output-reference.Output) > 1e-8 || math.Abs(residual(output)) > 1e-11 {
				t.Fatal("amplifier differs from high-precision scalar KCL", output, reference.Output, residual(output))
			}
			t.Logf("output %.12g V", output)
		})
	}
}
