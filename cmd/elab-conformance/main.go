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
	Name       string               `json:"name"`
	Equivalent bool                 `json:"equivalent_to_in_process"`
	Workflows  []conformance.Result `json:"workflows"`
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
	result := report{GoVersion: runtime.Version(), Platform: runtime.GOOS + "/" + runtime.GOARCH, Limitations: []string{"engineering workflow parity on this workstation; not complete native action coverage or participant acceptance", "numerical parity uses shared pinned kernels; independent model/reference and refinement qualification remain separate", "native socket path verifies semantic operations in real app processes; keyboard/focus/screen-reader and visible latency are separate", "all sessions are newly created and owned; pre-existing applications/endpoints are untouched"}}
	var reference []conformance.Result
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
		result.Adapters = append(result.Adapters, adapterResult{mode.name, true, workflows})
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
