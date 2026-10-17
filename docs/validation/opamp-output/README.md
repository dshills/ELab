# Amplifier output qualification evidence

See the [receipt](../opamp-output-validation.json) and [six-step native review](../../../research/ui-audit/opamp-output/README.md). This is an unreleased source/candidate checkpoint. No physical print job or canonical app replacement is part of it.

The normal candidate uses the SDK verified by `sdk-final.json`. `source-index-final.json` freezes 630 Go/Python/module/script/lock inputs used during the main qualification. The only subsequent source change wraps the new native test in a cleanup-safe subtest; `test-cleanup-source-delta.json` records it, `source-index-post-cleanup.json` records the final hashes, and separate native race/lint/vet logs cover it. Production source/binary inputs did not change. The reviewed native ledger changes only the drawing source hash, with exact equality of every site and command set.

`conformance-report.json.gz` is the complete original JSON. `review.go.txt` reruns the typed current-source comparators through every adapter; `audit.py.txt` separately checks prior numerical evidence while permitting only the enumerated coordinate changes and variable `operation.inspect` polling. It also compares actual native/headless output hashes and PDF content. The original PDFs retain their differing timestamps and document IDs. No other PDF bytes are normalized for comparison.

`outputs.tar.gz` contains 43 regular files: all ten headless SVG/PDF captures and thirteen PNG pages, plus the ten original native PDFs. Duplicate preview PNGs and byte-identical native SVG/PNG copies are represented by their independently checked original hashes in both public reports. `file-fixtures.tar.gz` retains 296 generated JSON/project files from the public electronics run, excluding endpoint descriptors and recovery directories. Each archive has a complete SHA-256 index; every regular member was reopened and checked after creation. No archive extraction was needed for qualification.

The initial failed regression log uses the three retained `*.before.txt` source files through a Go overlay. Earlier quality logs preserve two preparation issues: a regression-source directory visible to `go ./...`, and an unreviewed native-source hash. The directory was moved under a dot-prefixed name and the exact source/site delta was reviewed. No test, layout reference or ingress gate was disabled. The first output harness also compared implicit preview metadata against explicit `page_index:0`; pixels matched, and using matching options restored exact PNG bytes.

To repeat the public output checks, build the current source and use new paths:

```sh
GOPROXY=off GOFLAGS=-p=1 ./scripts/build.sh
GOPROXY=off GOFLAGS=-p=1 ./scripts/check.sh
python3 scripts/opamp-output-demo.py \
  --artifacts build/new-amplifier-output \
  --output build/new-amplifier-output.json
./build/elab-conformance --opamp-suite --opamp-storage-suite \
  --inductor-suite --electrical-suite --native-binary build/elab \
  --artifacts build/new-amplifier-conformance
build/elab --smoke --smoke-timeout 2m
```

The output demo's `--connect` mode requires a **fresh, explicitly owned disposable native host**. It checks admission before destructive setup or cleanup and closes that host afterward. General user/agent editing remains unrestricted by demo admission. The CUA inspection fixture's launcher source is retained separately; it execs the exact candidate host, without behavior overlays, with recovery and saved appearance disabled. Saved window geometry remains enabled, so its screenshot size is not default-window evidence.

For typed report review, copy `review.go.txt` to `main.go` inside a hidden directory beneath this repository's `build/`, then `go run` that directory with the current report and prior storage report as its two arguments. The Python audit requires `pypdf` from the bundled runtime and accepts an optional output root (default `build/opamp-app`). It expects the report and capture subdirectory names used by this run. Independent PDF visual review uses `pdftoppm -r 120 -png`; the retained contact sheets and reference pages are in the research review's `output/` folder.

The candidate is private, unsigned and unnotarized. Native text readability, tooltip behavior, first-click behavior, external AX coverage, physical printer/device output, visible latency and maximum capacity remain explicit follow-ups. Historical backend timing evidence is not reclassified as current visible-latency or launch evidence.
