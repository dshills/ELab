# Captured measurement entry recovery /214

October 23, 2026. Existing captured interval controls, without new commands,
components, models or file formats.

## Reproduced failures

The retained-widget baseline discards a negative start (`-1 ms`), an end past
the capture duration (`30 ms` for a 20 ms capture), and an end past actual saved
timestamps (`20 ms` for the accepted short-tail import). Each field immediately
returns to its previous accepted text, losing the entry the user needs to fix.
The cached measurement and complete accepted service snapshot remain unchanged.

A fourth mounted case types an uncommitted `4 ms` start, then lets an agent
select another captured channel with a zero start. The old private text follows
the newly selected channel instead of staying with its original measurement.

The [existing native /206 session](ui-audit/statistics-entry-214/README.md) also
replaces `-1 ms` with `0 s` after Return. In that old native session, re-entering
`0 s` does clear the error; a stuck native correction is not established. Its
temporary measurement is cleared and its inspector scroll restored. Full before
and after state differs only in revision counters; circuit, document, history,
captured data, cursors, windows, view and physics remain exact. No graphical
application is launched.

## Repair

Keep each rejected start/end entry separately from accepted measurements. A
refusal is retained only for its current command/document context, capture,
channel and accepted endpoint. Both ordinary invalid input and an uncovered
saved-time interval remain editable. A correction deletes only its own entry;
an unchanged rejected peer survives. A short note distinguishes last accepted
results from the currently rejected interval.

Retained entries are pruned when their endpoint, channel, capture, document or
recorded-view owner changes, and when measurements are turned off. Private
widget state now includes the captured measurement channel, matching the
existing threshold-control ownership pattern. An agent's channel selection
cannot acquire the old channel's uncommitted text.

Every accepted edit still uses `recording.statistics.set`. The service and its
public contract are unchanged. Stale parser failures cannot install text in a
newer form; already owned refusals remain correctable after unrelated agent
work. No saved rows, cached result or live circuit changes on refusal.

## Checks

Original mounted failures take 1.224 s. The four repaired mounted cases pass in
2.563 s. Final focused race checks pass in 4.360 s, including five stale callback
cases, rejected-peer preservation, six owner-pruning cases, exact imported
endpoint recovery, channel replacement and existing native-command parity.
One intermediate test run used a success-only UI helper for deliberately invalid
entries. Those driver setup failures are retained separately; the corrected
tests call the same production bound callback and explicitly assert refusal.
No application validation or expectation was relaxed.

The [complete quality gate passes](../docs/validation/statistics-entry-214/README.md):
lint/staticcheck, 67 Python tests, root/desktop race and vet, mounted race and
offscreen vet. Desktop race takes 182.462 s; mounted race 164.391 s. All
883 source/check inputs remain exact. The inventory retains 166 commands,
134 routes and 404 sites; the existing statistics ingress moves into its owned
entry helper, and native source count becomes 78. The SDK, backend source,
current /205 download and previous candidates are preserved. Changed-candidate
native acceptance remains separate from the old native baseline, source,
mounted and headless evidence. No new launch prompt or review request is queued.
