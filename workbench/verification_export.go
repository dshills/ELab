package workbench

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strconv"
)

type VerificationExportParams struct {
	ID                 string  `json:"id"`
	Path               string  `json:"path"`
	Format             string  `json:"format"`
	Overwrite          bool    `json:"overwrite,omitempty"`
	ExpectedGeneration *uint64 `json:"expected_document_generation,omitempty"`
}
type VerificationDocument struct {
	Metadata VerificationMetadata `json:"metadata"`
	Rows     []VerificationRow    `json:"rows"`
}

func encodeVerificationExport(record *verificationRecord, format string) ([]byte, error) {
	var err error
	var data []byte
	if format == "json" {
		data, err = json.MarshalIndent(VerificationDocument{Metadata: record.metadata, Rows: record.rows}, "", "  ")
		if err != nil {
			return nil, fileError(err)
		}
		data = append(data, '\n')
	} else {
		var b bytes.Buffer
		writer := csv.NewWriter(&b)
		header := []string{"operation_id", "input_revision", "electrical_revision", "engine", "led_model", "npn_model"}
		m := record.metadata
		modelIDs := []string{}
		if m.Timer555Model != nil {
			header = append(header, "timer555_model")
			modelIDs = append(modelIDs, m.Timer555Model.ID)
		}
		if m.LogicModel != nil {
			header = append(header, "logic_model")
			modelIDs = append(modelIDs, m.LogicModel.ID)
		}
		if m.ControlModel != nil {
			header = append(header, "control_model")
			modelIDs = append(modelIDs, m.ControlModel.ID)
		}
		if m.DisplayModel != nil {
			header = append(header, "display_model")
			modelIDs = append(modelIDs, m.DisplayModel.ID)
		}
		if m.SequentialModel != nil {
			header = append(header, "sequential_model")
			modelIDs = append(modelIDs, m.SequentialModel.ID)
		}
		if m.OpAmpModel != nil {
			header = append(header, "opamp_model")
			modelIDs = append(modelIDs, m.OpAmpModel.ID)
		}
		if m.PNPModel != nil {
			header = append(header, "pnp_model")
			modelIDs = append(modelIDs, m.PNPModel.ID)
		}
		if m.PMOSModel != nil {
			header = append(header, "pmos_model")
			modelIDs = append(modelIDs, m.PMOSModel.ID)
		}
		if m.NMOSModel != nil {
			header = append(header, "nmos_model")
			modelIDs = append(modelIDs, m.NMOSModel.ID)
		}
		if m.SwitchModel != nil {
			header = append(header, "switch_model")
			modelIDs = append(modelIDs, m.SwitchModel.ID)
		}
		if m.WaveformModel != nil {
			header = append(header, "waveform_model")
			modelIDs = append(modelIDs, m.WaveformModel.ID)
		}
		if m.PotentiometerModel != nil {
			header = append(header, "potentiometer_model")
			modelIDs = append(modelIDs, m.PotentiometerModel.ID)
		}
		if m.ZenerModel != nil {
			header = append(header, "zener_model")
			modelIDs = append(modelIDs, m.ZenerModel.ID)
		}
		if m.SchottkyModel != nil {
			header = append(header, "schottky_model")
			modelIDs = append(modelIDs, m.SchottkyModel.ID)
		}
		if m.DiodeModel != nil {
			header = append(header, "diode_model")
			modelIDs = append(modelIDs, m.DiodeModel.ID)
		}
		header = append(header, "input_mode", "duration_s", "sample_step_s", "time_s")
		for _, probe := range record.metadata.Probes {
			header = append(header, probe.ID+" ["+probe.Unit+"]")
		}
		if err := writer.Write(header); err != nil {
			return nil, fileError(err)
		}
		npnModel := ""
		if m.NPNModel != nil {
			npnModel = m.NPNModel.ID
		}
		for _, sample := range record.rows {
			row := []string{m.OperationID, strconv.FormatUint(m.InputRevision, 10), strconv.FormatUint(m.ElectricalRevision, 10), m.Engine, m.LEDModel.ID, npnModel}
			row = append(row, modelIDs...)
			row = append(row, m.InputMode, number(m.Duration), number(m.Settings.StepSeconds), number(sample.Time))
			for _, value := range sample.Values {
				row = append(row, number(value))
			}
			if err := writer.Write(row); err != nil {
				return nil, fileError(err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return nil, fileError(err)
		}
		data = b.Bytes()
	}
	return data, nil
}
