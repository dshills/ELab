package workbench

import (
	"encoding/json"
	"fmt"
)

// ProjectCapture is an owned native-format checkpoint for an application host.
// Capturing does not mark the document saved, change its file association,
// create history, restart physics, or include unfinished analysis results.
type ProjectCapture struct {
	Data            []byte
	CommandRevision uint64
	Generation      uint64
	Dirty           bool
}

func (s *Service) CaptureProject() (ProjectCapture, error) {
	s.mu.Lock()
	document := s.documentLocked()
	capture := ProjectCapture{CommandRevision: s.commandRevision, Generation: s.documentGeneration, Dirty: s.dirtyLocked()}
	s.mu.Unlock()
	data, err := json.Marshal(document)
	if err != nil {
		return capture, err
	}
	if len(data)+1 > MaxProjectBytes {
		return capture, fmt.Errorf("project exceeds %d bytes", MaxProjectBytes)
	}
	capture.Data = append(data, '\n')
	return capture, nil
}
