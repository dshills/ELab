package workbench

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const DocumentFormat = "elab.schematic"
const DocumentVersion = 4
const MaxDocumentBytes = 256 * 1024

type Document struct {
	PotentiometerModel *PotentiometerModel `json:"potentiometer_model,omitempty"`
	DiodeModel         *DiodeModel         `json:"diode_model,omitempty"`
	SwitchModel        *SwitchModel        `json:"switch_model,omitempty"`
	Recordings         *ProjectRecordings  `json:"recordings,omitempty"`
	Paused             *bool               `json:"paused,omitempty"`
	ScopeProbes        *ProbeParams        `json:"scope_probes,omitempty"`
	Format             string              `json:"format"`
	Version            int                 `json:"version"`
	Graph              Graph               `json:"graph"`
	View               View                `json:"view"`
	NPNModel           *NPNModel           `json:"npn_model,omitempty"`
	LEDModel           LEDModel            `json:"led_model"`
	TransientSettings  *TransientSettings  `json:"transient_settings,omitempty"`
}
type DocumentState struct {
	SchematicDirty  bool   `json:"schematic_dirty"`
	RecordingsDirty bool   `json:"recordings_dirty"`
	RecordingCount  int    `json:"recording_count"`
	RecoveredFrom   string `json:"recovered_from,omitempty"`
	Path            string `json:"path"`
	Dirty           bool   `json:"dirty"`
	FormatVersion   int    `json:"format_version"`
	Generation      uint64 `json:"generation"`
}
type SaveParams struct {
	ExpectedGeneration *uint64 `json:"expected_document_generation,omitempty"`
	Path               string  `json:"path,omitempty"`
	Overwrite          bool    `json:"overwrite,omitempty"`
}
type LoadParams struct {
	ExpectedGeneration *uint64 `json:"expected_document_generation,omitempty"`
	Path               string  `json:"path"`
	DiscardChanges     bool    `json:"discard_changes,omitempty"`
}
type ReplaceParams struct {
	DiscardChanges bool `json:"discard_changes,omitempty"`
}

