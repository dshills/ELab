# PRD implementation audit

Audited October 13, 2026 against [PRD draft 0.3](07-product-requirements.md), source at local checkpoint `90927b3`, and the hardening changes following it. This is an engineering conformance audit, not a participant or screen-reader study. Passing existing demonstrations does not establish MVP completion. Proposed targets and original scope are preserved.

**Status terms:** Implemented means the stated behavior has direct source/test evidence within the documented envelope. Partial means some acceptance conditions remain unmet or unverified. Pending means evidence is absent or a release gate has not been performed. No row below declares an unperformed human, clean-installation, platform or visible-latency test passed.

## Functional requirement matrix

| ID | Current evidence | Status and remaining work |
|---|---|---|
| UI-01 | Independent root/native modules, Vellum baseline, shared dispatcher; build and packaged GPU smoke | Partial: current workstation builds; clean SDK/dependency bootstrap and supported distribution remain unqualified |
| EDIT-01 | Native `smokeConstruct`, drawing regressions, headless/desktop drawing fixtures | Implemented visible blank LED construction; novice discoverability still requires Gate D |
| EDIT-02 | Named terminals, explicit junction tests, crossings/near misses, geometry-only moves, pointer-generation guards | Implemented within one sheet and existing bounds |
| EDIT-03 | Resistance/capacitance parser, invalid-value preservation, named batches/history; potentiometer slider | Partial: no common text time-value parser; native slider control events prove one commit/undo per drag; mounted-driver invalidation/focus conformance remains open |
| EDIT-04 | Object selector, named terminal controls, pointer canvas, arrows/pan/zoom/fit/escape | Partial: keyboard net selection and complete focus/navigation scenarios need implementation/evidence |
| SIM-01 | Automatic restart after valid electrical edits; asynchronous specialized LED/NPN solves, live kernels | Partial: bounded general MNA/step work still occurs under the service lock; responsiveness to difficult solves is unmeasured |
| SIM-02 | Fresh/stale flags, electrical/run identities, pause/edit/restart regressions, input-event traces | Partial: switches now queue until the next accepted step, with paused pending/coalescing/atomicity evidence and separate actuation provenance. Wiper events remain immediate; complete mounted gesture invalidation and state-table qualification remain open |
| SIM-03 | Text status, paused/stale readouts, simulated time/playback controls, explicit failures | Implemented behavior in tested families; interpretation with people remains pending |
| SIM-04 | Independent electrical fixtures, exact RC/DC kernels, adaptive kernels with explicit work limits; sampling warnings/fit | Partial: release reference grid and timestep-refinement evidence must be consolidated across all curated families, including fast transitions |
| SIM-05 | Independent bounded Verify, captured models/settings, operations, kept results, native controls | Implemented within declared duration/step/storage envelope |
| PART-01 | Versioned curated models/pins and fixtures; per-part assumptions/IDs now visible; hobbyist aliases added | Implemented generic/ideal inventory; no physical parts/ratings claim |
| VIEW-01 | Selected component readouts; eight persisted live voltage/current cards with signed units/reference lines, stale/unavailable states, shared commands, guarded undoable positioning and GPU evidence | Partial: pinned behavior is implemented; consistent direct selected pin/net inspection and complete keyboard selection remain open |
| VIEW-02 | Illustrative LED brightness, text readings | Partial: wire color is currently static; no animation/reduced-motion preferences or current-direction animation |
| SCOPE-01 | Six labeled traces and component plot actions; shared source focus with object/terminal/net IDs, missing-source/undo and fresh native GPU evidence | Implemented for selected live sources; recorded view suppresses current-schematic highlights; interactive captured-schematic highlighting is future work |
| SCOPE-02 | Retained/fixed intervals, exact sample cursors, deltas/extrema, CSV; finite frequency guards | Implemented sampled measurement behavior; general text time entry remains in EDIT-03 |
| HELP-01 | Structured missing/open/model/unsupported/solve diagnostics and quiet error text; retained editable graph | Partial: known diagnostic objects are not consistently highlighted on the schematic; separate technical-detail disclosure needs review |
| HELP-02 | LED/RC/NPN/Blink/Flasher templates, reset, visible model limitations | Partial: each of the three core examples still needs its concise question and expected qualitative behavior, with reset tracked to the current example |
| FILE-01 | Format 4 full models/records, older versions readable, strict unsupported-format refusal, fresh-process offline tests | Partial: fresh process on this workstation is proven; clean supported installation remains unqualified |
| FILE-02 | Atomic manual saves, conflict guards, private immutable recovery snapshots, process-crash and fault tests | Implemented tested process/filesystem behavior; no power-loss qualification claimed |
| FILE-03 | Schematic SVG includes pinned cards, reference lines and measurement provenance; SI CSV/source identity, guarded immutable exports and rendered visual QA | Implemented for current visible schematic/pin output; future display preferences must preserve export parity |
| ACCESS-01 | Named semantic controls, native focusable fields/selectors/buttons, keyboard shortcuts | Partial: full keyboard path and actual pilot-platform screen-reader checks are pending; labels alone do not prove access |
| AGENT-01 | Public versioned catalog, same dispatcher, stdio/private socket, native parity regressions and scripts | Partial: many workflows proven; exhaustive registered-action/function/preference coverage manifest is not yet established |
| AGENT-02 | Typed schemas/local registry, structured snapshots, units, part defaults/models/aliases/limits | Partial: command examples, semantic bounds/preconditions/error-code discovery need a complete machine-readable coverage audit |
| AGENT-03 | Headless and packaged-socket workflows, in-process/native parity tests | Partial: consolidate all three empty-session workflows through one release conformance harness, including fault/close/reopen |
| AGENT-04 | Verify operation IDs/progress/cancel, captured input versions, bounded sample paging | Partial: synchronous file/step/general solve paths are not all observable asynchronous jobs; state events are snapshot polling rather than a documented subscription |
| AGENT-05 | Revision preconditions, retry cache, atomic batches, shared named history | Implemented: exact history-tip guards, original actor identity and explicit cross-actor acknowledgment prevent silent reversal of intervening work |
| AGENT-06 | Shared validation/model limits, actor attribution, common undo/recovery and private authorized transport | Implemented in tested paths; inherits unresolved behavior/coverage requirements above |

