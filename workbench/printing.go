package workbench

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dshills/elab/internal/printspool"
)

const MaxPrintJobs = 8

type PrintDestinationParams struct {
	PrinterID string `json:"printer_id"`
	Copies    int    `json:"copies"`
}
type PrintSubmitParams struct {
	PreviewID string `json:"preview_id"`
	PrinterID string `json:"printer_id"`
	Copies    int    `json:"copies"`
}
type PrintWork struct {
	ID     string   `json:"id"`
	Kind   string   `json:"kind"`
	Status string   `json:"status"`
	JobID  string   `json:"job_id,omitempty"`
	Actor  string   `json:"actor"`
	Error  *Failure `json:"error,omitempty"`
}
type PrintJob struct {
	PageCount          int           `json:"page_count"`
	ID                 string        `json:"id"`
	PreviewID          string        `json:"preview_id"`
	PrinterID          string        `json:"printer_id"`
	Copies             int           `json:"copies"`
	Status             string        `json:"status"`
	SpoolID            int           `json:"spool_id,omitempty"`
	CancelRequested    bool          `json:"cancel_requested"`
	Page               SchematicPage `json:"page"`
	RunID              uint64        `json:"run_id"`
	ElectricalRevision uint64        `json:"electrical_revision"`
	DocumentGeneration uint64        `json:"document_generation"`
	Actor              string        `json:"actor"`
	PDFSHA256          string        `json:"pdf_sha256,omitempty"`
	PDFBytes           int           `json:"pdf_bytes"`
	Error              *Failure      `json:"error,omitempty"`
}
type PrintState struct {
	Destinations []printspool.Destination `json:"destinations"`
	PrinterID    string                   `json:"printer_id"`
	Copies       int                      `json:"copies"`
	Work         *PrintWork               `json:"work,omitempty"`
	Jobs         []PrintJob               `json:"jobs"`
}
type printWork struct {
	view   PrintWork
	ctx    context.Context
	cancel context.CancelFunc
	job    *PrintJob
}