func contentHash(g Graph, settings TransientSettings) [32]byte {
	data, _ := json.Marshal(struct {
		Graph    Graph
		Settings TransientSettings
	}{g, settings})
	return sha256.Sum256(data)
}
func (s *Service) dirtyLocked() bool {
	return contentHash(s.graph, s.transientSettings) != s.savedGraph || s.recordingsDirtyLocked()
}
func (s *Service) requireCleanLocked(discard bool) error {
	if s.dirtyLocked() && !discard {
		return &actionError{"unsaved-changes", "Save your schematic and recorded results first, or explicitly discard unsaved changes."}
	}
	return nil
}
func (s *Service) detachDocumentLocked() {
	s.inputMode = "auto"
	s.pendingInputMode = ""
	s.documentGeneration++
	s.clearProjectOperationsLocked()
	s.recoveredFrom = ""
	s.documentPath = ""
	s.transientSettings = DefaultTransientSettings()
	s.savedGraph = contentHash(EmptyGraph(), s.transientSettings)
	s.diskHash = [32]byte{}
}
func documentPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, 0) || len(path) > 4096 {
		return "", fmt.Errorf("a nonempty local file path (max 4096 bytes) is required")
	}
	return filepath.Abs(path)
}
func fileError(err error) error { return &actionError{"file-error", err.Error()} }
func readDocumentFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fileError(err)
	}
	if !info.Mode().IsRegular() {
		return nil, fileError(fmt.Errorf("choose a regular file; links and special files are not supported"))
	}
	if info.Size() > MaxProjectBytes {
		return nil, &actionError{"invalid-document", "project file exceeds 8 MiB"}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fileError(err)
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil {
		return nil, fileError(err)
	}
	if !actual.Mode().IsRegular() || !os.SameFile(info, actual) {
		return nil, fileError(fmt.Errorf("file changed while opening; try again"))
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxProjectBytes+1))
	if err != nil {
		return nil, fileError(err)
	}
	if len(data) > MaxProjectBytes {
		return nil, &actionError{"invalid-document", "project file exceeds 8 MiB"}
	}
	return data, nil
}
func decodeDocument(data []byte) (Document, error) {
	var d Document
	if len(data) > MaxProjectBytes {
		return d, fmt.Errorf("project exceeds 8 MiB")
	}
	if err := params(data, &d, "format", "version", "graph", "view", "led_model"); err != nil {
		return d, fmt.Errorf("invalid schematic: %w", err)
	}
	var raw struct {
		Graph    json.RawMessage `json:"graph"`
		View     json.RawMessage `json:"view"`
		Settings json.RawMessage `json:"transient_settings"`
	}
	_ = json.Unmarshal(data, &raw)
	var graphFields struct {
		Components []json.RawMessage `json:"components"`
		Wires      []json.RawMessage `json:"wires"`
	}
	if err := params(raw.Graph, &graphFields, "components", "wires"); err != nil {
		return d, err
	}
	for _, component := range graphFields.Components {
		if err := params(component, &Component{}, "id", "kind", "name", "terminals", "position", "value", "rotation"); err != nil {
			return d, err
		}
		if err := validatePointParam(component, "position"); err != nil {
			return d, err
		}
	}
	for _, wire := range graphFields.Wires {
		if err := params(wire, &Wire{}, "id", "from", "to", "connected"); err != nil {
			return d, err
		}
	}
	if err := params(raw.View, &View{}, "center", "zoom"); err != nil {
		return d, err
	}
	if err := validatePointParam(raw.View, "center"); err != nil {
		return d, err
	}
	if d.Format != DocumentFormat || (d.Version < 1 || d.Version > DocumentVersion) {
		return d, fmt.Errorf("unsupported schematic format/version; expected %s version %d", DocumentFormat, DocumentVersion)
	}
	if d.Version < 4 {
		if len(data) > MaxDocumentBytes {
			return d, fmt.Errorf("legacy schematic exceeds 256 KiB")
		}
		if d.Recordings != nil || d.Paused != nil {
			return d, fmt.Errorf("recordings and paused preference require version 4")
		}
	} else {
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(data, &fields)
		if d.Paused == nil {
			return d, fmt.Errorf("version 4 requires paused preference")
		}
		recordings, err := validateProjectRecordings(fields["recordings"])
		if err != nil {
			return d, err
		}
		d.Recordings = recordings
	}
	if d.Version >= 2 {
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(data, &fields)
		if err := params(fields["transient_settings"], &TransientSettings{}, "step_s", "high_s", "low_s", "initial_voltage_v", "seconds_per_wall_second"); err != nil {
			return d, err
		}
	}
	if d.TransientSettings == nil {
		if d.Version >= 2 {
			return d, fmt.Errorf("version 2–4 requires transient settings")
		}
		settings := DefaultTransientSettings()
		d.TransientSettings = &settings
	}
	if err := d.TransientSettings.Validate(); err != nil {
		return d, err
	}
	if d.Version >= 3 && d.ScopeProbes == nil {
		return d, fmt.Errorf("version 3/4 requires scope_probes")
	}
	if d.ScopeProbes != nil {
		if d.Version < 3 {
			return d, fmt.Errorf("scope probes require version 3")
		}
		if d.ScopeProbes.IDs == nil || len(d.ScopeProbes.IDs) > MaxScopeProbes {
			return d, fmt.Errorf("invalid saved probe selection")
		}
		available := map[string]bool{}
		for _, p := range probeInventory(d.Graph) {
			available[p.ID] = true
		}
		seen := map[string]bool{}
		for _, id := range d.ScopeProbes.IDs {
			if !available[id] || seen[id] {
				return d, fmt.Errorf("invalid saved probe %q", id)
			}
			seen[id] = true
		}
	}
	if d.Graph.Components == nil || d.Graph.Wires == nil {
		return d, fmt.Errorf("graph requires components and wires arrays")
	}
	if len(d.Graph.Components) > MaxComponents || len(d.Graph.Wires) > MaxWires {
		return d, fmt.Errorf("schematic exceeds component/wire limits")
	}
	if d.PotentiometerModel != nil && *d.PotentiometerModel != LinearPotentiometerModel() {
		return d, fmt.Errorf("unsupported potentiometer model parameters/version")
	}
	for _, part := range d.Graph.Components {
		if part.Kind == "potentiometer" && (d.PotentiometerModel == nil || *d.PotentiometerModel != LinearPotentiometerModel()) {
			return d, fmt.Errorf("potentiometers require supported potentiometer_model parameters")
		}
	}

	if d.DiodeModel != nil && *d.DiodeModel != SiliconDiodeModel() {
		return d, fmt.Errorf("unsupported diode model parameters/version")
	}
	for _, part := range d.Graph.Components {
		if part.Kind == "diode" && (d.DiodeModel == nil || *d.DiodeModel != SiliconDiodeModel()) {
			return d, fmt.Errorf("diode schematics require supported diode_model parameters")
		}
	}

	if d.SwitchModel != nil && *d.SwitchModel != SwitchDeviceModel() {
		return d, fmt.Errorf("unsupported switch model parameters/version")
	}
	for _, c := range d.Graph.Components {
		if c.Kind == "switch" && (d.SwitchModel == nil || *d.SwitchModel != SwitchDeviceModel()) {
			return d, fmt.Errorf("switch schematics require supported switch_model parameters")
		}
		if c.Kind == "npn" && (d.NPNModel == nil || *d.NPNModel != TransistorModel()) {
			return d, fmt.Errorf("NPN schematics require supported npn_model parameters")
		}
	}
	if d.NPNModel != nil && *d.NPNModel != TransistorModel() {
		return d, fmt.Errorf("unsupported NPN model parameters or version")
	}
	if d.LEDModel != Model() {
		return d, fmt.Errorf("unsupported LED model parameters or version")
	}
	if !finite(d.View.Zoom) || d.View.Zoom < .01 || d.View.Zoom > 4 || !finite(d.View.Center.X) || !finite(d.View.Center.Y) || absOutOfBounds(d.View.Center) {
		return d, fmt.Errorf("invalid saved viewport")
	}
	ids := map[string]bool{}
	positions := map[Point]bool{}
	for _, c := range d.Graph.Components {
		value := ""
		if c.Kind == "net-label" {
			value = c.Label
		}
		if c.Unit != "" {
			value = fmt.Sprintf("%.17g %s", c.Value, c.Unit)
		}
		expected, err := MakeComponent(c.ID, c.Kind, c.Position, value)
		if err != nil {
			return d, err
		}
		if c.Rotation != 0 && c.Rotation != 90 && c.Rotation != 180 && c.Rotation != 270 {
			return d, fmt.Errorf("invalid rotation on %s", c.ID)
		}
		expected.Rotation = c.Rotation
		if c.WiperPosition != nil {
			if c.Kind != "potentiometer" || !finite(*c.WiperPosition) || *c.WiperPosition < 0 || *c.WiperPosition > 1 {
				return d, fmt.Errorf("invalid potentiometer wiper on %s", c.ID)
			}
			v := *c.WiperPosition
			expected.WiperPosition = &v
		}
		if c.Kind == "potentiometer" && c.WiperPosition == nil {
			return d, fmt.Errorf("potentiometer requires explicit wiper_position")
		}

		if c.SwitchClosed != nil {
			if c.Kind != "switch" {
				return d, fmt.Errorf("switch state belongs only to a switch")
			}
			closed := *c.SwitchClosed
			expected.SwitchClosed = &closed
		}
		if c.Kind == "switch" && c.SwitchClosed == nil {
			return d, fmt.Errorf("switch requires an explicit boolean state")
		}
		if c.InitialVoltage != nil {
			if c.Kind != "capacitor" || !finite(*c.InitialVoltage) || math.Abs(*c.InitialVoltage) > 12 {
				return d, fmt.Errorf("invalid capacitor initial voltage on %s", c.ID)
			}
			expected.InitialVoltage = c.InitialVoltage
		}
		if !reflect.DeepEqual(expected, c) {
			return d, fmt.Errorf("invalid component metadata, model, value, or grid position on %s", c.ID)
		}
		if ids[c.ID] || positions[c.Position] {
			return d, fmt.Errorf("duplicate object ID or occupied position on %s", c.ID)
		}
		ids[c.ID] = true
		positions[c.Position] = true
	}
	pairs := map[string]bool{}
	for _, w := range d.Graph.Wires {
		if !ValidID(w.ID) || ids[w.ID] {
			return d, fmt.Errorf("invalid or duplicate wire ID")
		}
		if _, _, ok := d.Graph.Terminal(w.From); !ok {
			return d, fmt.Errorf("unknown terminal %s", w.From)
		}
		if _, _, ok := d.Graph.Terminal(w.To); !ok {
			return d, fmt.Errorf("unknown terminal %s", w.To)
		}
		a, b := w.From, w.To
		if a > b {
			a, b = b, a
		}
		pair := a + "\x00" + b
		if a == b || pairs[pair] {
			return d, fmt.Errorf("self connection or duplicate wire")
		}
		ids[w.ID] = true
		pairs[pair] = true
	}
	return d, nil
}
func absOutOfBounds(p Point) bool { return p.X < -1e6 || p.X > 1e6 || p.Y < -1e6 || p.Y > 1e6 }

