# First LED implementation slice

This is the historical foundation record. The current app starts blank and includes schematic construction and a redesigned interface; see the [schematic drawing update](schematic-drawing.md).

**Implemented October 13, 2026.** ELab now has an independent Vellum desktop application and a portable headless application service. This completes the requested app foundation and simple LED experiment. Gate A in the PRD remains in progress because general circuit construction, transient simulation, and the RC/NPN fixtures still need implementation.

## Working behavior

The desktop starts with a 5 V ideal DC source, 1 kΩ resistor, generic red LED, and ground. A committed value or connection change automatically calculates the operating point on a cancellable worker. The window exposes current, voltage, power, component information, connection diagnostics, pause/resume/recalculate, reset, undo/redo, selection, and viewport navigation.

The solver uses `Is=1e-20 A`, `N=2`, `Rs=1 Ω`, and 27 °C, matching the research fixture. It solves the monotonic series-circuit equation by bounded bisection and reports model/engine identity, node voltages, KVL residual, and input/run revisions. It supports this series DC topology; it is not the eventual general transient solver.

All semantic UI edits and canvas navigation invoke `workbench.Service.Dispatch`. In-process clients and both agent transports call the same dispatcher. Every currently shipped semantic operation appears in the public versioned catalog, with structured state and input/output schemas. Mutations require revision preconditions and idempotency keys; successful electrical edits share one history. Old solver results cannot overwrite a newer edit, a paused state, or a closed session.

Native code is isolated in the `desktop` module. The root module has no third-party dependencies and can run the agent session without Vellum, a window, or a GPU. The local socket adapter is explicit, has bounded admission and private credentials, and the reference client can attach to a live desktop session. Document/file operations will need the same separation in the next slice.

## Validation

Validation on macOS 26.6.2, darwin/arm64, Go 1.27.2, using the [recorded Vellum baseline](vellum-baseline.json):

| Check | Observed result |
|---|---|
| Root and desktop unit tests with race detection | Passed |
| Root and desktop `go vet` | Passed |
| Native app build | Passed; linker emits an upstream duplicate `-lobjc` warning |
| Native smoke | GPU schematic pixels verified; the actual UI resistance callback updated the live calculation |
| Model-matched electrical reference | 1 kΩ: 2.916796577 mA; 2 kΩ: 1.476723549 mA |
| Electrical bounds and conservation | Tests cover 0, 0.2, 5, and 12 V with 10, 1000, 2000, and 100000 Ω; check KVL and total power |
| UI/agent equivalence | UI value callback and external-style dispatcher sequence produce identical circuit values and numerical results |
| Recovery and concurrency | Tests cover invalid values, disconnect/undo, pause/edit/resume, obsolete worker results, stale revision rejection, retry-key reuse, and detached snapshots |
| Agent transport | Real private socket and stdio tests pass, including schema discovery, credentials, descriptor permissions, and ordinary cleanup |
| Interactive UI inspection | Visible 1 kΩ/2 kΩ updates, incomplete/stale feedback, undo recovery, and paused edits/resume confirmed; semantic controls/readings appear in the accessibility tree |
| External agent controlling the visible app | Complete example/reset/edit/retry/disconnect/undo/pause/edit/resume/selection/view sequence passed; updated selection, model inspector, and agent attribution appeared in the actual window |

Tests do not establish a complete screen-reader experience, latency percentiles, clean-machine packaging, or general solver performance. The native smoke makes a bounded rendering/electrical assertion, not a complete screenshot or usability audit. See [agent protocol](agent-api.md) and the runnable `scripts/agent-demo.py` for repeatable headless/desktop command sequences.

Saved evidence: [headless workflow](validation/headless-led.json), [desktop agent workflow](validation/desktop-agent-led.json), and [verified desktop screenshot](validation/led-desktop.jpg). The bundled application is a private local development build.

## Next implementation work

1. Replace the fixed topology with a graph of components, terminals, wires, and explicit nets; support placing and connecting the LED circuit from blank with the same commands.
2. Add exact value-entry and gesture transaction behavior across UI and agents, preserving named undo actions and atomic batches.
3. Select and integrate the general solver against the PRD's LED, RC, NPN, and invalid-circuit acceptance suite; introduce genuine transient time and scope data.
4. Add window-independent project persistence, recovery, portable model dependencies, and exports through the shared command layer.
5. Extend schema/job/event support and UI/agent conformance with each shipped capability; perform accessibility, capacity, latency, and packaged-runtime acceptance.

The full [PRD](../research/07-product-requirements.md) remains the MVP acceptance contract.

## Selectable probes and cursors

The [scope implementation](scope-probes.md) adds private immutable full-probe frames to retained transient samples and exports only the selected channel arrays. Stable component/pin descriptors cover voltages and signed branch/device currents across RC, NPN pulse, and mixed engines. Four typed scope commands share native/agent admission, revision and retry behavior. Session-only selection/cursor state preserves electrical time and document history; measurements use recorded samples with explicit rolling/stale validity. Native scope rendering, plot gestures, and inspector controls use these same commands. Discovery and all socket responses remain bounded to 1 MiB.

Current follow-on scope and validation: [ten hobbyist workflow improvements](../research/09-next-ten-implementation.md). These extend Gate B/C; the full MVP gates remain open.
