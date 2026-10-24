# Complete desktop race suite

On October 24, 2026, the complete desktop module passed an uncached run with
race detection and all offscreen mounted-control tests enabled:

```sh
cd desktop
go test -race -tags=offscreen -timeout=30m -count=1 -json ./...
```

**423 top-level tests /1,262 tests including subtests passed in 651.999 seconds
(10 minutes 52 seconds), with zero failures, skips or race reports.** The
previously interrupted `TestMountedVerificationCorrectionToAcceptedTime` group
passes as part of this complete run. No application or assertion fixes were
needed; the earlier run had exhausted Go's ten-minute suite deadline.

[Result metadata](result.json) identifies the source revision, command,
environment, counts and slowest tests. [The complete raw event stream](desktop-race.jsonl.gz)
is gzip-compressed without filtering and its uncompressed SHA-256 is recorded
in the metadata. Its build-output event retains the harmless duplicate
`-lobjc` linker warning; the separate stderr log is empty.

`scripts/check.sh` now uses a 30-minute runner timeout for both desktop race
runs. Its offscreen run includes the entire module rather than selecting a
subset of mounted tests by name. `sh -n` and `git diff --check` pass.

**Simulation limits are unchanged:** 512 integration attempts and 100 ms per
integration budget. The existing bound-regression test passes separately;
its output is retained in `integration-bounds.txt`. No production Go code,
numerical tolerance, test assertion, dependency, app package or user setting
was changed. This offscreen run did not take control of the desktop.
