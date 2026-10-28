package workbench

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"time"
)

const LiveSpectrumInterval = 250 * time.Millisecond

// An end-exclusive FFT needs N intervals (N+1 retained rows).
const MaxLiveSpectrumSamples = MaxTraceSamples / 2

type ScopeViewParams struct {
	View string `json:"view"`
}
type LiveSpectrumParams struct {
	Enabled     bool    `json:"enabled"`
	Hold        bool    `json:"hold,omitempty"`
	ProbeID     string  `json:"probe_id,omitempty"`
	Window      string  `json:"window,omitempty"`
	Samples     int     `json:"samples,omitempty"`
	Fundamental float64 `json:"fundamental_hz,omitempty"`
	Harmonics   int     `json:"harmonics,omitempty"`
}

// LiveSpectrumInput accepts unit text through dispatch normalization; output settings stay numeric.
type LiveSpectrumInput LiveSpectrumParams

type LiveSpectrumState struct {
	Measurements *SpectrumMeasurements `json:"measurements"`
	Settings     LiveSpectrumParams    `json:"settings"`
	Status       string                `json:"status"`
	Reason       string                `json:"reason,omitempty"`
	RunID        uint64                `json:"run_id"`
	Start        float64               `json:"start_s"`
	End          float64               `json:"end_s"`
	Result       *SpectrumResult       `json:"result,omitempty"`
}
type liveSpectrumKey struct {
	Generation, Electrical, Run, Actuation, Settings uint64
	Probes                                           string
	Window                                           ScopeWindow
}
type liveSpectrumInput struct {
	key        liveSpectrumKey
	record     *verificationRecord
	params     SpectrumParams
	start, end float64
}

