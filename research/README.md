# Electronic design tools: research library

Research completed: **October 12, 2026**. Primary audience: **hobbyists and makers, with room to grow**. Starting preference: **iCircuit’s live simulation while drawing**.

This first research phase covers **24 product families**, **82 selected source records**, a reusable feature inventory, and a plan for hands-on validation. It is an extensive qualitative desk study, not a representative survey or a tested ranking. Some products have substantially better user evidence than others.

## Start here

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
