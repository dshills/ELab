# ELab product requirements

**Status:** Draft 0.3 for product and engineering planning  
**Date:** October 13, 2026  
**Audience:** Product, design, and engineering  
**Working product name:** ELab

ELab is an electronics workbench for hobbyists and makers. Its first release lets someone draw a small schematic, see it simulate as the circuit becomes valid, change values, understand the result, and save a reproducible experiment. **Live simulation and easy editing are the core product.** Numerical feedback, understandable errors, and visible model assumptions make that experience useful beyond a demonstration.

**Complete agent operability is also a core requirement.** Every application capability available to a human must be available programmatically to an authorized AI agent, whether built into ELab or connected externally. An agent must be able to complete the entire workflow without relying on mouse automation, screenshots, or a person operating a dialog.

This PRD defines a deliberately narrow MVP. It draws on the [research synthesis](01-findings.md), [live simulation contract](03-live-simulation.md), and [hands-on comparison](hands-on/README.md). Full user/agent capability parity is an explicit product-owner requirement added October 12, 2026. The product owner selected **Vellum as the UI framework** on October 13, 2026. Other requirements and numerical targets below are proposed product decisions, not capabilities already implemented or performance established by the research. ELab is a working name; the solver and external agent transport remain open engineering decisions.

### Selected UI framework

Build the desktop UI in Go using Vellum and its native Cogent driver, with macOS as the first validation target. Use Vellum's public application, UI, command, task, drawing, and workbench APIs where they satisfy the product contract. Keep circuit data, simulation, persistence, and the shared semantic command/query contract usable without a native window or GPU. UI controls, built-in agents, and external agents must all reach the same application behavior.

Vellum's existing local automation provides a useful transport candidate, but does not by itself satisfy ELab's complete agent contract. ELab must supply capability schemas, job/result access, retry deduplication, atomic edit batches, structured decisions, and headless lifecycle support. Framework selection does not waive accessibility, performance, or agent-parity acceptance. See the [Vellum adoption and integration decision](08-vellum-integration.md) for inspected sources, responsibilities, limitations, and the first implementation gate.

## 1 Product outcome

A maker should be able to answer “What happens if I change this?” without setting up an analysis every time, and “Can I rely on this result?” without leaving the circuit to investigate hidden assumptions.

The first complete experience is:

**Draw → observe → adjust → measure → recover → save.**

Success means the user can build and explain a simple circuit, identify when a result is current, recover from an incorrect connection, and reopen the experiment. Enjoying an animation alone is insufficient. A successful simulation is not a claim that physical hardware has been validated.

## 2 Users and jobs

| User | Relevant starting point | Job ELab should serve |
|---|---|---|
| Primary: learning maker | Recognizes a battery, resistor, LED, and breadboard; may be new to schematics | Understand connections and the effect of changing a component without mastering simulator setup |
| Primary: practicing hobbyist | Builds small circuits and knows basic voltage/current relationships | Sketch and check an idea quickly, make exact measurements, and keep a usable record |
| Secondary: growing maker | Begins using named parts, microcontrollers, or PCB tools | Preserve work and extend it into deeper analysis and other tools as ELab develops |
| Agent acting for a maker | Built-in or external agent with authorized access to the application | Discover available operations, inspect state, construct and modify circuits, simulate, diagnose, measure, and save using the same capabilities as the maker |

The MVP serves small analog experiments. It does not promise professional signoff, arbitrary circuit simulation, or coverage of every maker project. A person whose immediate job is firmware debugging or PCB production will need another tool in the first release.

## 3 Evidence informing the requirements