func DefaultLiveSpectrumParams() LiveSpectrumParams {
	return LiveSpectrumParams{Enabled: true, Window: "hann", Harmonics: 5}
}
func (s *Service) scopeViewLocked() string {
	if s.scopeView == "spectrum" {
		return "spectrum"
	}
	return "scope"
}
func (s *Service) liveSpectrumKeyLocked() liveSpectrumKey {
	window := ScopeWindow{}
	if s.scopeWindow != nil {
		window = *s.scopeWindow
	}
	return liveSpectrumKey{s.documentGeneration, s.electrical, s.run, s.actuationRevision, s.liveSpectrumSerial, strings.Join(s.plotProbeSelectionLocked(), "\x00"), window}
}
func (s *Service) liveSpectrumViewLocked() *LiveSpectrumState {
	if s.liveSpectrum == nil {
		return nil
	}
	out := *s.liveSpectrum
	out.Result = cloneSpectrumResult(out.Result)
	if out.Result != nil && (s.transient == nil || s.transient.Stale || s.liveSpectrumResultKey != s.liveSpectrumKeyLocked()) {
		out.Result = nil
		out.RunID, out.Start, out.End = 0, 0, 0
		out.Status = "waiting"
		out.Reason = "Waiting for fresh samples for the current circuit, run and scope selection"
	}
	if out.Result == nil {
		out.RunID, out.Start, out.End = 0, 0, 0
	}
	if !out.Settings.Enabled {
		out.Status = "off"
	} else if out.Settings.Hold {
		out.Status = "held"
	} else if out.Result != nil && s.paused {
		out.Status = "paused"
	}
	return &out
}
func (s *Service) setScopeViewLocked(p ScopeViewParams) error {
	if p.View != "scope" && p.View != "spectrum" {
		return fmt.Errorf("view must be scope or spectrum")
	}
	s.scopeView = p.View
	if p.View == "spectrum" && s.liveSpectrum == nil {
		return s.setLiveSpectrumLocked(DefaultLiveSpectrumParams())
	}
	return nil
}
func (s *Service) setLiveSpectrumLocked(p LiveSpectrumParams) error {
	if p.Window == "" {
		p.Window = "hann"
	}
	if p.Harmonics == 0 {
		p.Harmonics = 5
	}
	if err := (SpectrumParams{Window: p.Window, Samples: p.Samples, Fundamental: p.Fundamental, Harmonics: p.Harmonics}).validate(); err != nil {
		return err
	}
	if p.Samples > MaxLiveSpectrumSamples {
		return fmt.Errorf("live samples must be automatic or a power of two from 32 to %d", MaxLiveSpectrumSamples)
	}
	previousProbe := ""
	if s.liveSpectrum != nil {
		previousProbe = s.liveSpectrum.Settings.ProbeID
	}
	if p.ProbeID != "" && p.ProbeID != previousProbe {
		available := false
		for _, probe := range probeInventory(s.graph) {
			available = available || probe.ID == p.ProbeID
		}
		if !available || !slices.Contains(s.plotProbeSelectionLocked(), p.ProbeID) {
			return &actionError{"not-found", "Choose a probe selected in snapshot.scope.selected"}
		}
	}
	// A removed/unselected existing choice remains visible for undo. It must
	// never trap the user in live mode or prevent Hold/Recorded from working.
	if s.liveSpectrum == nil {
		s.liveSpectrum = &LiveSpectrumState{Status: "waiting", Reason: "Collecting at least 32 sample intervals"}
	}
	old := s.liveSpectrum.Settings
	old.Enabled, old.Hold = p.Enabled, p.Hold
	if old != p {
		s.liveSpectrumSerial++
		s.liveSpectrum.Result = nil
		s.liveSpectrum.Status = "waiting"
		s.liveSpectrum.Reason = "Collecting samples for these spectrum settings"
	}
	s.liveSpectrum.Settings = p
	if p.Enabled && s.liveSpectrumCancel == nil {
		ctx, cancel := context.WithCancel(context.Background())
		s.liveSpectrumCancel = cancel
		s.liveWorkers.Add(1)
		go s.runLiveSpectrum(ctx)
	}
	return nil
}
func (s *Service) runLiveSpectrum(ctx context.Context) {
	defer s.liveWorkers.Done()
	ticker := time.NewTicker(LiveSpectrumInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refreshLiveSpectrum()
		}
	}
}
func (s *Service) liveSpectrumInputLocked() (*liveSpectrumInput, string) {
	if s.transient == nil || s.transient.Stale {
		return nil, "Waiting for a fresh transient run"
	}
	c, _, _ := s.compileGraphLocked()
	scope := s.scopeLocked(c)
	if !scope.WindowValid {
		return nil, "Scope window is outside retained history"
	}
	cfg := s.liveSpectrum.Settings
	id := cfg.ProbeID
	if id == "" && len(scope.Selected) > 0 {
		id = scope.Selected[0]
	}
	for _, trace := range scope.Traces {
		if trace.Probe.ID != id {
			continue
		}
		if len(trace.Points) < 33 {
			return nil, "Collecting at least 32 sample intervals"
		}
		rows := make([]VerificationRow, len(trace.Points))
		for i, p := range trace.Points {
			rows[i] = VerificationRow{Time: p.Time, Values: []float64{p.Value}}
		}
		start, end := rows[0].Time, rows[len(rows)-1].Time
		return &liveSpectrumInput{key: s.liveSpectrumKeyLocked(), start: start, end: end, params: SpectrumParams{ID: fmt.Sprintf("live-run-%d", s.run), ProbeID: id, Start: start, End: end, Window: cfg.Window, Samples: cfg.Samples, Fundamental: cfg.Fundamental, Harmonics: cfg.Harmonics}, record: &verificationRecord{metadata: VerificationMetadata{OperationID: fmt.Sprintf("live-run-%d", s.run), DocumentGeneration: s.documentGeneration, ElectricalRevision: s.electrical, Probes: []Probe{trace.Probe}}, rows: rows}}, ""
	}
	return nil, "Select a scope probe with retained samples"
}
func (s *Service) refreshLiveSpectrum() {
	s.mu.Lock()
	if s.closed || s.liveSpectrum == nil || !s.liveSpectrum.Settings.Enabled || s.liveSpectrum.Settings.Hold {
		s.mu.Unlock()
		return
	}
	input, reason := s.liveSpectrumInputLocked()
	if input == nil {
		if s.liveSpectrum.Reason != reason || s.liveSpectrum.Result != nil {
			s.liveSpectrum.Result = nil
			s.liveSpectrum.Status = "waiting"
			s.liveSpectrum.Reason = reason
			s.revision++
			s.signalLocked("measurements")
		}
		s.mu.Unlock()
		return
	}
	if s.liveSpectrum.Result != nil && s.liveSpectrumResultKey == input.key && s.liveSpectrum.Start == input.start && s.liveSpectrum.End == input.end {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	result, err := analyzeSpectrum(input.record, input.params)
	if result != nil {
		result.Assumptions[0] = "Bounded rolling accepted live samples; no live circuit or time change; independent updates at most four times per second"
		// Keep live and immutable recording identities distinct, including sample windows.
		hash := sha256.Sum256([]byte(result.ID))
		result.ID = fmt.Sprintf("live-spectrum-%x", hash)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publishLiveSpectrumLocked(input, result, err)
}

func (s *Service) publishLiveSpectrumLocked(input *liveSpectrumInput, result *SpectrumResult, err error) {
	if s.closed || s.liveSpectrum == nil || !s.liveSpectrum.Settings.Enabled || s.liveSpectrum.Settings.Hold || s.liveSpectrumKeyLocked() != input.key || s.transient == nil || s.transient.Stale {
		return
	}
	if err != nil {
		if s.liveSpectrum.Reason == err.Error() && s.liveSpectrum.Result == nil {
			return
		}
		s.liveSpectrum.Result = nil
		s.liveSpectrum.Status = "waiting"
		s.liveSpectrum.Reason = err.Error()
	} else {
		if s.liveSpectrum.Result != nil && s.liveSpectrumResultKey == input.key && s.liveSpectrum.Result.ID == result.ID {
			return
		}
		s.liveSpectrum.Result = result
		s.liveSpectrum.Status = "ready"
		s.liveSpectrum.Reason = ""
		s.liveSpectrum.RunID = input.key.Run
		s.liveSpectrum.Start = input.start
		s.liveSpectrum.End = input.end
		s.liveSpectrumResultKey = input.key
	}
	s.revision++
	s.signalLocked("measurements")
}

func addLiveSpectrumCapabilityDetails(id string, schema map[string]any) bool {
	if id != "scope.view.set" && id != "spectrum.live.set" {
		return false
	}
	if id == "scope.view.set" {
		inputProperty(schema, "view")["enum"] = []string{"scope", "spectrum"}
	} else {
		inputProperty(schema, "window")["enum"] = []string{"", "hann", "rectangular"}
		inputProperty(schema, "samples")["enum"] = []int{0, 32, 64, 128, 256}
		frequency := inputProperty(schema, "fundamental_hz")
		delete(frequency, "type")
		delete(frequency, "$ref")
		frequency["anyOf"] = []any{map[string]any{"type": "number", "minimum": 0, "maximum": 1e6}, map[string]any{"type": "string", "minLength": 1, "maxLength": 128}}
		frequency["description"] = "Frequency in Hz, numeric SI or decimal text with Hz/kHz/mHz; zero leaves THD unset"
		inputProperty(schema, "harmonics")["enum"] = []int{0, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		schema["x-probe-inventory"] = "snapshot.scope.selected; empty chooses its first available trace"
	}
	schema["x-command-contract"] = commandContractDetail{Version: 1, Scope: "session-scope-spectrum", Effects: []string{"Preserves graph, time, physical state, edit history, saved recordings and dirty membership", "First Spectrum tab selection enables default live analysis if not previously configured; the tab itself is session-only", "Live settings replace the complete draft; hold stops FFT publication without pausing simulation"}, Errors: []commandFailureDetail{{"invalid-params", "admission", "Invalid tab, missing enabled flag, grid, window, frequency or harmonic count"}, {"not-found", "admission", "Explicit live probe is not selected in the live scope"}}, Completion: &commandCompletionDetail{Mode: "immediate settings/view update; independent bounded asynchronous live FFT", Observe: []string{"project.inspect"}}}
	schema["x-live-spectrum"] = map[string]any{"update_interval_ms": 250, "maximum_retained_samples": MaxTraceSamples, "source": "accepted scope trace points within snapshot.scope.window; no simulation advancement", "observe": "snapshot.spectrum.live; snapshots and analysis do not trigger FFT computation", "lifecycle": "session only; document replacement clears settings/result; closing cancels the one bounded worker; hold freezes publication but never preserves stale source validity", "settings": "spectrum.live.set replaces settings with Hann/automatic/five-harmonic defaults; enabled is required; recorded spectrum commands remain independent"}
	return true
}

func cloneLiveSpectrumState(in *LiveSpectrumState) *LiveSpectrumState {
	if in == nil {
		return nil
	}
	out := *in
	out.Measurements = cloneSpectrumMeasurements(in.Measurements)
	out.Result = cloneSpectrumResult(in.Result)
	return &out
}
