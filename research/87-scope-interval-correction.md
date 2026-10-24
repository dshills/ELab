# Correcting scope intervals without losing input

October 23, 2026. Continuing the existing measurement and input-contract work
from [the overnight plan](80-overnight-work-plan.md). This repairs live and
recorded start/end entry; it adds no analysis mode, device, command or format.
The qualified /205 download and held /209 candidate remain preserved.

## Reproduction and native baseline

All four mounted fields discard rejected input on repaint. A start of −1 ms
becomes the prior 0 s; an end of 30 ms becomes the prior 20 ms. Re-entering the
accepted text can then produce no native Change, leaving the prior refusal
visible. Shared command feedback also hides the actual available interval.
The baseline tests fail independently for live and recorded fields.

The already-running owned /206 native fixture reproduces the live end case:
typing 30 ms and Return restores 20 ms and shows generic refusal text. No new
application is launched. A temporary valid 18 ms edit clears the refusal, then
20 ms and Follow live history restore the exact original scope. Fresh agent
reads compare the complete before/after snapshots: only presentation and
command revision counters change. Graph, samples, results, document, history,
recording, cursors, viewport and all other snapshot fields are unchanged.
[Native baseline captures](ui-audit/scope-interval-210/README.md) are from old
code and do not qualify the changed binary.

## Repair and ownership

Rejected interval text remains private to its document, live run or selected
recording and accepted window. Repainting and rolling live history preserve it;
changing either endpoint of a fixed interval, restarting a live run, replacing
the document or selecting another capture invalidates it. A recorded draft
survives an unrelated live restart. Stale parser failures cannot install text
in the new owner's field.

Correcting a field to its prior accepted value still submits once. Correcting
one endpoint preserves its peer's unchanged rejected entry. Explicit Follow
live history / Last 200 ms / Show complete recording actions clear the target
interval's private entries after the shared command succeeds.

The first-click reset tests also reproduce a valid blur commit followed by the
reset conflicting with its own older revision. An accepted field response now
provides a bounded continuation for its related reset. Exact current command,
document, run/recording and accepted-window checks prevent this continuation
from crossing an agent edit. Mounted held-press and retained-callback tests
prove that the agent's interval survives and a fresh retry still succeeds.

## Shared feedback and validation

Strict, current `scope.window.set` and `recording.window.set` refusals explain
the actual history/capture bounds, start-before-end relationship and time-unit
syntax. Expired live history uses its current lower bound. A DC recording
explains its one time point. Ill-typed, missing, extra or duplicate fields,
wrong modes and stale request contexts do not acquire invented range guidance.

Pure shared domain helpers preserve the existing validation, roundoff,
recorded-boundary normalization and technical error strings. The guidance uses
those same validators without mutating state. Original error details remain
available. Valid commands, numeric SI output and file schemas are unchanged.

## Validation checkpoint

Focused source guidance, legacy window/cursor contracts and mounted correction,
peer preservation, explicit reset, private-entry first-click, held-agent and
owner-change checks pass. The complete final mounted focus run takes 16.008 s.
The public command/alias/route sets remain unchanged: 166 commands, 134 source
routes and 404 source sites. Seven changed native sources include one new
private interval helper, for 77 production files; counts alone are not full
dynamic conformance evidence.

The [complete source gate passes](../docs/validation/scope-interval-210/README.md):
lint/staticcheck, 67 Python tests, full root/desktop race and vet, mounted UI race
and offscreen vet. Desktop race takes 180.336 s; mounted race 160.790 s.
All 873 captured source/check inputs remain exact. The SDK and /205 ZIP
are unchanged. The raw receipt preserves original failures, focused passes,
native before/after snapshots and the corrected fixture assumptions separately.

Fresh native acceptance of the changed interval behavior and the accumulated
/207–/209 changes remains separate. Preserve /205 until those paths pass; no
new owner approval or unattended launch prompt is requested. The current owned
old-code fixture remains paused and restored for applicable subsequent checks.

The clean 0.181.9 /210 app and matching agent build from `c9c72f2c`.
Strict ad-hoc bundle verification passes. The matching packaged agent passes
refusal/correction and unchanged-state checks for live and recorded intervals.
Both 4–16 ms windows contain 13 samples; 52 voltage/current readings match
independent RC equations. Maximum errors are 8.88178e-16 V and
8.13152e-20 A. Workers join, the agent exits zero, all 873 inputs
and the app bundle remain exact. [Receipt](../docs/validation/scope-interval-210/candidate-report.json).
Native acceptance remains pending; no graphical candidate launch is used.
