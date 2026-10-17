# Op-amp performance qualification evidence

The [parent receipt](../opamp-performance-validation.json) binds the hardware, source freeze, binaries and limitations. `quality.json` / `build.log.txt` / `check.log.txt` retain ordinary build and formatting/imports/Staticcheck/race/vet/Python checks. `measurement-process.json` / `measurement.log.txt` retain the sequential four-adapter run after those checks finished. Timings exclude process launch and circuit construction; they do not measure visible latency or maximum capacity.

`conformance-report.json.gz` is gzip of the exact original JSON report, including all 640 timing samples, eight circuit variants per adapter, settings/models/probes, numerical captures, calls and 931 reference checks per adapter. The receipt pins both compressed bytes and original decoded SHA-256. Decompression was compared byte-for-byte, and the same typed review produced identical results for raw and compressed input.

`review.go.txt` is the executed audit source. It recompiles each recorded graph, recomputes its canonical Go JSON SHA-256, checks that the only timed changes are the declared resistors, checks sample model/probe/engine associations and warm identity advances, and recomputes every percentile and target. It also reruns the numerical conformance comparison from the retained rows. `sample-audit.json` contains its result. To reproduce from the repository root with a new owned review directory:

```sh
mkdir build/.opamp-report-review
cp docs/validation/opamp-performance/review.go.txt build/.opamp-report-review/main.go
GOWORK=off go run ./build/.opamp-report-review \
  docs/validation/opamp-performance/conformance-report.json.gz
```

`review-negative-checks.py.txt` documents the executed eight corruption cases; `reviewer-negative-checks.json` records each expected refusal. The test driver uses the original build-local raw report and compiled reviewer paths. `invalid-repetitions.json` separately confirms the actual CLI rejects 19 repetitions before creating an artifacts directory. Historical build-local paths are retained faithfully; choose new output paths for a fresh measurement run.

The native socket sessions were newly created and closed through the public lifecycle. The canonical 0.105 app predates op-amps and remains unchanged. Full physical input/accessibility, physical device behavior, distribution and application promotion remain separate gates.
