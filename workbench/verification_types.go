package workbench

import (
	"context"
	"encoding/json"
)

type CapacitorInitial struct {
	ID      string  `json:"id"`
	Voltage float64 `json:"voltage_v"`
}
type InductorInitial struct {
	ID      string  `json:"id"`
	Current float64 `json:"current_a"`
}
type VerificationSettings struct {
	Inductors      []InductorInitial  `json:"inductor_initials,omitempty"`
	Duration       float64            `json:"duration_s"`
	Step           float64            `json:"sample_step_s"`
	InputMode      string             `json:"input_mode"`
	InitialVoltage *float64           `json:"initial_voltage_v,omitempty"`
	Capacitors     []CapacitorInitial `json:"capacitor_initials"`
}

func DefaultVerificationSettings() VerificationSettings {
	return VerificationSettings{Duration: .1, InputMode: "auto", Capacitors: []CapacitorInitial{}}
}
func cloneVerificationSettings(in VerificationSettings) VerificationSettings {
	out := in
	out.Capacitors = append([]CapacitorInitial{}, in.Capacitors...)
	out.Inductors = append([]InductorInitial(nil), in.Inductors...)
	if in.InitialVoltage != nil {
		v := *in.InitialVoltage
		out.InitialVoltage = &v
	}
	return out
}

