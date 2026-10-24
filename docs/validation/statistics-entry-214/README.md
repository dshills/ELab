# Captured measurement entries /214

October 23, 2026. [Receipt](report.json), [source inputs](source-inputs.json),
[repair notes](../../../research/91-statistics-entry-recovery.md).

The mounted baseline loses three rejected time entries and lets an old private
start follow an agent's channel selection. The existing [native /206 baseline](../../../research/ui-audit/statistics-entry-214/README.md)
also replaces `-1 ms` with `0 s`; re-entering `0 s` succeeds there. No stuck
native correction is claimed. Its restored full snapshot differs only in
revision counters. The owned app remains open; no graphical app was launched.

Rejected measurement endpoints now retain their own text, accepted endpoint,
capture/channel and document owner. Ordinary invalid and uncovered-time entries
are correctable; one correction preserves its unchanged rejected peer. A note
identifies last accepted results. Private widget state is scoped to the measured
channel. Stale callbacks cannot install new entries after agent work.

Four mounted repairs pass in 2.563 s. Final focused race checks pass in 4.360 s,
including five stale callbacks, six pruning cases, peer preservation and existing
shared-command equality. A success-only helper initially rejected the tests'
deliberately invalid input; that harness error is preserved and corrected without
relaxing application validation. [Raw checks and producers](raw.tar.gz),
[member hashes](raw-manifest.json), and native captures retain the failures.

The complete gate passes lint/staticcheck, 67 Python tests, root/desktop race and
vet, mounted race and offscreen vet. Desktop race takes 182.462 s; mounted
race 164.391 s. All 883 source/check inputs match. Inventory retains
166 commands, 134 routes and 404 sites; one existing ingress moves to the owned
entry helper and native sources become 78. Backend source, SDK and /205 remain
unchanged. Fresh native acceptance of the changed candidate remains separate;
no unattended launch prompt or owner review is requested.
