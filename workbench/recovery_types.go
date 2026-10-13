package workbench

import "encoding/json"

const RecoveryFormat = "elab.recovery"
const RecoveryVersion = 1
const MaxRecoveryBytes = MaxDocumentBytes + 16*1024
const MaxRecoveryEntries = 128

type RecoveryConfigureParams struct {
	Directory string  `json:"directory"`
	Enabled   bool    `json:"enabled"`
	Interval  float64 `json:"interval_s"`
}
type RecoveryChoiceParams struct {
	ID                 string  `json:"id"`
	SHA256             string  `json:"sha256"`
	DiscardChanges     bool    `json:"discard_changes,omitempty"`
	ExpectedGeneration *uint64 `json:"expected_document_generation,omitempty"`
}
type RecoveryCandidate struct {
	ID             string `json:"id"`
	DocumentKey    string `json:"document_key"`
	SavedAt        string `json:"saved_at"`
	SourcePath     string `json:"source_path,omitempty"`
	SHA256         string `json:"sha256"`
	CurrentSession bool   `json:"current_session"`
	ActiveSession  bool   `json:"active_session"`
	Components     int    `json:"components"`
}
type RecoveryProblem struct {
	ID      string `json:"id"`
	SHA256  string `json:"sha256,omitempty"`
	Message string `json:"message"`
}
type RecoveryState struct {
	Directory    string              `json:"directory,omitempty"`
	Enabled      bool                `json:"enabled"`
	Interval     float64             `json:"interval_s"`
	LastSnapshot string              `json:"last_snapshot,omitempty"`
	LastSavedAt  string              `json:"last_saved_at,omitempty"`
	Error        string              `json:"error,omitempty"`
	Selected     string              `json:"selected"`
	Candidates   []RecoveryCandidate `json:"candidates"`
	Problems     []RecoveryProblem   `json:"problems"`
}
type recoveryCapsule struct {
	Format          string          `json:"format"`
	Version         int             `json:"version"`
	ID              string          `json:"id"`
	DocumentKey     string          `json:"document_key"`
	Session         string          `json:"session"`
	Sequence        uint64          `json:"sequence"`
	CommandRevision uint64          `json:"command_revision"`
	SavedAt         string          `json:"saved_at"`
	SourcePath      string          `json:"source_path,omitempty"`
	DocumentSHA256  string          `json:"document_sha256"`
	Document        json.RawMessage `json:"document"`
}
type recoveryCapture struct {
	document                    Document
	generation, commandRevision uint64
	sourcePath                  string
	dirty                       bool
}
type RecoveryInspection struct {
	Candidate RecoveryCandidate `json:"candidate"`
	Document  Document          `json:"document"`
}

func cloneRecovery(in RecoveryState) RecoveryState {
	out := in
	out.Candidates = append([]RecoveryCandidate{}, in.Candidates...)
	out.Problems = append([]RecoveryProblem{}, in.Problems...)
	return out
}
