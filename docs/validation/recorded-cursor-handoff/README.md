# Recorded cursor edit → Clear/Select handoff

October 18, 2026. A bounded repair to an existing measurement workflow. The initial hypothesis that typing `0 s` could not place a recorded startup cursor was disproved: the mounted test already passes. No startup-field behavior was changed.

## Reproduced failure

On a paused RC circuit, create a 20 ms recorded verification. Set cursor A to 4 ms, type `8 ms` into its native field, then click Clear once. Real mounted mouse/key routing commits A to 8 ms on focus loss, advancing command revision 6→7; the held Clear callback still expects 6 and refuses. A remains placed, with `expected command revision 6; current command revision is 7`. [Original failing run](before.txt).

## Change

Recorded Select/Clear now use the same asynchronous field-settlement path as live scope cursors. The action waits for both recorded time fields and continues only through their consecutive accepted responses. Each continuation and pending action retains its recording ID; another recording, live-field command, agent edit or document replacement breaks the original context. No generic revision refresh/retry is introduced. The field still invokes public `recording.cursor.set` with its captured operation ID; Select/Clear invoke the existing public recording commands with command/document guards.

The common helpers preserve the existing live-scope behavior. The measurement-physics invariant was moved from the offscreen-only test file into the ordinary scope test helpers so both test modes enforce it. The source-inventory parser recognizes the recorded factory and shared six-argument relay, with literal/dynamic/unknown-command tests. There are 68 production native files, 393 sites, 160 requiring binding review, 207 literal routes, 162 public commands and 132 referenced IDs. Public command/alias/coverage sets are unchanged; the two new sites are internal relay calls, not new product capabilities. Only `cursor_actions.go` and `recording.go` change production native hashes.

## Validation

The mounted RC test uses actual routed pointer/keyboard input and Cogent field settlement. For Clear and Select, it types privately, verifies no public state changes before commit, then sends one click. Exactly the time commit plus requested action occur. The resulting recorded scope exactly matches an independent in-process public agent. Run/electrical identities, components, wires, transient/mixed physics, document and circuit Undo/Redo stay unchanged. Startup-time Enter also matches the agent.

Deferred-field tests require A and B to finish before Select and reject duplicate activation/completion. Intervening agent cursor edits, another selected recording, live-scope changes, document replacement, invalid time and field admission/composition failure preserve the intervening state. Existing RC/flasher live-scope cases still pass. [Focused race run](focused.txt).

The development native executable passes the existing full smoke ([log](native-smoke.txt)). Its build metadata is explicitly dirty-tree ([build info](development-build-info.txt)); it is not a new release or package qualification. No CUA/physical OS typing, IME, reader or visible-p95 claim is made. Retained native input across all recording/view switches remains a broader qualification item. The owner walkthrough is still “Not tried yet,” and P0 remains 17 implemented /10 partial. /154, canonical /120 and the frozen SDK remain preserved. A clean replacement package is the next release step.

The complete final `scripts/check.sh` passes: PRD/source inventory freshness, import formatting, Staticcheck, 51 Python tests, root/native race tests and vet, required mounted tests and tagged vet. Native race: 151.518 s; mounted race: 16.310 s; focused live/recorded cursor race: 6.491 s. [Full quality log](full-check.txt), [checkpoint](checkpoint.json).
