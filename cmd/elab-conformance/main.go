// elab-conformance compares the PRD's three reference workflows through public
// in-process, headless stdio, headless socket and optional native socket paths.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/dshills/elab/internal/conformance"
)

type adapterResult struct {
	Name       string                        `json:"name"`
	Equivalent bool                          `json:"equivalent_to_in_process"`
	Workflows  []conformance.Result          `json:"workflows"`
	Electrical *conformance.ElectricalResult `json:"electrical,omitempty"`
}
type report struct {
	Passed      bool            `json:"passed"`
	GoVersion   string          `json:"go_version"`
	Platform    string          `json:"platform"`
	Adapters    []adapterResult `json:"adapters"`
	Limitations []string        `json:"limitations"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	binary := flag.String("agent-binary", "build/elab-agent", "built external reference client/server binary")
	native := flag.String("native-binary", "", "optional native binary; creates/closes six owned temporary native sessions")
	directory := flag.String("artifacts", "", "new empty output directory (required; never replaces existing data)")
	electrical := flag.Bool("electrical-suite", false, "also run frozen independent electrical references and mixed flasher timestep refinement")
	flag.Parse()
	if *directory == "" {
		return fmt.Errorf("--artifacts must name a new directory")
	}
	dir, err := filepath.Abs(*directory)
	if err != nil {
		return err
	}
	if err = os.Mkdir(dir, 0700); err != nil {
		return fmt.Errorf("create new artifacts directory: %w", err)
	}
	agentPath, err := filepath.Abs(*binary)
	if err != nil {
		return err
	}
	modes := []struct {
		name    string
		factory conformance.Factory
	}{{"in-process", conformance.InProcess}, {"headless-stdio", processFactory(agentPath, "stdio")}, {"headless-socket", processFactory(agentPath, "socket")}}
	if *native != "" {
		path, err := filepath.Abs(*native)
		if err != nil {
			return err
		}
		modes = append(modes, struct {
			name    string
			factory conformance.Factory
		}{"native-socket", processFactory(path, "native")})
	}
	result := report{GoVersion: runtime.Version(), Platform: runtime.GOOS + "/" + runtime.GOARCH, Limitations: []string{"engineering workflow parity on this workstation; not complete native action coverage or participant acceptance", "workflow parity uses shared pinned kernels; optional electrical suite adds independent frozen educational-model references and same-kernel flasher refinement, not real-device/SPICE qualification", "native socket path verifies semantic operations in real app processes; keyboard/focus/screen-reader and visible latency are separate", "all sessions are newly created and owned; pre-existing applications/endpoints are untouched"}}
	var reference []conformance.Result
	var electricalReference *conformance.ElectricalResult
	for _, mode := range modes {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		workflows, runErr := conformance.Run(ctx, mode.factory, filepath.Join(dir, mode.name))
		cancel()
		if runErr != nil {
			return writeFailure(dir, result, fmt.Errorf("%s: %w", mode.name, runErr))
		}
		if reference == nil {
			reference = workflows
		} else if err = conformance.Compare(reference, workflows); err != nil {
			return writeFailure(dir, result, fmt.Errorf("%s: %w", mode.name, err))
		}
		adapter := adapterResult{Name: mode.name, Equivalent: true, Workflows: workflows}
		if *electrical {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			qualified, qualificationErr := conformance.RunElectrical(ctx, mode.factory, filepath.Join(dir, mode.name, "electrical"))
			cancel()
			adapter.Electrical = &qualified
			if qualificationErr == nil {
				if electricalReference == nil {
					electricalReference = &qualified
				}
				qualificationErr = conformance.CompareElectrical(*electricalReference, qualified)
			}
			if qualificationErr != nil {
				result.Adapters = append(result.Adapters, adapter)
				return writeFailure(dir, result, fmt.Errorf("%s electrical suite: %w", mode.name, qualificationErr))
			}
		}
		result.Adapters = append(result.Adapters, adapter)
		fmt.Fprintf(os.Stderr, "%s: three workflows passed\n", mode.name)
	}
	result.Passed = true
	return writeReport(dir, result)
}
func writeFailure(dir string, result report, cause error) error {
	return errors.Join(cause, writeReport(dir, result))
}
func writeReport(dir string, result report) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "report.json"), append(data, '\n'), 0600)
}
