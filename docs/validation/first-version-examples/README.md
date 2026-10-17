# First-version package 1: example entry points

October 17, 2026. This checkpoint closes command/startup behavior for [release package 1](../../../research/34-first-version-plan.md). It does not close physical UI input, accessibility, visible latency, installation or full MVP acceptance.

The new `TestRepeatedExampleLoadsAlwaysStartFreshAfterPause` first reproduced the legacy LED defect: `example.reset` changed document generation 2→3 but left run 2→2 and electrical revision 2→2. The loader cleared existing results without starting a replacement solve. All other legacy loaders already restarted. The fix shares the legacy command-to-template mapping between loading and the restart decision; no numerical models or schemas changed.

After repair, all 25 focused cases pass: seven legacy commands and 18 generic catalog loads. Every case reloads an unchanged paused template and checks fresh run/electrical/document identities, running preference, exact components/wires and available simulation results.

Validation completed:

- `go test ./workbench -run '^TestRepeatedExampleLoadsAlwaysStartFreshAfterPause$' -count=1` passed.
- `scripts/check.sh` passed lint/imports/Staticcheck, 38 Python tests, root/native race tests and vet. Root conformance tests took 120.230 s, workbench 75.418 s, native desktop 142.776 s; these are test-suite durations, not app-latency measurements.
- `scripts/build.sh` passed with the current pinned Vellum SDK. The native linker emitted its existing duplicate `-lobjc` warning.
- Updated `scripts/examples-demo.py` passed through [fresh headless stdio](headless.json) and [an owned native private socket](native.json). Both cover all 18 templates, edited/reset behavior, fresh paused startup, dirty/stale guards, edited project save/fresh reopen, and all seven repeated legacy loads. The final owned project is saved and paused. [Process outcome](process.json) records normal native exit with no forced cleanup.

Local demonstration projects/logs are in ignored `build/first-version-examples/`; the reports here preserve the bounded outcomes. The native host used a private recovery directory and disabled appearance persistence. Existing user sessions and the canonical/preserved app bundles were not changed. The main development executables were rebuilt; no new versioned app candidate was created.

Next: complete the actual UI construction/measurement/file walkthrough in release package 2. Callback/API and numerical results do not prove mouse/keyboard readability or usability.
