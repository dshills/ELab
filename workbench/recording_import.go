package workbench

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"
)

const MaxRecordingBytes = 8 * 1024 * 1024

type RecordingImportParams struct {
	Path               string  `json:"path"`
	ExpectedGeneration *uint64 `json:"expected_document_generation,omitempty"`
}

func decodeRecording(data []byte) (*verificationRecord, error) {
	var d VerificationDocument
	if len(data) > MaxRecordingBytes {
		return nil, fmt.Errorf("recording exceeds 8 MiB")
	}
	if err := DecodeStrict(data, &d); err != nil {
		return nil, err
	}
	if err := recordingFields(data, reflect.TypeOf(d)); err != nil {
		return nil, err
	}
	m := d.Metadata
	if m.Format != "elab.verification" || m.Version != 1 || !ValidID(m.OperationID) {
		return nil, fmt.Errorf("unsupported recording format/version or origin ID")
	}
	if _, err := time.Parse(time.RFC3339Nano, m.CreatedAt); err != nil {
		return nil, fmt.Errorf("invalid recording timestamp")
	}
	// Reuse the schematic validator without applying live playback constraints
	// to the offline sample step. Raw graph fields preserve required-field checks.
	var raw struct {
		Metadata map[string]json.RawMessage `json:"metadata"`
	}
	_ = json.Unmarshal(data, &raw)
	envelope := map[string]any{"format": DocumentFormat, "version": 1, "graph": raw.Metadata["graph"], "view": View{Zoom: 1}, "led_model": m.LEDModel}
	if m.WaveformModel != nil {
		envelope["waveform_model"] = m.WaveformModel
	}
	if m.PotentiometerModel != nil {
		envelope["potentiometer_model"] = m.PotentiometerModel
	}
	if m.DiodeModel != nil {
		envelope["diode_model"] = m.DiodeModel
	}
	if m.OpAmpModel != nil {
		envelope["opamp_model"] = m.OpAmpModel
	}
	if m.PNPModel != nil {
		envelope["pnp_model"] = m.PNPModel
	}
	if m.PMOSModel != nil {
		envelope["pmos_model"] = m.PMOSModel
	}
	if m.NMOSModel != nil {
		envelope["nmos_model"] = m.NMOSModel
	}
	if m.SwitchModel != nil {
		envelope["switch_model"] = m.SwitchModel
	}
	if m.NPNModel != nil {
		envelope["npn_model"] = m.NPNModel
	}
	encoded, _ := json.Marshal(envelope)
	if _, err := decodeDocument(encoded); err != nil {
		return nil, fmt.Errorf("recorded schematic: %w", err)
	}
	c, _, diagnostic := m.Graph.Compile()
	if diagnostic != nil || c.Analysis != m.Analysis || !recordingEngineCompatible(m.Analysis, m.Engine) {
		return nil, fmt.Errorf("unsupported or inconsistent recorded analysis/engine")
	}
	if err := validateRecordingSettings(m.Settings, IsTransientAnalysis(m.Analysis)); err != nil {
		return nil, err
	}
	if err := m.Settings.validateForGraph(m.Graph, false); err != nil {
		return nil, err
	}

	if !verificationInputModeValid(m.Analysis, m.InputMode) || !finite(m.Duration) || m.Duration < 0 || m.Duration > 10 {
		return nil, fmt.Errorf("invalid recorded mode or duration")
	}
	if IsTransientAnalysis(m.Analysis) {
		if m.Duration <= 0 || verificationOutputSteps(m.Duration, m.Settings.StepSeconds) > MaxVerificationSteps {
			return nil, fmt.Errorf("recorded transient duration exceeds output bounds")
		}
	} else if m.Duration != 0 || len(d.Rows) != 1 {
		return nil, fmt.Errorf("DC recording requires one sample at time zero")
	}
	if m.Probes == nil || len(m.Probes) > MaxScopeProbes || m.Statistics == nil || len(m.Statistics) != len(m.Probes) {
		return nil, fmt.Errorf("invalid recorded probes/statistics")
	}
	inventory := map[string]Probe{}
	for _, p := range probeInventory(m.Graph) {
		inventory[p.ID] = p
	}
	seen := map[string]bool{}
	for _, p := range m.Probes {
		if inventory[p.ID] != p || seen[p.ID] {
			return nil, fmt.Errorf("unknown, duplicate or inconsistent recorded probe")
		}
		seen[p.ID] = true
	}
	if len(d.Rows) < 1 || len(d.Rows) > MaxVerificationFrames || m.SampleCount != len(d.Rows) {
		return nil, fmt.Errorf("recorded sample count is inconsistent or exceeds bounds")
	}
	previous := -1.
	duplicates := 0
	for _, row := range d.Rows {
		if !finite(row.Time) || row.Time < 0 || row.Time > m.Duration || row.Time < previous || row.Values == nil || len(row.Values) != len(m.Probes) {
			return nil, fmt.Errorf("recorded rows must be ordered, complete and inside the recorded duration")
		}
		if row.Time == previous {
			duplicates++
		} else {
			duplicates = 0
		}
		if duplicates > 2 {
			return nil, fmt.Errorf("too many duplicate edge timestamps")
		}
		for _, v := range row.Values {
			if !finite(v) || math.Abs(v) > 1e12 {
				return nil, fmt.Errorf("recorded measurement exceeds finite display bounds")
			}
		}
		previous = row.Time
	}
	if d.Rows[0].Time != 0 {
		return nil, fmt.Errorf("recording does not cover its complete interval")
	}
	if IsTransientAnalysis(m.Analysis) {
		// A recording must include every requested output step, not only endpoints.
		steps := verificationOutputSteps(m.Duration, m.Settings.StepSeconds)
		index := 0
		for step := 0; step <= steps; step++ {
			target := math.Min(m.Duration, float64(step)*m.Settings.StepSeconds)
			spacing := m.Settings.StepSeconds
			if step > 0 {
				previous := math.Min(m.Duration, float64(step-1)*m.Settings.StepSeconds)
				if target > previous {
					spacing = math.Min(spacing, target-previous)
				}
			}
			if step < steps {
				next := math.Min(m.Duration, float64(step+1)*m.Settings.StepSeconds)
				if next > target {
					spacing = math.Min(spacing, next-target)
				}
			}
			for index < len(d.Rows) && d.Rows[index].Time < target && !recordingGridTimeMatches(d.Rows[index].Time, target, spacing) {
				index++
			}
			if index == len(d.Rows) || !recordingGridTimeMatches(d.Rows[index].Time, target, spacing) {
				return nil, fmt.Errorf("recording is missing a requested output step")
			}
		}
	}
	for i, p := range m.Probes {
		points := make([]ScopePoint, len(d.Rows))
		for j, row := range d.Rows {
			points[j] = ScopePoint{Time: row.Time, Value: row.Values[i]}
		}
		expected := scopeStatistics(points, p.ID)
		actual := m.Statistics[i]
		if actual.ProbeID != p.ID || actual.Count != expected.Count || !recordingNear(actual.Minimum, expected.Minimum) || !recordingNear(actual.Maximum, expected.Maximum) || !recordingNear(actual.Mean, expected.Mean) || !recordingNear(actual.RMS, expected.RMS) {
			return nil, fmt.Errorf("recorded statistics do not match recorded samples")
		}
	}
	if err := validateRecordedResults(m, c); err != nil {
		return nil, err
	}
	// Every stored final result and the last plotted row describe the same
	// accepted instant, including the right side of a source edge. Reconstruct
	// probe references from the saved result only; never rerun an imported solve.
	var frame TraceSample
	switch {
	case m.Mixed != nil:
		frame = mixedSample(c, m.Mixed, 0)
	case m.NPN != nil:
		frame = npnTraceSample(c, m.NPN, m.NPN, m.Duration, 0)
	case m.LED != nil:
		frame = ledTraceSample(c, *m.LED)
	}
	if frame.frame != nil {
		last := d.Rows[len(d.Rows)-1]
		for i, probe := range m.Probes {
			value, ok := frameValue(frame, probe)
			if !ok || !recordingNear(value, last.Values[i]) {
				return nil, fmt.Errorf("recorded final state disagrees with its last probe sample")
			}
		}
	}
	return &verificationRecord{metadata: cloneVerificationMetadata(m), rows: d.Rows}, nil
}

