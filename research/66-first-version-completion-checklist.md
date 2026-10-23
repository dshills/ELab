# First-version release checklist

Updated October 23, 2026. **ELab 0.181.3 /204 is the current first usable private
macOS release in the owner-selected unsigned channel.** [Handoff](79-first-version-release-closeout.md),
[final evidence](../docs/validation/first-version-completion-204/README.md),
[app](../build/releases/ELab-0.181.3-macos-arm64/ELab.app),
[ZIP](../build/releases/ELab-0.181.3-macos-arm64.zip).

This retains the cut line in [the delivery plan](34-first-version-plan.md): draw,
run live simulation, edit, measure, undo, save/reopen and export offline, through
both the user interface and public agent commands. No new feature was added in
this closeout. Owner review and clean-account installation retain their explicit
deferrals. This is not a claim of complete PRD MVP or general public-installation
acceptance. The [previous detailed checklist](66-first-version-completion-checklist-history-2026-10-23.md)
is preserved with its original checkpoints.

## Required release work, in order

| # | Required item | Current disposition and evidence |
|---|---|---|
| 1 | Native input fixes | Complete for the declared classes. /194–195 correction/ownership evidence is retained; /204 actual scope/sweep/divider/secondary-click checks and current keyboard construction/value correction/Undo/Redo/Save/Open pass. [Mouse receipt](../docs/validation/ui-input-204/native-ui.json), [keyboard receipt](../docs/validation/first-version-completion-204/keyboard-report.json). |
| 2 | Usable ordinary UI | Current native Light/Dark minimum views at 1053×762 and working-size views at 1221×969 pass; the original theme/size and clean blank document are restored. Populated keyboard tasks also pass at 1053×802; Inspect/Simulate/Document controls are reachable. Broader /198 layout evidence is retained for unaffected styles. This is not exhaustive accessibility or clean-account acceptance. [Captures](ui-audit/first-version-completion-204/README.md). |
| 3 | Four reference workflows | Current keyboard-built LED reads 2.917 mA at 1 kΩ and 1.477 mA at 2 kΩ; exact native Save/Open and fresh packaged-agent reopening preserve its graph. RC/flasher native walkthroughs are retained for unchanged controls; exact extracted /204 LED/RC/NPN workflows, project contracts, independent electrical references and separate flasher timestep refinement pass across four adapters. Output evidence is reused with unchanged output engines. [Closeout](79-first-version-release-closeout.md). |
| 4 | Shared user/agent behavior | Current extracted native socket, headless stdio/socket and in-process reference tasks/project contracts agree. /204 browser category/search/purity/retry tests cover the added presentation commands; /198 inspector tests remain valid. Full dynamic every-binding qualification remains a broader MVP obligation. [Runtime receipt](../docs/validation/first-version-completion-204/runtime-report.json). |
| 5 | Data, captured results, recovery and cleanup | Four fresh packaged-agent reopens preserve all three bundled examples and the actual native-created project. Current project contracts and full source regressions pass. Unchanged storage/recovery workers retain their scoped earlier evidence. All 45 conformance subprocesses exit gracefully with code zero; no endpoint remains. The separately owned GUI test closes, its PID disappears and its saved project remains. [Examples](../docs/validation/first-version-completion-204/examples-report.json), [source reuse](../docs/validation/first-version-completion-204/source-carryover.json). |
| 6 | Numerical and state regressions | Current exact-package electrical references and flasher timestep refinement pass on four adapters; the unchanged tolerances and model limits remain. LED UI readings agree. No solver/model/sampling code changed in the UI maintenance series. Shared-kernel parity is distinguished from independent references. |
| 7 | Source/repository quality | /204 passed lint/static analysis, 67 Python tests, root/desktop race tests, vet and configured mounted tests. All 690 recorded source/module/SDK/check inputs still match. This closeout changes instructions/evidence only, so no rebuild or redundant full test run is needed. [Quality receipt](../docs/validation/ui-input-204/report.json). |
| 8 | Final package and offline operation | Exact newly extracted /204 bytes run with source/SDK/toolchain/caches/shared geometry/outbound IP denied for external test processes; controller/in-process adapter remain unrestricted. The 127 archive members, modes, app/agent hashes, dependency notices, examples and strict ad-hoc signature remain intact. [Runtime and package receipts](../docs/validation/first-version-completion-204/README.md). |
| 9 | Startup/distribution boundary | Recorded. Ordinary /204 startup was observed in the mouse maintenance check; separate clean-account/default installation remains **deferred by owner, not passed**. The unsigned-copy assessment rejects as expected; no trust override or quarantined launch occurred. Developer ID/notarization are not requirements of the owner-selected channel. [Assessment](../docs/validation/first-version-completion-204/distribution-report.json). |
| 10 | Actual owner workflow results | **Deferred by instruction; unobserved.** Owner review does not block this engineering release. Earlier “Not tried yet” responses remain unchanged; no usability or participant-study pass is inferred. [Optional tasks](35-first-version-review.md). |
| 11 | Blockers, one release and rollback | No reproduced blocking defect remains in the declared reference tasks. /204 is the unambiguous current app/agent/ZIP; earlier versions remain preserved. The /203 fast-drag failure is retained and not promoted. Instructions, limitations and rollback are available. [Current release](79-first-version-release-closeout.md). |