func (s *Service) saveLocked(p SaveParams) error {
	if s.verificationActiveLocked() {
		return &actionError{"busy", "Finish or cancel the active verification before saving the project"}
	}
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	path := p.Path
	if path == "" {
		path = s.documentPath
	}
	if path == "" {
		return &actionError{"unavailable", "Choose a destination for this schematic."}
	}
	path, err := documentPath(path)
	if err != nil {
		return err
	}
	replace := false
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fileError(fmt.Errorf("save destination must be a regular file, not a link or directory"))
		}
		if path == s.documentPath && !p.Overwrite {
			data, err := readDocumentFile(path)
			if err != nil {
				return err
			}
			if sha256.Sum256(data) != s.diskHash {
				return &actionError{"file-conflict", "The file changed outside ELab. Save As, or explicitly overwrite it."}
			}
		} else if !p.Overwrite {
			return &actionError{"file-exists", "Destination exists. Choose another path or explicitly allow overwrite."}
		}
		replace = true
	} else if !os.IsNotExist(err) {
		return fileError(err)
	}
	d := s.documentLocked()
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if len(data) > MaxProjectBytes {
		return &actionError{"invalid-document", "project file exceeds 8 MiB"}
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".elab-save-*.tmp")
	if err != nil {
		return fileError(err)
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return fileError(err)
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return fileError(err)
	}
	if err = f.Close(); err != nil {
		return fileError(err)
	}
	if replace {
		err = os.Rename(temp, path)
	} else {
		err = os.Link(temp, path)
	}
	if err != nil {
		if os.IsExist(err) {
			return &actionError{"file-exists", "Destination was created while saving; choose another path."}
		}
		return fileError(err)
	}
	// Flush directory metadata where supported. A completed atomic publication is
	// considered saved even on filesystems that cannot sync their directory handle.
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	s.recoveredFrom = ""
	s.documentPath = path
	s.savedGraph = contentHash(s.graph, s.transientSettings)
	s.savedRecordings = s.recordingMembershipHashLocked()
	s.diskHash = sha256.Sum256(data)
	if s.recoveryStore != nil {
		capture := s.recoveryCaptureLocked()
		capture.commandRevision++ // Blocks dirty captures made before this save.
		if _, _, err := s.recoveryStore.checkpoint(capture); err != nil {
			s.recovery.Error = err.Error()
		}
		// A saved file remains successful if recovery cleanup fails.
		if err := s.scanRecoveryLocked(); err != nil {
			s.recovery.Error = err.Error()
		}
	}
	return nil
}
func (s *Service) loadLocked(p LoadParams) error {
	if err := s.checkDocumentGenerationLocked(p.ExpectedGeneration); err != nil {
		return err
	}
	if err := s.requireCleanLocked(p.DiscardChanges); err != nil {
		return err
	}
	path, err := documentPath(p.Path)
	if err != nil {
		return err
	}
	data, err := readDocumentFile(path)
	if err != nil {
		return err
	}
	d, err := decodeDocument(data)
	if err != nil {
		return &actionError{"invalid-document", err.Error()}
	}
	s.installDocumentLocked(d)
	s.recoveredFrom = ""
	s.documentPath = path
	s.savedGraph = contentHash(s.graph, s.transientSettings)
	s.savedRecordings = s.recordingMembershipHashLocked()
	s.diskHash = sha256.Sum256(data)

	return nil
}

