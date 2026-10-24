# Recorded-position feedback stays with its request

October 23, 2026. Continuing input/agent contract work from the overnight plan.
This repairs stale seek feedback and late private editor input. Commands,
validation, captured rows, models, formats and the pinned SDK remain unchanged.

## Reproductions

The exact /210 packaged agent captures 20 ms, then selects a new 10 ms capture.
A malformed time request carrying the older command revision reports the new
capture's 0–10 ms / indices 0–10 range. Its accepted state is unchanged, but the
guidance describes the wrong request context. The disposable agent shuts down
normally and exits zero; no graphical application is launched.

UI callback tests independently retain an old Recorded seek time field, apply
an agent cursor, speed or live-component edit, then deliver the old field's
malformed text. All three cases install `oops` in the newer editor despite the
stale origin. Original failing logs and the packaged response are preserved.

## Repair

Request diagnostics now check all supplied presentation, command and document
guards before attaching target locations or specializing feedback. Cursor,
interval, component-value and recorded-position help share this rule. Parser
error precedence and the original technical error remain unchanged; an old
quantity parser error is not relabeled a conflict.

Position-specific help requires a strictly decoded seek shape with numeric
index or numeric/unit-text time intent. Unknown, duplicate or ill-typed fields
keep generic help. Fractional or oversized finite numeric indices, absent/null
seek alternatives and invalid time quantities retain useful whole-index/time
guidance for the current capture. Existing DC guidance is preserved.

The UI retains a rejected playback entry only while its captured command,
document, recording, visibility and accepted field value still match. An old
parser failure cannot install text or create a continuation in the new editor.
A fresh 3 ms correction still succeeds and preserves the agent's other edits.

## Validation checkpoint

Focused guidance/cursor/interval/value tests pass in 1.574 s. Callback and
existing mounted playback/private-buffer tests pass in 5.931 s. The command
inventory changes only one source hash and following line locations; every
semantic site, command, alias and route set remains identical (166 commands,
134 routes, 404 sites, 77 production native files).

One added test initially expected raw JSON `1e309` to receive range help. The
shared strict decoder cannot represent that number, so it correctly remains a
generic decoder failure. The corrected test distinguishes it from the valid
quantity-string intent `"1e309 s"`, which receives current-range guidance.
Neither decoding nor accepted values were relaxed to make the test pass.

The [full source gate passes](../docs/validation/recorded-position-211/README.md):
lint/staticcheck, 67 Python tests, root/desktop race and vet, mounted UI race and
offscreen vet. Desktop race takes 170.080 s; mounted race 153.997 s. All
876 captured source/check inputs remain exact. The SDK, current /205
download and earlier candidates are preserved.

Changed-binary native acceptance remains separate from source/offscreen and
headless-agent evidence. Use the saved authorization without a new permission
request or unattended launch prompt. The owned native session is untouched by
this checkpoint.
