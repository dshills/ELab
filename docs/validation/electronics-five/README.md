# Analysis and device expansion qualification

October 27, 2026. Five requested additions are implemented through shared
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
