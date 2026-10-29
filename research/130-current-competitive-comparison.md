# ELab competitive comparison — October 29, 2026

**Assessment:** ELab is now a credible small-circuit experimentation workbench,
with substantially more feature breadth than the original MVP. It has not yet
established that it is easier for hobbyists than iCircuit/Falstad, or a replacement
for LTspice, KiCad or Fusion. Its best product opportunity is a local workbench
where a maker and an external AI can draw, inspect, change and reproduce the same
experiment through one explicit command contract.

That is a proposed position, not a proven market preference or exclusivity claim.
Live simulation, FFT/Bode tools, offline operation and agent access all exist in
other products. Our particular combination, execution semantics and usability
need to earn the preference.

## Scope and evidence

- Audience: hobbyists and makers, with room to grow, as in the
  [original PRD](07-product-requirements.md).
- ELab baseline: source `7f164ce8` (documentation update), with the peak/cursor
  implementation from `7408ac43`; current runtime discovery confirms **61 kinds,
  87 examples and 226 commands**. This comparison uses source, retained tests,
  the actual native screenshots captured today, and a new headless smoke.
- Competitor baseline: current official product pages, vendor release notes,
  documentation and developer APIs checked October 29. The original
  [October 12 hands-on study](hands-on/README.md) remains evidence for its exact
  prepared LED/RC/NPN fixtures and installed versions: iCircuit 1.15.1,
  CircuitJS 4.1.5js, LTspice Mac 26.0.2 and KiCad 10.0.3.
- This is **not a fresh all-product desktop benchmark**. No new competitor
  installations, matched-model sweep of all engines, hardware measurements,
  recruited user sessions or comparative task-time series were performed.
  Current vendor capabilities are distinguished from observed behavior.
- Public source availability, a locally qualified unsigned package and a
  polished, independently accepted public release are different milestones.
  Current source includes changes that the preserved /282 app predates.

## Two important corrections to the competitive picture

### iCircuit 1.16 is a broader competitor than the installed version we tested