// File choices may outlive the project they were made for. A generation guard
// survives harmless clock ticks but refuses to act on a replacement project.
func (s *Service) checkDocumentGenerationLocked(expected *uint64) error {
	if expected != nil && *expected != s.documentGeneration {
		return &actionError{"conflict", fmt.Sprintf("Schematic changed while this file action was pending (expected document generation %d, current %d). Choose the file again.", *expected, s.documentGeneration)}
	}
	return nil
}

func (s *Service) documentLocked() Document {
	settings := s.transientSettings
	compiled, _, _ := s.graph.Compile()
	probes := ProbeParams{IDs: s.scopeLocked(compiled).Selected}
	paused := s.paused
	d := Document{Recordings: s.projectRecordingsLocked(), Paused: &paused, ScopeProbes: &probes, Format: DocumentFormat, Version: DocumentVersion, Graph: s.graph.Clone(), View: s.view, LEDModel: Model(), TransientSettings: &settings}
	for _, part := range s.graph.Components {
		if part.Kind == "potentiometer" {
			model := LinearPotentiometerModel()
			d.PotentiometerModel = &model
			break
		}
	}
	for _, part := range s.graph.Components {
		if part.Kind == "diode" {
			model := SiliconDiodeModel()
			d.DiodeModel = &model
			break
		}
	}
	for _, c := range s.graph.Components {
		if c.Kind == "switch" {
			model := SwitchDeviceModel()
			d.SwitchModel = &model
			break
		}
	}
	for _, c := range s.graph.Components {
		if c.Kind == "npn" {
			m := TransistorModel()
			d.NPNModel = &m
			break
		}
	}
	return d
}

