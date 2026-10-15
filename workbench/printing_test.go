package workbench

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/dshills/elab/internal/printspool"
	"github.com/dshills/elab/internal/schematicpdf"
)

type fakePrintSpool struct {
	list    func(context.Context) ([]printspool.Destination, error)
	submit  func(context.Context, string, string, string, printspool.Options) (printspool.Receipt, error)
	inspect func(context.Context, int) (printspool.Receipt, error)
	cancel  func(context.Context, int) error
}

func (f *fakePrintSpool) Destinations(ctx context.Context) ([]printspool.Destination, error) {
	if f.list != nil {
		return f.list(ctx)
	}
	return []printspool.Destination{{ID: "lab", Name: "Lab printer", Default: true, Accepting: true, State: "idle"}}, nil
}
func (f *fakePrintSpool) Submit(ctx context.Context, p, path, title string, o printspool.Options) (printspool.Receipt, error) {
	return f.submit(ctx, p, path, title, o)
}
func (f *fakePrintSpool) Inspect(ctx context.Context, id int) (printspool.Receipt, error) {
	if f.inspect != nil {
		return f.inspect(ctx, id)
	}
	return printspool.Receipt{ID: id, State: "pending"}, nil
}
func (f *fakePrintSpool) Cancel(ctx context.Context, id int) error {
	if f.cancel != nil {
		return f.cancel(ctx, id)
	}
	return nil
}
func awaitPrintWork(t *testing.T, s *Service, status string) *PrintState {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		state := s.Snapshot().Printing
		if state != nil && state.Work != nil && !operationActive(state.Work.Status) {
			if state.Work.Status != status {
				t.Fatalf("print work: %+v", state.Work)
			}
			return state
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("print work did not settle")
	return nil
}

// A completed preview fixture isolates printing from the platform renderer, so
// capture/transaction tests also execute without cgo. Real PDF parity is separate.
func preparedPrintService(t *testing.T, spool printspool.Backend) *Service {
	t.Helper()
	s := pausedRC(t)
	s.printBackend = spool
	page, layout, _ := normalizeSchematicPage(nil)
	s.mu.Lock()
	s.preview = &previewJob{cancel: func() {}, view: PrintPreview{ID: "captured", Visible: true, Status: "completed", Page: page, RunID: s.run, ElectricalRevision: s.electrical, DocumentGeneration: s.documentGeneration}, data: []byte("preview"), capture: exportCapture{graph: s.graph.Clone(), kind: "schematic-png", settings: s.transientSettings, page: layout, artifact: Artifact{Page: &page}}}
	s.mu.Unlock()
	call(t, s, "print.printers.refresh", struct{}{}, "printers")
	awaitPrintWork(t, s, "completed")
	return s
}
func TestPrintingCapturedPDFSpoolLifecycleAndOwnedState(t *testing.T) {
	var received []byte
	var temporary string
	var options printspool.Options
	spool := &fakePrintSpool{submit: func(ctx context.Context, printer, path, title string, o printspool.Options) (printspool.Receipt, error) {
		if printer != "lab" || title != "ELab print-1" {
			t.Error("submission not pinned", printer, title)
		}
		var err error
		received, err = os.ReadFile(path)
		temporary = path
		options = o
		if info, e := os.Stat(path); e != nil || info.Mode().Perm() != 0600 {
			t.Error("print file is not private")
		}
		return printspool.Receipt{ID: 41}, err
	}}
	s := preparedPrintService(t, spool)
	defer s.Close()
	s.printRender = func(ctx context.Context, c exportCapture) ([]byte, error) {
		part, _ := c.graph.Component("R1")
		if c.kind != "schematic-pdf" || c.page.RasterDPI != 0 || part.Value != 10000 {
			t.Error("preview capture changed", part.Value)
		}
		return []byte("%PDF-captured"), nil
	}
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "later-source")
	before := s.Snapshot()
	response := call(t, s, "print.submit", PrintSubmitParams{"captured", "lab", 2}, "submit")
	state := awaitPrintWork(t, s, "completed")
	if state.Jobs[0].SpoolID != 41 || state.Jobs[0].Status != "submitted" || state.Jobs[0].Actor != "test" || state.Work.Actor != "test" || options.Copies != 2 || options.Orientation != "landscape" || options.Paper != "letter" || !bytes.Equal(received, []byte("%PDF-captured")) {
		t.Fatal("submission metadata/options differ", state, options)
	}
	if _, err := os.Stat(temporary); !os.IsNotExist(err) {
		t.Fatal("temporary print PDF leaked")
	}
	sum := sha256.Sum256(received)
	if state.Jobs[0].PDFSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("PDF identity missing")
	}
	retry := call(t, s, "print.submit", PrintSubmitParams{"captured", "lab", 2}, "submit")
	if !retry.Duplicate || len(s.Snapshot().Printing.Jobs) != 1 || retry.Result.(Snapshot).Printing.Work.ID != response.Result.(Snapshot).Printing.Work.ID {
		t.Fatal("retry submitted twice")
	}
	state.Jobs[0].Page.Paper = "tampered"
	*state.Jobs[0].Page.MarginMM = 35
	state.Destinations[0].Name = "tampered"
	owned := s.Snapshot().Printing
	if owned.Jobs[0].Page.Paper != "letter" || *owned.Jobs[0].Page.MarginMM != 10 || owned.Destinations[0].Name != "Lab printer" {
		t.Fatal("print state aliases caller")
	}
	call(t, s, "print.job.refresh", IDParams{ID: "print-1"}, "observe")
	awaitPrintWork(t, s, "completed")
	if s.Snapshot().Printing.Jobs[0].Status != "pending" {
		t.Fatal("spool state missing")
	}
	spool.inspect = func(context.Context, int) (printspool.Receipt, error) {
		return printspool.Receipt{ID: 41, State: "canceled"}, nil
	}
	call(t, s, "print.job.cancel", IDParams{ID: "print-1"}, "cancel-spool")
	state = awaitPrintWork(t, s, "completed")
	if !state.Jobs[0].CancelRequested || state.Jobs[0].Status != "canceled" {
		t.Fatal("cancellation not observed")
	}
	refuse(t, s, "print.job.cancel", IDParams{ID: "print-1"}, "unavailable")
	after := s.Snapshot()
	if before.Document != after.Document || before.RunID != after.RunID || !reflect.DeepEqual(before.Components, after.Components) || !reflect.DeepEqual(before.Transient, after.Transient) || !reflect.DeepEqual(before.UndoEntry, after.UndoEntry) {
		t.Fatal("printing mutated live circuit/history/dirty state")
	}
}
func TestPrintSubmissionAcknowledgmentWinsLocalCancellation(t *testing.T) {
	for _, ack := range []bool{false, true} {
		t.Run(map[bool]string{false: "unknown", true: "acknowledged"}[ack], func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			spool := &fakePrintSpool{submit: func(context.Context, string, string, string, printspool.Options) (printspool.Receipt, error) {
				close(entered)
				<-release
				if ack {
					return printspool.Receipt{ID: 50}, nil
				}
				return printspool.Receipt{}, printspool.ErrUncertain
			}}
			s := preparedPrintService(t, spool)
			defer s.Close()
			s.printRender = func(context.Context, exportCapture) ([]byte, error) { return []byte("%PDF"), nil }
			call(t, s, "print.submit", PrintSubmitParams{"captured", "lab", 1}, "submit")
			<-entered
			work := s.Snapshot().Printing.Work
			refuse(t, s, "print.printers.refresh", struct{}{}, "busy")
			promptLiveCommand(t, s, "component.select", IDParams{ID: "C1"})
			call(t, s, "print.work.cancel", IDParams{ID: work.ID}, "cancel")
			close(release)
			want := "failed"
			if ack {
				want = "completed"
			}
			state := awaitPrintWork(t, s, want)
			if ack {
				if state.Jobs[0].Status != "submitted" || state.Jobs[0].SpoolID != 50 {
					t.Fatal("acknowledged job lost during local cancel")
				}
			} else {
				if state.Jobs[0].Status != "unknown" || state.Work.Error.Code != "submission-unknown" || state.Jobs[0].PDFSHA256 == "" {
					t.Fatal("ambiguous submission mislabeled")
				}
			}
		})
	}
}
func TestPrintCancelBeforeSpoolingAndCloseJoin(t *testing.T) {
	spool := &fakePrintSpool{submit: func(context.Context, string, string, string, printspool.Options) (printspool.Receipt, error) {
		t.Error("canceled preparation submitted")
		return printspool.Receipt{}, nil
	}}
	s := preparedPrintService(t, spool)
	entered, release := make(chan struct{}), make(chan struct{})
	s.printRender = func(ctx context.Context, c exportCapture) ([]byte, error) {
		close(entered)
		<-ctx.Done()
		<-release
		return nil, ctx.Err()
	}
	call(t, s, "print.submit", PrintSubmitParams{"captured", "lab", 1}, "prepare")
	<-entered
	closed := make(chan struct{})
	go func() { s.Close(); close(closed) }()
	select {
	case <-closed:
		t.Fatal("Close did not join preparation")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close remained blocked")
	}
	if state := s.Snapshot().Printing; state.Jobs[0].Status != "canceled" || state.Jobs[0].SpoolID != 0 {
		t.Fatal("pre-spool cancellation differs")
	}
}
func TestPrintDiscoveryFailurePurityAndStaleIDs(t *testing.T) {
	var mu sync.Mutex
	fail := false
	spool := &fakePrintSpool{list: func(context.Context) ([]printspool.Destination, error) {
		mu.Lock()
		defer mu.Unlock()
		if fail {
			return nil, errors.New("offline")
		}
		return []printspool.Destination{{ID: "lab", Accepting: true}}, nil
	}}
	s := preparedPrintService(t, spool)
	defer s.Close()
	call(t, s, "print.destination.set", PrintDestinationParams{"lab", 1}, "select")
	before := s.Snapshot()
	for _, tc := range []struct {
		id   string
		p    any
		code string
	}{
		{"print.submit", PrintSubmitParams{"old", "lab", 1}, "not-ready"},
		{"print.submit", PrintSubmitParams{"captured", "missing", 1}, "unavailable"},
		{"print.submit", PrintSubmitParams{"captured", "lab", 0}, "invalid-params"},
		{"print.job.cancel", IDParams{ID: "42"}, "not-found"},
		{"print.work.cancel", IDParams{ID: "expired"}, "not-found"},
	} {
		refuse(t, s, tc.id, tc.p, tc.code)
		if !reflect.DeepEqual(before, s.Snapshot()) {
			t.Fatal("refusal changed state", tc.id)
		}
	}
	mu.Lock()
	fail = true
	mu.Unlock()
	call(t, s, "print.printers.refresh", struct{}{}, "offline")
	state := awaitPrintWork(t, s, "failed")
	if state.PrinterID != "lab" || len(state.Destinations) != 1 {
		t.Fatal("failed discovery discarded last usable inventory")
	}
}

