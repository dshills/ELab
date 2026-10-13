package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

type Snapshot struct {
	Recording            *RecordingView       `json:"recording,omitempty"`
	Recovery             RecoveryState        `json:"recovery"`
	VerificationSettings VerificationSettings `json:"verification_settings"`
	Operations           []OperationSummary   `json:"operations"`
	SelectedOperation    string               `json:"selected_operation"`
	LastOperation        string               `json:"last_operation,omitempty"`
	Verification         *VerificationPreview `json:"verification,omitempty"`
	LastExport           *Artifact            `json:"last_export,omitempty"`
	PartSearch           string               `json:"part_search"`
	MatchingParts        []Part               `json:"matching_parts"`
	UndoName             string               `json:"undo_name"`
	RedoName             string               `json:"redo_name"`
	Scope                Scope                `json:"scope"`
	Mixed                *MixedResult         `json:"mixed,omitempty"`
	NPN                  *NPNResult           `json:"npn,omitempty"`
	Transient            *Transient           `json:"transient,omitempty"`
	TransientSettings    TransientSettings    `json:"transient_settings"`
	Document             DocumentState        `json:"document"`
	CommandRevision      uint64               `json:"command_revision"`
	Revision             uint64               `json:"revision"`
	ElectricalRevision   uint64               `json:"electrical_revision"`
	Circuit              Circuit              `json:"circuit"`
	Components           []Component          `json:"components"`
	Wires                []Wire               `json:"wires"`
	Nets                 []Net                `json:"nets"`
	Drawing              DrawingState         `json:"drawing"`
	Model                LEDModel             `json:"model"`
	Selected             string               `json:"selected"`
	View                 View                 `json:"view"`
	Status               string               `json:"status"`
	Analysis             string               `json:"analysis"`
	Paused               bool                 `json:"paused"`
	RunID                uint64               `json:"run_id"`
	Result               *Result              `json:"result,omitempty"`
	Diagnostics          []Diagnostic         `json:"diagnostics"`
	CanUndo              bool                 `json:"can_undo"`
	CanRedo              bool                 `json:"can_redo"`
	LastActor            string               `json:"last_actor"`
	Closed               bool                 `json:"closed"`
}
type savedResponse struct {
	fingerprint [32]byte
	response    Response
}
type Service struct {
	savedRecordings                  [32]byte
	restorePaused                    bool
	recording                        recordingState
	recovery                         RecoveryState
	recoverySelectionCleared         bool
	recoveryStore                    *recoveryStore
	recoveryCancel                   context.CancelFunc
	recoveredFrom                    string
	discardRecoveryOnClose           bool
	operations                       map[string]*verificationOperation
	operationOrder                   []string
	operationSerial                  uint64
	selectedOperation, lastOperation string
	verificationSettings             VerificationSettings
	lastExport                       *Artifact
	scopeWindow                      *ScopeWindow
	partSearch                       string
	nextHistoryName                  string
	scopeProbes                      []string
	scopeConfigured                  bool
	cursorA, cursorB                 *float64
	activeCursor                     string

	mixed                       *MixedResult
	mixedState                  *mixedState
	npnHigh, npnLow             *NPNResult
	npn                         *NPNResult
	mu                          sync.Mutex
	documentPath                string
	documentGeneration          uint64
	savedGraph, diskHash        [32]byte
	revision, electrical, run   uint64
	commandRevision             uint64
	circuit                     Circuit
	graph                       Graph
	drawing                     DrawingState
	usedIDs                     map[string]bool
	selected                    string
	view                        View
	status                      string
	paused, closed              bool
	result                      *Result
	diagnostics                 []Diagnostic
	undo, redo                  []historyEntry
	transient                   *Transient
	transientSettings           TransientSettings
	inputMode, pendingInputMode string
	actor                       string
	cache                       map[string]savedResponse
	cacheOrder                  []string
	changed                     chan struct{}
	cancel                      context.CancelFunc
	solver                      func(context.Context, Circuit) (Result, error)
}