func clonePrintState(in *PrintState) *PrintState {
	if in == nil {
		return nil
	}
	out := *in
	out.Destinations = append([]printspool.Destination{}, in.Destinations...)
	out.Jobs = append([]PrintJob{}, in.Jobs...)
	copyFailure := func(e *Failure) *Failure {
		if e == nil {
			return nil
		}
		v := *e
		return &v
	}
	for i := range out.Jobs {
		out.Jobs[i].Page = cloneSchematicPage(out.Jobs[i].Page)
		out.Jobs[i].Error = copyFailure(out.Jobs[i].Error)
	}
	if in.Work != nil {
		v := *in.Work
		v.Error = copyFailure(v.Error)
		out.Work = &v
	}
	return &out
}
func (s *Service) printingStateLocked() *PrintState {
	out := clonePrintState(s.printState)
	if out == nil {
		return nil
	}
	if s.printWork != nil {
		v := s.printWork.view
		if v.Error != nil {
			e := *v.Error
			v.Error = &e
		}
		out.Work = &v
	}
	out.Jobs = []PrintJob{}
	for _, job := range s.printJobs {
		v := *job
		v.Page = cloneSchematicPage(job.Page)
		if v.Error != nil {
			e := *v.Error
			v.Error = &e
		}
		out.Jobs = append(out.Jobs, v)
	}
	return out
}
func (s *Service) printJobLocked(id string) *PrintJob {
	for _, j := range s.printJobs {
		if j.ID == id {
			return j
		}
	}
	return nil
}
func (s *Service) printCommandLocked(command string, raw json.RawMessage) error {
	if command == "print.work.cancel" {
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		if s.printWork == nil || p.ID != s.printWork.view.ID {
			return &actionError{"not-found", "Print work is no longer current"}
		}
		if !operationActive(s.printWork.view.Status) {
			return &actionError{"unavailable", "Print work is already terminal"}
		}
		s.printWork.view.Status = "canceling"
		s.printWork.cancel()
		return nil
	}
	if s.printWork != nil && operationActive(s.printWork.view.Status) {
		return &actionError{"busy", "Finish or cancel active print work first"}
	}
	var target *PrintJob
	var capture exportCapture
	switch command {
	case "print.printers.refresh":
		if err := params(raw, &struct{}{}); err != nil {
			return err
		}
	case "print.destination.set":
		var p PrintDestinationParams
		if err := params(raw, &p, "printer_id", "copies"); err != nil {
			return err
		}
		if p.Copies < 1 || p.Copies > 10 {
			return fmt.Errorf("copies must be 1–10")
		}
		if p.PrinterID != "" && !s.printDestinationLocked(p.PrinterID, false) {
			return &actionError{"not-found", "Refresh printers and choose a listed destination"}
		}
		if s.printState == nil {
			s.printState = &PrintState{Destinations: []printspool.Destination{}, Jobs: []PrintJob{}}
		}
		s.printState.PrinterID, s.printState.Copies = p.PrinterID, p.Copies
		return nil
	case "print.submit":
		var p PrintSubmitParams
		if err := params(raw, &p, "preview_id", "printer_id", "copies"); err != nil {
			return err
		}
		if p.Copies < 1 || p.Copies > 10 {
			return fmt.Errorf("copies must be 1–10")
		}
		if !s.printDestinationLocked(p.PrinterID, true) {
			return &actionError{"unavailable", "Choose a listed printer accepting jobs; refresh printers if needed"}
		}
		if s.preview == nil || !s.preview.view.Visible || s.preview.view.ID != p.PreviewID || s.preview.view.Status != "completed" || len(s.preview.data) == 0 {
			return &actionError{"not-ready", "Printing requires the exact visible completed preview"}
		}
		if len(s.printJobs) >= MaxPrintJobs {
			index := -1
			for i, j := range s.printJobs {
				if printspool.Terminal(j.Status) {
					index = i
					break
				}
			}
			if index < 0 {
				return &actionError{"record-limit", "Eight unresolved print jobs retained; inspect or cancel known spool jobs first"}
			}
			s.printJobs = append(s.printJobs[:index], s.printJobs[index+1:]...)
		}
		capture = s.preview.capture
		capture.page.PageIndex = nil
		capture.artifact.PageIndex = nil
		capture.kind, capture.page.RasterDPI, capture.artifact.Format, capture.artifact.DPI = "schematic-pdf", 0, "pdf", 0
		s.printJobSerial++
		pview := s.preview.view
		target = &PrintJob{PageCount: pview.PageCount, ID: fmt.Sprintf("print-%d", s.printJobSerial), PreviewID: p.PreviewID, PrinterID: p.PrinterID, Copies: p.Copies, Status: "preparing", Page: cloneSchematicPage(pview.Page), RunID: pview.RunID, ElectricalRevision: pview.ElectricalRevision, DocumentGeneration: pview.DocumentGeneration}
		s.printJobs = append(s.printJobs, target)
	case "print.job.refresh", "print.job.cancel":
		var p IDParams
		if err := params(raw, &p, "id"); err != nil {
			return err
		}
		target = s.printJobLocked(p.ID)
		if target == nil {
			return &actionError{"not-found", "Print job is not retained by this ELab session"}
		}
		if target.SpoolID <= 0 {
			return &actionError{"unavailable", "No acknowledged spool job ID; check the system print queue before retrying"}
		}
		if command == "print.job.cancel" && printspool.Terminal(target.Status) {
			return &actionError{"unavailable", "Spool job is already terminal"}
		}
	default:
		return fmt.Errorf("unknown print command")
	}
	if s.printState == nil {
		s.printState = &PrintState{Destinations: []printspool.Destination{}, Copies: 1, Jobs: []PrintJob{}}
	}
	s.printSerial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	w := &printWork{ctx: ctx, cancel: cancel, job: target, view: PrintWork{ID: fmt.Sprintf("print-work-%d", s.printSerial), Kind: command, Status: "queued"}}
	if target != nil {
		w.view.JobID = target.ID
	}
	s.printWork = w
	backend, render := s.printBackend, s.printRender
	s.liveWorkers.Add(1)
	go s.runPrintWork(w, backend, render, capture)
	return nil
}
func (s *Service) printDestinationLocked(id string, accepting bool) bool {
	if s.printState != nil {
		for _, d := range s.printState.Destinations {
			if d.ID == id && id != "" {
				return !accepting || d.Accepting
			}
		}
	}
	return false
}
func (s *Service) runPrintWork(w *printWork, backend printspool.Backend, render func(context.Context, exportCapture) ([]byte, error), capture exportCapture) {
	defer s.liveWorkers.Done()
	defer w.cancel()
	s.mu.Lock()
	if w.ctx.Err() == nil {
		w.view.Status = "running"
		s.revision++
		s.signalLocked("printing", "jobs")
	}
	s.mu.Unlock()
	var destinations []printspool.Destination
	var receipt printspool.Receipt
	var err error
	var pdf []byte
	cancelAcknowledged := false
	if err = w.ctx.Err(); err == nil {
		switch w.view.Kind {
		case "print.printers.refresh":
			destinations, err = backend.Destinations(w.ctx)
		case "print.submit":
			pdf, err = render(w.ctx, capture)
			if err == nil && (len(pdf) == 0 || len(pdf) > MaxProjectBytes) {
				err = fmt.Errorf("print PDF must be 1 byte–8 MiB")
			}
			if err == nil {
				err = w.ctx.Err()
			}
			if err == nil {
				receipt, err = s.submitPrintPDF(w, backend, pdf)
			}
		case "print.job.refresh":
			receipt, err = backend.Inspect(w.ctx, w.job.SpoolID)
		case "print.job.cancel":
			err = backend.Cancel(w.ctx, w.job.SpoolID)
			cancelAcknowledged = err == nil
			if err == nil {
				receipt, err = backend.Inspect(w.ctx, w.job.SpoolID)
			}
		}
	}
	if err == nil && w.ctx.Err() != nil && !(w.view.Kind == "print.submit" && receipt.ID > 0) && !cancelAcknowledged {
		err = w.ctx.Err()
	}
	if w.view.Kind == "print.submit" && receipt.ID <= 0 && err == nil {
		err = printspool.ErrUncertain
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if cancelAcknowledged {
		w.job.CancelRequested = true
	}
	// An acknowledged submission wins over local cancellation/deadline: paper may
	// already be printing. Never discard the ID or label it locally canceled.
	if w.view.Kind == "print.submit" && receipt.ID > 0 {
		w.job.SpoolID, w.job.Status = receipt.ID, "submitted"
		err = nil
	} else if w.view.Kind == "print.submit" {
		w.job.Status = "failed"
		if errors.Is(err, printspool.ErrUncertain) {
			w.job.Status = "unknown"
		} else if errors.Is(err, context.Canceled) {
			w.job.Status = "canceled"
		}
	} else if err == nil && w.job != nil {
		w.job.Status = receipt.State
		w.job.Error = nil
	}
	w.view.Status = "completed"
	if err != nil {
		code := "print-failed"
		switch {
		case errors.Is(err, printspool.ErrUncertain):
			code = "submission-unknown"
		case errors.Is(err, printspool.ErrUnsupported):
			code = "unsupported"
		case errors.Is(err, printspool.ErrNotFound):
			code = "spool-job-unavailable"
		case errors.Is(err, context.DeadlineExceeded):
			code = "timeout"
		case errors.Is(err, context.Canceled):
			code = "canceled"
		}
		w.view.Status = "failed"
		if code == "canceled" {
			w.view.Status = "canceled"
		}
		w.view.Error = &Failure{Code: code, Message: err.Error()}
		if w.job != nil {
			e := *w.view.Error
			w.job.Error = &e
		}
	} else if w.view.Kind == "print.printers.refresh" && !s.closed {
		s.printState.Destinations = append([]printspool.Destination{}, destinations...)
		if !s.printDestinationLocked(s.printState.PrinterID, false) {
			s.printState.PrinterID = ""
		}
	}
	s.revision++
	s.signalLocked("printing", "jobs")
}
func (s *Service) submitPrintPDF(w *printWork, backend printspool.Backend, pdf []byte) (printspool.Receipt, error) {
	directory, err := os.MkdirTemp("", "elab-print-")
	if err != nil {
		return printspool.Receipt{}, err
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "schematic.pdf")
	if err = os.WriteFile(path, pdf, 0600); err != nil {
		return printspool.Receipt{}, err
	}
	if err = w.ctx.Err(); err != nil {
		return printspool.Receipt{}, err
	}
	s.mu.Lock()
	w.job.Status = "submitting"
	sum := sha256.Sum256(pdf)
	w.job.PDFSHA256, w.job.PDFBytes = hex.EncodeToString(sum[:]), len(pdf)
	s.revision++
	s.signalLocked("printing", "jobs")
	s.mu.Unlock()
	return backend.Submit(w.ctx, w.job.PrinterID, path, "ELab "+w.job.ID, printspool.Options{Paper: w.job.Page.Paper, Orientation: w.job.Page.Orientation, Copies: w.job.Copies})
}