func TestPrintRetentionProtectsUnresolvedJobsAndEvictsTerminalOnly(t *testing.T) {
	spool := &fakePrintSpool{submit: func(context.Context, string, string, string, printspool.Options) (printspool.Receipt, error) {
		return printspool.Receipt{ID: 77}, nil
	}}
	s := preparedPrintService(t, spool)
	defer s.Close()
	s.printRender = func(context.Context, exportCapture) ([]byte, error) { return []byte("%PDF"), nil }
	for i := 0; i < MaxPrintJobs; i++ {
		call(t, s, "print.submit", PrintSubmitParams{"captured", "lab", 1}, fmt.Sprintf("submit-%d", i))
		awaitPrintWork(t, s, "completed")
	}
	before := s.Snapshot()
	refuse(t, s, "print.submit", PrintSubmitParams{"captured", "lab", 1}, "record-limit")
	if !reflect.DeepEqual(before, s.Snapshot()) {
		t.Fatal("full print admission changed retained jobs")
	}
	spool.inspect = func(context.Context, int) (printspool.Receipt, error) {
		return printspool.Receipt{ID: 77, State: "completed"}, nil
	}
	call(t, s, "print.job.refresh", IDParams{ID: "print-1"}, "terminal")
	awaitPrintWork(t, s, "completed")
	call(t, s, "print.submit", PrintSubmitParams{"captured", "lab", 1}, "replacement")
	state := awaitPrintWork(t, s, "completed")
	if len(state.Jobs) != 8 || state.Jobs[0].ID != "print-2" || state.Jobs[7].ID != "print-9" {
		t.Fatal("retention removed unresolved job")
	}
}

