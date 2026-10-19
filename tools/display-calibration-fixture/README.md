# Owned visual calibration fixture

This standalone Swift helper draws four distinct 16×16 corner patterns and one
96×64 checkerboard inside an explicitly verified ELab window rectangle. It is
instrumentation for the owner-authorized cropped-display calibration described
in [the measurement method](../../research/36-visible-latency-method.md).

`sh tools/display-calibration-fixture/check.sh` compiles with Swift 6 and warnings
as errors, then runs seven pure layout checks. It creates no UI or recording.

Runtime requires `--show --target /owned-target.json --output /new/private/tmp/path
--seconds 1...30`. The target JSON names the exact owned PID, CG window ID,
absolute executable, SHA-256 and global window bounds. Panels are opaque,
nonactivating, mouse-transparent and shadowless; every patch stays inside that
rectangle. Target identity and actual patch geometry are checked every 50 ms.
The helper creates no capture, input event, global listener or permission request.

After two preparation checks, `ready.json` records panel IDs, draw observations,
geometry and Mach timestamps. To finish, atomically create `stop.request` with
`{"not_before_mach_ticks":"TICKS"}`. Use the largest supplied frame display time
to keep the fixture present through prospective timestamps. The bounded lifetime
still applies. `closed.json` and the process status must both indicate success.
The helper closes only its own panels. A failed guard invalidates the attempt;
do not retry to select favorable timing or omit the failed evidence.

The [native comparison](../../docs/validation/display-frame-probe-edges-occlusion/README.md)
retains an initial target-guard failure and one diagnostic repeat. Static pixel
mapping and occlusion results do not establish event-to-visible latency.
