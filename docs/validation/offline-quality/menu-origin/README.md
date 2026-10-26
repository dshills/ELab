# Examples menu command-context repair

October 26, 2026. A routed, retained-driver reproduction shows that an Examples
category choice can erase a newer agent search. The application now retires this
managed menu when its command context changes. No Vellum, public command,
component, electrical-model, file-format or solver-limit change is made.
The qualified /267 and every earlier release remain preserved.

## Reproduction and scope

The test extracts the actual assembled example-category selector, including
production field guards, document ownership and scoped theme, into a minimal
host. ELab intentionally keeps this sheet control managed by Vellum; ordinary
macOS selectors use its separate native proxy and revision guard. Actual driver
pointer events open the real menu and choose Timers. One MouseUp is routed while
its posted owner action remains queued. A public service agent then searches for
motor. Redraw refreshes the retained descriptor before the owner action drains.
Before repair, query motor becomes empty and Start here becomes Timers under a
new user revision. A pointer held across that agent edit reproduces the same loss.
[First failure](before.txt), [final baseline regressions](before-final-regressions.txt).

The broader baseline reproduces nine failures: queued search in four themes,
a queued choice after an unrelated editor edit, and held release in four themes.
Document replacement already refuses stale input. The existing toolbar More
menu also preserves an intervening agent pan against queued Fit, and fresh Fit
works. Those passing controls are retained without altering their implementation.

The browser category's ControlStateKey now includes its displayed command
revision, following the existing secondary-toolbar policy. An intervening agent
revision retires the popup and its queued action. Fresh independent selection
still calls examples.category.set, clears search and receives one user revision.
The general SDK selector, Switch/ToggleButton, programmatic button and non-list
group contracts remain unchanged.

The first repair caused an observed focus regression: a normal category choice
previously retained example-category focus, but context replacement lost it.
[Baseline focus](baseline-focus.txt), [intermediate regression](repaired-focus.txt).
The final callback requests that same control's focus only after acceptance.
The normal-selection and all six queued scenarios now verify focus explicitly.
[Final focused race checks](focused-final.txt) pass the six queued cases, four
held cases, queued overflow control and ordinary focus control.

These are actual routed queues/widgets and public service agent interleavings.
They are **not native macOS event-queue or hardware-rate acceptance**. The reduced
host does not qualify modal placement. Original native checks3/6/9/11 remain open;
no unbounded CUA attachment or identical Save-panel trial is repeated.

## Review and qualification

The initial full quality attempt correctly caught a stale checked-in UI inventory
and stopped at that failure. Its partial receipts are preserved in
initial-full-quality/. The inventory is reviewed before regeneration: only
example_browser.go's source hash, four line positions and its category callback
expression change. All187 public commands,456 sites and source routes remain.
The changed expression adds accepted-choice focus return around the same public
category command. [Inventory review](native-inventory-review.json).

The first expanded fixture assumed the browser remained visible after project
replacement; it actually closes, so the host now renders that real closed state
and explicitly reopens before the fresh-input check. That fixture failure is
preserved in after.txt; it is not counted as an additional product defect.

[Scope review](scope-review.json) verifies that1,353 of the prior1,354 compiled/
test inputs stay exact; the only changed production input is example_browser.go.
The new offscreen test is additional. The pinned SDK lock/archive are exact and
its complete /267 SDK qualification is carried, with no SDK gate repeat.
[Carried SDK scope](framework-complete.json). Nine prior package/SDK hashes remain
exact. Recipes are retained as text; inspect absolute workstation paths before
replaying them in another disposable workspace.

Complete application source qualification passes: portable, ordinary desktop
and rendered desktop race suites contain867 /339 /470 top-level tests, with
1,950 /839 /1,446 including subtests. Lint/imports,68 Python tests, PRD review,
all three vet modes and the complete build pass. [Final quality](full-quality/quality-complete.json),
[test counts](test-counts.json). These are the final reviewed inventory and source;
initial partial results remain labeled separately. /267 remains the qualified
download until the exclusive /268 copied-runtime/archive checks complete.
