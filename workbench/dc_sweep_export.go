package workbench

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strconv"
)

func encodeSweepExport(record *SweepRecord, format string) ([]byte, error) {
	if format == "json" {
		data, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	}
	var buffer bytes.Buffer
	w := csv.NewWriter(&buffer)
	header := []string{"operation_id", "engine", "document_generation", "electrical_revision", "component_id", "parameter", "parameter_unit", "input", "kcl_residual_a", "power_residual_w"}
	for _, p := range record.Result.Probes {
		header = append(header, p.ID+" ["+p.Unit+"]")
	}
	if record.Metadata.Settings.Parameter == "frequency" {
		for _, id := range record.Metadata.Settings.ProbeIDs {
			header = append(header, id+" real [V/V]", id+" imaginary [V/V]", id+" phase_defined")
		}
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, p := range record.Result.Points {
		m := record.Metadata
		r := record.Result
		row := []string{m.OperationID, m.Engine, strconv.FormatUint(m.DocumentGeneration, 10), strconv.FormatUint(m.ElectricalRevision, 10), r.Target.ComponentID, r.Target.Parameter, r.Target.Unit, number(p.Input), number(p.KCLResidual), number(p.PowerResidual)}
		for _, v := range p.Values {
			row = append(row, number(v))
		}
		for _, z := range p.AC {
			row = append(row, number(z.Real), number(z.Imaginary), strconv.FormatBool(z.PhaseDefined))
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buffer.Bytes(), w.Error()
}
