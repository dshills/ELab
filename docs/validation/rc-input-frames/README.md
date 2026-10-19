# RC manual input-edge probe frames

October 18, 2026. The source repair rebuilds the immutable probe frame whenever a manual RC input change is consumed. Scope, cursors, extrema and public CSV now agree with the solver at the discontinuity. This is development-source and public-agent evidence; clean replacement qualification remains next. The preserved /160 review is still affected.

The [native /160 failure](../../../research/ui-audit/first-version/native-rc-160/README.md) retains correct scalar current while displaying/exporting the previous positive current for the falling input edge. `applyRCInputLocked` changed scalar input/current without updating the private frame. Calling `rcSample` constructs a fresh frame with the existing node associations and pin orientation, preserving capacitor voltage, energy, time and earlier history. No new command, solver model, saved format, native binding or Vellum change is required. [Patch](source-repair.patch.gz).

[Regression source](../../../workbench/rc_input_frames_test.go) covers queued input consumed by Step, queued input consumed by Resume, and input while running, with all four resistor/capacitor pin orientations. Discharge, Charge and return to Auto preserve earlier probe points and storage state. Every retained scope sample and CSV row is checked against voltage/current laws; cursors select the right-hand edge and extrema include it. Consecutive actions without a clock tick may retain several events at one timestamp. The test checks the latest event's two sides and preserves the others.

The final regression fails on the original source in all 12 entry-path/orientation cases and passes with the repair. [Original-source failure](before-final-regression.txt), [targeted race result](after.txt). Two earlier test-development errors are retained separately: incorrect statistics field names, then a fixture incorrectly assuming exactly two samples per timestamp. Neither is reported as a product failure.

The [public reproduction](reproduce.py) starts a fresh owned agent, loads RC, selects six voltage/current probes, charges for 10 ms, switches to discharge and steps another 10 ms. It checks all 23 rows against the independent exponential RC solution and Ohm's law, including both sides of the edge, cursor tie behavior, extrema and public CSV. The same producer fails against preserved /160 and passes against the rebuilt development agent:

| At the 10 ms discharge edge | Preserved /160 | Repaired development agent |
|---|---:|---:|
| Solver current | −0.3160602794 mA | −0.3160602794 mA |
| Scope current | +0.1839397206 mA | −0.3160602794 mA |
| CSV current | +0.1839397206 mA | −0.3160602794 mA |

[Failing public report](old-public/report.json), [passing public report](fixed-public/report.json), [development identity](development-identity.json). Both owned agent processes exit 0; the failing producer exits 1 because its assertions detect wrong measurements. The development binary is based on `71c7f30` with `vcs.modified=true`; it is not a clean release. Raw build metadata is preserved compressed.

[Full quality result](quality.json) and [output](quality.txt) cover formatting/imports, Staticcheck, 27-requirement audit, 51 Python tests, root/native race and vet, mounted control regressions and tagged vet. The first sandboxed run passed simulator race tests but local Unix socket binds were denied; its failure is preserved in `sandbox-quality.*`. The rerun has permission for those test sockets.

No new physical native input replay, copied runtime, whole-bundle, screen-reader/IME, visible latency, distribution or owner acceptance is inferred. Owner remains “Not tried yet.” SCOPE-02 and FILE-03 stay partial until a clean replacement passes the edge reproduction and integration gates. No review is promoted.
