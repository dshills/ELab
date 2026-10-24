# Queued field callback ownership

October 24, 2026. Repair in the existing Vellum compatibility layer; no new
commands, components, models or formats.

While investigating the remaining native parser checks, source review found
that ordinary queued text/number Change events replace their originating callback
with the latest descriptor's callback at owner delivery. Thus an edit admitted
before agent work can borrow a later repaint's command context, bypassing ELab's
otherwise correct old-callback refusal guard.

Two retained native-widget queue regressions reproduce this deterministically:
admit Change, reconcile a newer callback without replacing the widget, then drain
the owner queue. Both fields call the newer callback before the repair. The fix
retains the admitted callback for text and number fields, as sliders already do.
Fresh independent events still use the new callback and retain their exact
payload. Existing settlement epochs and record/window lifetime checks remain.

Both regressions and focused related checks pass. Complete framework preparation,
normal/offscreen race and vet checks pass. An initial sandboxed broad run could
not bind temporary Unix sockets; its log is preserved, and the authorized rerun
passes. The SDK export contains 8,440 exact members, with changes confined to
controls.go, the new regression and generated provenance. Release metadata,
dependencies, notices and verifier/bootstrap bytes are unchanged.

ELab now pins private elab14 from clean source
`23063545f871d8de0724077744d53f8f90d32c69`. Full application quality passes: lint,
67 Python checks, root/desktop race and vet, mounted-control race checks and
offscreen vet. All 900 frozen application/check inputs are unchanged through the
gate; both old and new SDK caches are reverified. [Evidence and exact logs](../docs/validation/queued-field-origin/README.md).

The owner returned and instructed us to finish the current work and stop. This
source repair is saved; no new candidate is built, launched or promoted. The
installed /223 executable, matching agent, original candidate executable, saved RC
fixture and preserved /205 ZIP retain their exact hashes, and the installed app's
strict signature passes. The four original native checks (3, 6, 9 and 11) remain
open; framework queue evidence does not itself close native event-queue
qualification. Affected candidate/native checks are deferred. Work stops here.
