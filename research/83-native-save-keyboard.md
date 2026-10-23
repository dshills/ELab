# Native Save paths and keyboard selection

October 23, 2026. Baseline for overnight item 4, collected in the isolated /205
test app while the cursor repair's quality gate ran. The Save implementation is
byte-identical to current source; this does not qualify the not-yet-built /206
native bundle. [Evidence](../docs/validation/save-panel-205/README.md).

The default basename, an explicit `.elab` filename, a compound Unicode name and
an intentionally repeated `.elab.elab` suffix all save exactly as shown in the
native dialog. Matching public-agent loads reopen all four blank schematics
cleanly. A fifth direct-agent save preserves a requested extensionless path.
Original fixture hashes stay unchanged. No path-stripping repair is justified;
it would destroy an explicitly requested repeated suffix.

A related keyboard defect is reproduced: Command-A in the native filename field
leaves the initial basename selected without its extension. Replacing the full
name through AX selection works. The app's menu bar contains File, Examples and
Appearance but no standard Edit menu. The pinned native menu service replaces
the application's main menu, and the native panel uses AppKit's own text field.
Missing responder-chain editing actions are a cause to test, not yet a verified
repair. Preserve the first failure and compare Command-A before/after a narrowly
scoped native-panel change. Do not implement global filename normalization.

The initial fixture verification incorrectly required the panel to return a
`/private/tmp` spelling; AppKit returned `/tmp`. Resolved paths identify the same
owned directory. Direct-agent paths are still checked as exact strings. One CUA
click used an index retired by a full-tree capture and did nothing; fresh IDs
were read before saving. The keyboard replacement attempt that appended to the
basename is retained as a failure, not a keyboard-save success.

Next: finish packaging the cursor repair, then investigate a standard AppKit
editing menu for native panels. It must respect the first responder, restore
the previous application menu on completion/cancellation, and preserve existing
Vellum field shortcuts and all explicit user/agent paths. Test the actual keys
and native modal lifecycle, including Cancel and owned-session shutdown.
