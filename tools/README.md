# Project-local quality tools

This separate Go module pins Staticcheck and goimports. `scripts/lint.sh` builds
those tools into ignored `build/tools/` and checks the root web host plus shared
engine, transport and agent code. It does not change globally installed tools.

The versions and checksums are recorded in this directory's `go.mod`/`go.sum`.
CI prepares the declared Go toolchain and caches these dependencies. The quality
runner executes lint before the race-tested numerical suite.

The retired native display/frame probes and baseline-admission experiments are
available in the [native archive](../docs/native-archive.md).
