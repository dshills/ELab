# Continued MVP implementation and code quality

**Started October 13, 2026.** The product owner asked for continued backlog work until they return and explicitly stop it, with attention to reviews, lint, structure and Go practices. This record tracks completed and active slices; it does not declare the entire MVP complete.

## Quality checkpoint and review

The previously uncommitted implementation/research was checkpointed locally on `codex/mvp-continuation` (`8ef5cf4`). No changes were pushed. Subsequent green slices receive separate local commits so their scope is reviewable and reversible.

The initial source review found and repaired:

- **File picker/project replacement race.** Save/Open/export choices could apply to a project an agent loaded while the picker was open. Shared file commands now accept `expected_document_generation`; native pickers capture/preserve it, including pending unsaved-change decisions. A mismatch refuses before reading, writing or publishing. Tests replace a project while the native picker adapter waits.
- **Cursor endpoint precision.** Fixed-window cursors could be called unavailable when decimal endpoints differed only by floating-point timestamp roundoff. Cursor validity now uses the same inclusive time predicate as plots, statistics and CSV.
- **Duplication at bounds.** The default copy position could fail at the positive corner despite nearby free space. Bounded search now tries eight directions while preserving the usual first placement and all existing validation.
- **Lint findings.** Removed an unused hash helper and dead axis initialization; replaced deprecated title-casing of known transistor states with explicit labels; normalized imports and checked previously ignored simulation/CSV return values.

## Repeatable quality gate

`scripts/check.sh` runs project-local Staticcheck and import formatting, then root/desktop race tests and Go vet. `scripts/lint.sh` is separately callable. The nested `tools` module pins Staticcheck 0.8.1, x/tools 0.50.0 and checksums; runtime core dependencies remain standard-library-only. Tools build into ignored `build/tools`, leaving global installations unchanged.

The older installed analyzers could not read the Go 1.27.2 export-data format. Compatible tools were built from existing cached modules and validated offline. No lint checks were suppressed to hide that incompatibility. A numerical race test hit the mixed engine's existing 100 ms guard while a heavy analyzer build ran concurrently; the serial quality gate passes with the guard unchanged. Heavy analysis/build tasks are kept separate from numerical acceptance tests.

## Verify, operation lifecycle and recorded exports

The next slice implements [bounded Verify](../docs/verification.md) with captured graph/startup/settings, independent kernels, operation IDs/progress/cancellation, retained immutable results and paged SI samples. Native settings/readouts and external agents use the same commands. JSON/CSV exports retain the captured source after editor changes. Live physical state, graph, undo and dirty state are preserved. Limits and failure semantics are explicit; partial failed/canceled computations are never published as recorded results.

Discovery uses a complete local schema registry (`schema_version:2`) rather than repeating the full Snapshot definition per command. This preserves the existing 1 MiB response bound and keeps operation discovery scalable.

Completed queue items: bounded Verify, operation lifecycle, and portable recorded exports. Dedicated record-file import/replay remains a follow-on item. Validation: [headless report](../docs/validation/headless-verification.json), [desktop report](../docs/validation/desktop-verification.json), core/native parity regressions and native GPU/UI smoke.

Packaged slice: **0.10.0 / 24**, with [final validation](../docs/validation/verification-validation.json). The prior application bundle was preserved.

## Autosave and crash recovery

Implemented [autosave/recovery](../docs/recovery.md) through seven shared commands and native Inspector controls. Native startup defaults to two-second private snapshots; headless sessions opt in explicitly. Snapshots contain complete validated portable schematics, with immutable publication, checksums, per-session ownership and two-copy retention per document. Saved projects are preserved. Restoring opens a dirty unassociated copy for Save As; dirty/generation/hash guards and the existing native unsaved-change prompt apply.

Review fixes include serializing final shutdown with pending writes so explicit discard cannot be undone by a late autosave, rejecting old captures after an explicit save, retaining a valid older snapshot if the newest is damaged, preserving other live sessions' snapshots (including damaged bodies with identifiable ownership), bounding scans/diagnostics, preserving an explicitly cleared candidate selection, and returning through CLI deferred cleanup before choosing a nonzero process exit. Filesystem work for automatic capture occurs outside the editor mutex. Shared document serialization/installation was extracted from persistence rather than duplicated.

