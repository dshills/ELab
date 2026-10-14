package workbench

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
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

func (s *Service) exportVerificationLocked(p VerificationExportParams) error {
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	operation, ok := s.operations[p.ID]
	if !ok {
		return &actionError{"not-found", "Operation not found or no longer retained"}
	}
	if operation.record == nil {
		return &actionError{"not-ready", "Export requires a completed verification"}
	}
	if p.Format != "json" && p.Format != "csv" {
		return fmt.Errorf("verification format must be json or csv")
	}
	path, err := documentPath(p.Path)
	if err != nil {
		return err
	}
	if path == s.documentPath {
		return fmt.Errorf("verification export must not replace the schematic")
	}
	record := operation.record
	var data []byte
	if p.Format == "json" {
		data, err = json.MarshalIndent(VerificationDocument{Metadata: record.metadata, Rows: record.rows}, "", "  ")
		if err != nil {
			return fileError(err)
		}
		data = append(data, '\n')
	} else {
		var b bytes.Buffer
		writer := csv.NewWriter(&b)
		header := []string{"operation_id", "input_revision", "electrical_revision", "engine", "led_model", "npn_model"}
		m := record.metadata
		modelIDs := []string{}
		if m.SwitchModel != nil {
			header = append(header, "switch_model")
			modelIDs = append(modelIDs, m.SwitchModel.ID)
		}
		if m.PotentiometerModel != nil {
			header = append(header, "potentiometer_model")
			modelIDs = append(modelIDs, m.PotentiometerModel.ID)
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
			return fileError(err)
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
				return fileError(err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return fileError(err)
		}
		data = b.Bytes()
	}
	if len(data) > 8*1024*1024 {
		return fmt.Errorf("verification export exceeds 8 MiB")
	}
	if err := publishExport(path, data, p.Overwrite); err != nil {
		return err
	}
	s.lastExport = &Artifact{Path: path, Format: "verification." + p.Format, OperationID: p.ID, ElectricalRevision: record.metadata.ElectricalRevision, Bytes: len(data)}
	return nil
}
