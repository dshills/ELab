package workbench

import (
	"fmt"
	"strings"
)

type ButtonParams struct {
	ID      string `json:"id"`
	Pressed bool   `json:"pressed"`
}

func (s *Service) buttonLocked(id string, pressed, click bool) error {
	p, ok := s.graph.Component(id)
	if !ok || !IsButtonKind(p.Kind) {
		return fmt.Errorf("choose a momentary pushbutton")
	}
	if s.mixedState == nil || s.mixed == nil || s.mixed.Stale || s.transient == nil || s.transient.Stale {
		return &actionError{"unavailable", "wait for a fresh circuit before operating a pushbutton"}
	}
	if s.activeInputWorkLocked() {
		return &actionError{"unavailable", "wait for current input work to complete"}
	}
	old := s.mixedState.net.controlMemoryFor(p)
	if !click && old.Pressed == pressed && old.ReleaseAt == 0 {
		return nil
	}
	if click && (old.Pressed || old.Bouncing || old.ReleaseAt > 0) {
		return &actionError{"unavailable", "release this button and let contact bounce finish before another click"}
	}
	mode := "release"
	if pressed {
		mode = "press"
	}
	if click {
		mode = "click"
	}
	return s.startInputWorkLocked("", nil, s.graph, "button:"+id+":"+mode)
}
func (n *mixedNetwork) stageButton(event string, at float64) error {
	fields := strings.Split(event, ":")
	if len(fields) != 3 {
		return fmt.Errorf("invalid button event")
	}
	for k, p := range n.controls {
		if p.ID == fields[1] && IsButtonKind(p.Kind) {
			m := n.controlMemory[k]
			m.Pressed = fields[2] != "release"
			m.StartedAt = at
			m.ReleaseAt = 0
			m.Bouncing = p.Kind == "bouncing-pushbutton"
			if fields[2] == "click" {
				m.ReleaseAt = at + ControlledDeviceModel().ClickSeconds
			}
			n.controlMemory[k] = m
			return nil
		}
	}
	return fmt.Errorf("button no longer exists in the captured circuit")
}
