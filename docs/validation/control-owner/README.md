# Private numeric edits and presses belong to their logical owner

October 18, 2026. **Release blockers reproduced on /145 source.** The preceding text-only repair did not cover numeric editors or a pressed button.

- After typing private `75` into a 50% wiper field, replacing/reselecting the document or selecting P2 before one frame retains the numeric widget. Return changes the replacement/P2 wiper from 50% to 75%.
- MouseDown on R1's Rotate, agent selection of R2, redraw, then MouseUp rotates R2. The unchanged-owner control rotates R1 once.
- MouseDown on Restart, agent replacement/advance of the RC document, redraw, then MouseUp restarts the new experiment.
- One normal Apply click after typing `2.2k` can fail: focus loss accepts the draft at command revision 4, but the click still holds revision 3. This also fails against the archived original /145 desktop source and original SDK, so it is not introduced by the new identity work.

The retained original failures are [numeric](numeric-before.txt), [Rotate](rotate-before.txt) and [archived /145 Restart/Apply](archived145-restart-apply-before.txt). The archive run exports commit `0216eb8`'s desktop module into an owned build directory, adds only the new regression, and resolves dependencies to the original `178945c` SDK. Current root/domain production source is unchanged from that checkpoint.

The SDK adds `ControlStateKey` for twelve ordinary managed controls: button, checkbox, switch, toggle, selector, slider, number field, text field, radio group, segmented choice, list box and menu button. Different logical owners remount local state with the same public ID. Empty keys retain existing behavior. Caller-owned editor/grid/tree/placement models refuse this key; the earlier text-only key stays compatible. ELab applies exact document generation/object ID strings. Keys do not depend on command revision, so an own field commit does not cancel its Apply press.

Apply now retains a constant-size chain of consecutive accepted value-field contexts. A rendered Apply may advance only within that chain, with the exact current document, selected ID and draft. It uses accepted command responses, never a later snapshot to claim another actor's update. Failed draft commits break the chain. Agent-before/after, another user field and document replacement still refuse. Existing asynchronous settlement and failure paths remain in place.

The [isolated after](isolated-after.txt) checks real native widgets and pointer/keyboard routing, explicit fresh focus/edit/click, unchanged-owner controls, held replacement Restart, single-click Apply and earlier slider/text regressions. The SDK matrix verifies same-key retention and changed-key replacement for all twelve supported kinds. Root/driver race/vet and focused offscreen race/vet pass. The [full consumer gate](full-check.txt) passes audit freshness, lint/imports/Staticcheck, 43 Python tests, root/native race/vet, all mounted owner/text/slider regressions and tagged vet. Native race took 147.211 s; tagged race took 9.213 s. Replacement runtime qualification remains separate. [Checkpoint](checkpoint.json).

The [SDK member diff](sdk-diff.json), [portable patch](vellum.patch), [raw commit](vellum.commit.txt), [prior lock](prior-sdk.lock.json) and [command-ledger review](ledger-review.json) retain exact scope. No public command ID/schema, model or project format changes. Earlier caches, archives and sibling Vellum remain untouched. The native command ledger remains 62 production sources, 389 sites, 162 public IDs and 132 source routes; the Apply callback is the substantive binding change.

This is deterministic event-router evidence, not physically interleaved OS delivery, window blur, IME/VoiceOver, visible p95 or exhaustive mounted conformance. Owner walkthrough remains pending. /145 stays immutable and affected until a clean qualified replacement is handed over; no P0 status is upgraded.
