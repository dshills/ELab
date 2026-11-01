# Web release-confidence gate

This milestone prepares `exp-web` for promotion. It does not merge main or replace
the qualified native alpha or the preserved web alpha archive.

## Changes and internal review

- Native buttons replace the parts library's incomplete listbox semantics.
- All dialogs have accessible names; tabs identify their panels, use one tab stop
  per group and support arrows/Home/End through their existing activation paths.
- Drawing shortcuts respect text composition, consumed events, modal dialogs and
  browser modifier shortcuts. Command-K cannot stack a command dialog over another
  modal. Text editing retains normal undo behavior.
- Choosing a part moves focus to the schematic for Enter placement; the canvas
  itself respects composition/modifier guards. Drawing tools expose pressed state.
- Schematic accessible names contain the visible symbol annotations, identifier
  and value in drawing order, followed by the component type. The brand's spoken
  name matches its visible wording and omits its decorative mark.
- Focusable panel dividers expose their current size.
- Locked test-only Playwright/axe dependencies and independent CI jobs cover three
  browser engines. The runtime and packaged app acquire no npm dependencies.

Internal review checked shortcut propagation, dynamic tab rendering, native dialog
focus behavior, private test-library ownership, server shutdown, browser evidence
scope, pinned dependency integrity and workflow permissions. Tests activate the
real UI and native host without bypassing commands through test-only app APIs.
The browser host is opt-in locally and refuses to reuse an existing server.

## Local checks

- All 27 frontend logic tests and module syntax checks pass.
- All 108 Python tests pass, including dependency-lock qualification inputs.
- `go test -race ./exp-web -count=1` passes (12.074 s); `go vet ./exp-web` passes.
- `actionlint .github/workflows/quality.yml` and `git diff --check` pass.
- Playwright discovers 15 journeys, five for each engine, without launching a
  browser. Actual browser execution is delegated to hosted CI.
- No desktop launch, browser opening or screen capture was performed locally;
  the owner's subscription-video pause remains in effect.

The [first hosted run](https://github.com/dshills/ELab/actions/runs/41355009855)
at `814eba14` passed all four functional journeys on all three engines (12 total).
All three accessibility journeys caught the same visible-label/spoken-name
mismatch in the brand and schematic parts. Those defects are fixed in the
follow-up, with a new frontend regression; the gate remains enabled. A second run
passed the functional journeys including keyboard placement, but showed that
reordering spoken wording alone did not fix the names: adjacent positioned SVG
text nodes had no whitespace boundaries, and the visible decorative brand mark
was included by the visible-label comparison. The next fix adds real text-node
boundaries and lets the brand derive its name from its nondecorative contents.
The gate remains enabled for the next run.

The [final browser run](https://github.com/dshills/ELab/actions/runs/41355615048)
at `e8508429f507b1147d8223e1ed36598ff652554e` passes all 15 journeys without
retries. Inspection of the archived reports confirms all six accessibility scans
(workbench and projects modal on each engine) contain zero violations, including
moderate/minor findings. [Durable results](browser-results.json) retain individual
tests, durations, scan results and original HTML-report checksums. Reports are
kept in GitHub artifacts for 14 days; local copies are under the ignored build
directory. These results cover the tested views and do not certify the whole app.

The source runtime is unchanged by these evidence-only updates. The Linux
portable job passed all seven gates in 10m46s. Its
[original quality receipt](portable-results.json) confirms the full shared race
suite passed (585.737 s), vet passed, and source inputs remained unchanged.

The macOS job started at 22:15:13 UTC and remains in progress. The previous
successful main run's macOS job took about 62 minutes, so it is left running;
this record does not claim it passed. The CLI watch log is retained locally at
`build/web-ci-watch-final.txt`. Check the linked run's final result before
promotion. The preserved alpha packages have not been rebuilt or replaced.
Before this milestone, GitHub had only main; its
latest quality run at `cd6dcdcdd123c1356c8aa8e525a146b47ed96f22` passed. That result
does not qualify the web branch.

## Promotion gates

1. All branch CI jobs pass, including portable/native suites and browser journeys.
2. After the video pause ends, check the packaged candidate in installed macOS
   browsers, keyboard focus/VoiceOver and pointer resizing. WebKit on Linux is not
   a substitute for installed Safari, and axe is not a screen-reader evaluation.
3. Run an everyday-use pilot with disposable circuits: create a small LED circuit;
   edit/record/analyze RC; inspect the four-MOSFET bridge; save, quit/reopen and
   recover; export/import a native project. Record completion, confusing steps,
   data loss, errors and any advantage over the native reference. No data loss or
   blocked core workflow is acceptable for promotion.

The pilot requires human use and cannot be declared passed from automation.
Rebuild and qualify a new web package from the eventual promoted commit; the
existing alpha archive remains an immutable earlier candidate.

References: [WAI tabs](https://www.w3.org/WAI/ARIA/apg/patterns/tabs/),
[Playwright CI](https://playwright.dev/docs/ci),
[automated accessibility limits](https://playwright.dev/docs/accessibility-testing).
