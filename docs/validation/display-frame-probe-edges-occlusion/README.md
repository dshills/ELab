# Cropped display edges and owned occlusion

October 19, 2026. The static mapping/occlusion comparison succeeds in one
diagnostic repeat on unchanged signed ELab 0.156.0 /173. The initial helper guard
failure remains invalid and unexplained. This is not a latency trial or p95 result.

The owner authorized tightly cropped display recordings for this calibration.
Both attempts request three seconds per recorder, local files only, no audio or
permission request. The explicit crop is the independently verified entire owned
window: global `(2034,303,1053,802)`, display 3, 5120×1440 at native 1×. Nothing
outside that rectangle is output. The only CUA input is AXRaise; no electrical
value is edited. App, agent, model, SDK and both recorder binaries stay frozen.

The helper creates four distinct 16×16 patterns two points inside the corners and
a 96×64 opaque checkerboard over part of the ground symbol. Its panels are
mouse-transparent, nonactivating and shadowless. CG metadata checks their actual
positions. `ready.json` records geometry and draw observations; the helper checks
target identity and patch bounds every 50 ms. The controller starts the display
and window-only recorders concurrently. A successful stop waits through the
largest supplied frame display timestamp before hiding the patches.

| Attempt | Display frames / unique PNGs | Window-only frames / unique PNGs | Result |
|---|---:|---:|---|
| Initial guard failure | 162 / 143, plus one metadata-only prospective frame beyond the deadline | 175 / 3 | Helper exits 1 with `Exact frozen owned target no longer available`; five window-only frames fall after fixture closure. Entire attempt rejected. |
| Diagnostic repeat | 55 / 51 | 86 / 3 | Both recorders and helper exit 0. Every recorded image timestamp falls inside fixture lifetime. |

The repeat changes only the helper's failure text to report each observed target
property. No guard is removed or relaxed. The first failure does not reproduce;
its cause is not established. The repeat's different callback counts are retained
without frame-rate, dropped-frame or performance conclusions. The original helper
binary was not retained before the diagnostic rebuild; its recorded SHA remains
in the initial plan. The current diagnostic binary is preserved in
`build/display-edge-occlusion-diagnostic-work/calibration-fixture`.

An independent Pillow replay checks **every complete image row**, including all
distinct PNGs from both attempts, against hardcoded patterns and coordinates.
All five display patterns match every expected black/white pixel exactly, with
zero variation within either color class. In the window-only stream the patterns
are absent: corner matches are 0.4609375–0.5390625 and checkerboard match is
0.5009765625. The underlying ground-region image correlates 0.999516 with the
independent pre-fixture CUA screenshot. That baseline has nonzero variance; a
blank region cannot pass. Eight analysis counterexamples reject a one-pixel
horizontal shift for each pattern, doubled output dimensions, a clipped region
and blank underlying pixels. These are analysis checks, not additional captures.

Manual inspection of the repeat's two representative images agrees with the
replay: composition contains the patches; desktop-independent window content
contains the ground symbol. All original PNGs are retained losslessly in per-stream
`frames.tar.xz`, with hashes, byte counts and 0600 archive-member modes. Archives
are read in memory; the verifier never extracts pixels or runs a recorder.

Both synthetic native sessions save their final project and close through the
public `app.close` command. Cleanup reports completed workers/exit readiness,
`open -W` exits 0, owned PIDs/endpoints disappear, and shared geometry plus copied,
review and canonical bundles stay unchanged. LaunchServices owns native process
exit status, which remains unavailable. Initial/final public graph, circuit,
model and result match the separately run packaged agent exactly.

Swift 6 warnings-as-errors compilation, seven pure layout cases and strict
Swift-format pass. Current /173 admission and the earlier raw recorder proof
also pass read-only verification; all 650 Go source hashes remain unchanged.
Initial compiler failures, executed controllers, logs and cleanup reports remain
in this evidence set. No application tests are repeated for this tooling-only change.

This establishes the observed static crop origin/scale, unclipped marker mapping
and composition-versus-window occlusion behavior on this recorded display mode.
It does not establish physical scanout, unseen overlay absence during future
trials, live causal acknowledgment/fresh-result timing, recorder fault behavior,
other monitors/scales, accessibility, owner acceptance or the hobbyist study.
All recorder eligibility/calibration flags remain false, eligible observations
remain zero and the latency gate remains null. Next: one acknowledged CUA
edit/causal-frame sequence with this composition mode, then the predeclared
LED/RC/NPN cohorts if that method qualifies.

Read-only replay (Pillow required):

```sh
python3 -B docs/validation/display-frame-probe-edges-occlusion/verify.py
```
