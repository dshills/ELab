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