func TestPrintRealPreviewRendersCapturedVectorPDF(t *testing.T) {
	if !schematicpdf.Supported() {
		t.Skip("macOS PDF renderer")
	}
	s := pausedRC(t)
	defer s.Close()
	var delivered []byte
	s.printBackend = &fakePrintSpool{submit: func(_ context.Context, _ string, path, _ string, _ printspool.Options) (printspool.Receipt, error) {
		var err error
		delivered, err = os.ReadFile(path)
		return printspool.Receipt{ID: 81}, err
	}}
	call(t, s, "print.preview.start", PreviewParams{}, "preview")
	preview := awaitPreview(t, s, "completed").PrintPreview
	call(t, s, "print.printers.refresh", struct{}{}, "printers")
	awaitPrintWork(t, s, "completed")
	call(t, s, "component.set", ValueParams{ID: "R1", Value: "20k"}, "source-edit")
	render := s.printRender
	s.printRender = func(ctx context.Context, capture exportCapture) ([]byte, error) {
		part, _ := capture.graph.Component("R1")
		if part.Value != 10000 || capture.artifact.RunID != preview.RunID || capture.artifact.Format != "pdf" || capture.page.RasterDPI != 0 {
			t.Error("vector print did not retain preview source")
		}
		return render(ctx, capture)
	}
	call(t, s, "print.submit", PrintSubmitParams{preview.ID, "lab", 1}, "print")
	state := awaitPrintWork(t, s, "completed")
	if !bytes.HasPrefix(delivered, []byte("%PDF-")) || len(delivered) < 1000 || state.Jobs[0].PDFBytes != len(delivered) || s.Snapshot().PrintPreview.Current {
		t.Fatal("real captured preview was not delivered as vector PDF")
	}
}