Validation includes core/desktop race tests, Staticcheck/import formatting, Go vet, native/agent control parity and actual GPU/UI smoke. An owned headless process was killed with SIGKILL and reopened: [crash recovery report](../docs/validation/crash-recovery.json) shows successful automatic capture, validated copy restoration, Save As and byte-for-byte preservation of the original `.elab` file. This is process-crash validation, not power-loss qualification.

Packaged slice: **0.11.0 / 25**, with [final validation](../docs/validation/recovery-validation.json) and [desktop endpoint validation](../docs/validation/desktop-recovery.json). Recovery copies, leases, build artifacts and running older application sessions remain local; no user files or earlier bundles were removed. Recorded results/history are not recovery payloads; Windows liveness and public distribution remain unqualified.

## Recorded-result import and replay

Implemented [native record import and replay](../docs/recording.md), with strict whole-file validation, immutable original metadata/rows, local provenance and IDs, exact edge/index/time inspection, independent playback, probe selection, visible intervals, sample statistics, timing cursors and captured source inspection. Users and agents use the same commands; the live schematic/state/history/save point remain intact.

Recorded plots preserve ordered bucket extrema and label reduction. Full data remains available for cursors, statistics, paged inspection and exports. Imported revision counters never claim association with a live source. Import retention preserves an active Verify job. Review repaired typed Go requests collapsing explicit empty lists into omission, rejected null probe lists, preserved valid DC metadata-only sample overrides, grouped private playback state and separated plotting from command/timer code. Queued ticks cannot rewind playback after a new position/rate. Import checks every interior output step as well as endpoints, and extremely close cursor timestamps cannot publish an infinite frequency. Native GPU smoke found and repaired a view-switch mount error by retaining the scope panel/canvas identity across live and recorded modes.

Packaged slice: **0.12.0 / 26**, with [final validation](../docs/validation/recording-validation.json), [headless workflow](../docs/validation/headless-recording.json) and [desktop workflow](../docs/validation/desktop-recording.json). This slice reopens standalone recorded JSON. Project embedding was the immediate next slice and is now implemented below.

## Embedded project recordings and lifecycle

Implemented [project format 4](../docs/project-format.md): kept completed recordings, complete captured metadata/rows, canonical dependency hashes, selected recorded view and saved pause preference travel with the project. Versions 1–3 remain readable. The complete project and every embedded recording validate before replacement. Live reopening starts at zero and respects pause, without resuming recorded physics.

Completed Verify/import results become unsaved project assets; shared Include/Exclude controls leave excluded session replay/export available. Kept recordings are protected from eviction, capacity refuses explicitly, and restored membership restores a clean save point. Saving refuses pending verification. Project replacement cancels old workers and blocks late publication. Recovery includes kept records and exposes bounded metadata/sample queries. Native and agent save/open/inclusion paths share the same dispatcher.

Packaged slice: **0.13.0 / 27**, with [final validation](../docs/validation/project-recordings-validation.json), [headless portability](../docs/validation/headless-project-recordings.json), [desktop portability](../docs/validation/desktop-project-recordings.json) and [record-only crash recovery](../docs/validation/record-only-crash-recovery.json). Participant usability, hardware/power-loss validation, full device-library breadth, accessibility and distribution qualification remain future gates.

## Explicit schematic primitives

Implemented [junction/net-label workflows](../docs/schematic-primitives.md): single-node tray objects, atomic named-route junction insertion, pointer crossing review plus X/Y explicit joins, case-sensitive remote labels, stable physical net identities/order, primitive voltage probes, editing/undo/batches, SVG and portable project/record validation. Helpers never hide open circuit pins or establish a ground reference by name alone, and are omitted from physical device stamping.

Validation includes all supported circuit families, coupled transient numerical equivalence, named supply values, crossing/near-miss and orphan/reference cases, persistence/record hashes, native/agent parity and native drawing smoke. Packaged slice: **0.14.0 / 28**, with [final validation](../docs/validation/primitives-validation.json), [headless workflow](../docs/validation/headless-primitives.json) and [desktop workflow](../docs/validation/desktop-primitives.json). This advances EDIT-02 and the P0 primitive requirements without claiming broader physical topology support.

## SPST switches and bounded passive networks