| Research observation | Product decision | Evidence |
|---|---|---|
| iCircuit and Falstad updated LED current after a resistor edit without a separate Run action | Live simulation is enabled by default when a circuit is valid | [iCircuit steps 1–2](hands-on/findings.md#icircuit), [Falstad steps 5–6](hands-on/findings.md#falstad-circuitjs) |
| Plotting could be reached from a component or measurement; some controls required discovery | Use explicit Plot voltage and Plot current actions beside readings | [Hands-on scope workflows](hands-on/findings.md) |
| KiCad's editable cursor time and LTspice's saved measurements supported precise checks | Include a small measurement layer in the same workspace | [RC and transistor measurements](hands-on/measurements.md) |
| Different LED models gave different currents; iCircuit's bandwidth setting changed displayed RC results | Expose model identity, simulation status, and resolution limits | [Model and resolution results](hands-on/measurements.md) |
| KiCad put operating-point readings onto the schematic | Keep measurements associated with visible nodes and components | [KiCad step 13](hands-on/findings.md#kicad) |
| The desk study found editing, model setup, and project ownership important across products | Treat drawing, recovery, and portable files as MVP requirements | [Research priorities](01-findings.md), [feature inventory](04-feature-inventory.md) |

The hands-on study used prepared circuits and agent-operated workflows. It did not establish how novices draw from blank or how quickly they learn the controls. The MVP must validate those behaviors with people; no competitor ranking or preference percentage is implied.

## 4 Product principles

1. **Keep the circuit central.** Editing, readings, errors, and plots remain visibly connected to the schematic.
2. **Make the next action discoverable.** Core operations have visible controls and keyboard equivalents; shortcuts and context menus supplement them.
3. **Make incomplete work comfortable.** A missing wire is a normal drawing state. Explain it without interrupting every edit.
4. **Keep time and results honest.** Show when a result is running, paused, stale, restarting, or failed, and distinguish simulated time from wall time.
5. **Reveal depth when needed.** Exact parameters and model details are available without occupying the initial workspace.
6. **Preserve the experiment.** Saving includes the circuit's dependencies and analysis settings. Reopening must not depend on a remote service or a changing library.
7. **Give users and agents the same capabilities.** The UI and agent interfaces use a shared application command/query contract. A feature is incomplete until an agent can discover, invoke, inspect, and recover from it programmatically.

## 5 MVP boundary

P0 means required for MVP acceptance. P1 means a candidate for the next release after the core workflow is validated. Later means explicitly outside the initial scope. Cutting scope should reduce breadth before weakening result validity, editing recovery, or file ownership.

### Included in the MVP

| Area | P0 boundary |
|---|---|
| UI foundation | Vellum Go UI with its native Cogent driver; macOS first; domain and agent workflows runnable without the native driver |
| Workspace | One schematic sheet, searchable component tray, contextual inspector, compact simulation status, optional docked scope |
| Editing | Place, connect, select, move, rotate, duplicate, delete, pan, zoom, fit, undo, redo, and exact value entry |
| Simulation | Automatic DC operating-point calculation and time-domain simulation for the curated library; pause, resume, restart, and bounded verification |
| Interaction | Value editing and bounded sliders; an interactive switch; voltage color and optional current animation paired with numerical readings |
| Measurement | Node voltage relative to ground; component voltage, current, and power where supported; up to four simultaneous scope traces; cursors and interval extrema |
| Guidance | Three working examples, contextual explanations, connection/model diagnostics, and example reset |
| Persistence | Local project save/open, crash recovery, a self-contained documented project format, saved experiment settings and result snapshots |
| Export | Native project file, schematic SVG, and CSV for a bounded result with units and metadata |
| Accessibility | Core workflow available with keyboard, text alternatives for electrical state, optional animation, and accessible component/property navigation |
| Agent operation | Complete, documented, versioned access to all application commands, queries, settings, and results for built-in and external clients; interactive and headless operation; no GUI-only steps |

### Curated component library

Ship generic, simulation-ready parts with tested pin mappings and inspectable parameters. Search by familiar terms such as “battery,” “LED,” and “square wave” as well as technical names.

| Component family | Required behavior |
|---|---|
| Resistor and potentiometer | Resistance and bounded wiper control; explicit pin identity |
| Capacitor | Capacitance and configurable initial voltage; stored-state behavior documented |
| DC voltage source | Exact voltage; battery search alias; clearly labeled ideal source without battery discharge behavior |
| Sine and pulse voltage sources | Amplitude/levels, offset where applicable, frequency/period, duty, and finite pulse rise/fall times |
| Ground reference | Explicit common voltage reference and understandable missing-ground diagnosis |
| SPST switch | Open/closed state with documented finite on/off resistance; actuation during live simulation |
| Diode and red LED | Generic nonlinear models; forward behavior and reverse/model limits visible; LED brightness presented as illustrative |
| NPN transistor | Generic nonlinear model with named base, collector, and emitter pins; readable state explanation where supported |

Wire, junction, and net labels are schematic primitives. Probes are measurement controls and do not load the circuit. Every shipped model needs a version, parameter defaults, pin map, known limitations, and a regression fixture. The MVP does not need a catalog of purchasable parts or editable custom models.

### Deferred scope

| Priority | Capability | Reason for deferral |
|---|---|---|
| P1 | Schematic output and printing | Extend existing SVG output with PDF/image output and native printing; share paper, scale, margin and content settings with agents |
| P1 | Validated KiCad schematic handoff for a supported subset | Preserve a path to hardware after the core workflow is proven; mapping and round-trip expectations need their own specification |
| P1 | Vendor SPICE model import with guided pin mapping | Important for real devices, but broad compatibility would expand the MVP substantially |
| P1 | AC analysis, DC sweeps, reusable subcircuits, run comparisons | Useful growth tools after the first measurement workflow is reliable |
| P1 | Expanded parts including op-amps, MOSFETs, inductors, and selected ICs | Each introduces additional model and numerical validation obligations |
| Later | MCU/firmware simulation, digital buses, sensor/peripheral emulation | A separate maker workflow with substantial support boundaries |
| Later | PCB layout, footprints, routing, manufacturing ordering, enclosure integration | The MVP is an experimental schematic workbench |
| Later | Breadboard/3D views, cloud accounts, hosted sharing, collaboration | Validate need before adding another representation or service |
| Later | Packaged AI circuit-generation assistant, automatic design certification, plugins, enterprise controls | A shipped conversational assistant and autonomous design behavior are separate from the required MVP agent interface; external agents can already construct circuits through that interface |

Native project exchange is in scope; importing arbitrary KiCad, LTspice, or other proprietary projects is not. No universal SPICE compatibility claim is permitted.

## 6 Primary hobbyist workflow

### Start a circuit

The start surface offers **New circuit** and three examples: LED and resistor, RC charging, and transistor switch. Neither account creation nor a setup wizard blocks entry. Each example includes a short question, an expected qualitative behavior, and one visible model limitation. Opening an example creates an editable copy; Reset example restores the starting circuit.

### Draw and connect

The user searches for parts, places them, and draws wires between highlighted terminals. Move and rotate controls are visible on selection. A connected move preserves the net unless the user explicitly detaches it. Junction dots indicate electrical joins; crossing wires do not connect without an explicit junction. Placement previews and canceled gestures do not mutate the circuit.

The incomplete LED circuit displays a quiet explanation such as “Connect the LED cathode to ground.” Once its topology is valid, simulation begins automatically. Reference designators are assigned automatically; beginners do not need a separate annotation step.

### Observe and experiment

Selecting the LED shows its current, voltage, power, model identity, and a brief explanation of what is being approximated. Selecting the resistor exposes its exact value and an optional slider. Changing `1k` to `2k` starts an updated simulation without pressing Run. The UI states when the experiment restarts and when the displayed result belongs to the updated circuit.

### Measure a waveform

Selecting the RC output or capacitor offers **Plot voltage** directly. A trace appears in the docked scope, with units and a clear relationship to the selected object. The user can enter a cursor time such as `10 ms`. Selecting the trace highlights its source on the circuit. A time-window control and a measure-interval selection distinguish startup from later cycles.

### Recover from a mistake

Deleting a wire marks the affected connection and changes simulation status to Incomplete. Existing readings are labeled as previous results and stop appearing live. Undo restores the wire and automatically restarts the valid circuit. A solver failure supplies an actionable explanation when known, keeps the circuit editable, and offers technical details separately.

### Save and return

Save project preserves the schematic, embedded model dependencies, input values, initial conditions, simulation settings, and selected probes. **Verify** runs a specified simulated duration from explicit initial conditions and saves a result snapshot. Reopening shows saved results as recorded results; live simulation starts a new run, never silently resumes a previous physical state.

### Work with an agent

A maker can let an external agent create the LED circuit, adjust its resistor, plot current, explain a diagnostic, and save the experiment. The agent discovers commands and part schemas, reads structured state, and invokes the same operations used by the UI. In an interactive session, its committed changes appear immediately on the canvas and in the shared undo history. The maker can continue editing; the agent detects intervening changes before applying an outdated action.

The same workflow can run without a visible application window. The agent can read numerical results, request a rendered schematic or plot, save the project, and return an artifact the maker can open. Built-in agents, when present, use the same public contract and do not receive a separate, more capable path into the application. The MVP must support both client types; it need not bundle a model provider or conversational assistant.

## 7 Functional requirements and acceptance

All requirements in this section are P0. IDs are stable references for implementation and validation.

| ID | Requirement | Acceptance condition |
|---|---|---|
| UI-01 | Vellum application foundation | Desktop application consumes Vellum's public APIs from an independent Go application; both visible and headless clients use the same semantic operations; a recorded, reproducible SDK/dependency baseline builds on a clean supported development environment |
| EDIT-01 | Discoverable circuit construction | A user can construct the LED circuit from blank using visible controls without entering SPICE syntax or using a context menu |
| EDIT-02 | Predictable connectivity | Tests distinguish endpoints, explicit junctions, crossings, and near misses; moving a connected component preserves its net; invalid placement remains a preview until committed |
| EDIT-03 | Exact, reversible edits | `1k`, `2.2k`, `10 ms`, and `1 µF` parse into canonical values with visible units; ambiguous or invalid input is explained without changing the previous valid value; one undo restores each committed edit, including a complete slider drag |
| EDIT-04 | Consistent navigation and selection | Mouse and keyboard can select parts/nets, inspect properties, pan, zoom, fit, and escape an unfinished operation without altering the circuit |
| SIM-01 | Automatic analysis | Completing or electrically editing a valid supported circuit triggers calculation without a separate Run step; drawing remains responsive while calculating |
| SIM-02 | Explicit state semantics | All actions follow the state contract below; traces from different electrical revisions are never joined into an apparently continuous result |
| SIM-03 | Honest status and time | Status uses text as well as color; simulated time and playback speed are visible; failure or lack of progress never becomes an apparently valid zero reading |
| SIM-04 | Appropriate numerical resolution | Default analysis passes the electrical acceptance suite; insufficient resolution, solver failure, or unsupported conditions are shown explicitly; the app never silently swaps to a simpler model to maintain speed |
| SIM-05 | Bounded verification | Verify accepts duration and initial conditions through ordinary controls, records engine/model/settings versions, and produces a reproducible result for the supported suite |
| PART-01 | Ready parts and visible assumptions | Every palette part has a working model and correct pin mapping; its inspector identifies generic/ideal behavior and model limitations before a user relies on it |
| VIEW-01 | Measurements on the circuit | Selected and pinned readings expose units, sign convention, reference direction, and simulation status; a stale pin is visibly stale |
| VIEW-02 | Optional explanatory animation | Animation can be disabled without losing information; polarity/current direction remains available numerically; LED brightness and moving dots are labeled as illustrative rather than calibrated physical rendering |
| SCOPE-01 | Discoverable plotting | A selected node/component has labeled plot actions; up to four traces retain labels, units, and source associations; removing a source marks its trace unavailable |
| SCOPE-02 | Precise and bounded measurement | Users can type cursor times, read values/deltas, and obtain min/max over a specified interval; the measurement shows its interval and does not silently include earlier startup history |
| HELP-01 | Actionable recovery | Missing reference, disconnected required terminal, invalid parameter, missing/corrupt model, and nonconvergence each produce a relevant explanation; a known affected object is highlighted; no modal interrupts a temporary wire gesture |
| HELP-02 | Working examples | All three examples open, run, explain one limitation, accept changes, and reset without modifying the original example |
| FILE-01 | Complete local ownership | A project exported on one supported installation opens offline on a clean installation without fetching a model; unsupported newer formats open read-only when possible or fail clearly without rewriting the file |
| FILE-02 | Reliable saves and recovery | Writes are atomic; failed saves preserve the last valid project; recovery offers the newest complete autosave without replacing an explicitly saved file silently |
| FILE-03 | Useful output | SVG represents the visible schematic with readable labels; bounded-result CSV includes time, signal names, units, and result identity; exports preserve the source project |
| ACCESS-01 | Nonvisual and keyboard alternatives | Component search, placement, connection by named terminal, value editing, plotting, save, and recovery are operable with keyboard and semantic controls; core labels/readings are checked with a screen reader on the pilot platform |
| AGENT-01 | Complete capability parity | Every shipped user command, function, setting, and observable result has an equivalent documented agent operation; release coverage is 100%, including application/project lifecycle, editing, view controls, simulation, measurement, diagnostics, recovery, and export |
| AGENT-02 | Discoverable structured contract | Clients can enumerate supported capabilities and versions, obtain typed input/output schemas, units, defaults, preconditions, and error codes, and inspect application/project state by stable IDs without screen scraping |
| AGENT-03 | Interactive and headless execution | A reference external client completes all three circuit workflows from an empty project without UI input; the same command sequences work through the in-process client interface and produce equivalent project state and numerical results |
| AGENT-04 | Observable asynchronous operations | Every long-running operation returns an operation ID, exposes progress/completion/error/cancellation, and identifies the input revision and resulting run/artifacts; accepted work is distinguishable from completed work |
| AGENT-05 | Reliable edits and shared recovery | Mutations support revision preconditions and retry deduplication; atomic edit batches commit fully or leave state unchanged; successful batches appear as named undoable actions and cannot silently overwrite intervening user changes |
| AGENT-06 | Shared behavior and visible control | User, built-in, and external clients share validation, units, model limits, simulation state rules, and authorized capabilities; interactive agent changes are visible, attributed, and recoverable through the ordinary undo/recovery controls |

### Agent interface contract

The application must expose semantic operations, not merely keyboard shortcuts or simulated clicks. Coverage includes creating/opening/closing projects, searching and inspecting libraries, placing and connecting components, exact value edits and live input events, layout and selection, scope configuration, cursor measurements, solver settings, preferences, examples, diagnostics, undo/redo, save/recovery, verification, rendering, and every available export. Future features inherit this requirement when they ship.

The user interface and agent adapters must invoke the same application command/query layer. This does not require exposing private implementation helpers or raw solver memory. It requires exposing every supported application behavior and its meaningful state, without a privileged internal-only capability or a human-only control path.

| Contract area | Required behavior |
|---|---|
| Connection and lifecycle | Document how an external process starts or attaches to ELab, discovers sessions/projects, and closes its session. Local headless operation must not require an AI provider account or a visible window |
| Discovery | Publish a versioned capability catalog and machine-readable schemas with descriptions and examples. Return explicit unsupported-version/capability responses rather than silently ignoring a request |
| State inspection | Return connectivity, component/model details, properties, selection/view state, diagnostics, simulation status, available measurements, and artifact references as structured data with units and stable identities |
| Command results | Return command ID, outcome, affected objects, applicable state revision, and structured errors. Invalid input must preserve the prior valid state just as it does in the UI |
| Events and jobs | Provide structured changes to project state, diagnostics, simulation state, and job completion. Allow bounded queries for waveform samples; clients must be able to resynchronize from a snapshot after disconnecting |
| Concurrent editing | Reject a stale mutation with the current revision and conflict information. A retry with the same request key must not duplicate a component, edit, run, or export. Project changes and session/view changes have appropriate version preconditions |
| Batches and undo | Validate an atomic batch against its intended final state, commit once, and apply the ordinary simulation restart rules once. Log the actor and affected revision; grouped undo must detect intervening edits instead of blindly reversing another actor's work |
| Dialogs and decisions | Any application decision normally collected by a dialog must also accept structured input or return a structured decision request that an authorized agent can resolve. Ordinary authorized work must not require a GUI click |
| Results and rendering | Provide numerical values, waveforms, diagnostics, snapshot metadata, and saved artifacts directly. Rendered views are available for presentation, but are not the only way to learn what happened |

Publish a supported external transport/API and a runnable reference client with the MVP. Choose the transport during Gate A; the requirement does not depend on a particular agent framework, protocol, or model vendor. If multiple adapters are offered, they must preserve the same capability coverage and semantics. Access uses the user's authorized application/project context; agent support does not imply publishing an unauthenticated network service. Connecting a remote AI provider and transmitting project data are separate, explicit user choices.

## 8 Live simulation state contract

**Live** means automatic analysis and visible feedback after committed changes. It does not promise simulation progresses at wall-clock speed or that every edit preserves stored energy.

New transient runs default to zero capacitor voltage and time zero at the configured source phase. Users can change these initial conditions in experiment settings. A DC operating-point result is labeled as a steady-state calculation; it must not silently replace the requested transient startup state. Live meters show the current transient sample, while recorded operating-point readings remain separately identified.

### Status shown to the user

| Status | Meaning and required behavior |
|---|---|
| Incomplete | Required connectivity/reference is missing; identify the gap and do not produce new results |
| Updating | A valid change is being checked or solved; previous results remain clearly labeled with their prior revision |
| Running | Current valid circuit is advancing in simulated time; displayed samples identify this run |
| Paused | Simulated time is frozen; measurements remain inspectable and explicitly paused |
| Error | Validation or solving failed; preserve work and distinguish the failure from an electrical result |
| Unsupported | The imported project or requested operation exceeds supported models/features; explain the boundary without inventing results |

Stale is a property of a result, not a competing circuit status. Recorded verification results are always labeled Recorded, with their circuit revision and settings; editing the circuit does not erase them or relabel them as current.

### Effects of user actions

| Action | State and history behavior |
|---|---|
| Move, rotate without changing connectivity, rename, pan, or zoom | Preserve simulated state; update presentation only |
| Commit topology, model, source-definition, resistor, or capacitor value change while running | Invalidate current result, validate, and restart from configured initial conditions; begin a new trace segment/run with a visible restart marker |
| Drag a component-value slider | Acknowledge the preview immediately; coalesce changes while dragging, visibly mark results updating, and solve the final accepted value; do not replay an obsolete queue of values |
| Toggle the live switch | Apply an event at the next accepted simulation step, retain capacitor state, and mark the event on the trace; this is input actuation, not rewiring |
| Pause, then edit an electrical parameter | Remain paused with stale results; Resume starts a new run from initial conditions after validation |
| Toggle a switch while paused | Show a pending input; apply it on Resume at the next simulation step; do not advance time while paused |
| Resume without an electrical edit | Continue stored state and simulated time |
| Restart | Start at time zero with configured initial conditions and current source/switch settings; clearly separate prior history |
| Undo or redo | Apply the corresponding electrical or presentation rule; never restore unlabelled solver memory from an earlier revision |
| Remove a plotted object | Retain any recorded snapshot, mark the live trace unavailable, and offer removal or reassignment |
| Reopen project | Restore configuration and recorded snapshots; begin a new live run from saved initial conditions, subject to saved paused preference |

The MVP intentionally restarts after most electrical edits. Preserving capacitor state through arbitrary topology changes is deferred. Automatic restart must remain visible and requires no confirmation dialog. A live capture may contain switch events; it is a recorded observation, not a claim that reopening resumes that history. Bounded Verify runs use a fixed captured configuration and exclude further live input changes; electrical edits cancel that run or create a new revision, without contaminating its result. Presentation changes do not invalidate the calculation.

## 9 Simulation trust and project data

Each project stores stable identities for components, terminals, nets, and probes; the symbol; model identity/version and permitted embedded dependency content; pin mapping; parameter values and units; initial conditions; source definitions; and analysis settings. Rename operations must not break references.

A result snapshot stores circuit revision, model/dependency hashes, engine version, solver settings, simulated interval, measurement definitions, and result data. Live captures additionally record input events and their simulated times. Project saving does not need to serialize internal solver memory or unlimited waveform history.

Live history uses a bounded rolling window. The scope identifies retained intervals and any display reduction; reducing plotted samples must preserve visible extrema and must not change the solver's integration step. CSV and saved captures state their sampling policy. Bounded verification checks duration/storage limits before starting and reports truncation or cancellation explicitly rather than presenting a partial run as complete.

The MVP uses only curated models that can be distributed with the product and retained in exported projects. A component inspector distinguishes electrical model terminals from physical package pins when applicable; absent physical part/footprint information is explicit. Generic models must not acquire an implied real part identity from a familiar symbol.

Diagnostics should distinguish an incomplete drawing, a physically unusual configuration, unsupported behavior, and numerical failure. Avoid claiming a circuit is “safe,” “build ready,” or physically accurate simply because it converges. Rating/thermal validation and physical hardware comparison are outside MVP claims.

## 10 Quality targets

These are proposed launch targets. Engineering must measure them on a recorded reference configuration before treating them as achievable commitments. Delivery is a native Vellum desktop/laptop application, validated first on macOS with mouse/trackpad and keyboard. This does not commit simultaneous Windows/Linux/mobile support; framework availability alone does not establish ELab platform support.

| Area | Proposed target and measurement |
|---|---|
| Supported size | At least 50 simulated components, 100 electrical nodes, and four traces using the curated library; publish the validated envelope and explain excess complexity without losing edit/save access |
| Editing feedback | 95th-percentile visible acknowledgment within 100 ms for supported edits; measure input event to visible acknowledgment, independently of solver completion |
| Live update | 95th-percentile time from accepted edit to the first valid updated sample within 500 ms on the three reference circuits; show Updating until then |
| Verification | Complete the reference 200 ms transient runs within 2 seconds on the locked reference configuration; this is wall time for a bounded test, not a universal real-time promise |
| Responsiveness | A slow or failed solve must not block navigation, cancel, save, or undo; benchmark these actions under a deliberately difficult fixture |
| Numerical checks | Meet the tolerances in the release suite below with default settings; tighten numerical resolution without materially changing accepted measurements |
| Recovery | Autosave complete committed changes within 5 seconds while storage is available; show save failure; fault-injection tests preserve the last successful manual save |
| Portability | Native save/open round-trip retains connectivity, values, models, probes, initial conditions, and recorded results across clean installations of the supported version |
| Privacy | No account or network dependency for the core workflow; project contents stay local unless the user explicitly exports or shares them; no mandatory telemetry |
| Accessibility | Status is understandable without color or motion; complete the core keyboard and screen-reader scenarios on the pilot platform; do not infer support from screenshots alone |
| Agent parity | 100% of shipped user capabilities covered by the public command/query catalog and conformance tests; no core workflow requires UI automation; both client paths obey the same numerical and state-contract acceptance criteria |

Reference hardware, OS, build, model versions, circuit files, cold/warm state, and repetitions must accompany performance results. If targets cannot be met, narrow the supported circuit envelope or revise the target explicitly; never conceal stale results to appear faster.

## 11 Release validation

### Electrical and workflow acceptance suite

The existing [circuit fixtures and protocol](hands-on/protocol.md) provide starting topologies. Port and validate them for ELab; their mere ability to load is not acceptance. Reference values must use ELab's exact pinned models and stimuli.

| Scenario | Required result |
|---|---|
| LED and resistor | Build from blank; measure current; change 1 kΩ to 2 kΩ and see a lower current automatically. With the specified educational diode model at 27 °C, the research reference is approximately 2.9168 mA at 1 kΩ. Require agreement within 1% against an independently computed or reference-solver result for the identical model; check KVL/KCL separately |
| RC charging | 10 kΩ, 1 µF, 0–5 V pulse. For an ideal step, Vout at 10 ms is 3.160602794 V; account for finite source rise time. Require error within max(1 mV, 0.1% of reference magnitude) at predefined samples, and verify approximately 33.464 mV/4.966536 V periodic extrema using the matching stimulus |
| NPN switch | Base through 10 kΩ, collector load 1 kΩ to 5 V. With `NPN(Is=1e-14 Bf=100)`, compare on/off states and transition timing against an independent reference. Research on-state reference is approximately 70.4 mV; proposed tolerance is max(1 mV, 1% of reference magnitude) |
| Live edit and pause | Exercise every state-contract row, including invalidating a paused circuit, undo, repeated slider edits, switch events, and deletion of a probe source; stale or cross-revision values are never presented as current |
| Fault recovery | Remove ground, miss a connection, enter an invalid value, corrupt a model dependency, and provoke nonconvergence. Verify truthful status, relevant explanation, retained work, and undo/recovery |
| Save and export | Reopen on a clean offline installation; rerun bounded verification; compare against the same tolerances; inspect native, SVG, and CSV output and recovery after interrupted writes |
| Capacity and accessibility | Run a valid 50-component fixture with four traces, plus node-limit boundary cases; evaluate latency, cancellation, bounded history, and the keyboard/screen-reader path |
| Agent end to end | From an empty session, an external client creates each reference circuit, inspects models/connectivity, changes values, runs and measures it, fixes an introduced fault, saves/exports, closes, and reopens it without GUI input; repeat through the in-process interface and compare semantic project state and results |
| Agent coverage and recovery | Check every registered user action/query against the public catalog; test repeated requests, atomic batch failure, user/agent revision conflicts, undo, job cancellation, disconnect/reconnect, stale readings, and structured dialog decisions; no missing command or result is waived because screenshots could supply a workaround |

Freeze independent reference outputs before acceptance testing. Where practical, use analytical solutions plus a separately configured solver, and check model pin mapping. Agreement between two engines alone is not proof of real-device accuracy. Timestep refinement must preserve the accepted result; the suite must include fast transitions that an attractive slow animation could hide.

### Formative hobbyist test

Recruit an initial group of 10 hobbyists: five new to schematic simulators and five with some circuit-building experience. This is a formative release gate, not a population estimate. Use a consistent device, counterbalance task order where practical, and record assistance as well as completion. Obtain consent for any recordings; manual observation is sufficient and does not require product telemetry.

| Outcome | Proposed gate |
|---|---|
| First useful circuit | At least 8/10 overall and 4/5 newcomers build the LED circuit from blank and obtain a changing current reading within five minutes without facilitator intervention |
| Meaningful experiment | At least 8/10 correctly predict or explain why increasing resistance reduces LED current after using the tool |
| First waveform | At least 8/10 plot capacitor voltage and enter a cursor time without being told where the plot action is |
| Recoverability | At least 8/10 repair an intentionally disconnected wire and undo an accidental deletion without losing work |
| Result interpretation | At least 9/10 distinguish Running, Paused, and a stale/failed result in prepared scenarios; every observed case of failure presented as valid requires investigation before release |
| Assumptions and ownership | At least 8/10 identify one relevant model limitation and save/reopen the experiment without missing dependencies |

Track task completion, first-success time, help requests, errors, explanation accuracy, and confidence alongside satisfaction. The product's primary success measure is **unassisted completion of the full draw–observe–adjust–measure–recover–save loop**, with counts reported by experience group. No user-baseline metric has yet been measured.

## 12 Delivery sequence

| Gate | Deliverable | Exit condition |
|---|---|---|
| A: feasibility | Vellum editor/solver spike and shared command/query contract with LED, RC, NPN, and deliberately broken circuits | Demonstrate automatic updates, truthful state, nonblocking edits, model identity, an external agent running a circuit, and plausible numerical/latency targets; validate native canvas input and headless operation, pin the SDK/dependencies, and select engine approach and external transport |
| B: complete vertical slice | New circuit through save/reopen, including a scope and recovery, through both UI and agent clients | All core workflows executable with the initial curated models; no handcrafted hidden setup or GUI-only step needed to demonstrate them |
| C: MVP hardening | Full curated library, verification snapshots, exports, accessibility, capacity, recovery, and agent conformance | Functional, electrical, and complete agent-parity acceptance suites pass on the recorded support configuration |
| D: hobbyist validation | Formative sessions and prioritized corrections | Proposed user gates met or explicitly revised with evidence; no unresolved issue that presents a failed/stale solve as a current valid result, corrupts connectivity, or loses saved work |

No calendar estimates are assigned until Gate A resolves engineering feasibility and team capacity. Passing the demonstration circuits is necessary but insufficient: invalid edits and ordinary recovery are release criteria, not optional polish.

## 13 Decisions and risks to resolve

| Decision or risk | MVP planning position | Resolution point |
|---|---|---|
| UI framework and delivery | Vellum selected by the product owner; native Go desktop UI using the Cogent driver; macOS pilot; core available offline without login | Decision recorded October 13, 2026; Gate A integration evidence and Gate C packaging/accessibility acceptance |
| Framework maturity | Inspected Vellum identifies as experimental alpha. Canvas interaction, semantic accessibility, SDK reproducibility, native dependency distribution, and target runtime behavior need ELab-specific evidence | Gate A feasibility and Gate C release checks; track framework changes explicitly without reducing the product contract |
| Solver and live architecture | Select against latency, model fidelity, restart semantics, portability, and distribution constraints. Do not presume two engines; a single engine is preferred if it satisfies the contract | Gate A technical decision with benchmark evidence |
| Agent integration | Full parity is required. Select external transport, schema versioning, session lifecycle, and headless packaging around the shared command/query contract; built-in assistant packaging remains deferred | Gate A technical decision and Gate C conformance tests |
| State resets may surprise users | Explicit restart markers and separated traces are required; arbitrary topology state preservation is deferred | Test RC edits with participants before Gate D |
| Component breadth may feel too narrow | Guarantee three complete jobs and every shipped model; broaden only after observing blocked real tasks | Review pilot task requests after Gate D |
| Generic models may invite overconfidence | Model identity and limitations are visible; physical build readiness is not claimed | Interpretation tests at Gates C and D |
| Drawing ease is not yet validated | First circuit must be built from blank in user tests, not supplied as an imported fixture | Gate B design iteration and Gate D test |
| Growth into physical hardware | Stable identities and portable project data now; scoped KiCad handoff specification later | P1 planning after core validation |
| Business model and pricing | Outside this PRD; must not remove MVP local ownership or require an account for basic experimentation | Separate product decision before distribution |

## 14 Traceability and scope changes

The broader [48-feature inventory](04-feature-inventory.md) remains a research catalog. This PRD promotes F01–F08, F10–F12, F16–F21, F26, F29–F30, F35–F36, and F45 into the core experience or its foundations, with deliberately limited breadth. For F18, discoverable default shortcuts are required; customizable keymaps are deferred. F24–F25 are narrowed to a curated generic part/model experience; purchasable-device and footprint management are deferred. Time-domain and operating-point portions of F31, and basic measurement portions of F32, enter MVP; frequency analysis and advanced comparisons remain P1.

Draft 0.2 adds AGENT-01 through AGENT-06 from the product owner's explicit requirement for complete built-in and external agent operation. This is a foundational MVP requirement, separate from F48's possible AI explanation or design-assistance experience. It applies to every shipped capability, including capabilities introduced after the MVP.

Draft 0.3 records the product owner's Vellum selection, adds UI-01, selects native desktop delivery, and updates the implementation gates. The [integration decision](08-vellum-integration.md) separates verified framework interfaces from proposed ELab work. This choice does not add PCB, 3D, cloud, or other scope merely because the framework supports related UI features.

Use [hands-on design opportunities](hands-on/design-opportunities.md) for the reasoning behind later candidates. A scope change must state which user job it serves, what existing requirement it displaces or extends, and how it will be validated. Adding a competitor feature by itself is not sufficient justification. A new user capability cannot ship without corresponding agent access and conformance coverage.

## Owner addition — schematic output and printing (October 14, 2026)

After the current UI testing round, resume electronics-focused work. Add schematic output and printing to the next-release backlog, extending FILE-03 without displacing the existing P0 acceptance gates. SVG output exists. The 0.48.0 checkpoint adds printable macOS PDF output and shared page settings; image output, native print preview and printer discovery/submission remain planned. See [PDF evidence and remaining scope](../docs/schematic-pdf.md).

The job is to share, document or print the circuit a hobbyist has drawn. Output must include the complete schematic, readable component IDs/values, terminals and wire routes. Support portrait/landscape, A4/Letter, margins, fit-to-page and explicit scale, plus a print preview. Default output should remain legible on white paper and in grayscale regardless of the app theme. Optional measurement overlays must be explicit and labeled as live or recorded. Export and printing must preserve the source project, simulation and dirty state, and work for incomplete schematics.

Users and agents must share document preparation, output settings, printer discovery, destination selection and print submission, including inspectable success/cancellation/failure outcomes. Saving a PDF and submitting a printer job are distinct operations. Validate representative LED, RC and transistor/flasher schematics through rendered PDF/image inspection, complete-bounds/label checks, matching user/agent output, and native print-preview/cancellation tests. Actual printer output requires a separate device test; a generated PDF does not prove physical printing. PCB/EDA handoff remains a separate candidate.
