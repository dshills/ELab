# Electronic design tools: research library

Research completed: **October 12, 2026**. Primary audience: **hobbyists and makers, with room to grow**. Starting preference: **iCircuit’s live simulation while drawing**.

This first research phase covers **24 product families**, **82 selected source records**, a reusable feature inventory, and a plan for hands-on validation. It is an extensive qualitative desk study, not a representative survey or a tested ranking. Some products have substantially better user evidence than others.

## Start here

**Inspector tooltip fix:** [Owned popup retirement](115-inspector-tooltip-retirement.md) — October 25, 2026. Numeric bounds/arrow help clears with a removed or replaced inspector, preserving shared commands and normal help.

**Vellum widgets implemented:** [Five UI migrations](114-vellum-widget-migrations.md) — October 25, 2026. Search/Clear, field feedback, compact steppers, semantic appearance and secondary-action overflow, retaining shared commands.

**Vellum widget review:** [Recommended UI adoptions](113-vellum-widget-adoption-review.md) — October 25, 2026. Prioritized search, field feedback, compact steppers, semantic appearance and constrained toolbar overflow, with integration limits and deferred widgets.

**Component browser:** [Common, categories, search and schematic symbols](76-component-browser.md) — October 23, 2026. Organization of the existing 18 kinds with the same user/agent browser commands; no new device models.

**Remaining first-version work:** [Completion checklist](66-first-version-completion-checklist.md) — October 20, 2026. Detailed native repair, workflow, agent, data, quality, packaging and owner-acceptance tasks, with pass conditions and the additional full-MVP gates.

**Current delivery plan:** [First usable version](34-first-version-plan.md) — updated October 20, 2026. Frozen hobbyist alpha scope, six ordered work packages, release checks and a concrete app handoff. This is the active implementation priority; full PRD acceptance remains separate.

**Product definition:** [ELab product requirements](07-product-requirements.md) — Draft 0.3, October 13, 2026. Defines the hobbyist workflow, live simulation contract, complete user/agent capability parity, Vellum UI foundation, MVP boundaries, acceptance criteria, and release validation.

**Framework decision:** [Vellum adoption and integration](08-vellum-integration.md) — October 13, 2026. Records the selected Go UI framework, inspected baseline, application responsibilities, agent-interface gaps, and first implementation gate.

**Implementation:** [First LED application slice](../docs/implementation.md) — October 13, 2026. Vellum app foundation, live LED DC calculation, shared user/agent commands, validation results, and remaining MVP work. Launch instructions are in the [project README](../README.md).

**NPN implementation:** [Live transistor-controlled LED switch](../docs/npn-switch.md) — October 13, 2026. Drawable B/C/E pins, generic nonlinear DC model, shared user/agent commands, editable base drive, persistence, and validation against the saved resistive-load reference.

**NPN pulse implementation:** [Blinking LED and live scope](../docs/npn-transient.md) — October 13, 2026. Pulse-driven, instantaneous NPN switching with exact edges, shared controls, saved pulse settings, and headless/native validation.

**Mixed RC/NPN implementation:** [Capacitor-timed autonomous flasher](../docs/mixed-transient.md) — October 13, 2026. Coupled nonlinear transient simulation, signed startup conditions, two alternating LEDs, shared user/agent controls, and numerical/native validation.

0. [Hands-on comparison](hands-on/README.md) — three hobbyist circuits run in iCircuit, Falstad, LTspice, and KiCad, with screenshots, native fixtures, measurements, and design opportunities. Added October 12, 2026, after the desk study.
1. [Findings and product direction](01-findings.md) — the main patterns, tradeoffs and priorities.
2. [Product profiles and comparison](02-product-profiles.md) — what users appreciate, documented capabilities and lessons from each application.
3. [Live simulation deep dive](03-live-simulation.md) — what to preserve from iCircuit and how to avoid misleading feedback.
4. [Feature inventory](04-feature-inventory.md) — 48 candidate capabilities, linked to evidence and clearly marked as observations or hypotheses.
5. [User evidence ledger](05-user-evidence.md) — concise, traceable observations with caveats and a few short quotations.
6. [Method and next research](06-method-and-next-research.md) — limitations, source handling, benchmark tasks and interview guide.
7. [Source register](sources.md) and [structured sources](sources.json) — dates, links, access quality and contribution of every selected source.

## Main conclusion

The most promising hypothesis is an **interactive electronics workbench that helps a maker understand a circuit immediately and carry that understanding into a real build**. Prioritize live experimentation, discoverable editing, usable simulation models, clear feedback and portable projects. Add professional depth progressively. This is a proposed direction to validate, not a finding that one combination will satisfy everyone.

