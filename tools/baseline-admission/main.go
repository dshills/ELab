package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("baseline-admission", flag.ContinueOnError)
	flags.SetOutput(stderr)
	profilePath := flags.String("profile", "", "declared baseline-region JSON profile")
	imagePath := flags.String("image", "", "existing recorder PNG; no capture is performed")
	imageSHA := flags.String("sha256", "", "PNG SHA-256 from the recorder frame journal")
	write := func(result admission, status int) int {
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return status
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return write(admission{Version: 1, Code: "invalid-arguments"}, 1)
	}
	if flags.NArg() != 0 || *profilePath == "" || *imagePath == "" || *imageSHA == "" {
		return write(admission{Version: 1, Code: "invalid-arguments", Message: "require --profile, --image and --sha256"}, 1)
	}
	p, err := readBounded(*profilePath, maxProfileBytes)
	if err != nil {
		return write(admission{Version: 1, Code: "invalid-profile-file", Message: err.Error()}, 1)
	}
	img, err := readBounded(*imagePath, maxPNGBytes)
	if err != nil {
		return write(admission{Version: 1, Code: "invalid-image-file", Message: err.Error()}, 1)
	}
	result := inspectBaseline(p, img, *imageSHA)
	if result.Admitted {
		return write(result, 0)
	}
	return write(result, 2)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
