package workbench

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strconv"
)

func encodeSweepExport(record *SweepRecord, format string) ([]byte, error) {
	if record.Result.Noise != nil {
		if err := noiseResultShape(&record.Result); err != nil {
			return nil, err
		}
	}
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
	if IsFrequencySweep(record.Metadata.Settings.Parameter) {
		for _, id := range record.Metadata.Settings.ProbeIDs {
			header = append(header, id+" real [V/V]", id+" imaginary [V/V]", id+" phase_defined")
		}
	}
	if record.Result.Noise != nil {
		for _, p := range record.Result.Probes {
			header = append(header, p.ID+" input [V/sqrt(Hz)]", p.ID+" input_defined")
		}
		for _, c := range record.Result.Noise.Contributors {
			for _, p := range record.Result.Probes {
				header = append(header, c.ID+" -> "+p.ID+" [V^2/Hz]")
			}
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
		if p.Noise != nil {
			for j, v := range p.Noise.InputDensity {
				row = append(row, number(v), strconv.FormatBool(p.Noise.InputDefined[j]))
			}
			for _, psd := range p.Noise.PSD {
				for _, v := range psd {
					row = append(row, number(v))
				}
			}
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buffer.Bytes(), w.Error()
}