Implemented [SPST controls and network simulation](../docs/switches.md): open/closed placement state, shared `switch.set`, finite on/off resistance, device probes/readouts, native symbols, SVG, project/model validation and embedded Verify recordings. Direct fresh transient actuation stages an algebraic solve before committing and preserves capacitor voltage, time, run identity and the live clock. No-op operations preserve physics. Ordinary schematic batch/history edits retain restart semantics.

The existing bounded MNA kernel now accepts switched LED/RC/NPN circuits and passive resistor networks such as voltage dividers. Topology compilation was separated from numerical integration. Bounds and ideal-constraint restrictions remain explicit. New-network integration uses tighter local capacitor tolerances; existing mixed RC/NPN tolerances are unchanged. Review also corrected obsolete README statements about already implemented features.

Packaged slice: **0.15.0 / 29**, with [final validation](../docs/validation/switch-validation.json), [headless workflow](../docs/validation/headless-switch.json) and [packaged desktop workflow](../docs/validation/desktop-switch.json). The switched RC reference differs by about 1.4 mV at 20 ms; headless and native numerical outputs match exactly. Root/desktop race tests, lint, vet, native control parity and packaged GPU/UI smoke pass. Prior bundles and user sessions were preserved.

## Generic silicon diode and rectification

Implemented the [generic silicon diode](../docs/diodes.md) with named anode/cathode pins, fixed inspectable versioned Shockley/series-resistance parameters, signed reverse leakage, analytic conductance, native/SVG symbols, readouts, probes and complete project/record/recovery metadata. The model combines with the existing bounded network devices and supports up to eight diodes. The red LED's existing model is unchanged.

Independent inverse-Shockley and RK4 fixtures cover DC and nonlinear capacitor charging. Tests also check reverse hold/leakage, derivative agreement, monotonicity, conservation, internal finite series resistance, strict model imports and captured CSV values after live edits. Review removed unnecessary division by a valueless nonlinear component, made small branch readings visible with significant digits, and added switch/diode model attribution to recorded CSVs.

Captured metadata schemas now share a local registry definition, keeping the original catalog size bound instead of increasing it. Registry and command references are checked for resolution; protocol and discovery versions remain compatible.

Packaged slice: **0.16.0 / 30**, with [final validation](../docs/validation/diode-validation.json), [headless fixture](../docs/validation/headless-diode.json) and [packaged desktop fixture](../docs/validation/desktop-diode.json). Headless/native numerical outputs match exactly; nonlinear RC charging differs from the independent reference by under 1 mV at 10 ms. Automatic pulse rectification, source edge events, independent Verify, standalone record import, two embedded recordings and project reopening pass, alongside root/desktop race tests, lint, vet and packaged GPU/UI smoke. Prior bundles/sessions remain preserved.

## Linear potentiometer and live wiper controls

Implemented the [three-pin linear potentiometer](../docs/potentiometers.md): editable total resistance, bounded slider/exact percentage field, shared `potentiometer.wiper.set`, finite endpoint resistance, unloaded divider and rheostat wiring, pin voltage/current/power readings, probes, native/SVG symbols and complete model/record/project ownership. Direct fresh transient wiper updates preserve charge/time/run and the clock; no-op updates preserve physics.

Switch and wiper updates now share one staged network-edit helper. Normal paused schematic edits retain stale results until restart/resume; batches create one named history entry. Review added useful primary readings for pot-only/diode-only networks. Independent divider/Thevenin fixtures cover loaded outputs and RC charging before/after movement, with conservation, endpoint totals, unused/tied pins, strict imports, snapshot/duplicate independence, batch undo and running-clock checks.

Packaged slice: **0.17.0 / 31**, with [final validation](../docs/validation/potentiometer-validation.json), [headless fixture](../docs/validation/headless-potentiometer.json) and [packaged desktop fixture](../docs/validation/desktop-potentiometer.json). Headless/native numbers match exactly. Divider midpoint is 2.5 V unloaded and 2 V with a 10 kΩ load; RC references differ by under 0.7 mV. Native controls/agent parity, standalone JSON/CSV, two embedded records, saved wiper reopening, lint/vet/race tests and packaged GPU/UI smoke pass.

One full race run hit the existing flasher's 100 ms integration guard during heavy host indexing activity. The isolated flasher test and subsequent full serial gate pass; the guard was not changed. Prior bundles/sessions remain preserved.

