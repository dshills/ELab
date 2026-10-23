# Component browser /201

October 23, 2026. ELab 0.181.0. See the [receipt](report.json),
[packaged-agent checks](protocol-report.json), [native checks](native-ui.json),
and [package validation](package-report.json). The user/agent contract is in
[component-browser.md](../../component-browser.md).

Common, six families and global name/alias search organize the unchanged 18
kinds. Rows reuse the sheet's symbol geometry. A local presentation adapter
adds artwork to the existing managed ListBox; the pinned SDK is unchanged.
The final All view scrolls within that bounded list, with category/search above.
No device models, file format, simulation limits or editing behavior were added.

Shared category/search commands publish session-only state and preserve the
graph, simulation, history and document dirty flag. Tests cover UI/agent parity,
category inventories, global aliases, invalid/stale admission, isolated metadata,
keyboard search/selection, and symbol coverage. The packaged agent additionally
checks retained retry ownership and direct placement outside the visible group.

Two initial default package-parallel quality runs hit the existing flasher
integration work-time limit. The identical full suite passed with GOFLAGS=-p=1,
including conformance, race, vet, mounted tests and all 67 Python tests. No limits
were changed and no tests were skipped. Logs retain all three attempts.

Owned native previews verified Common/category names, vector symbols, global
search and keyboard selection, plus external-client category changes. Preliminary
captures include blank/stale images and an unsuccessful full-height outer-scroll
experiment; they are retained under early-ui and do not qualify the final layout.
Final packaged UI observations and images identify the released binary separately.
CUA wheel/thumb coordinate interactions and End were inconclusive; pointer
scrolling is not claimed as qualified. Ordinary Down/Return did reveal/select
the final row, and Shift-Tab/type/Return verified global search. The owner app's documents were preserved. Old packages were not replaced.

ZIP extraction into a Unicode/spaces path preserves members and modes and passes
strict ad-hoc signature verification. Developer ID/notarization, clean-account
installation, complete PRD and VoiceOver/IME acceptance remain outside this work.
