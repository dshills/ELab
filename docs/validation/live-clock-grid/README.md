# Live-clock grid repair · October 17, 2026

The exact 0.124.0 /141 source, exercised in a private native harness with the frozen capacity graph, stopped at 1.0999999999999897 s. Its diagnostic was `mixed timestep too small at 1.1, h 1.04e-14`; results were correctly marked stale. Repeated addition of the 1 ms output interval drifted short of the 1.1 s source corner. The real shared-clock regression reproduces the same failure before the fix.

`workbench/transient.go` now computes each requested time from the launch origin plus an integer count of accepted output samples times the interval. Discarded work does not advance that count; a resumed clock starts at the committed physical time. No solver tolerance, source-corner tolerance, work budget, model, command, format or SDK changed.

The focused race checks and full `scripts/check.sh` pass: lint/Staticcheck, 43 Python tests, root/native race and vet. Native race takes 142.400 s. The real-clock regression also compares the loaded RC response with an independent voltage law (10 µV bound), and another test checks a nonzero resume origin. Logs are retained here.

The repaired private native executable was built from `95712d4` plus the recorded source changes; build metadata correctly says modified. It is not a clean replacement release. Native Resume, Pan via Right, Escape to cancel Pan, resistor movement, Undo, Save and Pause operated while the clock advanced. Final public observation: **40.9 s**, paused, both result objects fresh, no diagnostic, actor user, original components/wires/settings restored and original run/electrical identity preserved. Native Save completed before Pause. `capacity-pulse.elab` is the subsequent public Save As copy. The previous owned test file was overwritten by native Save; graph/settings match the old raw fault, but no byte-identical original-file claim is made.

Evidence:

- [Checkpoint and source/binary hashes](../../../research/ui-audit/first-version/live-clock-grid-checkpoint.json).
- `states.json.gz`: raw old fault, repaired baseline/intermediate/final observations, and the unsuccessful overlap/dense trials below.
- `regression-before.txt`, `focused-race.txt`, `full-check.txt`: failure-before and passing validation.
- `*.ax.txt` and `paused.jpg`: actual native controls/observations; screenshot includes a hover tooltip, so use public state for exact readings.

The first manual-work UI trial is **ineligible** as an under-work span: before was running at 1/1000, after was already completed at 1000/1000. It does not qualify native work cancellation or response timing. Two denser 64-part/100-node trials (1 ms and 1 µs output interval) reached the existing 100 ms integration budget before a staged sample. Both preserve exact graph and committed physics. `dense-fixture-builder.py` is retained source provenance from its original build directory, not a standalone runnable script here; it imports the existing public capacity builder. No claim that all supported-size topologies converge follows from the successful 60-net circuit.

Neither the interaction checks nor the native UI capture tool establish event-to-pixel, first-visible-sample p95 or individual Newton-iteration overlap. The 0.124.0 /141 and canonical bundles were not changed. A clean replacement build and exact-runtime qualification remain the next release action. No additional paper was printed and no OS accessibility settings were changed.
