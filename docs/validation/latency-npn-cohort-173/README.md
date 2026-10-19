# Declared NPN responsiveness block

October 19, 2026. Frozen signed **ELab 0.156.0 /173**. All 20 original NPN
IDs have available software-composition bounds. No ID is repeated or dropped.
The original [collection declaration](../../../research/37-visible-latency-cohort-plan.md)
fixes the circuit, edit, binary hashes, display geometry, duration and thresholds.

| Marker → visible response, n=20 | Minimum | Median | Nearest-rank p95 | Maximum |
|---|---:|---:|---:|---:|
| Committed 5 V source value | 102.820 ms | 122.881 ms | 137.294 ms | 141.870 ms |
| Fresh valid NPN result | 116.814 ms | 132.464 ms | 169.661 ms | 170.081 ms |

The percentile is the 19th ordered value, `ceil(0.95 × 20)`, retaining every
observation. These intervals start at an acknowledged marker before the CUA
click, so they include controller overhead. They are not exact OS input/accepted
edit times or physical display scanout measurements.

The fresh-result p95 bound supports the **500 ms target for this fixture within
the validated software-composition method**. The acknowledgment p95 exceeds
100 ms and is inconclusive about actual product latency because the bound
includes controller overhead. Raw eligibility flags remain false and raw gate
outcomes null. This does not establish a general PRD performance pass: the LED
and RC blocks have unavailable outcomes, and cold interaction, difficult solves,
human typing/IME, other environments and accessibility retain separate scopes.

## Fixture, references and input

The declared circuit has a 5 V collector supply, 1 kΩ collector resistor,
10 kΩ base resistor, grounded emitter and generic pinned Ebers–Moll NPN. There
is **no LED** in this collector load. Every trial starts with `Drive` at 0 V
and makes one CUA Apply edit to 5 V while simulation is live.

The independent 70-digit Decimal damped-Newton reference already exists in
`scripts/freeze-electrical-reference.py`; its algorithm differs from ELab's
nested bisections. Read-only regeneration matches the frozen JSON exactly. Two
packaged headless preflight sessions match the cutoff and saturation references.
The on-state values are 4.929600780862 mA collector current and
70.399219138186 mV VCE. Additional independent Ebers–Moll KCL, base/load KVL,
emitter current and power-balance checks pass.

An initial producer draft assumes a nonexistent top-level `npn_model` field.
Inspection catches this before any trial; the complete nested `npn.model`
already participates in result parity. The initial source/declaration and final
correction are preserved. No cohort ID, app, agent, solver, model, SDK, format or
threshold is changed to accommodate it.

The owner has explicitly left the machine available. The protected native
session verifies exact PID/window/executable, fixed `(2034,303,1053,802)` crop,
display 3/native 1× and pinned host metadata. CUA raises only the owned window,
types privately, refreshes AX after the public before-state observation, waits
for a complete baseline frame and acknowledged marker, then clicks Apply once.
The first full original recording is inspected before continuing. All input uses
CUA; each local recording lasts three seconds, with no audio or new privacy grant.

The launcher's root-level `initial`, `baseline` and `agent-expected` files are a
synthetic LED registration prelude, excluded from the cohort. Each
`session/trials/npn-XX` directory contains its authoritative NPN baseline and
expected/accepted result. Public load restores the canonical NPN baseline without
reusing session-reserved IDs through Clear. All native wire IDs survive each
edit; cross-session endpoint/connected semantics, graph, full model/result and
one electrical revision/run increment match the packaged agent. Every original
baseline remains unchanged and every final project is saved separately.

## Image and cleanup replay

The 20 recordings retain **3,260 complete image frames and 3,130 original PNG
files**, losslessly archived per trial with member hashes, sizes and modes.
Every complete frame's relevant regions match one of three manually inspected
full originals:

1. Private 5 V draft, accepted source label 0 V, live cutoff/0.000 mA/5.000 V.
2. Accepted source label 5 V, explicit Solving/stale labels and old readings.
3. Accepted source label 5 V, live 4.930 mA/0.070 V/Saturation and current result identity.

There are twelve captured stale frames; fast solves need not expose one. Exact
source-label hashes identify acknowledgment independently of fresh-result
classification. Current, voltage, transistor-state, live/stale header and result
identity regions jointly identify freshness. Readout columns shift with their
content, so annotations retain each complete region rather than assuming an
unchanged text origin. There is no OCR, fuzzy matching or substituted screenshot.

All images and callback/marker order, prospective timestamps, Mach/presentation
conversion, crop and bytes replay. Nine corrupted-data classes are refused for
each recording: changed clock/crop/scale, missing image or acknowledgment,
forged eligibility, stale pixels labeled fresh, unfinished input and wrong run.

The worker and native session both exit 0. Public shutdown is completed and exit
ready; `open -W` exits 0, owned process/endpoint are absent, and no forced cleanup
occurs. Shared window settings, copied/review/canonical applications and all 650
Go source hashes remain unchanged. Native process exit status is unavailable
under LaunchServices; `open -W` is not presented as that status. CUA independently
reports no running ELab after shutdown.

Read-only replay requires Pillow; it launches no app, recorder or UI input:

```sh
python3 -B docs/validation/latency-npn-cohort-173/verify.py
```

The original 60-ID LED/RC/NPN collection is now recorded in full: 46 available
bounds and 14 unavailable outcomes. Do not backfill failures or pool them away.
Any new complete LED/RC cohort requires a separate declaration. Owner walkthrough
and broader PRD acceptance remain open; this checkpoint does not rebuild or
promote the app.
