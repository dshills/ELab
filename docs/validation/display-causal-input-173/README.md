# One causal edit in cropped display composition

October 19, 2026. **The explicit-crop recorder captures one acknowledged CUA
Apply, the committed value with stale labels, and the fresh matching LED result.**
This completes the nominal causal-image sequence for the new capture mode.
It is calibration, not a predeclared performance-cohort observation or p95 result.

An isolated signed ELab 0.156.0 /173 session starts with R1 = 1 kΩ. CUA types a
private `2k` draft; public inspection still reports 1 kΩ and fresh 2.916796577 mA.
The saved CUA controller signals the armed recorder, waits for a complete baseline
image, requests `composition_led01`, validates the fully written acknowledgment,
then clicks Apply once. There is no other electrical edit in the native session.
Input stays entirely in CUA. The recorder has unchanged SHA `a8cec4e…923f` and
uses existing capture access without a permission request, audio or upload.

The three-second output interval retains 166 callbacks: **163 complete image
frames and three idle rows**, with 158 original 1053×802 PNGs. There is one
marker, no collector/command/journal error and normal recorder exit 0. All frame
geometry agrees with the inspected crop `(2034,303,1053,802)` on display 3 at
native 1×. Every complete presentation timestamp equals the exact converted Mach
display timestamp. The 133 future-at-receipt timestamps remain explicit; none is
adjusted or treated as proof of physical presentation.

Manual inspection of the three representative full images establishes the
following roles. Exact hashes of four selected regions classify **every complete
frame** into one of those roles; every distinct full PNG is also hash/size checked.
There is no OCR or inferred reading from public state.

| First callback index | Visible state | Complete frames in that state |
|---:|---|---:|
| 1 | Private `2k`; schematic 1 kΩ; fresh 2.917 mA | 9 |
| 10 | Schematic 2 kΩ; `Solving… · stale reading`; `Previous reading · stale`; old 2.917 mA | 2 |
| 13 | Schematic 2 kΩ; live status; fresh 1.477 mA, 2.047 V, 3.02 mW | 152 |

The marker-to-committed/stale frame interval is **128.215167 ms**, and the
marker-to-fresh frame interval is **178.215292 ms**. These are arithmetic from
the before-input marker to the supplied composition-frame timestamp. The
controller does not have an OS input-event or accepted-edit timestamp; controller
overhead is included. There is no target pass/failure or percentile decision here.
An interval above an original target would be inconclusive about product latency.

Eight in-memory mutations are refused: shifted display clock, changed crop,
unacknowledged marker, missing image, wrong pixel scale, forged eligibility,
stale pixels relabeled fresh and unfinished input. These checks do not launch a
recorder, alter evidence or simulate an OS capture failure. The independent
[static edges/occlusion comparison](../display-frame-probe-edges-occlusion/README.md)
and [software clock check](../frame-probe-clock-check/README.md) retain their own
scope; this nominal trial does not erase their failures or qualify physical scanout.

After-state graph, circuit, model and full fresh result equal the separate packaged
agent's `component.set` result exactly. Electrical revision and run ID each increase
once. The original baseline save stays unchanged; the final project is saved
separately. Public `app.close` reports completed shutdown/exit readiness,
`open -W` exits 0, the owned PID/endpoint disappear, and CUA reports no running
ELab. Native exit status remains unavailable under LaunchServices. Shared geometry,
sealed review, copied bundle and canonical application stay unchanged during this
session. Settings differ from the earlier session's baseline, so no across-session
settings-equality claim is made and no settings are restored.

All original PNGs are retained losslessly in `capture/frames.tar.xz`, with a
0600 member/hash/byte manifest. Large text records are gzip-compressed. Endpoint
credentials and temporary executables are excluded. Source/controller, trial plan,
public snapshots, raw journals, annotations and cleanup are bound by the checkpoint.
Read-only replay requires Pillow:

```sh
python3 -B docs/validation/display-causal-input-173/verify.py
```

All 650 Go hashes remain identical to the passed source-quality checkpoint; no
app, solver, SDK, format or command changes occur. Existing review and recorder
verifiers still pass. Raw eligibility flags remain false, eligible observations
remain zero and the latency gate remains null. The next work is the fixed
[LED/RC/NPN cohort protocol](../../../research/37-visible-latency-cohort-plan.md),
using composition-frame bounds with explicit measurement limits. Physical scanout,
human typing/IME, other display configurations, owner walkthrough and the hobbyist
study remain separate.
