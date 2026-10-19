# Project-local quality tools

This separate Go module pins Staticcheck and goimports; the portable application module keeps its standard-library-only dependency boundary. `scripts/lint.sh` builds tools into the ignored `build/tools` directory and checks both application modules. It does not change global installed tools.

Staticcheck 0.8.1 uses x/tools 0.50.0 here because the pilot Go 1.27.2 compiler emits export-data format 5, which the older globally installed analyzers could not read. The tool module requires Go 1.26 or newer. Versions/checksums are recorded in go.mod/go.sum. Dependencies were already cached on the pilot machine; `GOPROXY=off ./scripts/lint.sh` reproduces an offline run there. A fresh development environment needs these pinned tool dependencies prepared normally.

Run heavy analyzers before, rather than concurrently with, the simulator's race-tested numerical suite. This keeps compiler/analyzer contention from consuming the mixed engine's explicit 100 ms per-output work budget. Such an exceeded budget remains a reported solver failure, never a fabricated result.

The separate [cropped display probe](display-frame-probe/README.md) implements the owner-authorized calibration method with an explicit fixed owned-window crop. Its offline checks do not capture UI or qualify latency. The older [window probe](frame-probe/README.md) remains frozen for comparison.
