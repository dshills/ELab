// elab-ui-audit records native source routes and verifies the checked-in ledger.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/dshills/elab/internal/uiaudit"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	directory := flag.String("desktop", "desktop", "native source directory")
	output := flag.String("output", "", "write a new inventory file; never overwrite")
	verify := flag.String("verify", "", "compare the current inventory with an existing file")
	flag.Parse()
	if (*output == "") == (*verify == "") {
		return fmt.Errorf("choose exactly one of --output or --verify")
	}
	report, err := uiaudit.Audit(*directory)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if *verify != "" {
		previous, err := os.ReadFile(*verify)
		if err != nil {
			return err
		}
		if !bytes.Equal(previous, data) {
			return fmt.Errorf("native inventory changed; review source/routes and generate a new candidate with --output before updating the ledger")
		}
	} else {
		file, err := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if len(report.Issues) > 0 {
		return fmt.Errorf("native inventory has unknown routes: %v", report.Issues)
	}
	encoded, _ := json.Marshal(map[string]any{"passed": true, "sources": len(report.Sources), "sites": len(report.Sites), "source_route_count": len(report.LiteralCoverage), "public_commands": len(report.PublicCommands), "scope": "source inventory; dynamic binding and runtime coverage remain separate"})
	fmt.Println(string(encoded))
	return nil
}
