package main

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

type sweepSettingsContinuation struct {
	from, to actionContext
	settings workbench.SweepSettings
}

type sweepFieldRejection struct {
	generation uint64
	target     string
	value      string
	text       string
	message    string
	help       *workbench.DiagnosticHelp
}

type sweepStartWork struct {
	context  actionContext
	settings workbench.SweepSettings
	field    int
	failed   bool
}

var sweepStartFields = []string{"sweep-range-start", "sweep-range-stop", "sweep-points"}

func sweepFieldValue(settings workbench.SweepSettings, id string) string {
	switch id {
	case "sweep-range-start":
		return settings.Start
	case "sweep-range-stop":
		return settings.Stop
	default:
		return strconv.Itoa(settings.Points)
	}
}

func sweepSettingsOwner(settings workbench.SweepSettings) string {
	return settings.ComponentID + ":" + settings.Parameter
}

// Continue Run only through this form's exact accepted responses. The snapshot
// passed to a field can be older than another field's completed settlement;
// retain the accepted settings as well as their context, never adopt new work.
func (s *screen) changeSweepField(state workbench.Snapshot, id, text string) {
	pending := s.pendingSweepStart
	origin := s.fieldContext
	if pending != nil && (origin == nil || *origin != pending.context || pending.field >= len(sweepStartFields) || sweepStartFields[pending.field] != id) {
		pending.failed = true
		s.sweepSettingsAccepted = nil
		s.errorMessage = "Sweep settings changed. Review them and try again."
		s.app.Invalidate("main")
		return
	}
	settings := state.DCSweep.Settings
	if pending != nil {
		settings = pending.settings
	}
	previousValue := sweepFieldValue(settings, id)
	var parseError bool
	switch id {
	case "sweep-range-start":
		settings.Start = text
	case "sweep-range-stop":
		settings.Stop = text
	case "sweep-points":
		points, err := strconv.Atoi(strings.TrimSpace(text))
		parseError = err != nil
		settings.Points = points
	}
	response := workbench.Response{}
	if parseError {
		s.errorMessage = "Enter a whole sweep point count"
		if origin != nil && !s.fieldContextCurrent(origin) {
			s.errorMessage = "Sweep settings changed. Review them and try again."
		}
		s.requestHelp = nil
		s.app.Invalidate("main")
	} else {
		response = s.invokeAtCommandRevision("sweep.settings.set", settings, state.CommandRevision, state.Document.Generation)
	}
	if parseError || response.Error != nil {
		s.sweepSettingsAccepted = nil
		if s.fieldContextCurrent(origin) && (parseError || response.Help != nil && response.Help.Code == "invalid-params") {
			if s.sweepFieldRejected == nil {
				s.sweepFieldRejected = make(map[string]sweepFieldRejection)
			}
			s.sweepFieldRejected[id] = sweepFieldRejection{
				generation: origin.generation, target: sweepSettingsOwner(settings),
				value: previousValue, text: text, message: s.errorMessage, help: s.requestHelp,
			}
		}
		if pending != nil {
			pending.failed = true
		}
		return
	}
	delete(s.sweepFieldRejected, id)
	if origin == nil {
		return
	}
	accepted := response.Result.(workbench.Snapshot)
	from := *origin
	if previous := s.sweepSettingsAccepted; previous != nil && previous.to == from {
		from = previous.from
	}
	s.sweepSettingsAccepted = &sweepSettingsContinuation{from: from, to: actionContext{accepted.CommandRevision, accepted.Document.Generation}, settings: accepted.DCSweep.Settings}
	if pending != nil {
		pending.context, pending.settings = s.sweepSettingsAccepted.to, accepted.DCSweep.Settings
	}
}

func (s *screen) sweepStartAction(state workbench.Snapshot) ui.Node {
	label := "Run DC sweep"
	if state.DCSweep.Settings.Parameter == "noise" {
		label = "Run noise analysis"
	}
	if state.DCSweep.Settings.Parameter == "loop-frequency" {
		label = "Run loop analysis"
	} else if state.DCSweep.Settings.Parameter == "frequency" {
		label = "Run AC sweep"
	}
	return buttonStyle(ui.Node{ID: "sweep-start", Kind: ui.Button, Text: label, Accessibility: ui.Accessibility{Name: label}, Disabled: s.pendingSweepStart != nil, Activate: func() {
		if s.pendingSweepStart != nil {
			return
		}
		pending := &sweepStartWork{context: actionContext{state.CommandRevision, state.Document.Generation}, settings: state.DCSweep.Settings}
		current := s.service.Snapshot()
		if accepted := s.sweepSettingsAccepted; accepted != nil && accepted.from.generation == pending.context.generation && accepted.from.revision <= pending.context.revision && pending.context.revision <= accepted.to.revision && accepted.to == (actionContext{current.CommandRevision, current.Document.Generation}) {
			pending.context, pending.settings = accepted.to, accepted.settings
		}
		if current.CommandRevision != pending.context.revision || current.Document.Generation != pending.context.generation {
			s.finishSweepStart(pending)
			return
		}
		s.pendingSweepStart = pending
		s.settleSweepStart(pending, 0)
	}}, false)
}

func (s *screen) settleSweepStart(pending *sweepStartWork, index int) {
	if s.pendingSweepStart != pending || pending.field != index {
		return
	}
	if pending.failed || index == len(sweepStartFields) {
		s.pendingSweepStart = nil
		if pending.failed {
			s.app.Invalidate("main")
			return
		}
		s.finishSweepStart(pending)
		return
	}
	done := func(err error) {
		if s.pendingSweepStart != pending || pending.field != index {
			return
		}
		pending.field++
		if err != nil {
			pending.failed = true
			s.errorMessage = err.Error()
		}
		s.settleSweepStart(pending, index+1)
	}
	if s.settleField == nil {
		done(nil)
	} else if err := s.settleField(sweepStartFields[index], done); err != nil {
		done(err)
	}
}

func (s *screen) finishSweepStart(pending *sweepStartWork) {
	current := s.service.Snapshot()
	s.pruneSweepFieldRejections(current)
	if current.CommandRevision == pending.context.revision && current.Document.Generation == pending.context.generation {
		for _, id := range sweepStartFields {
			if rejected, ok := s.sweepFieldRejected[id]; ok {
				s.errorMessage, s.requestHelp = rejected.message, rejected.help
				s.requestHelpRevision = current.CommandRevision
				s.app.Invalidate("main")
				return
			}
		}
	}
	s.invokeAtCommandRevision("sweep.start", workbench.SweepStartParams{Settings: &pending.settings}, pending.context.revision, pending.context.generation)
}

func (s *screen) pruneSweepFieldRejections(state workbench.Snapshot) {
	for id, rejected := range s.sweepFieldRejected {
		if state.DCSweep == nil || rejected.generation != state.Document.Generation || rejected.target != sweepSettingsOwner(state.DCSweep.Settings) || rejected.value != sweepFieldValue(state.DCSweep.Settings, id) {
			delete(s.sweepFieldRejected, id)
		}
	}
}

func (s *screen) continueSweepSettingsViewport(from actionContext, accepted workbench.Snapshot) {
	previous := s.sweepSettingsAccepted
	if previous == nil || previous.to != from || accepted.Document.Generation != from.generation || accepted.DCSweep == nil || !reflect.DeepEqual(previous.settings, accepted.DCSweep.Settings) {
		return
	}
	previous.to = actionContext{accepted.CommandRevision, accepted.Document.Generation}
	if pending := s.pendingSweepStart; pending != nil && pending.context == from {
		pending.context = previous.to
	}
}
