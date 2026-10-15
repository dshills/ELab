package workbench

import "fmt"

// mutationContextFailureLocked checks the common lifecycle/document/revision
// guards under s.mu. Preflight is not admission: dispatch checks again after a
// compatibility wait, and cached replies retain their existing precedence.
func (s *Service) mutationContextFailureLocked(r Request) *actionError {
	if s.closed && r.Command != "app.close.acknowledge" {
		return &actionError{"closed", "session is closed"}
	}
	if r.ExpectedDocumentGeneration != nil && *r.ExpectedDocumentGeneration != s.documentGeneration {
		return &actionError{"conflict", "document changed before this action; inspect the current document and retry"}
	}
	if r.ExpectedRevision != nil && *r.ExpectedRevision != s.revision {
		return &actionError{"conflict", fmt.Sprintf("expected revision %d; current revision is %d", *r.ExpectedRevision, s.revision)}
	}
	if r.ExpectedCommandRevision != nil && *r.ExpectedCommandRevision != s.commandRevision {
		return &actionError{"conflict", fmt.Sprintf("expected command revision %d; current command revision is %d", *r.ExpectedCommandRevision, s.commandRevision)}
	}
	return nil
}
