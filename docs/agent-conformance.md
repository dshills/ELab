# Three reference workflows through every client path

`elab-conformance` runs the PRD's LED, RC charging and NPN switch workflows from empty owned sessions using the same public command/query contract. It compares the in-process client with external headless stdio, headless private socket and, optionally, the packaged native app's private socket. It requires no GUI input, accounts, provider credentials or network service.

```sh
GOPROXY=off ./scripts/build.sh
./build/elab-conformance --artifacts /absolute/path/to/new-results \
  --native-binary /absolute/path/to/ELab.app/Contents/MacOS/ELab
```

The artifact directory must be new; the harness refuses existing data. `--agent-binary` defaults to `build/elab-agent`. Omitting `--native-binary` runs the three portable paths. Native mode creates six new app processes with isolated recovery directories/endpoints, closes those sessions through `app.close`, and waits for their process cleanup. Pre-existing apps, sockets, bundles and projects are untouched. Each adapter has a two-minute overall bound. Processes/logs/artifacts are owned by this run; a failed run writes `passed:false` and returns a failing exit status.

Each circuit discovers capability schemas and part model/pin information, places named parts and connects explicit terminals. LED edits automatically reduce accepted current; RC compares the 10 ms step response and slower 20 kΩ response, then measures a two-channel scope interval; NPN uses the PRD’s resistor-only collector load and exercises cutoff, active and saturation with conservation checks and the approximately 70.4 mV on-state reference. The existing transistor-controlled LED family remains covered by its separate fixtures. Each circuit deletes a connected wire, requires diagnostics and stale readings, restores it by ordinary undo, and restarts fresh physics. Structured live measurements and a bounded Verify capture inspect the same sources. Verify captures are checked against the admitted graph/revisions, and their kept recorded rows/models/hashes remain part of saved projects.

Every workflow writes a project, schematic SVG and recorded CSV; RC also writes live scope CSV. It requires a clean saved state, closes, starts a fresh empty session, reopens, checks live recalculation and pause/zero-time startup, saves again and requires exact unnormalized document equality. This exercises saved recorded data without confusing it with live physical state.

Cross-adapter comparison requires the same ordered logical commands and parameters. Before mutations the client queries the current revision/document guard; those admission queries are excluded from logical workflow evidence. Native layout reports can legitimately race admission, so explicit conflict refusals retry with a fresh guard and a distinct request/key, at most eight times. A changed document generation or electrical revision stops retries. Other errors and transport failures never retry automatically. Tests inject these competing actions into real service sessions and check that a successful retry applies the requested edit once.

Only artifact path roots differ; repeated identical adjacent operation-status polls collapse because scheduling changes their count. Semantic projects compare exactly after explicit normalization of independent session creation timestamps/operation IDs/publication revisions. Original provenance remains in all saved files and report data. Graphs, values, pin mappings, model identities/parameters/dependency hashes, preferences, scope selection, settings, recorded rows and numerical results are preserved in the comparison. Measured SI values must be finite and agree within `1e-10 * max(1, abs(reference))`; missing workflows, commands, parameters, measurements or recorded samples fail. Negative tests deliberately alter these categories to check rejection.

The shared runner lives in `internal/conformance`; fixtures, numerical experiments, comparison logic and client adaptation have separate files. The CLI owns subprocess/pipe lifetime and uses `transport.RemoteStdio` for private sockets. Its stdout reader is independently owned so process Wait cannot discard the final close response. The standard build produces the tool; formatting/lint/race/vet checks include it.

This establishes the three-circuit execution requirement on this workstation. It does not establish exhaustive native action/preference coverage, independent SPICE validation, timestep refinement, visible latency, keyboard/focus/screen-reader usability, clean installation/distribution or participant acceptance. Those release requirements remain open. See the [PRD audit](../research/11-prd-implementation-audit.md).

[Final validation](validation/agent-conformance-validation.json) and [four-adapter report/artifacts](validation/agent-conformance-prd/report.json) record the exact tested binaries and results.

## Powered op-amp suite

`--opamp-suite` adds four owned-session experiments: the guided follower and inverting amplifier, an RC-buffered NPN LED driver, and two cascaded buffers driving an RC/MOSFET LED circuit. All construction, deletion/replacement, wiring, probes, restart/step, Verify and project save/reopen use existing public commands. Deleted component IDs remain reserved; the mixed circuits replace the removed sine source with a distinct VRC identity.

The sine circuits require 101 samples over 100 ms. Each mixed pulse circuit preserves all 101 grid samples plus pre/post observations at the two edges, for **105 rows**. RC values satisfy the existing max(1 mV, 0.1%) analytic envelope. Independent scalar bisection checks follower output KCL against the declared current law and resistor loads, including finite-gain/rail/output-resistance effects; NPN base voltage is an observed boundary condition, not an independent NPN reference. The two guided peaks retain the frozen 90-digit Decimal references.

The suite records **931 numerical checks per adapter**, complete graphs/hashes, signed result observations, every row and logical command. Comparison recomputes checks from the rows/peaks, validates graph hashes and sample/edge coverage, then compares normalized evidence. It rejects missing/altered model, case, row, graph, command or numerical evidence. Native/headless transports must match; repeated identical status polls are the only command-sequence normalization. Run/electrical revision provenance is zeroed only in compared result observations; saved projects retain their own metadata.

The [four-adapter evidence](validation/opamp-adapter-conformance.json) stores common numerical/graph data once and retains each adapter's original calls. Its encoding explains reconstruction; the reconstructed JSON object was checked against the complete raw report. This extends engineering qualification without claiming repeated performance, physical input/accessibility, device accuracy, universal topology coverage or application-release acceptance.

## Capacity suite

`--capacity-suite` builds three fresh circuits through existing public placement/wiring/value/probe/simulation/file commands: 51 parts/100 nodes, 64 parts/64 nodes, and 64 parts/60 nodes with four RC stages driving an NMOS LED. Each runs four channels over 200 ms at 1 ms outputs, retaining both sides of source edges. The RC/MOSFET case also runs a 0.5 ms capture. Independent series/divider and matrix-exponential references, plus coarse/fine comparisons, produce 4,900 checks per adapter. An independently generated Decimal eigensystem validates the matrix reference.

Each fixture saves and freshly reopens a project and exact retained recording, then exceeds one capacity bound, verifies refused Step preserves accepted state, moves/saves the drawing, and freshly reopens it with the earlier recording intact. Comparison recomputes graph/engine/count/settings/probe/row/edge/refinement/reference/boundary/reopen coverage. The paused lifecycle and unsupported compilation are checked separately. This is representative capacity and transport evidence; repeated maximum-capacity timing, exhaustive native inputs and participant acceptance remain open. See [integration details](capacitor-integration.md).

## Capacity timing suite

`--capacity-performance-suite` adds twenty fresh/twenty warm measurements for each of the three capacity fixtures. Every sample retains a full 205-row reference-checked capture and exact graph/model/settings/probe associations. The validator recomputes variants, increasing warm identities, numerical references, statistics, percentiles and target flags; it does not trust aggregate success. See [protocol and scope](capacity-performance.md). Construction/process launch and visible-frame latency are outside these backend measurements.

## Copied runtime without development files or IP networking

The [macOS runtime rehearsal](isolated-runtime.md) runs these suites using copied external app/agent processes under OS-enforced development-file and outbound-IP denial. The in-process baseline/controller remains unrestricted. Saved models/recordings reopen through the same public commands. This extends workstation portability evidence without establishing a clean OS/account or distribution qualification.
