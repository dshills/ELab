# Invalid value recovery: native reproduction and repair

October 18, 2026. A new native check found a blocked correction path in the previously qualified /153 review. The source repair passes the focused regression and the same native flow in a development harness. A clean replacement review package remains a separate next step; /153 and canonical /120 are preserved.

## What failed

With the live LED example and R1 selected, enter `0` and click Apply once. The shared validator refuses it, preserving the accepted 1 kΩ circuit, 2.917 mA reading and circuit history. The UI displays “That request could not be applied.” with the allowed 10–100000 Ω range. Correct the field to `2.2k` and Apply once: /153 shows “expected command revision 8; current command revision is 9” and keeps the old circuit. No second Apply is used to hide the failure. [Original native journal](before-native-session.json), [UI observations](before-ui-observation.txt).

A draft commit cleared the validation message before asynchronous Apply completed. Removing that row enlarged the schematic, and its accepted `view.viewport.set` observation advanced the command revision between field settlement and component application. The deferred Apply correctly refused its now-stale context. The source regression initially fails with “draft correction hid its refusal before Apply finished” ([transcription](regression-before.txt)).

## Repair

`changeValueDraft` retains the related invalid-parameter refusal through that field’s own accepted draft response. It updates the refusal’s presentation revision using that exact response. A successful `component.set` clears the refusal normally. Other commands, agent changes and document replacement still invalidate captured intent; no revision guard, generic retry or public command changes.

The source ledger was reviewed: 68 production files, 391 sites, 162 public commands and 132 referenced command IDs remain. Only the changed source digest and subsequent line positions differ; all non-line site fields and command/alias/coverage sets match the previous ledger. The SDK stays frozen at `677548d`.

## Native result

The corrected development executable is SHA-256 `3590adf5d54a7e27a3e2b1d8100654806134c2b5a1c97752e70caf479bcd4dc0`. It is a dirty-tree development build, not a clean release or new product version ([build info](development-build-info.txt)). The separately identified harness retains copied /153 metadata solely for testing. Production app bundles are untouched.

Original owned PID 37672 runs with recovery/appearance persistence disabled. CUA enters `0`, observes refusal, selects that text, types exactly `2.2k`, verifies the draft and clicks Apply once. The error disappears, the field normalizes to 2.2 kΩ, history shows `Change value · by user`, and LED current changes to 1.345 mA. [UI observations](after-ui-observation.txt).

An independent reference generated through the exact /153 packaged agent before native input has the same validation refusal and corrected model state. The observer queries the original host until both stages match every component/wire/net field, selection, drawing, history labels, status, pause state and every LED result field except independent run/electrical identities. The invalid stage preserves graph, full result, run/electrical identities, document and history; correction restarts once. The saved [schematic](corrected-value.elab) matches the final graph. The original host closes normally with code 0. [Native journal](after-native-session.json), [agent reference](after-agent-reference.json), [verification](verification.json), [observer](after-observer.py).

The first corrected-harness observation attempt is excluded: `cua.getApp` stalled for 2594.2209 seconds, outliving the original PID 18201’s 300-second lease. That host closed normally with no tested input. The tool then opened a blank instance, which was closed through Cmd+W (App quit). The cached bundle selection in the independent retry completed in 0.1033 seconds. This is an observation-tool failure, not a passing product trial. [Excluded journal](excluded-selection-timeout-session.json).

## Limits

This is one automated native ASCII validation/correction flow, corroborated by same-process state and same-model agent parity. Screenshots were inspected in tool output but not exported; durable UI evidence is a transcription. Inspector controls disappear from the external CUA AX projection while the refusal is visible, even though screenshots show them; this repair does not close that separate accessibility issue. It does not establish physical keyboard/IME, reader acceptance, concurrent or blur behavior, visible p95, another environment or human study outcomes. Owner walkthrough remains “Not tried yet”; P0 remains 17 implemented /10 partial. No new OS settings or printer work.

## Quality result

The complete `scripts/check.sh` passes on the final repair/tests: Go formatting/imports, Staticcheck, PRD audit, 51 Python tests, root race tests/vet, native race tests (148.247 s)/vet and required mounted regressions (16.214 s)/offscreen vet. Focused value/layout/concurrency guards pass under race (3.715 s); the native development executable passes the existing full smoke. The earlier full run exposed a test helper that required the error to disappear on draft commit; that assertion was updated to require the intended retained refusal until successful Apply, with no further production change. [Quality log](quality.txt), [native smoke](native-smoke.txt), [checkpoint](checkpoint.json). `python3 -B verify.py` rechecks stored artifacts/state without replaying native input.
