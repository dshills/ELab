package main

import (
	"fmt"

	"github.com/dshills/elab/workbench"
	"github.com/dshills/vellum/ui"
)

func (s *screen) printingControls(state workbench.Snapshot) []ui.Node {
	nodes := []ui.Node{section("printer-title", "Print captured pages")}
	printing := state.Printing
	busy := printing != nil && printing.Work != nil && (printing.Work.Status == "queued" || printing.Work.Status == "running" || printing.Work.Status == "canceling")
	refresh := s.action("printers-refresh", "Refresh printers", "print.printers.refresh", struct{}{})
	refresh.Disabled = busy
	nodes = append(nodes, refresh)
	if printing == nil {
		return append(nodes, quiet("printer-discovery-help", "Find printers configured in macOS, then choose a destination."))
	}
	options := []ui.Option{{ID: "", Label: "Choose printer…"}}
	accepting := false
	for _, d := range printing.Destinations {
		accepting = accepting || d.ID == printing.PrinterID && d.Accepting
		name := d.Name
		if d.Default {
			name += " · default"
		}
		if !d.Accepting {
			name += " · not accepting jobs"
		}
		options = append(options, ui.Option{ID: d.ID, Label: name})
	}
	nodes = append(nodes, ui.Node{ID: "printer-destination", Kind: ui.Selector, Options: options, SelectedID: printing.PrinterID, Disabled: busy, Accessibility: ui.Accessibility{Name: "Print destination"}, Change: func(e ui.ChangeEvent) {
		s.invoke("print.destination.set", workbench.PrintDestinationParams{PrinterID: e.SelectedID, Copies: printing.Copies})
	}}, quiet("printer-copies-label", "Copies"), ui.Node{ID: "printer-copies", Kind: ui.NumberField, Number: float32(printing.Copies), Numeric: ui.Numeric{Min: 1, Max: 10, Step: 1}, Disabled: busy, Accessibility: ui.Accessibility{Name: "Printed copies"}, Change: func(e ui.ChangeEvent) {
		s.invoke("print.destination.set", map[string]any{"printer_id": printing.PrinterID, "copies": e.Number})
	}})
	if len(printing.Destinations) == 0 {
		nodes = append(nodes, quiet("printer-empty", "No printers listed. Add a printer in macOS Settings, then refresh."))
	}
	p := state.PrintPreview
	if p != nil {
		label := "Print captured pages"
		if p.PageCount > 0 {
			label = fmt.Sprintf("Print all %d pages", p.PageCount)
		}
		submit := s.action("printer-submit", label, "print.submit", workbench.PrintSubmitParams{PreviewID: p.ID, PrinterID: printing.PrinterID, Copies: printing.Copies})
		submit.Disabled = busy || p.Status != "completed" || !p.Visible || !accepting
		nodes = append(nodes, submit)
		if !p.Current {
			nodes = append(nodes, quiet("printer-old-source", "Print uses all pages of the captured circuit. Refresh preview to use your edits."))
		}
	}
	if printing.Work != nil {
		w := printing.Work
		nodes = append(nodes, quiet("printer-work-status", w.Status))
		if busy {
			cancel := s.action("printer-work-cancel", "Cancel preparation", "print.work.cancel", workbench.IDParams{ID: w.ID})
			cancel.Disabled = w.Status == "canceling"
			nodes = append(nodes, cancel)
		}
		if w.Error != nil {
			nodes = append(nodes, quiet("printer-work-error", w.Error.Message))
		}
	}
	for _, job := range printing.Jobs {
		nodes = append(nodes, quiet("printer-job-"+job.ID, fmt.Sprintf("%s · %s · %d copies\n%s", job.ID, job.Status, job.Copies, job.PrinterID)))
		if job.SpoolID > 0 {
			refresh := s.action("printer-refresh-"+job.ID, "Refresh job", "print.job.refresh", workbench.IDParams{ID: job.ID})
			refresh.Disabled = busy
			cancel := s.action("printer-cancel-"+job.ID, "Cancel print job", "print.job.cancel", workbench.IDParams{ID: job.ID})
			cancel.Disabled = busy || job.Status == "completed" || job.Status == "canceled" || job.Status == "aborted"
			nodes = append(nodes, refresh, cancel)
		}
		if job.Status == "unknown" {
			nodes = append(nodes, quiet("printer-unknown-"+job.ID, "Submission may have succeeded. Check the macOS print queue before printing again."))
		}
	}
	return append(nodes, quiet("printer-job-help", "Submitted means accepted by the spooler. Job completion and cancellation cannot guarantee what has already reached paper."))
}