## Reading the evidence

- **User evidence** means a person reported a preference or experience. It is not proof of prevalence or technical accuracy.
- **Documented** means a vendor or project describes the capability. It does not establish that users like it.
- **Inference/proposal** means our interpretation or a candidate design choice.
- Source IDs such as S24 link directly to the source; the register records retrieval limitations.

Keep future updates dated. Preserve historical observations instead of silently treating them as facts about current releases. No application was installed, benchmarked or visually audited during the initial desk-study phase. The subsequent [hands-on phase](hands-on/README.md) includes application use and a user-approved LTspice installation; it is not a timed usability benchmark.

Latest implementation: [schematic drawing and clean Vellum UI](../docs/schematic-drawing.md), including saved native screenshots and UI/agent validation reports.

Saving and loading: [portable schematic persistence](../docs/persistence.md), with the shared agent commands and validation evidence.

Capacitors and live RC simulation: [implementation, shared commands, and analytical validation](../docs/rc-transient.md).

- [Selectable probes and scope cursors](../docs/scope-probes.md): both flasher branches, retained signed measurements, sample-snapped timing, and shared native/agent commands. [Headless validation](../docs/validation/headless-probes.json) and [desktop validation](../docs/validation/desktop-probes.json).

- [Next ten implemented workflow items](09-next-ten-implementation.md): search, duplication, named history, atomic edits, direct plotting, fixed intervals/statistics, saved probes, CSV and SVG exports, with headless/native validation.

- [Continuing implementation and code quality](10-continuation-progress.md): local checkpoints, reviewed concurrency fixes, pinned lint tooling, bounded Verify operations and recorded exports.

- [First-version delivery plan](34-first-version-plan.md), [visible-latency method](36-visible-latency-method.md) and [fixed collection protocol](37-visible-latency-cohort-plan.md): current alpha cut line, calibration evidence and the original declared responsiveness trials. Separate current /185 [RC](47-current-rc-visible-results.md), [LED](50-current-led-visible-results.md) and [NPN](52-current-npn-visible-results.md) blocks each complete twenty observations below 500 ms within their warm-running/software-composition scopes.

- [Visible responsiveness results](38-visible-responsiveness-results.md): all scheduled outcomes and separately scoped software-frame percentiles; 100 ms acknowledgment and broader performance remain open.

- [Source modes, restart and captured experiments](42-source-modes-and-captures.md): twelve bounded /183 electronics/lifecycle cases with independent equations, immutable captures and fresh saved-file reopens; no source/version change.

- [Native MOSFET controls and signed readings](43-native-mosfet-review.md): ten /183 native states and fresh saved reopens, independent signed current/power checks, precise PMOS presentation documentation and preserved input-method limitations.

- [Bounded native binding follow-up](53-native-binding-followup.md): actuator/display/probe/sampling source review, preserved stale-audit-tool failure, exact current ledger verification and refreshed local helper; no new physical/runtime/full-binding qualification.

- [Structured malformed-file diagnostic](54-structured-corruption-review.md): 20,028 mutations, eighteen dirty-work-preserving public refusals and separately scoped guided/unguided fuzz evidence, with the original race timeout retained and the full unchanged race pass completed. Current actions are separated from the verbatim [engineering history](34-engineering-checkpoint-history.md).

- [Recorded sample-index validation](55-recorded-index-validation.md): reproduced fractional-index truncation, shared-command source repair, original failing/fixed passing mounted and callback regressions and full quality. Clean replacement qualification remains next.

- [0.169.0 /186 recorded-index replacement](56-recorded-index-review.md): clean app/agent, exact copied validation/preservation cases, strict signature, ZIP and example reopens; original /185 source/failures and all prior artifacts preserved. Owner walkthrough remains pending.

- [Native release acceptance and inspector repair](67-native-release-acceptance.md): actual /194 input and numerical passes, preserved recordings, the reproduced inspector blocker, and its native-verified /195 replacement with package checks.

- [Maintenance work plan](80-overnight-work-plan.md) and [scope interval correction](87-scope-interval-correction.md): retained invalid input, shared range guidance and first-click resets, with full source checks and 52 packaged-agent RC reference readings. The newer candidates retain separately pending native checks; the qualified /205 download remains preserved.

- [Recorded-position request context](88-recorded-position-context.md): stale seek feedback and late rejected drafts stay with their original context, with full checks and packaged immutable-capture/RC evidence after a live model edit.

- [Recorded interval coverage](89-recorded-statistics-coverage.md) and [analysis entry ownership](90-analysis-field-context.md): refuse statistics outside actual saved timestamps and prevent old form callbacks from retaining malformed text after agent edits. Full source and matching packaged-agent checks pass; native acceptance of the held candidates remains separate, and /205 stays the current download.

