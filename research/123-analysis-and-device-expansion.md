# Five electronics additions

Owner authorized all five on October 27, 2026, with complete user/agent access,
Prism review for each item and local commits. Existing qualified packages and
owner sessions remain preserved. No push/upload or unrelated feature work.

Implementation order:

1. Semiconductor AC sweeps: BJT, MOSFET and diode Jacobians at the solved DC
   bias, with existing captured Bode/cursor/export/project commands.
2. Captured spectrum/THD: explicit interval, sampling/window rules, harmonic
   amplitudes and THD (not THD+N); shared inspector and structured agent results.
3. Versioned transistor profiles: configurable gain, output conductance and
   junction capacitances, preserved generic v1 behavior, validated model contract
   and shared model controls; no unvalidated manufacturer identity claims.
4. Tolerance/Monte Carlo: deterministic seeded R/C/transistor variation,
   bounded independent trials, summaries and inspectable results with provenance.
5. Loudspeaker: electrical coil and declared resonance model, signed power and
   optional explicit audio preview, using the same user/agent operations.

Each item requires independent electrical references, examples, shared command
and discovery coverage, persistence/capture validation where applicable, native
handler parity and a Prism review. Findings must be resolved or documented with
concrete evidence before the corresponding local commit. Final broad race,
lint/vet/build and matching-package/native/agent checks follow the combined work.
Do not relax numerical accuracy/integration guards to make tests pass.

Prism is installed locally. The initial configured OpenAI provider doctor
responded, but automatic approval review blocked source-diff egress before the
review process started. The owner then explicitly selected Anthropic Claude
Opus 5.5. All actual feature reviews use `--provider anthropic --model
claude-opus-5-5` (the documented API ID). No code review diffs are sent to OpenAI.

Item 1: implementation and focused electrical/native-handler tests pass. The
initial Prism review (Anthropic Opus 5.5) found one medium test-goroutine issue
and three low test/documentation findings. All are addressed: assertions now
run outside UI dispatch, agent timeout is explicit, completed records are cloned
under the service lock, and machine-specific developer-path prose is removed.
An initial MOS test fixture accidentally grounded the drain; the final fixture
corrects that wiring without changing any electrical assertion or solver guard.
Three complete Anthropic reviews are recorded in
[AC evidence](../docs/validation/electronics-five/ac/README.md). Final review has
no medium/high findings. Remaining low testing observations have explicit
existing-test/electrical-reference dispositions. Focused core and desktop race
tests pass. Item 1 is complete; committing this reviewed batch.

Item 1 is committed as `ed98095f`.

Item 2: FFT/THD, captured-data inspector controls, exact-result JSON/CSV exports
and a clipping example are implemented. Independent Fourier-series/window tests,
recording export/reopen, stale result identity, unit input, user/agent parity
and ordinary/rendered focused tests pass. New discovery initially exceeded its
64 KiB catalog bound; shared input/part/operation schema references correct this
without dropping fields or relaxing the limit. Focused discovery/time/core race
tests now pass. Three complete Anthropic Prism reviews are recorded with explicit dispositions. Final source fixes actual-row defaults, omitted-parameter normalization, appearance invalidation, exact export identity/provenance, defensive grid/export checks and test assertions. Public unit-text and object-only protocol tests resolve the remaining medium observations. Focused core and desktop race tests pass. Item 2 is complete; committing the reviewed batch.

Items 3–5: planned, not yet qualified or committed.
