# Power, isolation and AC analysis qualification

The five requested additions are implemented with ten working examples. Native
controls and external agents use the same guarded commands. The catalog now has
48 components and 175 public commands. This is development candidate **ELab
0.190.2 /249**, built from clean source
`4ce8e7e336e37addeaff677e1aa5b3074f7d4514`.

The local application is
`build/ELab-0.190.2-249-Power/ELab.app`, with its matching `elab-agent` beside it.
It is ad-hoc signed and passes strict signature verification; it is not a
Developer ID signed or notarized distribution. The existing qualified unsigned
alpha download remains preserved. This candidate has not replaced that release.

| Capability | Examples | Electrical and interaction coverage |
| --- | --- | --- |
| Battery charge, internal resistance and voltage sag | `battery-led`, `battery-regulator` | Terminal voltage/ESR law, stored energy, discharge, LED fade, switched load and regulator dropout; charge slider and exact value entry |
| DC motor and 555 PWM control | `motor-dc`, `motor-pwm` | Startup/stall current, back EMF, torque, speed, mechanical loss, energy and live load continuity; locked rotor and duty adjustment |
| AC frequency sweeps and Bode plots | `ac-rc`, `ac-rlc`, `ac-opamp` | Complex gain/phase against analytical networks, half-power estimates and resonance; run, gain/phase menu and point cursor without disturbing transient state |
| LED/phototransistor optocoupler | `opto-switch` | Separate electrical ports, CTR, saturation/leakage, conservation and input/output probes; native CTR editing |
| Transformer and coupled inductors | `coupled-windings`, `transformer-supply` | Mutual inductance, dot polarity, winding resistance, loaded AC, startup, magnetic energy, bridge smoothing and 3.3 V regulation |

The model contracts describe commands, units, limits, solver representation and
omissions: [batteries](../../batteries.md), [motors](../../dc-motors.md),
[AC sweeps](../../ac-frequency-sweeps.md), [optocouplers](../../optocouplers.md)
and [transformers](../../transformers.md). These are generic educational models:
battery charge is a linear reservoir, the motor uses a viscous mechanical load,
AC analysis supports an explicit linear component set, and optical/magnetic
devices do not model isolation ratings or every physical nonideality.

## Automated qualification

[report.json](report.json) records exact counts, package timings and source
baselines; the corresponding raw Go JSON logs are included here.

| Complete race suite | Source baseline | Top-level tests | Including subtests | Result |
| --- | --- | ---: | ---: | --- |
| Root modules | `d691c086` | 835 | 1,858 | Passed; workbench 197.687 s, conformance 117.814 s |
| Standard desktop | `0baca004` | 321 | 744 | Passed in 627.990 s |
| Rendered offscreen desktop | `0baca004` | 437 | 1,288 | Passed in 932.227 s |

No tests failed or skipped in these completed runs. Two CLI packages have no
test files. Root, standard desktop and rendered desktop vet checks pass;
goimports/staticcheck pass; Python tooling passes 67 tests. The 27-requirement
PRD coverage audit also passes, which checks coverage records rather than
asserting complete product acceptance. Test processes used `GOMAXPROCS=2` and
the desktop runner's 30-minute timeout. Application runtime settings, numerical
tolerances, the 512-attempt guard and the 100 ms per-output limit were preserved.

The full suites precede three small repairs. These were checked separately,
rather than being represented as new full-suite runs:

- Seven sweep help/error/accessibility strings: focused root checks, complete UI
  inventory tests, rendered AC/DC sweep tests and lint pass. The string-only
  patch is retained in [post-suite-wording.patch](post-suite-wording.patch).
- Motor parts-list exports omitted saved parameters. Reproducing tests now
  verify parameter completeness, free/locked mode and bounded wrapped layout.
  Three race repetitions pass, followed by broader export/schematic/parts tests
  in 23.434 s, XML/motor checks and lint. The exported PNGs were inspected.
- Scientific-notation scope current labels clipped at the right edge. Rendering
  now reserves their measured text width, and cursor coordinates use the same
  plot bounds. Geometry checks cover Light/Dark/Apollo at 240, 320, 800 and
  1700 px; repeated cursor/scroll/Bode checks, mounted controls and lint pass.
  The final packaged native check confirms the repair.

The final matching packaged agent replays **all ten examples** through public
commands, with electrical assertions, captures, clean saving, fresh-process
reopening and schematic SVG/PNG/PDF exports. AC results also roundtrip JSON and
CSV. See [final-stdio.json](final-stdio.json); [export-manifest.json](export-manifest.json)
records the locally retained exported artifacts' hashes. Native executable and
agent hashes are in [final-build-final-result.json](final-build-final-result.json).
Its `graphical_app_launched: false` records the build-time step; subsequent
native execution is documented separately below.

## Actual desktop checks

[native-checks-248.json](native-checks-248.json) records physical CUA pointer and
keyboard input to the packaged /248 executable. Native charge, ESR, mechanical
load, locked rotor, PWM duty, CTR, turns ratio, winding polarity and AC controls
were compared with the matching agent's snapshots. Light, Dark and Apollo were
exercised. Live motor changes retained speed/current/time before integration;
load then reduced speed. Reversing the secondary dot changed the measured
5 ms winding voltage from +8.417514 V to −8.417514 V. The RC sweep's entire
transient result and run identity remained unchanged while its Bode cursor moved.

After the scope repair, the final **/249** executable displayed all amp-level
current labels including their units. A native click at the corrected plot's
right boundary selected exactly 40 ms. See
[native-checks.json](native-checks.json), [final state](states249/native-final.json),
and [final screenshot](ui249/motor-axis-cursor-final.png). Native states and AX
observations are retained beside the screenshots for inspection.

The component browser reaches all 48 entries by wheel, and the long motor
inspector scrolls by wheel and thumb. **Component-list thumb dragging remains
unqualified by this native method**: targeted CUA gestures selected list rows
instead. This does not establish how an ordinary hardware gesture behaves.
The /248 screenshots also contain a persistent automation tooltip overlay.
These method limitations are retained rather than classified as verified
hardware defects or hidden behind a blanket UI pass.

All test projects, recovery files and appearance settings were private disposable
fixtures. No owner projects or window dimensions were changed. Fixtures were
saved, both owned apps closed normally, and their socket endpoints disappeared;
CUA's running-app inventory contains no ELab session after cleanup. The initial
launcher identity lookup stalled for several hours; the actual running owned
application was found and reused without another permission request.

The preserved alpha ZIP's SHA-256 remains
`d13973f02f9c2895d353d24338ed060492a3a2200ff8a91e3606bfa867f4946b`.