// Roundoff must not let one saved row stand in for distinct requested times,
// including a very short final partial step. Ordinary grids retain the legacy
// decimal tolerance, capped by the distance to the nearest planned neighbor.
const maxRecordingGridRoundoffFraction = 1e-9

func recordingGridTimeMatches(at, target, spacing float64) bool {
	epsilon := math.Min(1e-12*math.Max(1, math.Abs(target)), spacing*maxRecordingGridRoundoffFraction)
	return math.Abs(at-target) <= epsilon
}

func recordingNear(a, b float64) bool {
	return finite(a) && finite(b) && math.Abs(a-b) <= 1e-12*math.Max(1, math.Abs(b))
}
func validateRecordedResults(m VerificationMetadata, c Compiled) error {
	switch m.Analysis {
	case "RC transient":
		if m.LED != nil || m.NPN != nil || m.Mixed != nil {
			return fmt.Errorf("RC recording contains unrelated final results")
		}
	case NPNAnalysis, NPNTransientAnalysis:
		r := m.NPN
		if r == nil || m.LED != nil || m.Mixed != nil || r.Stale || r.Engine != NPNEngineID || !supportedNPNModel(r.Model) || r.TransistorID != c.Transistor.ID || r.SupplyID != c.Source.ID || r.InputID != c.Input.ID || r.BaseResistorID != c.BaseResistor.ID || r.LoadResistorID != c.Resistor.ID || r.LEDID != c.LED.ID {
			return fmt.Errorf("invalid recorded NPN final result")
		}
	case MixedAnalysis, NetworkDCAnalysis, NetworkTransientAnalysis, WaveformAnalysis, InductorAnalysis:
		r := m.Mixed
		if r == nil || m.LED != nil || m.NPN != nil || r.Stale || r.Engine != m.Engine || !recordingNear(r.Time, m.Duration) || len(r.Nodes) > MaxMixedNodes+1 {
			return fmt.Errorf("invalid recorded mixed final result")
		}
		if !finite(r.NumericalDamping) || r.NumericalDamping < 0 || m.Engine == verificationEngineID(m.Analysis) && r.NumericalDamping != 0 {
			return fmt.Errorf("invalid recorded numerical damping for integration method")
		}
		if _, _, err := recordedDeviceNodes(c, r); err != nil {
			return err
		}
		if err := validateRecordedOpAmps(c, m.Graph, r); err != nil {
			return err
		}
		for _, v := range r.Inductors {
			p, ok := m.Graph.Component(v.ID)
			if !ok || p.Kind != "inductor" || !recordingNear(v.Energy, .5*p.Value*v.Current*v.Current) || !recordingNear(v.Voltage, r.Nodes[c.TerminalNet[v.ID+".1"]]-r.Nodes[c.TerminalNet[v.ID+".2"]]) {
				return fmt.Errorf("invalid recorded inductor identity, voltage or energy")
			}
		}
		for _, v := range r.Capacitors {
			p, ok := m.Graph.Component(v.ID)
			if !ok || p.Kind != "capacitor" || !recordingNear(v.Energy, .5*p.Value*v.Voltage*v.Voltage) || !recordingNear(v.Voltage, r.Nodes[c.TerminalNet[v.ID+".1"]]-r.Nodes[c.TerminalNet[v.ID+".2"]]) {
				return fmt.Errorf("invalid recorded capacitor identity, voltage or energy")
			}
		}
		if err := validateRecordedMOSFETs(c, m.Graph, r); err != nil {
			return err
		}

		if err := validateRecordedBipolars(c, m.Graph, r); err != nil {
			return err
		}
		for _, group := range []struct {
			kind string
			ids  []string
		}{{"capacitor", nil}, {"inductor", nil}, {"led", nil}, {"resistor", nil}, {"source", nil}, {"switch", nil}, {"diode", nil}, {"potentiometer", nil}} {
			switch group.kind {
			case "inductor":
				for _, v := range r.Inductors {
					group.ids = append(group.ids, v.ID)
				}
			case "capacitor":
				for _, v := range r.Capacitors {
					group.ids = append(group.ids, v.ID)
				}
			case "led":
				for _, v := range r.LEDs {
					group.ids = append(group.ids, v.ID)
				}
			case "resistor":
				for _, v := range r.Resistors {
					group.ids = append(group.ids, v.ID)
				}
			case "potentiometer":
				for _, v := range r.Potentiometers {
					group.ids = append(group.ids, v.ID)
				}
			case "diode":
				for _, v := range r.Diodes {
					group.ids = append(group.ids, v.ID)
				}
			case "switch":
				for _, v := range r.Switches {
					group.ids = append(group.ids, v.ID)
				}
			case "source":
				for _, v := range r.Sources {
					group.ids = append(group.ids, v.ID)
				}
			}
			if err := recordedObjectIDs(m.Graph, group.kind, group.ids); err != nil {
				return err
			}
		}
	default:
		r := m.LED
		if r == nil || m.NPN != nil || m.Mixed != nil || r.Stale || r.Engine != EngineID || r.Model != Model() || r.LEDID != c.LED.ID || r.SourceID != c.Source.ID || r.ResistorID != c.Resistor.ID {
			return fmt.Errorf("invalid recorded LED final result")
		}
	}
	return nil
}