type VerifyParams struct {
	Inductors      []InductorInitial  `json:"inductor_initials,omitempty"`
	Duration       *float64           `json:"duration_s,omitempty"`
	Step           *float64           `json:"sample_step_s,omitempty"`
	InputMode      string             `json:"input_mode,omitempty"`
	InitialVoltage *float64           `json:"initial_voltage_v,omitempty"`
	Capacitors     []CapacitorInitial `json:"capacitor_initials,omitempty"`
	ProbeIDs       []string           `json:"probe_ids,omitempty"`
}
type OperationSummary struct {
	FailureHelp        *DiagnosticHelp `json:"failure_help,omitempty"`
	KeptInProject      bool            `json:"kept_in_project"`
	SourcePath         string          `json:"source_path,omitempty"`
	SourceSHA256       string          `json:"source_sha256,omitempty"`
	ID                 string          `json:"id"`
	Kind               string          `json:"kind"`
	Status             string          `json:"status"`
	Actor              string          `json:"actor"`
	InputRevision      uint64          `json:"input_revision"`
	ElectricalRevision uint64          `json:"electrical_revision"`
	DocumentGeneration uint64          `json:"document_generation"`
	RequestedDuration  float64         `json:"requested_duration_s"`
	SimulatedTime      float64         `json:"simulated_time_s"`
	CompletedSteps     int             `json:"completed_steps"`
	TotalSteps         int             `json:"total_steps"`
	Progress           float64         `json:"progress"`
	SourceUnchanged    bool            `json:"source_unchanged"`
	Error              *Failure        `json:"error,omitempty"`
}
type VerificationRow struct {
	Time   float64   `json:"time_s"`
	Values []float64 `json:"values"`
}
type VerificationMetadata struct {
	ActuationRevision  uint64              `json:"actuation_revision,omitempty"`
	WaveformModel      *WaveformModel      `json:"waveform_model,omitempty"`
	PotentiometerModel *PotentiometerModel `json:"potentiometer_model,omitempty"`
	DiodeModel         *DiodeModel         `json:"diode_model,omitempty"`
	SchottkyModel      *DiodeModel         `json:"schottky_model,omitempty"`
	ZenerModel         *ZenerModel         `json:"zener_model,omitempty"`
	SwitchModel        *SwitchModel        `json:"switch_model,omitempty"`
	NMOSModel          *NMOSModel          `json:"nmos_model,omitempty"`
	PMOSModel          *PMOSModel          `json:"pmos_model,omitempty"`
	PNPModel           *PNPModel           `json:"pnp_model,omitempty"`
	OpAmpModel         *OpAmpModel         `json:"opamp_model,omitempty"`
	Timer555Model      *Timer555Model      `json:"timer555_model,omitempty"`
	LogicModel         *LogicModel         `json:"logic_model,omitempty"`
	SequentialModel    *SequentialModel    `json:"sequential_model,omitempty"`
	DisplayModel       *DisplayModel       `json:"display_model,omitempty"`
	Format             string              `json:"format"`
	Version            int                 `json:"version"`
	OperationID        string              `json:"operation_id"`
	CreatedAt          string              `json:"created_at"`
	InputRevision      uint64              `json:"input_revision"`
	ElectricalRevision uint64              `json:"electrical_revision"`
	DocumentGeneration uint64              `json:"document_generation"`
	Graph              Graph               `json:"graph"`
	Settings           TransientSettings   `json:"settings"`
	InputMode          string              `json:"input_mode"`
	Analysis           string              `json:"analysis"`
	Engine             string              `json:"engine"`
	LEDModel           LEDModel            `json:"led_model"`
	NPNModel           *NPNModel           `json:"npn_model,omitempty"`
	Probes             []Probe             `json:"probes"`
	Duration           float64             `json:"duration_s"`
	SampleCount        int                 `json:"sample_count"`
	Statistics         []ScopeStatistics   `json:"statistics"`
	LED                *Result             `json:"led,omitempty"`
	NPN                *NPNResult          `json:"npn,omitempty"`
	Mixed              *MixedResult        `json:"mixed,omitempty"`
}
type VerificationReading struct {
	Probe   Probe   `json:"probe"`
	Value   float64 `json:"value"`
	Minimum float64 `json:"min"`
	Maximum float64 `json:"max"`
}
type OperationView struct {
	Operation OperationSummary      `json:"operation"`
	Metadata  *VerificationMetadata `json:"metadata,omitempty"`
}
type VerificationPageParams struct {
	ID     string `json:"id"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}
type VerificationPage struct {
	ID         string            `json:"id"`
	Offset     int               `json:"offset"`
	NextOffset int               `json:"next_offset"`
	Total      int               `json:"total"`
	Probes     []Probe           `json:"probes"`
	Rows       []VerificationRow `json:"rows"`
}
type VerificationPreview struct {
	ID          string                `json:"id"`
	Engine      string                `json:"engine"`
	InputMode   string                `json:"input_mode"`
	Duration    float64               `json:"duration_s"`
	SampleCount int                   `json:"sample_count"`
	Readings    []VerificationReading `json:"readings"`
}
type verificationRecord struct {
	metadata VerificationMetadata
	rows     []VerificationRow
}
type verificationOperation struct {
	sourceActuationRevision uint64
	done                    chan struct{}
	keep                    bool
	inputMetadata           VerificationMetadata
	summary                 OperationSummary
	ctx                     context.Context
	cancel                  context.CancelFunc
	record                  *verificationRecord
}
type verificationInput struct {
	compiled Compiled
	settings TransientSettings
	mode     string
	probes   []Probe
	duration float64
	steps    int
	serial   uint64
	metadata VerificationMetadata
}

// Empty arrays explicitly clear selections/overrides; nil inherits defaults.
// encoding/json's slice omitempty otherwise collapses those distinct actions.
func (p VerifyParams) MarshalJSON() ([]byte, error) {
	type wire VerifyParams
	data, err := json.Marshal(wire(p))
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if p.ProbeIDs != nil {
		fields["probe_ids"], err = json.Marshal(p.ProbeIDs)
		if err != nil {
			return nil, err
		}
	}
	if p.Capacitors != nil {
		fields["capacitor_initials"], err = json.Marshal(p.Capacitors)
		if err != nil {
			return nil, err
		}
	}
	if p.Inductors != nil {
		fields["inductor_initials"], err = json.Marshal(p.Inductors)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(fields)
}
