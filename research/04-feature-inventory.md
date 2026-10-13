# Feature inventory

48 candidate capabilities for a maker-focused system. This is a collection of transferable ideas, **not a requirement to build all 48**. A feature’s presence in documentation is not evidence of user affection. “Hypothesis” explicitly marks our proposed response or extension.

Stages are provisional: **Prototype** tests the core experience; **Foundation** describes data/trust requirements to design early; **Next** extends a validated core; **Later** serves more advanced workflows; **Research** needs stronger evidence before commitment. No engineering estimates or validated market-priority scores are implied.

See [user evidence](05-user-evidence.md) for reported preferences and [sources](sources.md) for access quality. The same inventory is available as [JSON](feature-inventory.json).

## Experiment

| ID | Candidate | Basis | Proposed stage | References and design implication |
|---|---|---|---|---|
| F01 | Simulation responds during construction | User + documented | Prototype | iCircuit. Test the defining experience with incomplete as well as complete circuits. [S01](https://icircuitapp.com/index.html) [S04](https://apps.apple.com/us/app/icircuit/id454347770?mt=12&platform=mac&see-all=reviews) |
| F02 | Adjust parameters while running | User + documented | Prototype | Falstad, EveryCircuit. Make value changes feel immediate and keep the result state clear. [S10](https://www.reddit.com/r/ElectricalEngineering/comments/1t0mf5d/psa_falstad_is_a_thing_that_exist/) [S07](https://everycircuit.com/help) |
| F03 | Animate current and encode voltage on the canvas | User + documented | Prototype | Falstad. Help users explain a circuit; retain numbers and an animation-off option. [S12](https://www.reddit.com/r/AskElectronics/comments/1sfb5y3/what_is_everybodys_favorite_circuit_building_app/) [S05](https://ww.falstad.com/circuit/) |
| F04 | Show compact readings beside the circuit | Documented | Prototype | EveryCircuit. Test whether inline information reduces attention switching without clutter. [S06](https://everycircuit.com/?lang=en) |
| F05 | Probe a node or component directly | Documented | Prototype | LTspice, iCircuit. Make the first useful measurement easy to obtain. [S16](https://www.analog.com/en/resources/design-tools-and-calculators/ltspice-simulator.html?adicid=af_ww_technical+article__external+publications_2026) [S02](https://icircuitapp.com/usermanual/) |
| F06 | Move from a reading to a waveform | Documented | Prototype | iCircuit. Avoid making users configure a separate instrument for every signal. [S02](https://icircuitapp.com/usermanual/) |
| F07 | Exact values alongside exploratory controls | User + documented | Prototype | EveryCircuit, iCircuit. Support both experimentation and reproduction. [S54](https://www.reddit.com/r/AskElectronics/comments/1tug24e/any_website_great_for_simulating_circuits_using_a/) [S02](https://icircuitapp.com/usermanual/) |
| F08 | Pause, restart and inspect history | Documented | Prototype | EveryCircuit, iCircuit. Expose what continues and what resets after an edit. [S07](https://everycircuit.com/help) [S02](https://icircuitapp.com/usermanual/) |
| F09 | Virtual bench instruments | Documented + limited user | Next | Multisim, Proteus. Test whether familiar controls help or overwhelm new makers. [S44](https://www.ni.com/en/support/documentation/supplemental/07/explore-circuit-behavior-using-simulation-driven-instruments-in-.html) [S47](https://www.labcenter.com/instruments/) [S45](https://www.reddit.com/r/diyelectronics/comments/1k267ij) |
| F10 | Interactive switches and real inputs | User + documented | Prototype | Falstad, Proteus. Enable experiments that resemble manipulating a physical circuit. [S10](https://www.reddit.com/r/ElectricalEngineering/comments/1t0mf5d/psa_falstad_is_a_thing_that_exist/) [S47](https://www.labcenter.com/instruments/) |

## Learning

| ID | Candidate | Basis | Proposed stage | References and design implication |
|---|---|---|---|---|
| F11 | Open and modify a working example | Documented | Prototype | Falstad, Tinkercad. Start from an understandable circuit with a specific question. [S05](https://ww.falstad.com/circuit/) [S53](https://images.tinkercad.com/jl5ii4oqrdmc/1eTWASZYKjnMX9u5ioLh8Y/bb2dfb24cfc4bd66adc485baaecedf97/Tinkercad_Getting_Started_Guide_ISTE.pdf) |
| F12 | Restore an example after experimenting | Documented | Prototype | iCircuit. Encourage exploration without fear of destroying the starting point. [S02](https://icircuitapp.com/usermanual/) |
| F13 | Use physical component/breadboard views | User | Research | Tinkercad, Fritzing. Measure transfer to a real build and wiring speed, not visual appeal alone. [S55](https://www.reddit.com/r/AskElectronics/comments/f4sw0d/is_there_a_website_great_for_simulating_circuits/) [S58](https://www.reddit.com/r/arduino/comments/1t92v07/does_anyone_have_a_tip_on_how_to_get_fritzing_i/) |
| F14 | Reusable subcircuits with visible I/O | Documented | Next | SimulIDE, Logisim-evolution. Let a known-working block become a simpler building unit. [S59](https://simulide.com/p/) [S72](https://github.com/logisim-evolution/logisim-evolution/blob/main/README.md?plain=1) |
| F15 | Guided experiments with questions and expected behavior | Request + hypothesis | Research | CRUMB, CircuitLab. Validate learning and independence; do not assume a lesson catalog is sufficient. [S81](https://www.reddit.com/r/ECE/comments/1h44cb0) [S14](https://www.circuitlab.com/docs/) |
| F16 | Contextual explanation of a failing circuit | Hypothesis from friction | Prototype | Cross-product. Show the affected object, cause and a reversible next action. [S24](https://www.kicad.org/discover/spice/) [S70](https://www.reddit.com/r/PCB/comments/1bph9gd) |

## Editing

| ID | Candidate | Basis | Proposed stage | References and design implication |
|---|---|---|---|---|
| F17 | Visible move, rotate, duplicate and delete | User-derived hypothesis | Prototype | iCircuit, LTspice. Test first-time discovery without requiring a tutorial. [S03](https://apps.apple.com/us/app/icircuit/id383359044) [S17](https://www.reddit.com/r/ECE/comments/1wta1rm/wtf_is_the_uiux_of_ltspice/) |
| F18 | Fast, configurable keyboard commands | User | Prototype | LTspice. Support repeated expert work without making shortcuts mandatory. [S18](https://www.reddit.com/r/ElectricalEngineering/comments/1kazqi2) |
| F19 | Preserve intended connections while moving parts | Documented | Prototype | iCircuit. Measure unintended disconnections and control over rerouted wires. [S02](https://icircuitapp.com/usermanual/) |
| F20 | Clear selection, pan, zoom and properties | User-derived hypothesis | Prototype | LibrePCB, Flux. Keep navigation and editing predictable across views. [S67](https://www.reddit.com/r/AskElectronics/comments/1ku5mat/kicad_vs_librepcb/) [S70](https://www.reddit.com/r/PCB/comments/1bph9gd) |
| F21 | Granular undo and discoverable redo | User-derived hypothesis | Prototype | iCircuit. Recovery is part of experimentation, not just file management. [S03](https://apps.apple.com/us/app/icircuit/id383359044) |
| F22 | Interactive routing that respects constraints | User + documented | Later PCB | KiCad, Altium. Assist placement while preserving explicit control and reversible outcomes. [S27](https://forum.kicad.info/t/interactive-router-settings/61779) [S25](https://www.kicad.org/discover/pcbnew/) [S36](https://www.altium.com/documentation/altium-designer/pcb/routing) |
| F23 | Switch between assistance modes | User | Later PCB | KiCad. Make it clear when the tool will move surrounding objects. [S27](https://forum.kicad.info/t/interactive-router-settings/61779) [S79](https://forum.kicad.info/t/how-to-keep-the-router-from-moving-existing-tracks/66878) |

## Parts

| ID | Candidate | Basis | Proposed stage | References and design implication |
|---|---|---|---|---|
| F24 | Find an actual usable device quickly | User | Prototype | EasyEDA. Measure successful placement and simulation, not search-result volume. [S28](https://www.reddit.com/r/PCB/comments/1ryohby/anyone_here_start_with_easyeda_instead_of_kicad/) |
| F25 | Present symbol, model, footprint and physical identity together | Documented + hypothesis | Prototype | EasyEDA, LibrePCB. Extend unified part concepts with explicit simulation readiness. [S29](https://prodocs.easyeda.com/en/introduction/introduction/) [S64](https://librepcb.org/features/library-concept/) |
| F26 | Show model provenance, coverage and compatibility | Hypothesis from documented limits | Prototype | SPICE ecosystem. Reveal unsupported behavior before the user relies on a result. [S22](https://e2e.ti.com/support/tools/simulation-hardware-system-design-tools-group/sim-hw-system-design/f/simulation-hardware-system-design-tools-forum/692613/faq-tina-spice-note-about-encrypted-models-and-importing-models-from-one-simulator-to-another) [S24](https://www.kicad.org/discover/spice/) [S30](https://docs.easyeda.com/en/FAQ/Simulation/) |
| F27 | Guide importing a vendor model | Hypothesis from friction | Next | LTspice, QSPICE. Provide pin mapping, validation and a small test circuit. [S20](https://www.reddit.com/r/ElectricalEngineering/comments/1nwn1k3/is_ltspice_a_alternative_for_cadence_spice/) [S22](https://e2e.ti.com/support/tools/simulation-hardware-system-design-tools-group/sim-hw-system-design/f/simulation-hardware-system-design-tools-forum/692613/faq-tina-spice-note-about-encrypted-models-and-importing-models-from-one-simulator-to-another) |
| F28 | Install and update libraries in the application | Documented | Next | LibrePCB. Keep dependency management out of the first experiment. [S65](https://librepcb.org/features/library-management/) |
| F29 | Keep project references stable when parts are renamed | Documented | Foundation | LibrePCB. Avoid fragile file/name relationships; make changes explicit. [S64](https://librepcb.org/features/library-concept/) |
| F30 | Preserve the project’s required library content | Documented | Foundation | LibrePCB. Make reopening independent of later library removal or updates. [S66](https://librepcb.org/compare/) |

## Analysis

| ID | Candidate | Basis | Proposed stage | References and design implication |
|---|---|---|---|---|
| F31 | Time-domain, DC and frequency-domain analysis | Documented + user | Next | CircuitLab, KiCad, TINA-TI. Expose each analysis through a practical question and sensible starting values. [S15](https://www.reddit.com/r/ElectricalEngineering/comments/d2k2vz) [S24](https://www.kicad.org/discover/spice/) [S21](https://www.ti.com/tool/TINA-TI) |
| F32 | Compare measurements and computed signals | Documented | Next | CircuitLab. Make gain, differential voltage and power easy to inspect with units. [S13](https://www.circuitlab.com/?trk=public_post-text) |
| F33 | Sweeps, tolerances and stress checks | Documented | Later analysis | PSpice. Turn a nominal example into an investigation of expected variation. [S40](https://www.cadence.com/en_US/home/tools/pcb-design-and-analysis/analog-mixed-signal-simulation/pspice.html) [S41](https://www.cadence.com/en_US/home/tools/pcb-design-and-analysis/orcad/faqs.html) |
| F34 | Behavioral code blocks | User + documented | Later analysis | QSPICE. Provide an advanced extension path with clear limits and inspectable code. [S19](https://www.qorvo.com/design-hub/calculators-simulation/qspice) [S20](https://www.reddit.com/r/ElectricalEngineering/comments/1nwn1k3/is_ltspice_a_alternative_for_cadence_spice/) |
| F35 | Explicit valid, stale, incomplete and failed states | Hypothesis from reliability concerns | Prototype | Cross-product. Prevent stale or missing results being interpreted as valid outputs. [S10](https://www.reddit.com/r/ElectricalEngineering/comments/1t0mf5d/psa_falstad_is_a_thing_that_exist/) [S70](https://www.reddit.com/r/PCB/comments/1bph9gd) |
| F36 | Saved reproducible experiments | Hypothesis | Foundation | Cross-product. Bundle revision, models, stimuli, initial conditions and analysis settings. [S22](https://e2e.ti.com/support/tools/simulation-hardware-system-design-tools-group/sim-hw-system-design/f/simulation-hardware-system-design-tools-forum/692613/faq-tina-spice-note-about-encrypted-models-and-importing-models-from-one-simulator-to-another) [S66](https://librepcb.org/compare/) |

## Embedded

| ID | Candidate | Basis | Proposed stage | References and design implication |
|---|---|---|---|---|
| F37 | Run firmware with supported peripherals | User + documented | Next | Proteus, Wokwi. Start with a small validated hardware set relevant to maker projects. [S49](https://www.reddit.com/r/Altium/comments/kr493k) [S48](https://labcenter.s3.amazonaws.com/downloads/Tutorials_p9.pdf) [S50](https://docs.wokwi.com/) |
| F38 | Debug code, serial traffic and digital signals | Documented | Next | Wokwi, Proteus. Help separate firmware, wiring and protocol problems. [S50](https://docs.wokwi.com/) [S47](https://www.labcenter.com/instruments/) |
| F39 | State the boundary of digital/analog simulation | Documented limit + hypothesis | Foundation | Wokwi. Prevent an executing program from implying validated analog circuitry. [S51](https://docs.wokwi.com/chips-api/analog) |

## Build

| ID | Candidate | Basis | Proposed stage | References and design implication |
|---|---|---|---|---|
| F40 | Carry circuit identity into PCB layout | Documented | Next integration | KiCad, DipTrace. Avoid redrawing while checking model-to-footprint mapping. [S25](https://www.kicad.org/discover/pcbnew/) [S62](https://www.diptrace.com/diptrace-software/schematic-capture/) |
| F41 | Short path to fabrication and assembly | User + documented | Later PCB | EasyEDA. Provide a preview and portable outputs as well as optional ordering. [S28](https://www.reddit.com/r/PCB/comments/1ryohby/anyone_here_start_with_easyeda_instead_of_kicad/) [S31](https://prodocs.easyeda.com/en/quick-start.html) |
| F42 | Connect board geometry with enclosure design | User + documented | Later integration | Fusion. Make connector positions and fit visible during changes. [S34](https://www.reddit.com/r/Fusion360/comments/1rqym0s/does_anyone_use_fusion_for_electronics_design/) [S32](https://www.autodesk.com/products/fusion-360/blog/18-things-need-to-know-fusion-360-electronics/) |
| F43 | Cross-probe among schematic, board and physical view | User + documented | Later PCB | Altium, KiCad. Support debugging and assembly as well as design navigation. [S39](https://www.reddit.com/r/KiCad/comments/1s2h5cj/finally_switching_from_altium_to_kicad/) [S25](https://www.kicad.org/discover/pcbnew/) |
| F44 | Assembly variants and consistent documentation | Documented | Later professional | Altium. Avoid manual divergence as a maker project becomes a product family. [S37](https://www.altium.com/documentation/altium-designer/design-variants) |

## Ownership

| ID | Candidate | Basis | Proposed stage | References and design implication |
|---|---|---|---|---|
| F45 | Local access and complete export | User + documented | Foundation | KiCad, LibrePCB, SimulIDE. Preserve continuity independent of network or service lifetime. [S28](https://www.reddit.com/r/PCB/comments/1ryohby/anyone_here_start_with_easyeda_instead_of_kicad/) [S66](https://librepcb.org/compare/) [S61](https://www.reddit.com/r/arduino/comments/1t58g4x/offline_alernative_to_tinkercad_for/) |

## Collaboration

| ID | Candidate | Basis | Proposed stage | References and design implication |
|---|---|---|---|---|
| F46 | Share an executable experiment with controlled access | Documented + hypothesis | Next | Wokwi, Flux. Test recipient success and setup cost, not merely link generation. [S50](https://docs.wokwi.com/) [S69](https://docs.flux.ai/tutorials/tutorial-collaboration-deep-dive) |
| F47 | Review and manage changes across a team | User | Later professional | Altium. Keep small-project sharing simple; add governance only as required. [S38](https://www.reddit.com/r/Altium/comments/1utu88w/altium_vs_kicad/) |

## Assistance

| ID | Candidate | Basis | Proposed stage | References and design implication |
|---|---|---|---|---|
| F48 | Explain or suggest parts in project context | Limited user + hypothesis | Research | Flux. Require inspectable evidence and reversible suggestions; preserve direct editing. [S70](https://www.reddit.com/r/PCB/comments/1bph9gd) [S68](https://docs.flux.ai/flux/Introduction/getting-started-in-flux--schematic) |

## What should be tested first

Use a deliberately narrow prototype to test F01–F08, F11–F12, F17–F21, F24–F26, F35 and F45. Design the data needed for stable parts and reproducible experiments early, even if the initial interface exposes only a small subset. These selections express the project brief and research synthesis; they are not a statistically established feature ranking.

Do not accumulate features at the expense of a reliable first circuit. Success means the user can build, understand, change, recover and retain an experiment.