The current vendor site describes over 300 elements and live analog/digital
operation, along with scope frequency-domain views, programmable boards,
SPICE subcircuits and HDL. Its new Pico/SPICE/HDL functions are described for
Apple platforms; do not assume feature parity on every listed platform.
[Vendor overview](https://icircuitapp.com/).

The Mac App Store now lists 1.16, with an October 12 release entry. Its release
notes describe manufacturer-model import, Verilog/VHDL and Pico firmware support.
Our earlier native results are explicitly for **1.15.1**, not a test of those
new capabilities. This is a necessary version-baseline update, not evidence
that we measured their new implementations.
[Mac listing and release history](https://apps.apple.com/us/app/icircuit/id454347770?mt=12).

**Implication:** live drawing and a spectrum tab do not by themselves distinguish
ELab. We are behind iCircuit on reusable model compatibility and code-driven
maker projects. Component-count totals also use different definitions: an ELab
kind is not directly equivalent to each IC, board or element in another catalog.

### External-agent integration is already a competitive category

Flux documents an MCP connection that lets external clients read project data
and delegate design changes to its cloud agent. This includes a ready client
setup workflow, rather than only file generation or screenshot automation.
[Flux's current MCP announcement](https://www.flux.ai/p/blog/flux-mcp-in-your-ai-tools).

Its August update also describes external-agent integration and a conversational
editor experience. These are vendor capability statements; this review did not
validate the quality or completion rate of Flux-generated hardware.
[Flux integration/editor update](https://www.flux.ai/p/blog/summer-2026-updates-flux-mcp-server-and-chat-mode).

**Implication:** ELab's strength is its local, direct, guarded service design and
reproducible results. We should not advertise generic AI/agent capability as
unique. We also need an approachable supported connection/client experience;
a complete API is not itself a complete novice-facing AI product.

## Product-by-product comparison

| Product | Established or documented advantage | Where ELab stands for our audience |
| --- | --- | --- |
| **iCircuit** | Continuous interaction, broader device/board coverage, mobile/desktop reach and the 1.16 extensions above. | Closest direct competitor. We implement the desired edit–observe workflow and extensive experiment tooling, but have not demonstrated a usability win; generic models, capacity and maker-code coverage are narrower. |
| **Falstad / CircuitJS** | Immediate browser entry, visible current/voltage, sample circuits, portable text/link sharing and offline options. [Official simulator](https://www.falstad.com/circuit/), [file/link workflow](https://falstad.com/circuit/doc/overview.html). | Our native document/inspector, captured experiments and explicit command/result semantics are strengths. Falstad remains easier to try and share without building/installing ELab. No measured drawing-speed or overall-accuracy advantage established. |
| **LTspice** | Extensive analog model/demo ecosystem and deeper SPICE workflows, including transient, AC/noise and third-party models. [ADI overview](https://www.analog.com/en/resources/design-tools-and-calculators/ltspice-simulator.html). | Our default live loop avoids a separate Run step in supported experiments. We are behind for named devices, broader analyses/topologies and practical analog verification. Existing FFT/cursors are not a unique ELab advantage. |
| **KiCad** | Integrated ngspice with model mapping, and a direct route from schematic to PCB, rules and fabrication-oriented work. [Simulation](https://www.kicad.org/discover/spice/), [PCB workflow](https://www.kicad.org/discover/pcb-design/). | More focused on immediate experiments, with fewer setup concepts in our curated library. We lack a validated KiCad exchange path, footprints, PCB editing and manufacturing workflow. ELab is a companion candidate, not a substitute for board design. |
| **Fusion Electronics** | Connected schematic/PCB/mechanical design, libraries, DRC and SPICE in one product environment. [Autodesk overview](https://www.autodesk.com/products/fusion-360/electronics-engineer). | A lighter experimentation job, with local control and no requirement to enter a PCB/enclosure project. There is no comparable board/enclosure/manufacturing capability in ELab. We did not re-test Fusion's simulator here. |
| **EveryCircuit** | Animated interaction, on-circuit signal visualization, Bode/XY views and a shared web/mobile circuit gallery. [Official features](https://everycircuit.com/). | Good overlap in educational experiments and component families. Our saved experiment provenance and external service control offer a different workflow; reach, sharing and novice preference are not established advantages for ELab. |
| **Wokwi** | MCU/firmware projects, peripherals, digital debugging and project links; CI/CLI workflows are documented. [Overview](https://docs.wokwi.com/), [CI](https://docs.wokwi.com/wokwi-ci/getting-started). | ELab covers analog/transistor experimentation that is a different job. Without MCU execution, code editing or bus debugging, it does not cover many Arduino/ESP32/Pico maker projects. We should not treat a 555 flasher as equivalent firmware coverage. |
| **Flux** | Integrated AI design work, PCB context and external-agent connections described above. | Strong architectural opportunity in local experiments, direct semantic commands and provider-independent clients. We do not have its ready conversational/cloud-collaboration/PCB experience. Their documentation does not establish that all their operations share ELab's particular revision/retry semantics, or that their AI is more reliable. |

KiCad also has a remote-control IPC API. Its current developer documentation
separates the PCB-oriented KiCad 9/10 API from schematic/headless additions in
KiCad 11. These version distinctions matter; it would be incorrect to describe
KiCad as lacking automation altogether.
[KiCad add-on API status](https://dev-docs.kicad.org/en/apis-and-binding/ipc-api/for-addon-developers/).

Falstad likewise documents an embedding JavaScript interface for circuit elements,
measurements, external voltages and import/export. It is a relevant automation
baseline, though it does not establish the same complete shared command contract.
[Official JavaScript interface](https://www.falstad.com/circuit/jsinterface.html).

## Where ELab is strong

### 1. The original draw–observe–adjust loop is implemented

Placement previews, named terminals, moving/rotating wired parts, Common/search,
explicit plot actions and live readings support the original design hypothesis.
The current Light RC screenshot shows schematic, value editing and time cursors
in one workspace. That is concrete progress over a specification.
[Drawing workflow](../docs/workbench-workflows.md),
[RC screenshot](../docs/images/rc-scope.jpg).

It is an interaction advantage relative to the explicit setup/run loop observed
in the original SPICE workflows, not proof that ELab is easier than every editor
or faster for an experienced SPICE user.

### 2. Experiments have meaningful structure

Independent Verify captures, retained results, comparisons, interval/threshold
measurements, DC/AC/noise/tolerance analysis and exports provide useful depth.
The eight-BJT amplifier is a genuine connected 27-part educational network,
not an opaque amplifier component substituted for a discrete circuit.
[Amplifier topology](../docs/discrete-class-ab.md),
[recorded measurements](../docs/interval-measurements.md),
[bounded analysis reads](../docs/analysis-pages.md).

Our scope/spectrum now includes useful exact-bin frequency/amplitude measurements,
but the live FFT is bounded to 256 samples and recorded FFT to 4096. A spectral
plot and THD calculation do not imply calibrated audio analysis or THD+N.
[Measurement contract](../docs/spectrum-analysis.md).

### 3. Human/agent coordination has a serious foundation

The same Service.Dispatch controls both paths, with revision/document checks,
retry identities, edit previews, atomic batches, actor-aware history, observable
jobs and detached result data. That makes exact changes and repeatable automated
experiments possible without desktop driving.
[Protocol](../docs/agent-api.md), [architecture](../docs/architecture.md),
[latest code review/tests](../docs/validation/spectrum-markers-20261028/README.md).

These are verified capabilities in named paths. The older PRD audit still records
partial exhaustive native binding/behavior acceptance; 226 discoverable commands
must not be equated with proof of every command in every native state.
[Audit](11-prd-implementation-audit.md).

### 4. Work remains locally usable and inspectable

Projects and captures have documented identities/settings, recovery, exports and
readable model assumptions. Simulation itself needs no cloud AI provider. This
is a useful distinction against cloud-mediated design work, though offline use
and file ownership also exist in iCircuit and established desktop tools.

## Where we are behind or unproven

### Ease of use remains a hypothesis

We have automation, internal observations and owner feedback, not a completed
first-use study showing hobbyists succeed faster or with less help. The existing
10-person study summary is still blank. A large feature list cannot answer this.
[Study status](first-use-study/results-summary.md).

The current Apollo screenshot still exposes dense model settings; a long
configurable-transistor-profile checkbox label is visibly clipped in the right
rail. Basic value/plot actions and advanced model configuration need deliberate
prioritization and layout checks. This is a specific native screenshot observation,
not a broad accessibility audit or measured task failure.
[Native capture](../docs/images/class-ab-apollo.jpg).

### The supported-circuit envelope is restrictive

The 64-part/100-node total is only part of the story: **four capacitors, four
inductors, two combined MOSFETs, two op-amps and three sources** can block ordinary
maker experiments before the total is reached. A four-discrete-MOSFET H-bridge
is outside today's envelope even though the dedicated H-bridge component works.
A six-capacitor tone-control/filter experiment also exceeds it.
[Published limits](../docs/circuit-capacity.md), [runtime smoke](130-current-elab-smoke.json).

These are boundaries to qualify with representative circuits, not grounds for
silently weakening work guards or numerical tolerances. The existing 512-attempt/
100 ms integration accounting can still refuse difficult requests.
[Integration limits](../docs/integration-limits.md).

### Generic models do not answer the real-part question

A hobbyist eventually asks whether a specific transistor, amplifier or regulator
will work. We have generic/device-profile parameters and some optional losses
or temperature laws, not a general manufacturer-SPICE import workflow. Named
hardware ratings, packages and arbitrary topology coverage are not implied.
LTspice/KiCad's model ecosystems and iCircuit's newly documented import support
make this a substantial gap rather than a cosmetic library-count difference.

Single-level circuit blocks help reuse an experiment, but are not equivalent to
arbitrary nested/model-imported SPICE hierarchy.
[Block boundary](../docs/circuit-blocks.md).

### Real-world continuation is missing

Schematics, PDF/PNG/SVG and measurement exports help explain/share a circuit.
They do not preserve an editable design into KiCad/LTspice, assign footprints,
produce a usable parts-order BOM or finish a PCB. Firmware execution and
serial/I2C/SPI debugging are separate missing maker workflows. These should be
explicit positioning decisions instead of an implication that all hobbyist
projects fit ELab today.

### Adoption is still harder than trying the established products

Current public distribution is source-oriented. Native qualification is scoped
to Apple Silicon/macOS, and there is no newly published signed/notarized public
installer containing all current functionality. Browser competitors remove
installation friction; iCircuit has store distribution and a wider platform
footprint. Open native checks, installation/accessibility and actual-user gates
remain separately documented.
[Current README](../README.md),
[historical acceptance scope](../docs/validation/v1-acceptance/README.md).

The root also has no ELab LICENSE file. Public source visibility should not be
marketed as a settled open-source licensing decision. GitHub's guidance explains
the distinction; selecting a license and checking notices is an owner decision.
[GitHub licensing guidance](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/licensing-a-repository).

## Fresh ELab reference smoke and the limits of the original numbers

A new headless session reports:

| Check | Current observation |
| --- | --- |
| Catalog | 61 kinds / 87 examples / 226 commands |
| Default LED circuit, 1 kΩ | 2.916796577 mA |
| Default LED circuit, 2 kΩ | 1.476723549 mA |
| Ideal 10 kΩ / 1 µF RC, at 10 ms | 3.160602794 V |
| Default NPN example | Saturation with its high input, cutoff at zero input |
| Cleanup | Shared shutdown completed and the headless process exited normally |

The LED points align with the original SPICE-pair reference values at their
reported precision; the exact RC case follows its analytical law. This checks
these supported workflows. It does not establish overall engine equivalence.
The RC source edge differs from the original finite-ramp SPICE fixture, and the
default ELab NPN circuit includes an LED unlike the original switch comparison;
NPN voltages are deliberately not used for a cross-product accuracy ranking.
Likewise, the original different iCircuit/Falstad LED currents reflected different
models and were not evidence of inferior accuracy.
[Fresh results](130-current-elab-smoke.json),
[original measurement qualifications](hands-on/measurements.md).

## Recommended priorities

These are research-backed product recommendations, not an implementation order
already authorized by this comparison request.

1. **Prove the effortless workflow and finish the public-entry path.** Use the
   existing first-use protocol for draw LED → change value → plot RC → measure →
   save/reopen, recording completion/help/errors and input-to-visible behavior.
   Repair observed clipping/discovery problems. Provide a versioned ready-to-run
   build with accurate signing/platform status. Success is people completing
   the tasks unaided, not another added component family.
2. **Make the agent experience straightforward.** Provide a supported MCP adapter
   or equally simple client setup over the existing guarded commands, with
   reproducible build/measure/compare examples and bounded result reads. Measure
   completion without screen automation and verify stale edits remain refused.
   Avoid treating a chat surface or command-count claim as the acceptance test.
3. **Qualify a more useful circuit envelope.** Use actual circuits that currently
   hit per-device walls—four discrete MOSFETs, six-capacitor tone/filter networks,
   additional amplifier stages—as specific regression/capacity targets. Preserve
   correctness, cancellation and editable/savable refusals while extending support.
4. **Create a credible real-device/model path.** Specify a supported model subset,
   explicit pin mapping, provenance, limitations and matched reference cases.
   Evaluate guided vendor-model support or a trusted simulator handoff; validate
   chosen BJT/MOSFET/op-amp cases rather than promising arbitrary SPICE compatibility.
5. **Let a successful experiment leave ELab intact.** Define a bounded KiCad
   schematic/netlist handoff, identifying unassigned footprints/models, plus useful
   parts-list and experiment exports. Keep MCU/HDL development as an explicit
   subsequent product choice; its importance is clear, but it is a different
   scale of engineering than another small generic device.

## Product conclusion

The strongest near-term claim is **an approachable local electronics lab that a
person and their agent can operate together, with inspectable, reproducible
experiments**. Our current implementation makes that plausible.

An overall easiest/best claim is not established. To win against iCircuit or
Falstad, we need an easier demonstrated first experiment and fewer practical
walls. To serve users growing toward hardware, we need real-device/model and
handoff paths. To compete on agents, we need the ease of connection users now see
in tools such as Flux while retaining our direct local semantics. Those outcomes
matter more now than expanding the generic catalog simply to raise its count.

## Source limitations

Competitor descriptions above are first-party capability evidence, not fresh
usability or accuracy tests. No customer-satisfaction percentage, overall quality
score or market-share estimate is asserted. iCircuit offline availability is not
unique to ELab; model/library counts are not directly comparable; automation in
other tools is not assumed absent because its contracts differ.

The Flux MCP documentation URL appeared in current search results but returned
404 on direct opening; its accessible official October and August articles
independently establish the vendor's integration claim, not an operational test
of the service. A guessed obsolete Fusion landing URL also returned 404; the
cited current product/help pages were accessible. Historical Falstad overview
text still says Java applet, so its legacy error restrictions are not treated as
proof of current CircuitJS restrictions. No application/source/solver or release
bytes were changed for this research; no new external-provider review was used.
