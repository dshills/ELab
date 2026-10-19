# Repaired recorder: nominal journal/input/capture check

**The repaired recorder writes, captures and finalizes normally in one fresh protected LED trial.** Exact recorder hash `677c5e117c40ca8f60b5c54e43176324ce4c10f799cbcf8f56813f4532d4cbaa`, built by the [journal repair](../frame-probe-journal-checks/README.md), records the unchanged /170 app through its verified LaunchServices-to-sandbox route. Existing Screen Recording access is preflighted in the working terminal context; no permission is requested. CUA is bound before it signals the armed recorder to start. After reading/validating the complete `journal_runtime01` acknowledgment, it clicks Apply once.

Private text `2k` leaves the accepted 1 kΩ circuit/current unchanged before Apply. The three-second recording exits **0** with **175 complete timestamped frames**, **six distinct 1053×802 PNGs**, one marker and no collector/command error. The journal and all capture files are 0600 inside a 0700 directory. Frame indexes/timestamps, PNG hashes/bytes/dimensions and marker/controller order are checked from the raw records.

The images distinguish actual state transitions. Old-value images show schematic 1 kΩ /2.917 mA despite private `2k`; a dimmed Apply image still has those old values. **Frame 12** shows the committed schematic **2 kΩ**, header **Solving… · stale reading**, and **Previous reading · stale** under the old 2.917 mA. **Frame 13** first shows fresh **1.477 mA**, 2.047 V and 3.02 mW. The three new image hashes were manually inspected; three others are byte-identical to previously inspected calibration PNGs. Annotation sources are explicit and do not constitute an automated OCR oracle.

The marker-to-first-fresh-frame interval is **163.765042 ms**. This is marker/Window-Server-frame arithmetic, not OS input/accepted-edit latency, a qualifying upper bound, p95 or a performance pass/failure. Presentation/Mach arithmetic is consistent, but physical clock/pixel interpretation remains open. On-screen membership at receipt does not prove unobscured visibility at the earlier display timestamp. Every frame remains ineligible: **0 eligible latency observations**, decision **null**.

The public after-state has exactly the separate packaged-agent 2 kΩ graph and full LED result (1.476723549 mA), with one electrical revision and run increment. `baseline.elab` remains byte-identical and `final.elab` is saved separately. Public close reports completed cleanup, joined workers and exit readiness. Recorder exit is 0; `open -W` exits 0, the owned native/recorder PIDs are absent, and the private endpoint is removed without forced cleanup. **Native exit status is unavailable**, because LaunchServices owns it. Shared geometry hashes/modes, all 1,651 sealed /170 entries, copied/canonical app, SDK and 312 nonconformance runtime inputs are unchanged.

This supersedes the repaired recorder's **pending nominal runtime check only**. The earlier offline journal checks and original recorder/data remain immutable. There is no app rebuild, version increment, promotion, owner acceptance, broader UI/conformance qualification or signing/startup claim. Stream errors, blank/idle/dropped frames, frame/image budgets, physical clock/pixels, visibility and the required LED/RC/NPN cohorts remain open. The captured stale label qualifies this one value-edit transition, not every render/solver condition.

Raw journal/text records are losslessly compressed; all six PNGs remain original. `checkpoint.json` binds sources, recorder/app identities, before/after/reference snapshots, saves, controller events, image annotations, capture and cleanup/postflight. Endpoint descriptors/credentials and temporary executables are excluded. All bindings/PIDs/windows here are stopped and must not be reused.

Replay retained data without launching or capturing:

```sh
python3 docs/validation/frame-probe-journal-runtime/analyze.py \
  --artifacts docs/validation/frame-probe-journal-runtime --verify
```