## State-contract audit

| PRD action | Current implementation/evidence | Result |
|---|---|---|
| Move/rotate/pan/zoom preserve physics | Geometry stripped by electrical equality; regressions | Supported |
| Running topology/model/value/source edits restart | New run from configured startup, old result stale during invalid/async states | Supported; visible acknowledgment/completion latency unmeasured |
| Complete value-slider drag previews/coalesces/final-solves as one undo | Native control drag previews and one final command/undo pass; mounted driver redraw/focus still unqualified | Partial |
| Live switch event at next accepted step | Final desired switch group applies after the next accepted output sample; both event sides retain charge | Supported in core/native callback paths |
| Paused parameter edits stay stale until resume/restart | Core/native regressions | Supported |
| Paused switch stays pending until resume/next step | Desired/applied/pending state exposed; no physics/event update until Step/Resume accepts a sample | Supported |
| Resume unchanged / Restart from startup | Existing clock and explicit fresh-start tests | Supported |
| Undo/redo follow electrical/presentation rules | Shared history; paused edits stay stale | Supported semantics with actor identity, exact-tip guards and explicit cross-actor acknowledgment |
| Remove plotted source | Kept records retained; hardening now exposes unavailable requests and shared removal | Supported: unavailable trace marker, source focus suppression, and undo restoration |
| Reopen starts fresh and restores kept records/pause | Project/recovery fixtures | Supported same-workstation behavior; clean installation open |

## Quality and release gates

