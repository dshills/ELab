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

Cross-adapter comparison requires the same ordered commands and parameters. Only artifact path roots differ; repeated identical adjacent operation-status polls collapse because scheduling changes their count. Semantic projects compare exactly after explicit normalization of independent session creation timestamps/operation IDs/publication revisions. Original provenance remains in all saved files and report data. Graphs, values, pin mappings, model identities/parameters/dependency hashes, preferences, scope selection, settings, recorded rows and numerical results are preserved in the comparison. Measured SI values must be finite and agree within `1e-10 * max(1, abs(reference))`; missing workflows, commands, parameters, measurements or recorded samples fail. Negative tests deliberately alter these categories to check rejection.

The shared runner lives in `internal/conformance`; fixtures, numerical experiments, comparison logic and client adaptation have separate files. The CLI owns subprocess/pipe lifetime and uses `transport.RemoteStdio` for private sockets. Its stdout reader is independently owned so process Wait cannot discard the final close response. The standard build produces the tool; formatting/lint/race/vet checks include it.

This establishes the three-circuit execution requirement on this workstation. It does not establish exhaustive native action/preference coverage, independent SPICE validation, timestep refinement, visible latency, keyboard/focus/screen-reader usability, clean installation/distribution or participant acceptance. Those release requirements remain open. See the [PRD audit](../research/11-prd-implementation-audit.md).

[Final validation](validation/agent-conformance-validation.json) and [four-adapter report/artifacts](validation/agent-conformance-prd/report.json) record the exact tested binaries and results.
