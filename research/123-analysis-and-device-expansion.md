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

Item 2 is committed as `cd7ff887`.

Item 3: versioned educational NPN/PNP profiles, Early effect and explicit junction capacitance are implemented with shared undoable controls, saved/captured parameters, a working follower example and independent DC/Jacobian/AC references. Biased startup avoids imposing redundant junction voltage constraints; existing four-storage and total-part bounds remain unchanged. Three Anthropic Prism reviews and findings dispositions are retained. Actual inspector mounting, stale callbacks, core race and offscreen-build handler/theme tests pass. Early fixture/compiler and unrecognized-tag trials are excluded from successful evidence. Manufacturer SPICE-card fidelity is deliberately not claimed. Committing this reviewed batch.

Item 3 is committed as `0377a10b`.

Item 4: independent seeded tolerance analysis is implemented for R/C and explicit profile beta, in AC-gain or DC mode. Every trial exposes actual parameter draws and outcome, with failed counts and inspectable errors. Captured model-bearing source/hash, exact-result exports, cancellation, shared controls and an RC example are covered. Independent RC transfer/statistics, reproducibility, zero values, failure retention, stale callback, exact-seed and ordinary/offscreen-build tests pass. Three complete Anthropic Prism reviews are retained; final report has no medium/high findings, with all remaining observations repaired or concretely dispositioned. Catalog schema pooling preserves the 64 KiB bound. Initial fixture IDs, pointer-identity assertion, RC-vs-mixed helper assumption and expected-refusal runner mistakes are excluded from passing evidence. Committing item 4.

Item 4 is committed as `9cbad84d`.

Item 5: passive electrical loudspeaker equivalent, saved parameters, amplifier example, shared configuration and explicit normalized preview are implemented. Independent complex impedance and transient ODE references, strict energy/current balance, capture/project validation, source-guarded playback/exports, codec corruption, all-theme meshes, field minima and user/agent parity pass. The new fixture reproduced an existing op-amp startup-cut grouping defect; its true input and output/supply current groups are now represented separately, without changing numerical limits. Scope energy now includes profile junction energy exactly once. Three complete Anthropic Prism reviews and detailed dispositions are retained. Remaining medium observations are resolved by referenced-schema, validated-lowering and private/public-energy proofs; no demonstrated medium/high defect remains.

The original eight-BJT, 1 nF compensation plus speaker LC combination reaches the unchanged 100 ms work guard during startup. A 10 nF control also did. This is explicitly unqualified for waveform progress; its regression verifies bounded refusal and unchanged accepted state. The powered-follower speaker example and independent passive references are the passing waveform evidence. No limit or original amplifier example was changed to hide this.

Combined checks: the initial full portable gate caught missing spectrum contract version/scope and the expected native-inventory change; both are repaired. One existing RL conformance run reached its 100 ms budget; three isolated pre-optimization controls pass, so no electrical/causal repair is claimed. Default tolerance settings now avoid rebuilding the probe inventory twice per live snapshot. A ±12 V global capacitor-setting profile control passes without a production change. Final full gates are running against frozen source/test inputs. Qualified SDK/package bytes remain protected.