| Target/gate | Authoritative current state | Evidence needed next |
|---|---|---|
| 50 simulated parts / 100 nodes / four traces | 64 simulated parts / 100 electrical nodes including ground, within 100 drawing objects / 300 wires; 51-part/100-node and separate 64-part fixtures, four live traces, Verify, save/reopen and fresh GPU readback | Supported size demonstrated in representative valid fixtures; per-device/model/convergence bounds remain explicit and visible latency is a separate gate |
| Visible edit acknowledgment p95 ≤100 ms | Native controls tested for behavior; no input-to-visible timestamp series | Recorded hardware/build and repeated native timing measurements |
| First updated result p95 ≤500 ms | No consolidated repeated benchmark | Separate cold/warm edit-to-result measurements on the three circuits |
| 200 ms Verify ≤2 s | Bounded Verify workflows pass; no locked performance report | Hardware/OS/model/circuit hashes, repetitions and timings |
| Slow solve does not block navigation/cancel/save/undo | Live network integration runs outside the service mutex with stale-result/input guards; manual Step also stages outside the lock with atomic completion/cancellation; network/NPN phase initialization also computes outside the lock; algebraic input/wiper and some file work remain synchronous | Controlled blocked-kernel tests and actual flasher backend measurements cover live ticks; blocked manual/startup work tests and native admission parity cover Step and initialization; remaining synchronous paths and visible-latency measurements stay open |
| Recovery within 5 s / storage failures preserve save | Default 2 s scheduling, immutable snapshots, crash/fault tests | Consolidated repeated timing/failure evidence; process-crash evidence is not power-loss evidence |
| Clean offline portability | Fresh-process files/records pass on current machine | Clean packaged-installation/runtime dependency audit |
| Privacy/local ownership | Core stdlib-only; private local transport, no provider/account/telemetry requirement | Release conformance inventory and packaging audit |
| Keyboard/screen reader | Semantic nodes and shortcuts exist | Actual end-to-end keyboard/focus and screen-reader scenarios on the pilot platform |
| 100% agent coverage | Shared API foundation and many parity tests | Complete manifest mapping every native action/query/decision/result to catalog and conformance evidence |
| Gate A | Feasibility and baseline largely demonstrated | Nonblocking difficult solve/performance and clean environment evidence remain incomplete |
| Gate B | Core construction/save/scope/recovery through both paths largely demonstrated | Close remaining discoverability/recovery and consistent interaction gaps |
| Gate C | Curated families/records/export implemented | Capacity, accessibility, complete state/agent conformance and performance remain incomplete |
| Gate D | No recruited/consented participant study | Prepare protocol/assets; actual 10-person sessions and reported outcomes cannot be simulated or inferred |

## Repairs from this audit

- Added searchable, documented hobbyist aliases such as battery, variable resistor, square wave and BJT; multiword whitespace/case handling is deterministic and session-only.
- Added inspectable limitations for every library entry and visible selected-component model identity/assumptions, including ideal passive parts and references.
- Repaired duplication of a valid negative-level finite pulse: the clone starts from a valid factory seed, then restores the entire source definition/value with independent storage. Wires remain uncopied and undo restores the original graph.
- Used a stable `expm1` exact-RC update for representable tiny voltage changes and guarded live-cursor frequency overflow. Extremely close samples retain their finite time delta and valid cursors, while an unrepresentable reciprocal is absent; snapshots remain JSON-serializable.
- Default traces retain their associations through incomplete edits. Removed-source probe requests now remain explicitly unavailable until undo, explicit removal or reassignment. Single-probe removal preserves other requests and does not alter physics. Saving still stores the current valid probe selection; unavailable requests are session state.

## Prioritized continuation

1. Resolve switch pending/next-step semantics, native slider gesture evidence/coalescing, actor-aware history, and other state/agent correctness gaps.
2. Expand/measure physical circuit capacity and move difficult work off the interactive service lock, preserving model fidelity and bounded failure semantics.
3. Implement visible voltage/current presentation preferences, pinned readings/source highlighting, diagnostics and concise example guidance.
4. Consolidate complete agent conformance, keyboard/focus/screen-reader scenarios and reproducible packaging evidence.
5. Prepare formative-test materials. Actual participant/hardware validation remains a separate evidence gate.

This queue advances the original requirements; it does not replace them with the subset currently implemented or claim targets met because tests are green.

The trace-source checkpoint (0.25.0 / 39) closes the live source-association gap with [core/native/public/GPU validation](../docs/validation/trace-focus-validation.json). Functional coverage is now 11 implemented / 16 partial. This does not establish MVP completion or any pending release/user-study gate.

The pinned-reading checkpoint (0.26.0 / 40) supplies [core/native/public/GPU/export evidence](../docs/validation/pinned-readings-validation.json). It closes the arbitrary live-card part of VIEW-01 and preserves FILE-03 output parity. VIEW-01 remains partial for direct selected pin/net inspection; overall totals stay 11 implemented / 16 partial.