- [Captured measurement entry recovery](91-statistics-entry-recovery.md): retained invalid start/end entries, cached-result labeling and channel-owned private editing. Four mounted failures are repaired; the old native baseline is observed and restored. Full quality and the clean /214 matching agent pass, with changed-candidate native acceptance still separate.

- [Captured comparison entry recovery](92-comparison-entry-recovery.md): retained invalid times and private editing scoped to the selected/reference pair, preserving independent display preferences. Full quality and clean /215 matching-agent checks pass, including 138 RC references across different saved rates/resistances after a later live edit. Native acceptance remains separate; /205 stays preserved.

- [Text editing and application shortcuts](93-text-undo-boundary.md): preserve initial text at the Undo boundary, exercise actual Command modifier mappings, and distinguish local text history from circuit Undo/Redo. The elab9 compatibility pin, focused checks and full application quality gate pass; native acceptance remains separate.

- [Number fields and wheel scrolling](94-number-field-wheel.md): native reproduction of focused wheel input changing playback, current-source initial-voltage/playback regressions, and a narrow managed-control repair preserving drafts and explicit keyboard edits. The elab10 pin and full source/offscreen gates pass; fresh native acceptance remains separate.

- [RC-driven MOSFET reference checks](95-gate-common-mode.md): seven packaged-agent circuits, 1,722 independent scalar comparisons, active-work Save/Undo/Pause/cancellation and a fresh saved-file reopen. Existing educational models and protected live state are preserved.

- [Verification output-step boundary](96-verification-output-grid.md): repair the false refusal of 1 ms at 1 µs by counting the represented worker grid, preserving distinct final intervals and the existing limits. Focused regressions, full quality and clean /218 matching-agent grid/storage checks pass; native acceptance remains separate.

- [Sliders and pane scrolling](97-slider-wheel.md): old-native/current-source reproduction of accidental wiper edits, normal text/selector/thumb checks, and a narrow managed-slider repair. The elab11 pin, full quality and clean /219 public-actuation/capture checks pass; fresh native acceptance remains separate.

- [Scope scrolling and schematic zoom](98-plot-wheel-validation.md): unchanged native plot-wheel paths preserve circuit state, with a focused routing/flag-restoration regression. No product or package change; /219's separate native acceptance list remains open.

- [Manual-step progress counts](99-manual-step-progress.md): match planned totals to the worker’s existing target grid, preserving timing and bounds. Focused and full gates pass; the original flasher wall-budget failure and numerical-test clock adjustment remain explicit.

- [Validation layout and numeric drafts](103-validation-scroll-and-drafts.md): focused interval visibility and retained unfinished values across layout, with unchanged electrical commands and full source checks.

- [Decimal number steps](104-decimal-number-steps.md): keep stepped values and displayed decimals consistent, with bounded arithmetic and unchanged public command types.

- [Native interval reset and exit classification](105-native-reset-and-exit-classification.md): both fields/views, timed agent interventions, immutable captures, and the AppKit Quit-menu evidence for the earlier exit.

- [Native recorded-position checks](106-native-recorded-position.md): refusal/correction, bounded agent interleavings, exact public protocol and the remaining native queue-origin qualification.

- [Native captured measurements](107-native-captured-measurements.md): exact endpoint defaults, cached-result correction, native/agent channel ownership, independent electrical references and seven remaining native checks.

- [Native captured comparison](108-native-captured-comparison.md): reference ownership, refusal/correction, independent preferences, same-reference reset, and 138 RC/statistics references; six native checks remain.

- [Native keyboard and text history](109-native-keyboard-history.md): initial/boundary selection, local text editing, actual application-history routing, exact state preservation and four remaining native checks.

- [Queued field callback ownership](110-queued-field-origin.md): reproduced text/number queue-context defect, private SDK elab14 repair and passing full framework/application quality. Owner returned and requested a stop; new candidate/native checks are deferred and the existing release remains preserved.

- [PWM, amplifier dynamics, shift registers and sound plan](111-hobbyist-devices-plan.md): the four additions following the H-bridge; [model and command guide](../docs/hobbyist-devices.md) documents examples, shared controls, numerical scope and explicit audio preview.

- [Reliability, workflow, release and examples closeout](116-reliability-release-and-examples-plan.md): all five ordered items complete; ELab 0.195.3 /264 unsigned download, unchanged 53 component kinds and 62 examples, shared browser commands, native keyboard/mouse/save checks, full race/rendered gates and extracted offline qualification.
