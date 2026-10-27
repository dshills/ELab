# Analysis and device expansion qualification

October 27, 2026.

Runnable unsigned/ad-hoc candidate **ELab 0.198.0 /276** is
[available here](../../../build/ELab-0.198.0-276-Analysis/ELab.app), with its matching
agent and **67** saved example projects. Clean binary source:
`323f5e7a9e9969629a23289391666dfc41d0c29b`. The current catalog has **54** component
kinds and **203** shared commands. The downloadable /270 and every previous
candidate remain preserved. [Package identities](package.json), [summary](summary.json).

Five requested additions are implemented through shared
user/agent commands. Existing component, storage, numerical accuracy and
integration work limits remain unchanged. The SDK and all earlier packages are
preserved. No push or public upload.

| Addition | Implementation commit | Evidence and review dispositions |
| --- | --- | --- |
| Semiconductor AC/Bode | `ed98095f` | [AC](ac/README.md) |
| Captured FFT/THD | `cd7ff887` | [Spectrum](spectrum/README.md) |
| Versioned BJT profiles | `0377a10b` | [Profiles](profiles/README.md) |
| Seeded tolerance trials | `9cbad84d` | [Tolerances](tolerance/README.md) |
| Loudspeaker load/preview | `f70ca402` | [Speaker](speaker/README.md) |

Each addition has three complete Prism reviews using the explicitly selected
**Anthropic `claude-opus-5-5`** provider/model. Raw reports and findings are
retained in the corresponding directory. Demonstrated defects are repaired;
remaining observations have concrete test/schema/model dispositions. Several
raw final reports still contain contextual medium findings; this is not a claim
that Prism emitted no medium findings. No review was suppressed or baselined.
The initial attempted OpenAI review was blocked before process creation; the
owner then authorized Anthropic. No code-review diff was sent to OpenAI.

## Combined qualification

Final exact-source coverage includes **914 portable,
354 ordinary desktop and 491 offscreen**
top-level race tests (2043,
859 and 1530 with subtests).
Lint/imports, all three vets, 77 Python tests, PRD audit and native command
inventory verification pass. The final **1409** frozen executable/test/module
inputs are pinned in [qualified inputs](qualified-inputs.json.gz).
[Coverage report](application-report.json.gz) and [aggregation controller](aggregate_checks.py.txt).
This is aggregated exact-source coverage, **not** a claim that each initial
ordered invocation passed. The failed invocations and controls are retained.

The initial complete portable run found missing spectrum contract version/scope
and a stale native UI inventory. Both are repaired. The second complete run
found a stale test's atomic-edit allowlist count (27 instead of 29); its corrected
fixture now verifies both new device-edit commands explicitly. The new complete
workbench run passes. A separate RL/RLC conformance workflow reached its unchanged
100 ms work guard in the combined ordered run. Three isolated workflow controls
and a full isolated conformance-package run pass. No cause or electrical repair
is established; initial failed runs remain evidence, not passing gates. Final
portable coverage combines exact unchanged packages from the complete run with
fresh complete workbench and conformance-package runs.

The ordinary full run also failed an older flasher UI-step test at 115 ms; its
helper printed only an error pointer. Three exact-source isolated full workflow
controls pass. In the full offscreen run, one flasher cursor case explicitly hit
the 100 ms work guard at its switching edge. Three isolated full cursor workflows
pass. No numerical correctness failure or causal production repair is claimed.
The other offscreen failure was a stale Passives count (9 instead of 10 after
adding the speaker). The corrected fixture checks presence and actually selects
the speaker through the mounted keyboard list; three repetitions pass. An initial
extended fixture incorrectly assumed focus survived removal of the MOSFET row;
final independent focus is explicit, and that failed assumption is preserved.
Only the atomic-edit and browser test fixtures changed during this continuation;
all production and other test inputs retained for coverage are exact.

## Exact package and normal-runtime checks

The exact matching packaged headless agent and copied native service both pass
all five workflows: 101-point semiconductor AC, 401-row clipping capture,
7.5929% declared-model THD, exact saved-recording spectrum recomputation,
profile edit/undo/redo/reopen, 16 seeded tolerance trials repeated exactly,
21-row speaker capture and a silent, explicit 524-byte preview render/read.
They also each complete all **70** original flasher steps in the normal build.
Preview playback was not requested; no audio hardware claim is made.
[Headless checks](headless-checks.json), [native checks](native-checks.json),
[public-command controller](new_workflows.py.txt).
The native feature receipt hashes its sandbox forwarding wrapper under
`agent_sha256`; the actual matching packaged agent is independently pinned in
[package](package.json) and [runtime](runtime-report.json.gz) receipts.

The copied native smoke and all four public conformance adapters pass under
outbound IP and development-file denial, with shared geometry writes denied
only inside the fixture. All **102** external processes exit 0 normally.
The separate owned feature app exits 0 and removes its endpoint. No owner
session, project or preference was used as a fixture.
[Runtime](runtime-report.json.gz), [public adapters](public-report.json.gz),
[normal feature close](native-final-process-state.json).

All **67** saved projects reopen paused/clean with exact graphs using the exact
matching agent. All **126** old project/SVG files and all 63 old catalog entries
remain exact. Only the copied Examples README/catalog are extended with four new
examples. All **25** protected package/SDK identities and all **1409** frozen
source/test/module inputs remain exact. Source and copied app seals verify
strictly. [Examples](examples.json), [preservation](preservation-before.json),
[seals](final-signatures.txt), [source inputs](qualified-inputs.json.gz).

The initial sandboxed icon conversion reported Invalid Iconset; the exact same
pinned packager passed outside that sandbox, preserving the original icon hash.
No packager, SDK, permission or security setting was changed. Original build
logs retain nonfatal module stat-cache write warnings and the existing duplicate
`-lobjc` linker warning. Early workflow-controller trials used the wrong spacing
spelling, omitted required audio start, used an incorrect snapshot key and
compared state against the pre-Undo run identity. Each was a controller error;
all are retained and excluded from passing product evidence. Final receipts use
the corrected explicit schema and a settled post-Undo baseline.

## Material limitations

- Configurable transistor profiles are educational constant-capacitance models,
  with no validated manufacturer identity or thermal/noise/rating claims.
- Spectrum results depend on the retained recording, adequate sampling and a
  coherent steady interval. THD is not THD+N or calibrated measurement hardware.
- Tolerance samples are independent uniform educational draws, not guaranteed
  worst cases or manufacturing statistics. Failed trials remain visible.
- Speaker preview illustrates filtered/normalized terminal voltage, not SPL,
  enclosure response or hardware audio qualification. No automatic playback.
- The original 1 nF, eight-BJT class-AB circuit combined with speaker LC reaches
  the existing integration guard at startup; a 10 nF control also does. This
  combination is not qualified for waveform progression. Its regression checks
  safe refusal and unchanged accepted state. The powered-follower speaker
  example and independent passive ODE/impedance references are positive evidence.
- Original physical native-input checks **3/6/9/11**, participant and separate-account
  acceptance remain open. Routed, offscreen and native-service command checks
  do not prove OS queue timing or hardware-rate mouse behavior.
