# Solver timeout under CPU load

2026-11-01, local macOS arm64 / Go 1.27.3. This fixes the shared solver used by
native and web hosts. It does not publish a new native alpha or modify earlier
release evidence.

## Cause and correction

The old 100 ms per-output budget measured elapsed wall time. Waiting for another
goroutine/process or a host scheduling pause could consume that entire budget
without meaningful numerical work. In the reproducible one-CPU flasher fixture,
eight owned CPU burners stopped a valid run after 0.23 simulated seconds: its
second attempted integration interval saw 200.091 ms of wall time.

On macOS and Linux the budget now uses `CLOCK_THREAD_CPUTIME_ID`, with the solver
goroutine pinned to its OS thread for the complete staged output sample. Source
corners, hybrid events and Newton stages share that same meter and attempt count.
The pin is released on success, numerical refusal and cancellation.

The policy is now explicit in `snapshot.limits` and detailed command discovery:

- **100 ms of worker-thread CPU per output sample.** Other workers and time spent
  waiting for CPU do not consume it. This is not a process-wide CPU measurement.
- **512 adaptive attempts**, plus the existing per-Newton iteration and all
  convergence, error estimation, conservation and model bounds.
- **5 seconds of wall time per output sample** as a separate stalled-work backstop.
  Existing operation deadlines remain; cancellation reaches adaptive/Newton and
  hybrid-event checkpoints before publication.
- **Strict 100 ms wall fallback** if a thread clock is unavailable or fails, rather
  than disabling the calculation budget. Other platforms retain this fallback.

Only a whole successful sample is published. A rejected/canceled step preserves
committed time, storage, trace and device state. There are no automatic retries
that discard real samples, relaxed numerical tolerances, or changes to device
models/integration equations.

`golang.org/x/sys` was already pinned in the module graph; it is now a direct
import for the OS clock. No dependency version was upgraded. Historical capacity
reports retain compatibility only for their exact original 100 ms wall-only
policy tuple, with all other capacity/numerical validation unchanged.

## Results

| Fixture | Original wall-only policy | CPU-accounted policy |
| --- | --- | --- |
| Doubled-capacitance flasher, idle | Completes 6 simulated seconds | Completes 6 simulated seconds |
| Same flasher, eight burners / one Go scheduler CPU | Stops at 0.23 s with `integration-time-limit` | Completes 6 s; all 600 accepted output results hash identically to idle |
| 555 piezo tone, eight burners / one scheduler CPU | Original live symptom was a false elapsed-time refusal | Completes 10,000 samples / 100 simulated ms; exact sample hash matches idle |
| Live web 555 clock, same contention | Old per-sample wall accounting was load-sensitive | Advances at least 1 simulated ms while HTTP inspection/pause work; accepted KCL/power bounds hold |
| Deliberate 200 ms scheduling pause during a sample | Exceeds the old 100 ms wall budget | RC/NPN, 555 and RLC results/trace match uninterrupted results exactly |

The flasher result hash is
`61f5254aa8943df724ce73dc06d63a5789665d858dc6ad1af119c4ab5effe2bf`.
Its settled measured period remains 0.98 s and its prior KCL/power assertions pass.
The 555 sample hash is
`a0d0b71eab6b95a4efb489b8a7f9b651771ab7f66c72d6fe9389dd598d676d16`.
Recorded 555 benchmark times were 3.182 s idle and 27.617 s with contention: the
load slows execution while preserving the same accepted samples.

[Before report](before.json) and [after report](after.json) contain the generated
flasher measurements. `passed` in the historical audit report means its
measurement/state-preservation assertions passed; its `bounded-failure` outcome
explicitly records the original solver refusal. The after run requires completion
and checks exact idle/load result hashes.

## Validation and reproduction

- Full portable `go test -race ./... -count=1 -timeout=30m`: passed; the native core
  took about 329 s. [Captured output](portable-race.txt).
- Final timing discovery, CPU/wall/clock failure, cancellation, hybrid timer and
  sequential, and live-web load regressions: passed after the final context wiring
  ([output](final-targeted-race.txt)).
- Native client's shared pulse/live-input/manual-step race tests: passed.
- `go vet ./...`, web/core/conformance staticcheck, goimports, and frontend checks:
  passed. Command inventory verification remains 229 commands.
- Darwin tests with `CGO_ENABLED=0`: passed for the real thread clock and scheduling
  pause cases. Linux amd64 `CGO_ENABLED=0` workbench tests cross-compile; no Linux
  runtime execution is inferred from that compile check.

To rerun the owned one-CPU synthetic load qualification:

```sh
ELAB_INTEGRATION_AUDIT_REQUIRE_COMPLETION=1 go test ./workbench -run '^$' \
  -bench '^(BenchmarkFlasherIntegrationBudgetAudit|Benchmark555ToneIntegrationLoad)$' \
  -cpu=1 -benchtime=1x -count=1
```

The burners are confined to their test process and joined on every outcome.
Synthetic contention and deterministic pauses establish numerical behavior for
these fixtures; they do not establish UI latency or success under arbitrary
machine suspension. A sample still refuses when its actual CPU budget, attempt
quota, wall backstop or enclosing operation deadline is exhausted.

Internal review checked thread pin ownership/unwinding, CPU clock scope/failure,
context propagation, shared budgets across event stages, atomic publication,
discovery accuracy and preservation of frozen report validation. No external
review service was used.
