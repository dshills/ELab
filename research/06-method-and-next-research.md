# Method, limitations and next research

## Scope

The research question was: **Which capabilities and experiences do users appreciate in schematic, circuit-simulation and adjacent PCB tools, and what could inform an especially easy system for hobbyists and makers with room to grow?**

The starting preference was iCircuit’s live simulation while drawing. Coverage expanded to 24 product families to include analog experimentation, detailed analysis, board production, firmware, physical breadboards, digital logic and collaboration. These are adjacent jobs, not interchangeable product categories.

Research was conducted on **2026-10-12**. Current documentation and 2024–2026 discussion were prioritized where available. Older specific accounts were retained when they explained a useful interaction or filled an evidence gap. No claim is made that the user sample is representative.

## Source selection and workflow

1. Searched product documentation and public first-person discussion using combinations of product names with “like,” “prefer,” “intuitive,” “live simulation,” “models,” “library,” “routing,” and “integration.”
2. Used Reddit, KiCad.info, Autodesk forums, EEVblog, Hacker News, App Store reviews, Steam and official documentation/project repositories.
3. Sought both positive accounts and counterevidence. Avoided treating vendor marketing, comparison-site summaries or generic recommendation lists as independent satisfaction research.
4. Opened selected high-value sources to inspect context. Other sources were available as substantive search-index excerpts; the register marks this explicitly.
5. Separated user preference, documented capability and proposed design response. Checked notable edition/lifecycle differences against official sources.
6. Organized 82 selected records, 40 evidence observations and 48 feature candidates. Counts describe this collection, not unique participants or independent studies.

A search hit was not automatically accepted. Promotional or repetitive comments were omitted from the preference synthesis where identifiable. Comments from disclosed product authors were used as product context, not independent endorsements. Technical claims use official project/vendor material; user assertions about accuracy, compatibility or speed remain assertions.

## Evidence strength

**High confidence in a qualitative theme:** several distinct accounts plus corroborating product behavior, or an unambiguous official lifecycle fact. This still does not estimate how common a preference is.

**Moderate:** a specific first-person experience with corroborating capability documentation, but limited breadth or older versions.

**Low / exploratory:** one brief recommendation, historical family-level evidence, vendor-selected testimonials or inferred appeal without detailed user experience.

The profiles deliberately do not assign numerical “best” scores. Such a score would conceal different jobs, unequal source coverage and untested capabilities.

## Limits and unresolved questions

- English-language public communities dominate the sample. Silent users, non-English users, educators outside these communities and people who abandoned a tool may be underrepresented.
- Search terms aimed at likes can produce positive selection bias. Counterqueries and mixed discussions reduce but do not remove it.
- Several sources describe the same discussion or overlapping communities. Source count is not independent corroboration count.
- Reddit identity, role and experience are self-reported. Comments may contain errors, promotion or generated text.
- “Indexed” records contain only the retrieved excerpt; individual dates/authors may not be fully available. The Mac iCircuit review page returned HTTP 429. An LTspice thread open failed. The Tinkercad landing page yielded no text, so its official guide was used instead.
- Historical missing-feature complaints cannot characterize the current release. iCircuit release notes, DipTrace’s current SPICE description, EasyEDA editions and the EAGLE/Multisim Live transitions required particular care.
- No software was installed, no timed tasks were run, and no numerical fidelity, import fidelity, platform parity, accessibility or licensing audit was performed.
- Store ratings are not comparable across products and were not used to rank them. Current subscription costs and free-tier limits were not systematically audited.
- Specific positive user evidence is weakest for Qucs-S, SimulIDE, current Logisim-evolution, CRUMB, Wokwi’s detailed workflow and OrCAD/PSpice. These deserve targeted follow-up.
- Specialist RF/EM, power-electronics and IC-signoff suites were not deeply reviewed. ADS, AWR, PLECS, SIMetrix/SIMPLIS, Micro-Cap’s historical workflow, Circuit Wizard, CircuitMaker and other tools remain possible extensions if the target scope expands.

## Next: hands-on comparative study

Test **iCircuit, Falstad, EveryCircuit, LTspice, KiCad, EasyEDA Pro, Fusion Electronics and Wokwi** first. This is a research sample chosen for distinct workflows, not a purchase recommendation. Add DipTrace or LibrePCB if part management/editor interaction becomes the leading question.

Use common tasks only where the application supports them. Mark an unsupported task as a scope difference, not automatic failure.

| Task | Primary comparison | Evidence to collect |
|---|---|---|
| First LED circuit and a meaningful measurement | iCircuit, Falstad, EveryCircuit, LTspice | First-success time, help requests, connection errors, explanation of current |
| RC transient and value change | Same live/analog tools | Edit latency, state semantics, exact values, understanding of time constant |
| Amplifier with a named real device | LTspice, KiCad and applicable live tools | Model search/import, missing assumptions, reproducibility, clipping interpretation |
| Repair an intentionally broken circuit | All applicable schematic tools | Error localization, recovery, distinction between invalid and unsupported |
| Sensor + MCU + display | Wokwi, Proteus or SimulIDE as follow-up | Firmware loading, peripheral support, debugging and model boundaries |
| Turn a small validated circuit into a PCB | KiCad, EasyEDA Pro, Fusion | Re-entry work, footprint mapping, rule checks, exports and manufacturing readiness |
| Fit a board into an enclosure | Fusion and an export-based workflow | Placement changes, connector alignment and synchronization effort |
| Reopen and share an old project | All applicable tools | Model/library retention, export completeness, recipient setup and permissions |

Record exact app version, OS, input device, account tier and circuit/model files. Capture screen recordings and task timestamps with participant consent. Keep separate measures for first use and repeat use. For circuit fidelity, use analytical expected results and measured hardware where appropriate; document model assumptions and tolerances.

## Interview guide

Recruit a small formative group spanning: makers with little schematic experience, makers who have ordered a few PCBs, experienced hobbyists, and a few professional users who also build personal projects. A suggested first round is 8–12 participants; it supports issue discovery, not population-level conclusions.

Ask participants to show their last actual project before discussing hypothetical features:

1. What were you trying to build, and which tools did you use at each step?
2. Show the moment a tool helped you understand something you had been unsure about.
3. What did you copy, redraw, re-enter or look up outside the application?
4. How did you select a part, and how did you decide its model/footprint was trustworthy?
5. When did you choose simulation, a breadboard or a manufactured prototype, and why?
6. Show a mistake you recovered from. What feedback would have helped earlier?
7. When would immediate simulation distract you or make you overconfident?
8. How do you share a problem so another person can reproduce it?
9. Have account, price, export or product changes made you switch tools?
10. Which advanced capability did you eventually need after outgrowing your first workflow?

Avoid leading with “Would you like iCircuit plus KiCad?” That question encourages endorsement of a vague bundle. Observe whether the proposed workflow actually removes a costly step.

## Decisions the next phase should resolve

- Is the first product primarily an experimental workbench, a beginner PCB tool, or a unified journey that needs both immediately?
- Does always-on behavior outperform an explicit Run model once invalid edits and state resets are included?
- Which 30–50 components and a small set of MCU/peripheral models cover the first realistic maker projects?
- Is KiCad interchange sufficient for the initial path to a board, or does native layout produce a materially better outcome?
- Which tasks need a physical/breadboard representation, and which become easier in schematic form?
- Can local ownership and optional sharing satisfy both access preferences without confusing file state?

A useful next deliverable is a benchmark notebook of observed workflows and results, followed by a narrow product brief supported by those observations.
