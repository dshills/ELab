# Validation layout and numeric drafts /222

[Receipt](report.json), [frozen inputs](source-inputs.json),
[raw evidence](raw.tar.gz), [member hashes](raw-manifest.json).

The running /221 app reproduces a real inspector defect: rejecting a scope
interval inserts guidance above the scrolled controls and hides the newly
focused field. The production inspector regression fails before repair.
A one-shot post-layout reveal retains the current focus owner while keeping it
visible. Deliberate scrolling away and replaced/disabled editors cancel it.
Four live/recorded start/end production cases pass without changing accepted
state. A separate managed numeric-field regression reproduces lost private text
after a parent structure change; the pinned native updater no longer resets it.
Native source selection already retained the draft on /221 and is not claimed
as a native reproduction of that separate layout bug.

The complete framework normal/offscreen race/vet and preparation checks pass.
The complete application lint/staticcheck, 67 Python checks, root and desktop
race/vet, production mounted race and offscreen vet pass. All frozen inputs and
8,437 SDK member hashes match. Upstream dependency/notices/bootstrap bytes are
unchanged. Historical caches, packages, the /221 installation and /205 download
are preserved. No verified cache is edited. New candidate native qualification
is separate; this report does not promote a release or close all native checks.


## Native qualification

[Native receipt](native-report.json), [captures and raw checks](native-raw.tar.gz),
[hashes](native-raw-manifest.json), [remaining checks](native-acceptance-pending.json).

The clean 0.181.21 /222 candidate and matching agent are built from `2d3d56fb`.
Its ordinary update retains the existing test installation identity, launcher and
sandbox. Normal CUA Quit and reopen succeed without another owner question.
The tested copy has identical nonempty Mach-O sections to the original sealed
candidate, whose exact bundle bytes remain untouched. /221 is retained as backup.

Native live/recorded start and end fields retain both malformed and out-of-range
text, keep current focus visible and accept the first correction to the accepted
value. Actual 0–20 ms live and 0–1 ms recorded bounds and units are readable in
the Simulate guidance. Accepted physical/document/recording state stays exact;
viewport/revision/actor changes are reported separately. Focused and unfocused
numeric wheel checks leave the complete public snapshots identical. An unfinished
0.25 V draft survives wheel input, scope layout changes and agent repaint.
Deliberate scrolling away stays put; returning shows the same private draft.
Typing, plus and Up still commit through shared state. Their raw float32 step can
show excess digits (`0.031999998`); that cosmetic issue is recorded, not passed.

All 1,001 stored rows match the preceding native fixture. The 2,044 independent
live/recorded voltage references pass, maximum absolute error below 5e-16 V.
Signatures, original saved file, 894 source inputs, candidate and /205 download
remain unchanged. The app is left clean, paused, Light, with scope visible.
Checks 7 and 16 have fresh /222 native evidence; six other accumulated checks
retain their earlier scoped evidence. Ten of the original eighteen remain open,
including the native Save Command-A observation and controlled divider speeds.
This package stays held; it does not replace the qualified /205 download.
