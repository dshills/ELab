# Captured comparison entry recovery /215

October 23, 2026. Existing capture comparison controls; no new commands, parts,
models, formats or SDK changes.

The original retained-widget tests reproduce three lost comparison-time entries:
malformed `oops`, negative `-1 ms`, and out-of-range `30 ms` on two 20 ms
captures. Accepted comparison data and the full service snapshot stay unchanged,
but the field replaces the rejected text with `0 s`. A fourth case types a
private `4 ms` request, then lets an agent choose a different reference capture.
The private text follows that new reference instead of staying with its owner.

The comparison time now retains its rejected text with the accepted time,
selected/reference capture IDs and document generation. Retention requires the
failing callback's current command context; old parser failures cannot install
text after agent work. A short note distinguishes the last accepted comparison
from the rejected request. A fresh correction clears that entry and preserves
the agent's reference, draft, overlay and whole-run-statistics preferences.

Changing a display preference preserves a currently owned refusal. Changing the
time, reference, selected capture, document or recorded-view owner prunes it.
An explicit reference choice restarts at zero and clears the old rejected time,
including when the same reference and zero time were already accepted. Private
widget state is scoped to both selected and reference captures.

The original four mounted failures take 1.235 s; the repaired mounted checks
pass in 2.445 s. Final focused race checks pass in 4.847 s, including eight stale
callbacks, seven pruning/reset cases, preservation through preference changes
and existing shared-query equality. Original fixture and mounted-test bytes are
retained beside the failure logs. The public command/alias/route sets stay
unchanged: 166 commands, 134 routes and 404 sites. The existing comparison ingress
moves to its owned-entry helper; native source count becomes 79.

The [full quality gate passes](../docs/validation/comparison-entry-215/README.md):
lint/staticcheck, 67 Python tests, root/desktop race and vet, mounted race and
offscreen vet. Desktop race takes 176.064 s; mounted race 155.393 s. All
886 source/check inputs remain exact. The planned matching-agent check compares
different saved RC rates and resistances using each actual selected timestamp,
then edits the live resistor to verify capture immutability. This is separately
scoped from fresh native acceptance of the UI changes. The current /205 download,
all earlier candidates and the existing owned app are preserved; no graphical
launch or permission request is queued.
