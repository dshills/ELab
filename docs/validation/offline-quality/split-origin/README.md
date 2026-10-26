# Divider callback origin

October 25–26, 2026. Qualified additional unsigned **ELab 0.195.5 /266**,
clean binary source `a23e56f265c8d300df3ca9afae4b419c0bbbbff7`.
[Qualification receipt](qualification.json), [archive receipt](archive.json).

A rendered ELab regression reproduces a queued divider
request overwriting a later agent layout after the retained descriptor rebinds:
0.75 becomes 0.6699999570846558 and command revision 6 becomes 7, actor user.
The originating KeyChord is dispatched with a one-event budget; its owner
callback remains queued. A public service agent command then changes the layout,
and a benign routed event refreshes the descriptor before owner delivery.
This uses real driver queues and ELab's workspace callback/context binding, with
minimal pane contents. It is not a replay of a captured Change callback, nor
proof of a hardware/native OS queue interleaving.

A held off-center pointer gesture also borrows a refreshed callback after a
layout change, RC input change or document replacement. The corrected baseline
has four failures: one queued-key case and three held-agent cases. The initial
trial used invalid RC mode `high`; the fixture was corrected to `charge`, and
both raw logs are preserved. Unchanged repaints and fresh gestures are positive
controls, alongside forward/inverse quick and classified-slide geometry tests.

SDK elab6 captures the callback when a queued event is admitted and when a
pointer gesture begins. Latest bounds still clamp and disabled/retired or
reordered/reoriented splits still refuse. ELab's existing field context advances
only after its own accepted dispatch, so an uninterrupted held drag continues.
The driver's former latest-callback test is intentionally updated: the concrete
ELab/agent reproduction demonstrates why callback refresh is unsafe. A fresh
independent request still uses the rebound callback.

No public commands, component kinds, models, solver limits or project formats
change. The SDK's full archive member comparison changes only split.go,
split_test.go and generated source documentation; no dependency contents change.
All qualified packages and earlier SDK archives remain preserved.

## Complete qualification

Full portable, ordinary-driver and rendered-driver SDK race suites pass, with
both vet modes and focused callback compatibility checks. Restricted attempts
could not bind disposable Unix sockets; authorized repeats pass without security
setting changes. The first rendered attempt preceded final callback-closure
cleanup; the final authorized run qualifies exact SDK source 303cd0ca.
[Framework scope and raw logs](framework-complete.json).

Complete ELab race suites pass: 867 portable /339 ordinary desktop /465 rendered
top-level tests (1,950 /839 /1,426 including subtests). Lint/imports, Python, PRD,
all vet modes and build pass. [Counts](test-counts.json),
[quality gates](full-quality/quality-complete.json). All 1,353 frozen source/test
inputs stay exact through qualification; the packaged app and agent embed the
clean source identity. These are functional checks, not calibrated timing trials.

Representative actual native /266 off-center forward/inverse drags move the
canvas −60 px, then restore exact viewport geometry and the original share
within 1e-6. Graph, document, physics, history and measurements stay exact. The
native Down shortcut produces no reported AX change; its acceptance is not
claimed. Comparator corrections for unchanged actor and float32 share rounding
are retained in [the native receipt](native266/native-check.json). The owned
session closes normally. This does not establish controlled hardware rates or
explain the earlier CUA inverse-drag observation.

The exact copied bundle passes native smoke and four-adapter public electrical,
project, inductor, op-amp, storage and capacity suites with source/developer-file
and outbound-IP denial. All 102 external processes exit normally. Every final
ZIP member's bytes/mode and strict ad-hoc seal verify after macOS extraction;
the extracted matching agent freshly reopens all 62 paused examples cleanly with
exact graphs. [Runtime](runtime-report.json.gz), [public suites](public-report.json.gz),
[process exits](runtime-process-exits.json), [offline examples](offline-examples.json).

Final local ZIP: `build/releases/ELab-0.195.5-macos-arm64.zip`,
38,748,268 bytes /375 files. SHA-256:
`cbc3f687068d932fa148d2a580b4116f4755582184435b2ec6718d80be37a863`.
App: `build/ELab-0.195.5-266-Divider/ELab.app` and matching adjacent agent.
No Developer ID/notarization, upload or push. Qualified /265, /264, /260 and /205
bytes remain unchanged. Owner walkthrough and existing clean-account,
participant/screen-reader/IME and physical hardware deferrals remain.

## Packaging controller corrections

The first reopen invocation used a relative workspace extraction path after
changing cwd, so it never executed the agent. The unchanged draft ZIP was
re-extracted outside the source-denied tree and its examples passed using an
absolute path. A 374-file draft also lacked the separate rollback document;
its bytes/receipts remain in staging. The final 375-file ZIP includes rollback
guidance and passes fresh extraction/reopening. App and agent bytes stay exact.
[Initial extraction](archive-initial-workspace-extraction.json),
[draft receipt](draft-archive.json), [raw path failure](offline-relative-path-failure.txt).
These are controller/assembly corrections, not product import failures.

Original native checks 3, 6, 9 and 11 remain open. Routed queue evidence does not
qualify native pre-agent parser origin or Save-panel keyboard dispatch. Preserve
that distinction in subsequent work and do not repeat completed full gates for
unchanged source.
