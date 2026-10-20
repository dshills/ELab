# Current integration method in agent discovery

**NUMERICS-DISCOVERY-01:** detailed `component.place` and `component.set`
discovery in preserved **0.162.0 /179** says network circuits retain backward
Euler. Fresh mixed RC/NPN, general capacitor and waveform results actually name
their current SDIRK2 engines. The numerical implementation is already correct;
the agent-facing explanation is obsolete.

The source correction changes only the two filtered
`x-network-numerics.time_integration` strings. They now identify two-stage
SDIRK2 and its three current engine IDs, the separate single-RC exact exponential
and algebraic NPN phase paths, and the existing inductor Radau IIA path. Engine
names come from the existing constants. Historical backward-Euler captures
retain their engine and rows on import/export; discovery no longer presents
their method as the current live method.

No device law, solver, tolerance, work/time limit, file version, output schema,
ordinary catalog, command ID, handler, native binding or SDK changes. This is a
source-qualified metadata correction. **/179 is preserved with the old text;
clean replacement packaging remains pending.** No promotion follows.

## Evidence

- The final new regression fails against original production metadata in six
  live-family subtests. Both value-command descriptions must identify the
  observed engine/method and distinguish historical captures. A separate guard
  keeps detailed guidance out of the bounded ordinary catalog. Corrected focused
  race checks pass. This does not mean every original family description was
  individually wrong: the false capacitor statement and missing current-family
  context caused the failures.
- Exact copied /179 and corrected development agents each observe six families
  at time zero and after a 1 ms Step: flasher, high-pass RC, single RC,
  resistor-load NPN pulse, RLC and finite-pulse RC. All twelve complete
  result/mixed/transient/scope comparisons pass under the established comparator
  (independent run/electrical IDs excluded; relative 1e-9/absolute 1e-12 numeric
  comparison). Graph/settings and configured advice are exact. No numerical
  changes are found.
- Full ordinary discovery is identical. Removing just the one corrected string
  from each detailed response makes the **entire** old/new responses identical,
  including all other input annotations, output schemas and schema registry.
  Discovery preserves the complete before/after snapshot in each owned session.
- The standard `./scripts/check.sh` passes in **354.561 seconds**. Its result and
  before/after Go hashes are retained in `raw/stage/full-quality.json.gz`: 655
  application files plus three existing files in the separate baseline-admission
  tool module (658 hashes in the runner's inventory). The application gate includes lint,
  65 Python tests, root/desktop race and vet, registered mounted-control race
  checks and offscreen vet. Existing historical-engine import/export and
  numerical regressions remain part of that gate. The read-only verifier
  requires exit 0 and unchanged final inputs.
- Three owned agents close with completed, joined, recovery-finalized,
  exit-ready shutdowns and exit 0. All **115 raw requests** remain available.
  Both final collections contain all six cases; the initial partial collection
  is retained separately and is not pooled into them.

The initial public collector incorrectly used the legacy alias suffix
`npn-pulse` as a catalog example ID. The agent correctly refused it; the actual
ID is `blink`. Its three earlier observations, `not-found` response, traceback
and normal cleanup are preserved. A second invocation refused to overwrite that
output directory before starting an agent. The corrected producer uses a fresh
directory and the actual catalog ID; final collections complete normally. The
failed producers/logs remain intact. The corrected final general-network case
uses the capacitor-containing `rc-high-pass` template rather than the initial
capacitor-free PMOS template.

Both agents run under the existing policy denying repository/SDK/toolchain/cache
and shared geometry access and outbound IP. All synthetic files are private.
No native window/input, display capture, privacy grant, owner project or printer
operation occurs. The sealed /179 review, SDK and canonical /120 are unchanged.

## Read-only replay

```sh
python3 -B docs/validation/integration-discovery/verify.py
```

Replay checks hashes and original failures, exact production/test source delta,
full quality, protocol/lifecycle, query purity, two-field discovery isolation,
all twelve physics states and preserved artifacts. It also checks the retained
development and original runtime binary hashes; keep those owned artifacts for
this strict local replay. Optimized Python refuses.
`producers/` retains historical scripts as text; their private output paths are
not an unattended rerun recipe. No application is started by replay.

AGENT-02 remains partial: this repairs one inaccurate numerical-method
description, not exhaustive contract/binding acceptance. P0 remains 17
implemented/10 partial; native/IME/reader, performance, environment, owner
walkthrough and participant gates retain their existing scopes.
