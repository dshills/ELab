# Native window and pointer repair /199

ELab 0.180.1, October 23, 2026. [Receipt](report.json),
[source/build inputs](quality-report.json), [package verification](package-report.json),
[design and cause](../../../research/74-titlebar-pointer-repair.md).

The standard macOS title bar replaces the unified full-content style; the editor
inset is removed. Common pointers now use standard OS artwork through a pinned
Vellum compatibility patch. Custom shapes retain the raster fallback. No shared
commands or electronics features change.

Full source checks and focused framework cursor/pointer race tests passed. The
initial quality run stopped at an intentionally strict stale source inventory;
review confirmed only hashes/locations changed, then the complete check passed.
An initial software-only cursor fixture needed both native backends replaced;
that fixture was corrected before the passing final framework suite.

The isolated native app exposed title-bar controls and accepted native Search
input. CUA screenshot captures returned blank Metal content; drag and double-click
were attempted, but window displacement and native cursor appearance are **not
visually verified**. The first launcher-path attachment timed out; direct bundle
attachment worked. The screenshot is retained as evidence of this limitation.

LED and RC native endpoint results matched the exact packaged agent and historical
references. Flasher timing hit the existing bounded solver's 100 ms work budget
twice; that case is not qualified here. Both raw failures are retained. This repair
does not modify the solver or relax its limits.

The ZIP extracted into a Unicode/spaces path with all 127 members and modes intact;
its strict ad-hoc seal verified. The isolated app shut down with workers joined
and endpoint removed. The preceding /198 package and running owner application
were preserved. No public upload, installation or owner acceptance is asserted.

After isolated-session cleanup, the normal /199 release was opened separately.
A read-only CUA screenshot showed the canvas, library and inspector rendering
correctly, with the standard title bar. That screenshot and AX state are retained.
The new document was blank and untouched. A later Raise action timed out; no more
UI actions were attempted. Drag displacement and cursor appearance remain
unverified. The normal /199 app is left open for use; /198 was not closed.