## Per-source sine and finite pulse waveforms

Implemented [sine/finite pulse sources](../docs/waveforms.md): distinct ready-to-simulate parts, amplitude/offset/frequency/phase or level/phase/ramp controls, shared typed commands, per-component timing, exact source-corner retention, native/SVG symbols, probes and complete model/project/recovery/record ownership. Legacy ideal square sources and examples retain their identities and behavior.

The new waveform engine stamps smooth sources at endpoints, bounds internal steps by waveform features, and uses explicitly documented local capacitor tolerances. One work budget now spans the complete output sample and all source corners; failures cannot publish partial time/charge/history. Review repaired near-zero residual timesteps at floating-point corner boundaries and avoided artificial work for constant waveforms. Shared sampling-fit guidance reduces output step and, when necessary, playback speed. Legacy global phase/sample coupling applies only to graphs using that legacy source.

Independent exact sine/linear-ramp RC references, NPN held-phase comparison to the DC engine, independent source periods, conservation, negative levels, parameter cloning, custom sine persistence, strict imports and whole-sample refusal atomicity pass. Native smoke initially found a fixture attempting Step after loading a running second example; the fixture now explicitly pauses, preserving the correct disabled-control behavior.

Packaged slice: **0.18.0 / 32**, with [final validation](../docs/validation/waveforms-validation.json), [headless workflow](../docs/validation/headless-waveforms.json) and [packaged desktop workflow](../docs/validation/desktop-waveforms.json). Headless/native outputs match exactly; sine RC differs by under 1.5 mV at 30 ms and finite-ramp RC by under 0.4 mV at 5 ms. Native controls, sampling fit, JSON/CSV, two embedded recordings and source settings reopening pass, alongside lint, vet, race tests and packaged GPU/UI smoke. Prior bundles/sessions are preserved.

All curated component families now have implementations, but this does not establish full MVP acceptance. Remaining product, interaction, accessibility, capacity and distribution requirements need a current-state audit and their own evidence.

## PRD conformance audit and immediate repairs

Saved the [complete requirement/state/quality audit](11-prd-implementation-audit.md) and its [machine-readable matrix](11-prd-implementation-audit.json). The coverage checker verifies all 27 P0 functional IDs: 9 implemented within their tested envelope and 18 partial/unverified. It preserves original targets and does not infer clean-installation, participant, screen-reader or visible-latency evidence. Physical capacity, nonblocking general work, state/gesture contracts, pinned readings/presentation, guidance and complete agent conformance remain open.

Implemented [audit repairs](../docs/prd-hardening.md): multiword hobbyist aliases, fresh inspectable library assumptions, selected model identity/limits, negative finite-pulse duplication with independent parameters/value, unavailable deleted probes, explicit single-probe removal/reassignment and undo restoration. Default scope associations also survive incomplete wire edits. Live reciprocal-frequency overflow no longer poisons JSON snapshots; the exact RC update retains representable tiny voltage changes using `expm1`. Library construction now uses keyed fields as metadata grows.

Packaged slice: **0.19.0 / 33**, with [final validation](../docs/validation/hardening-validation.json), [headless workflow](../docs/validation/headless-hardening.json) and [packaged desktop workflow](../docs/validation/desktop-hardening.json). Root/native regressions, lint, vet and packaged GPU/UI smoke pass. One existing flasher race test hit its wall-time work guard during concurrent work; the serial rerun passes with the guard/model unchanged. The repeated desktop fixture ends in a clean paused RC project at a durable repository-local path. Earlier bundles, sessions and user files are preserved.

## Next work queue

1. Pending/next-step switch actuation, slider gesture evidence/coalescing and actor-aware history.
2. Physical circuit capacity and nonblocking difficult-work architecture, with measured latency fixtures.
3. Voltage/current presentation preferences, pinned readings, trace/diagnostic highlights and concise example guidance.
4. Complete agent conformance, keyboard/screen-reader scenarios and reproducible clean packaging evidence.
5. Formative-test preparation; actual participant/hardware evidence remains a separate gate.

Every shipped user operation continues to require the same public agent command/state path. Each slice adds meaningful regression checks and records its actual limits. The full [audit](11-prd-implementation-audit.md) is authoritative for remaining acceptance work; a green checkpoint is not MVP completion.