Items 9 and 10 are accurately scoped dispositions, not tests silently marked
successful. Work-loss, wrong-result and core user/agent parity requirements have
not been waived. This release stays feature-frozen.

## Additional requirements before calling it the complete PRD MVP

These remain part of the original product request. They are not prerequisites added by this checklist, and private-alpha delivery must not be reported as satisfying them.

| Remaining area | Required completion evidence |
|---|---|
| Complete user/agent coverage | Every shipped capability is discoverable and callable, with accurate schemas, units, defaults, preconditions, results and errors. Complete conformance covers dynamic bindings, state transitions, atomic failures, revisions, retries, undo, cancellation, reconnect, dialogs and teardown. UI automation is not the agent interface. Built-in and external clients use the same public contract; bundling a conversational AI provider is not required. |
| Keyboard, focus and accessibility | Complete the actual pilot-platform keyboard and screen-reader workflows, navigation, properties, diagnostics, plots and file decisions. Cover remaining focus/blur, fast typing, IME and interruption cases. AX labels or screenshots alone do not establish screen-reader support. |
| Performance and difficult solves | Establish the PRD's visible p95 acknowledgment target of 100 ms separately from the 500 ms first-valid-update target on the three reference circuits; retain existing warm-trial limitations. Validate the reference 200 ms Verify runs within two seconds and navigation/cancel/save/undo during a deliberately difficult solve. No repeated latency campaign without a concrete remaining measurement need and declared method. |
| Capacity and complete state contract | Validate the supported 50-component/100-node/four-trace envelope, boundaries, bounded history, cancellation and retained edit/save access. Finish remaining physical gesture/state-transition scenarios; a large simple fixture does not prove arbitrary convergence. Publish the actual supported envelope. |
| Clean setup and project portability | Reproducibly bootstrap the locked SDK/dependencies on a clean supported development environment. Perform ordinary default startup and supported offline installation in a separate account/VM; reopen self-contained projects without fetching dependencies. The private-alpha deferral does not pass this gate. |
| Distribution | Qualify intended installation/quarantine/Gatekeeper behavior and the appropriate signing/notarization/distribution process before public distribution. A local ad-hoc resource seal does not establish those properties. |
| Ten-person hobbyist study | Run five newcomers and five experienced hobbyists using the prepared study packet. Meet or explicitly revise with evidence the original gates: 8/10 overall and 4/5 newcomers build the LED in five minutes unassisted; 8/10 explain the resistance experiment; 8/10 plot a waveform and enter a cursor; 8/10 recover the deliberate mistakes; 9/10 distinguish result states; 8/10 identify a model limit and save/reopen. Investigate every observed invalid result presented as valid. Implement the resulting prioritized corrections. |
| Printing/output scope | Preserve readable schematic/CSV exports and the already confirmed complete LED print. Qualify measured margins/media/scale, tiling, other settings and cancellation before claiming those paths/devices supported. No additional physical print is required solely to add evidence. |
| Formal MVP acceptance | Reconcile all 27 P0 requirements, all original quality targets, the electrical/workflow suite and Gates C/D against final-build evidence. The current audit is 17 implemented/10 partial; those counts describe recorded engineering coverage, not product acceptance. |

## Continuing rules

Preserve successful unaffected evidence and original failures. Reopen a release
item only for a concrete change, reproduced failure or uncovered requirement.
Do not add devices, analysis modes, cloud work or another broad framework upgrade
to this release. Use the owner's standing run/desktop authorization in
[AGENTS.md](../AGENTS.md); do not ask for it again.
