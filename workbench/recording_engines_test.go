package workbench

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackwardEulerEngineCapturesRemainExactAndDoNotRelabel(t *testing.T) {
	for _, family := range []string{"mixed", "network", "waveform"} {
		t.Run(family, func(t *testing.T) {
			file, err := os.Open(filepath.Join("testdata", "legacy-engines", family+".json.gz"))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			reader, err := gzip.NewReader(file)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			original, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			record, err := decodeRecording(original)
			if err != nil {
				t.Fatal("genuine pre-change capture refused", err)
			}
			if !strings.HasSuffix(record.metadata.Engine, "-be.v1") || record.metadata.Engine == verificationEngineID(record.metadata.Analysis) || !IsNPNTransientEngine(record.metadata.Engine) {
				t.Fatal("legacy capture lost its historical engine identity")
			}
			encoded, err := encodeVerificationExport(record, "json")
			if err != nil || !bytes.Equal(encoded, original) {
				t.Fatal("legacy JSON changed after decoding/export", err)
			}
			var document VerificationDocument
			if err := json.Unmarshal(original, &document); err != nil {
				t.Fatal(err)
			}
			document.Metadata.Mixed.NumericalDamping = -1e-6
			negative, _ := json.Marshal(document)
			if _, err := decodeRecording(negative); err == nil || !strings.Contains(err.Error(), "damping") {
				t.Fatal("negative historical damping accepted", err)
			}
			if err := json.Unmarshal(original, &document); err != nil {
				t.Fatal(err)
			}
			document.Metadata.Mixed.Engine = verificationEngineID(document.Metadata.Analysis)
			changed, _ := json.Marshal(document)
			if _, err := decodeRecording(changed); err == nil || !strings.Contains(err.Error(), "mixed final result") {
				t.Fatal("mismatched captured/final engine identities accepted", err)
			}
		})
	}
}

func TestRecordedEngineAliasesStayWithinTheirAnalysisFamily(t *testing.T) {
	analyses := []string{MixedAnalysis, NetworkDCAnalysis, NetworkTransientAnalysis, WaveformAnalysis, InductorAnalysis, "RC transient", NPNTransientAnalysis, NPNAnalysis}
	legacy := map[string]string{MixedAnalysis: "elab.mixed-mna-be.v1", NetworkDCAnalysis: "elab.network-mna-be.v1", NetworkTransientAnalysis: "elab.network-mna-be.v1", WaveformAnalysis: "elab.network-waveforms-mna-be.v1"}
	for _, analysis := range analyses {
		for _, engine := range []string{MixedEngineID, NetworkEngineID, WaveformEngineID, InductorEngineID, RCEngineID, NPNTransientEngineID, NPNEngineID, "elab.mixed-mna-be.v1", "elab.network-mna-be.v1", "elab.network-waveforms-mna-be.v1", "future.v99"} {
			want := engine == verificationEngineID(analysis) || engine == legacy[analysis]
			if recordingEngineCompatible(analysis, engine) != want {
				t.Fatal("engine alias crosses analysis families", analysis, engine)
			}
		}
	}
}