// All required exported fields must be present, including zero-valued fields.
// DecodeStrict separately rejects unknown, duplicate and noncanonical keys.
func recordingFields(raw json.RawMessage, t reflect.Type) error {
	if strings.TrimSpace(string(raw)) == "null" {
		return fmt.Errorf("recording fields must not be null")
	}
	if t.Kind() == reflect.Pointer {
		return recordingFields(raw, t.Elem())
	}
	switch t.Kind() {
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
			return fmt.Errorf("recording requires a complete object")
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")
			name := tag[0]
			if name == "-" || name == "" {
				continue
			}
			value, present := fields[name]
			optional := len(tag) > 1 && tag[1] == "omitempty"
			if !present {
				if optional {
					continue
				}
				return fmt.Errorf("recording is missing %s", name)
			}
			if string(value) == "null" {
				return fmt.Errorf("recorded %s must not be null", name)
			}
			if err := recordingFields(value, f.Type); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	case reflect.Map:
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil || fields == nil || len(fields) > MaxComponents {
			return fmt.Errorf("recorded map exceeds bounds")
		}
		for name, value := range fields {
			if len(name) > 128 {
				return fmt.Errorf("recorded map key exceeds bounds")
			}
			if err := recordingFields(value, t.Elem()); err != nil {
				return err
			}
		}
	case reflect.Slice:
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil || rows == nil {
			return fmt.Errorf("recording requires an array")
		}
		for _, row := range rows {
			if err := recordingFields(row, t.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s *Service) retainOperationLocked(operation *verificationOperation) {
	if s.operations == nil {
		s.operations = map[string]*verificationOperation{}
	}
	if len(s.operationOrder) >= MaxVerificationOperations {
		for i, id := range s.operationOrder {
			if !s.operations[id].keep && !operationActive(s.operations[id].summary.Status) {
				delete(s.operations, id)
				s.operationOrder = append(s.operationOrder[:i], s.operationOrder[i+1:]...)
				break
			}
		}
	}
	s.operations[operation.summary.ID] = operation
	s.operationOrder = append(s.operationOrder, operation.summary.ID)
}

func recordedObjectIDs(graph Graph, kind string, ids []string) error {
	expected := map[string]bool{}
	for _, part := range graph.Components {
		if part.Kind == kind || kind == "source" && IsVoltageSource(part.Kind) {
			expected[part.ID] = true
		}
	}
	if len(ids) != len(expected) {
		return fmt.Errorf("incomplete recorded %s results", kind)
	}
	for _, id := range ids {
		if !expected[id] {
			return fmt.Errorf("unknown or duplicate recorded %s result", kind)
		}
		delete(expected, id)
	}
	return nil
}

func validateRecordingSettings(settings TransientSettings, transient bool) error {
	if !finite(settings.PlaybackSpeed) || settings.PlaybackSpeed < .001 || settings.PlaybackSpeed > 1 {
		return fmt.Errorf("invalid recorded playback setting")
	}
	if !transient {
		// A DC sample step override is metadata only; high/low durations need not
		// cover that step. Preserve every configuration accepted by DC Verify.
		if !finite(settings.StepSeconds) || settings.StepSeconds < 1e-6 || settings.StepSeconds > 1 || !finite(settings.HighSeconds) || settings.HighSeconds < 1e-6 || settings.HighSeconds > 10 || !finite(settings.LowSeconds) || settings.LowSeconds < 1e-6 || settings.LowSeconds > 10 || !finite(settings.InitialVoltage) || settings.InitialVoltage < 0 || settings.InitialVoltage > 12 {
			return fmt.Errorf("invalid recorded DC settings")
		}
		return nil
	}
	return settings.validateRanges()
}
