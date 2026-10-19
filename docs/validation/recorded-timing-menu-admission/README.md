# Recorded timing menu admission · /163 source

Fresh timing-direction callbacks match a separate in-process agent. Queued callbacks preserve state after an intervening agent setting, a user field commit, another recording, a replacement document, a reopened menu or file-busy state. Callback contexts are restored after delivery. Measurement changes preserve live graph, physics, document and undo history.

[Tests](../../../desktop/recorded_timing_menu_darwin_test.go) use the assembled macOS selector adapter and its actual admission function, with owned in-memory sessions. The fresh-menu test **commits the field before capturing the menu context**. The interference test models changes **after a popup's context was captured**. Neither drives an AppKit popup or proves that clicking a native selector settles a private field and opens the menu on the first attempt.

The [race run](race.txt) passes in 2.243 s; [native vet](vet.txt) passes with no output. Existing duplicate Objective-C linker warnings remain in the race output. Tests run automatically with regular macOS native-module checks. This adds source conformance evidence without a production/runtime, command, model, format, SDK or version change. No new package or full-gate rerun is required for this test-only addition; the current /163 manifest/source verifier still passes.

## Native interaction still pending

The intended whole-window test was to type a threshold and immediately choose an arrival direction from its native menu. [Automatic approval review](approval-review.json) could not approve starting its setup: review-model capacity failed once, followed by two review deadline timeouts. These were review infrastructure failures, not unsafe-action determinations or missing project authorization. The native setup was never executed; [read-only preservation checks](preservation.json) confirm its directory was not created and the stopped disposable bundle's three files remain unchanged. No permission check was bypassed.

Resume that native interaction when the approval path is available or guidance resolves the failure. These source tests do not substitute for it and do not establish a defect in first-click menu behavior. Owner walkthrough remains Not tried yet; full agent conformance, physical input, reader/IME and visible latency retain their existing open scopes.

Verify without launching an application:

```sh
python3 -B docs/validation/recorded-timing-menu-admission/verify.py
```