func (s *Service) installDocumentLocked(d Document) {
	s.installProjectRecordingsLocked(d.Recordings)
	s.inputMode = "auto"
	s.pendingInputMode = ""
	s.documentGeneration++
	s.transientSettings = *d.TransientSettings
	s.verificationSettings = DefaultVerificationSettings()
	s.scopeConfigured = d.ScopeProbes != nil
	s.scopeProbes = nil
	if d.ScopeProbes != nil {
		s.scopeProbes = append([]string{}, d.ScopeProbes.IDs...)
	}
	s.activeCursor = "a"
	s.cursorA = nil
	s.cursorB = nil
	s.scopeWindow = nil
	s.graph = d.Graph.Clone()
	s.view = d.View
	s.selected = ""
	s.drawing = DefaultDrawing()
	for _, c := range s.graph.Components {
		s.usedIDs[c.ID] = true
	}
	for _, w := range s.graph.Wires {
		s.usedIDs[w.ID] = true
	}
	// Results/history/session revisions are never trusted from a file. Restart
	// against the restored graph; live simulation resumes on opening.
	s.mixed = nil
	s.mixedState = nil
	s.npn = nil
	s.result = nil
	s.transient = nil
	s.paused = d.Paused != nil && *d.Paused
	s.restorePaused = s.paused
}