func New() *Service        { return newGraphService(SolveLED, EmptyGraph()) }
func NewExample() *Service { return newService(SolveLED) }
func newService(solver func(context.Context, Circuit) (Result, error)) *Service {
	return newGraphService(solver, ExampleGraph())
}
func newGraphService(solver func(context.Context, Circuit) (Result, error), graph Graph) *Service {
	s := &Service{revision: 1, electrical: 1, graph: graph.Clone(), drawing: DefaultDrawing(), usedIDs: map[string]bool{}, view: View{Zoom: 1}, cache: map[string]savedResponse{}, changed: make(chan struct{}), solver: solver, actor: "system"}
	for _, c := range graph.Components {
		s.usedIDs[c.ID] = true
	}
	for _, w := range graph.Wires {
		s.usedIDs[w.ID] = true
	}
	s.commandRevision = 1
	s.inputMode = "auto"
	s.transientSettings = DefaultTransientSettings()
	s.verificationSettings = DefaultVerificationSettings()
	s.recovery = RecoveryState{Interval: 2, Candidates: []RecoveryCandidate{}, Problems: []RecoveryProblem{}}
	s.documentGeneration = 1
	s.savedGraph = contentHash(EmptyGraph(), s.transientSettings)
	s.savedRecordings = s.recordingMembershipHashLocked()
	s.startLocked()
	return s
}
func (s *Service) signalLocked()      { close(s.changed); s.changed = make(chan struct{}) }
func (s *Service) Snapshot() Snapshot { s.mu.Lock(); defer s.mu.Unlock(); return s.snapshotLocked() }
func (s *Service) snapshotLocked() Snapshot {
	graph := s.graph.Clone()
	compiled, _, _ := graph.Compile()
	analysis := compiled.Analysis
	if analysis == "" {
		analysis = "DC operating point"
	}
	out := Snapshot{Recording: s.recordingViewLocked(), Recovery: cloneRecovery(s.recovery), VerificationSettings: cloneVerificationSettings(s.verificationSettings), Operations: []OperationSummary{}, SelectedOperation: s.selectedOperation, LastOperation: s.lastOperation, PartSearch: s.partSearch, MatchingParts: filteredParts(s.partSearch), Scope: s.scopeLocked(compiled), Mixed: cloneMixed(s.mixed), NPN: cloneNPN(s.npn), CommandRevision: s.commandRevision, Transient: cloneTransient(s.transient), TransientSettings: s.transientSettings, Document: DocumentState{SchematicDirty: contentHash(s.graph, s.transientSettings) != s.savedGraph, RecordingsDirty: s.recordingsDirtyLocked(), RecordingCount: s.recordingCountLocked(), Path: s.documentPath, Dirty: s.dirtyLocked(), FormatVersion: DocumentVersion, Generation: s.documentGeneration, RecoveredFrom: s.recoveredFrom}, Revision: s.revision, ElectricalRevision: s.electrical, Circuit: s.circuit, Components: graph.Components, Wires: graph.Wires, Nets: compiled.Nets, Drawing: s.drawing, Model: Model(), Selected: s.selected, View: s.view, Status: s.status, Analysis: analysis, Paused: s.paused, RunID: s.run, Diagnostics: append([]Diagnostic{}, s.diagnostics...), CanUndo: len(s.undo) > 0, CanRedo: len(s.redo) > 0, LastActor: s.actor, Closed: s.closed}
	for i := len(s.operationOrder) - 1; i >= 0; i-- {
		out.Operations = append(out.Operations, s.operationSummaryLocked(s.operations[s.operationOrder[i]]))
	}
	if operation := s.operations[s.selectedOperation]; operation != nil && operation.record != nil {
		record := operation.record
		preview := &VerificationPreview{ID: operation.summary.ID, Engine: record.metadata.Engine, InputMode: record.metadata.InputMode, Duration: record.metadata.Duration, SampleCount: record.metadata.SampleCount, Readings: []VerificationReading{}}
		for i, probe := range record.metadata.Probes {
			stat := record.metadata.Statistics[i]
			preview.Readings = append(preview.Readings, VerificationReading{Probe: probe, Value: record.rows[len(record.rows)-1].Values[i], Minimum: stat.Minimum, Maximum: stat.Maximum})
		}
		out.Verification = preview
	}
	if s.lastExport != nil {
		artifact := *s.lastExport
		out.LastExport = &artifact
	}
	if len(s.undo) > 0 {
		out.UndoName = s.undo[len(s.undo)-1].Name
	}
	if len(s.redo) > 0 {
		out.RedoName = s.redo[len(s.redo)-1].Name
	}
	if out.Transient != nil {
		out.Transient.Sample.frame = nil
		for i := range out.Transient.Samples {
			out.Transient.Samples[i].frame = nil
		}
	}
	if s.result != nil {
		r := *s.result
		r.Nodes = map[string]float64{}
		for k, v := range s.result.Nodes {
			r.Nodes[k] = v
		}
		out.Result = &r
	}
	return out
}
func (s *Service) startLocked() {
	allowPaused := s.restorePaused
	s.restorePaused = false
	s.cursorA = nil
	s.cursorB = nil
	s.scopeWindow = nil
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.run++
	if s.mixed != nil {
		s.mixed.Stale = true
	}
	if s.npn != nil {
		s.npn.Stale = true
	}
	if s.transient != nil {
		s.transient.Stale = true
	}
	if s.result != nil {
		s.result.Stale = true
	}
	s.diagnostics = nil
	s.circuit = s.graph.Summary()
	compiled, status, diagnostic := s.graph.Compile()
	if diagnostic != nil {
		s.diagnostics = []Diagnostic{*diagnostic}
		if s.paused {
			s.status = "paused"
		} else {
			s.status = status
		}
		return
	}
	if s.paused && !allowPaused {
		s.status = "paused"
		return
	}
	if compiled.Analysis == MixedAnalysis {
		s.startMixedLocked(compiled)
		return
	}
	s.mixed = nil
	s.mixedState = nil
	if compiled.Analysis == NPNTransientAnalysis {
		s.startNPNTransientLocked(compiled)
		return
	}
	if compiled.Analysis == NPNAnalysis {
		s.startNPNLocked(compiled, allowPaused)
		return
	}
	s.npn = nil
	if compiled.Analysis == "RC transient" {
		s.startRCLocked(compiled)
		return
	}
	s.transient = nil
	s.status = "updating"
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	c, revision, run := s.circuit, s.electrical, s.run
	go func() {
		result, err := s.solver(ctx, c)
		if err == nil {
			result = compiled.Annotate(result)
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed || s.paused && !allowPaused || s.electrical != revision || s.run != run || ctx.Err() != nil {
			return
		}
		s.cancel = nil
		cancel()
		if err != nil {
			s.status = "error"
			s.diagnostics = []Diagnostic{{"solve-failed", err.Error(), "D1"}}
		} else {
			result.RunID = run
			result.ElectricalRevision = revision
			s.result = &result
			if s.paused {
				s.status = "paused"
			} else {
				s.status = "running"
			}
		}
		s.revision++
		s.signalLocked()
	}()
}
func (s *Service) Close() { s.mu.Lock(); defer s.mu.Unlock(); s.closeLocked() }
func (s *Service) closeLocked() {
	if s.closed {
		return
	}
	if s.recoveryStore != nil {
		if err := s.recoveryStore.finish(s.recoveryCaptureLocked(), s.discardRecoveryOnClose, s.recovery.Enabled); err != nil {
			s.recovery.Error = err.Error()
		}
		if s.recoveryCancel != nil {
			s.recoveryCancel()
		}
	}
	s.closed = true
	if s.recording.cancel != nil {
		s.recording.cancel()
	}
	s.recording.playing = false
	for _, operation := range s.operations {
		if operationActive(operation.summary.Status) {
			operation.cancel()
			operation.summary.Status = "canceled"
		}
	}
	s.status = "closed"
	if s.cancel != nil {
		s.cancel()
	}
	s.revision++
	s.signalLocked()
}
func (s *Service) Wait(ctx context.Context) (Snapshot, error) {
	for {
		s.mu.Lock()
		state := s.snapshotLocked()
		changed := s.changed
		s.mu.Unlock()
		if state.Status != "updating" {
			return state, nil
		}
		select {
		case <-ctx.Done():
			return state, ctx.Err()
		case <-changed:
		}
	}
}

// Dispatch is the sole semantic entry point used by widgets and agent clients.
// Mutation preconditions and retry keys apply identically to both.
func (s *Service) Dispatch(ctx context.Context, r Request, actor string) Response {
	if r.Command == "simulation.wait" {
		if r.Version != ProtocolVersion || r.ID == "" || len(r.ID) > 128 {
			return s.failure(r, "invalid-request", "version 1 and a nonempty ID (max 128 bytes) required")
		}
		if err := params(r.Params, &struct{}{}); err != nil {
			return s.failure(r, "invalid-params", err.Error())
		}
		bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		state, err := s.Wait(bounded)
		if err != nil {
			return Response{Version: 1, ID: r.ID, Revision: state.Revision, Error: &Failure{"timeout", err.Error()}}
		}
		return Response{Version: 1, ID: r.ID, Revision: state.Revision, Result: state}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Response{Version: ProtocolVersion, ID: r.ID, Revision: s.revision}
	fail := func(code, message string) Response { out.Error = &Failure{code, message}; return out }
	if err := ctx.Err(); err != nil {
		return fail("canceled", err.Error())
	}
	if r.Version != ProtocolVersion || r.ID == "" || len(r.ID) > 128 {
		return fail("invalid-request", "version 1 and a nonempty ID (max 128 bytes) required")
	}
	cap, known := capabilityIndex[r.Command]
	if !known {
		return fail("unknown-command", "enumerate capabilities.list for supported commands")
	}
	if !cap.Mutation {
		if r.Command == "recovery.inspect" || r.Command == "recovery.samples" {
			var inspection any
			var err error
			if r.Command == "recovery.samples" {
				inspection, err = s.recoverySamplesLocked(r.Params)
			} else {
				inspection, err = s.recoveryInspectionLocked(r.Params)
			}
			if err != nil {
				var action *actionError
				if errors.As(err, &action) {
					return fail(action.code, action.message)
				}
				return fail("invalid-params", err.Error())
			}
			out.Result = inspection
			return out
		}
		if r.Command == "operation.inspect" || r.Command == "operation.samples" {
			result, err := s.verificationQueryLocked(r.Command, r.Params)
			if err != nil {
				var action *actionError
				if errors.As(err, &action) {
					return fail(action.code, action.message)
				}
				return fail("invalid-params", err.Error())
			}
			out.Result = result
			return out
		}
		if err := params(r.Params, &struct{}{}); err != nil {
			return fail("invalid-params", err.Error())
		}
		if r.Command == "capabilities.list" {
			out.Result = map[string]any{"version": ProtocolVersion, "schema_version": DiscoverySchemaVersion, "schemas": SchemaRegistry(), "capabilities": Catalog(), "retry_retention": "Latest 256 successful mutation keys per session; reconnect to the same process preserves them. Expired keys have no deduplication guarantee."}
		} else if r.Command == "parts.list" {
			out.Result = Library()
		} else {
			out.Result = s.snapshotLocked()
		}
		return out
	}
	if r.ExpectedRevision == nil || r.Key == "" || len(r.Key) > 128 {
		return fail("invalid-request", "mutations require expected_revision and key (1–128 bytes)")
	}
	canonical := map[string]any{}
	if err := DecodeStrict(r.Params, &canonical); err != nil {
		return fail("invalid-params", err.Error())
	}
	if canonical == nil {
		return fail("invalid-params", "params must be a JSON object")
	}
	data, _ := json.Marshal(struct {
		Command string
		Params  map[string]any
	}{r.Command, canonical})
	fingerprint := sha256.Sum256(data)
	key := actor + "\x00" + r.Key
	if old, ok := s.cache[key]; ok {
		if old.fingerprint != fingerprint {
			return fail("key-reused", "retry key already belongs to a different command")
		}
		out = old.response
		out.Result = cloneSnapshot(old.response.Result.(Snapshot))
		out.ID = r.ID
		out.Duplicate = true
		return out
	}
	if s.closed {
		return fail("closed", "session is closed")
	}
	if *r.ExpectedRevision != s.revision {
		return fail("conflict", fmt.Sprintf("expected revision %d; current revision is %d", *r.ExpectedRevision, s.revision))
	}
	before := s.historyLocked()
	s.nextHistoryName = ""
	if err := s.applyLocked(r.Command, r.Params); err != nil {
		var action *actionError
		if errors.As(err, &action) {
			return fail(action.code, action.message)
		}
		return fail("invalid-params", err.Error())
	}
	changedGraph := !reflect.DeepEqual(before.Graph, s.graph)
	changedSettings := before.Settings != s.transientSettings
	changedDocument := changedGraph || changedSettings
	forceRestart := r.Command == "project.load" || r.Command == "recovery.restore" || r.Command == "example.rc" || r.Command == "example.npn" || r.Command == "example.npn-pulse" || r.Command == "example.flasher"
	if changedDocument {
		if r.Command != "edit.undo" && r.Command != "edit.redo" {
			before.Name = historyName(r.Command)
			if s.nextHistoryName != "" {
				before.Name = s.nextHistoryName
			}
			s.undo = append(s.undo, before)
			s.redo = nil
			if len(s.undo) > 100 {
				s.undo = append([]historyEntry(nil), s.undo[len(s.undo)-100:]...)
			}
		}
		s.normalizeDrawingLocked()
		if !ElectricalEqual(before.Graph, s.graph) || changedSettings || forceRestart {
			s.electrical++
			s.startLocked()
		}
	}
	if forceRestart && !changedDocument {
		s.electrical++
		s.startLocked()
	}
	if r.Command == "project.clear" || r.Command == "example.reset" || forceRestart && r.Command != "simulation.restart" && r.Command != "project.load" && r.Command != "recovery.restore" {
		s.verificationSettings = DefaultVerificationSettings()
		s.scopeConfigured = false
		s.scopeProbes = nil
		s.activeCursor = "a"
		s.cursorA = nil
		s.cursorB = nil
	}
	if r.Command == "simulation.verify" || r.Command == "verification.import" {
		s.operations[s.lastOperation].summary.Actor = actor
	}
	s.actor = actor
	s.commandRevision++
	s.revision++
	s.signalLocked()
	out.Revision = s.revision
	out.Result = s.snapshotLocked()
	cached := out
	cached.Result = cloneSnapshot(out.Result.(Snapshot))
	s.cache[key] = savedResponse{fingerprint, cached}
	s.cacheOrder = append(s.cacheOrder, key)
	if len(s.cacheOrder) > 256 {
		delete(s.cache, s.cacheOrder[0])
		s.cacheOrder = s.cacheOrder[1:]
	}
	return out
}

func cloneSnapshot(state Snapshot) Snapshot {
	state.Recording = cloneRecordingView(state.Recording)
	state.Recovery = cloneRecovery(state.Recovery)
	state.VerificationSettings = cloneVerificationSettings(state.VerificationSettings)
	state.Operations = append([]OperationSummary{}, state.Operations...)
	for i := range state.Operations {
		if state.Operations[i].Error != nil {
			failure := *state.Operations[i].Error
			state.Operations[i].Error = &failure
		}
	}
	if state.Verification != nil {
		preview := *state.Verification
		preview.Readings = append([]VerificationReading{}, preview.Readings...)
		state.Verification = &preview
	}
	if state.LastExport != nil {
		artifact := *state.LastExport
		state.LastExport = &artifact
	}
	state.MatchingParts = filteredParts(state.PartSearch)
	state.Scope = cloneScope(state.Scope)
	state.Mixed = cloneMixed(state.Mixed)
	state.NPN = cloneNPN(state.NPN)
	state.Transient = cloneTransient(state.Transient)
	state.Components = append([]Component(nil), state.Components...)
	for i := range state.Components {
		state.Components[i].Terminals = append([]string(nil), state.Components[i].Terminals...)
		if v := state.Components[i].InitialVoltage; v != nil {
			copy := *v
			state.Components[i].InitialVoltage = &copy
		}
	}
	state.Wires = append([]Wire(nil), state.Wires...)
	state.Diagnostics = append([]Diagnostic{}, state.Diagnostics...)
	state.Nets = append([]Net{}, state.Nets...)
	for i := range state.Nets {
		state.Nets[i].Terminals = append([]string{}, state.Nets[i].Terminals...)
		state.Nets[i].Labels = append([]string{}, state.Nets[i].Labels...)
	}
	if state.Result != nil {
		r := *state.Result
		r.Nodes = map[string]float64{}
		for k, v := range state.Result.Nodes {
			r.Nodes[k] = v
		}
		state.Result = &r
	}
	return state
}
func (s *Service) failure(r Request, code, message string) Response {
	return Response{Version: 1, ID: r.ID, Revision: s.Snapshot().Revision, Error: &Failure{code, message}}
}
