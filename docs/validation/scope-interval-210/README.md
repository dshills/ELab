# Scope interval correction /210

October 23, 2026. [Receipt](report.json), [source/check inputs](source-inputs.json),
[repair notes](../../../research/87-scope-interval-correction.md),
[old native baseline](../../../research/ui-audit/scope-interval-210/README.md).

All four interval fields originally discard rejected text. The independently
running old /206 fixture reproduces the live end case through actual CUA input.
It is restored afterward: complete before/after snapshots differ only in the
presentation and command revision counters. No new graphical launch is used.

The repair retains private rejected text with its accepted window and owner,
allows correction back to the accepted value, preserves an unchanged rejected
peer, and makes explicit reset actions clear private interval entries. A valid
blur commit initially makes Reset conflict with its own revision; the narrow
own-field continuation repairs that first-click failure without crossing an
agent edit. Shared command guidance explains actual bounds, order and units;
malformed/stale requests do not borrow unrelated history or captures.

Focused tests cover both endpoints, live/recorded source ownership, sampling,
restart/capture replacement, stale parser callbacks, first-click resets, held
agent interleavings, retained callbacks and unchanged full physics/document/
history state. Corrections match an independent public service's scope. Shared
domain tests retain legacy boundary normalization and DC-point behavior.

The complete source gate passes: lint/staticcheck, 67 Python tests, full root
and desktop race/vet, mounted UI race and offscreen vet. Desktop race takes
180.336 s; mounted race takes 160.790 s. The final focused mounted race
run takes 16.008 s. Static inventory retains all 166 public commands, aliases
and 134 route IDs, with 404 sites and 77 production native sources. This does
not establish complete dynamic binding or accessibility acceptance.

[Raw records](raw.tar.gz) and [member hashes](raw-manifest.json) include original
failures, before/current source snapshots, corrections to test producers,
focused/full checks, native snapshots and packaging producers. Two formatting
commands failed in the wrong directory before tests ran. The initial held
fixture assumed blur precedes mouse release; the corrected physical test avoids
that assumption and a separate callback test proves the accepted-continuation
case. These fixture mistakes are separate from the reproduced product defects.

All 873 captured source/check inputs match. No SDK bytes, model,
command set, file format or current download changes. The /205 ZIP still hashes
to `d13973f02f9c2895d353d24338ed060492a3a2200ff8a91e3606bfa867f4946b`. Prior candidates are preserved.

Fresh changed-binary native interval/correction/reset acceptance and the
accumulated /207–/209 native checks remain pending. Offscreen evidence and old
native baseline cannot qualify those paths. No new owner permission or review
is requested and no unattended launch prompt is queued.

The [clean 0.181.9 /210 candidate](candidate-report.json) builds from
`c9c72f2ca69ac2206a8bb8ed618415b025ee7fc8` and verifies its strict ad-hoc bundle
signature. Its matching agent passes live/recorded interval refusal, unit
correction, exact protected-state and stale/malformed-context checks. The 4–16 ms
windows each contain 13 samples. All 52 scalar readings match independent
5 V / 10 kΩ / 1 µF RC equations; maximum errors are 8.88178e-16 V
and 8.13152e-20 A. Its shutdown joins workers and the agent exits zero.
All 873 source/check inputs and the complete app bundle still match.
[Fresh native acceptance](native-acceptance-pending.json) remains pending, so
the candidate is held and the /205 download stays unchanged.
