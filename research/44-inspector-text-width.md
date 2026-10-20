# Inspector text width follow-up

October 19, 2026. Issue: **UI-INSPECTOR-WIDTH-01**, P2 readability. The native /183 MOSFET review showed circuit guidance constrained to roughly 80 pixels despite the wider Inspector. This is a bounded repair to the existing first-version readable-controls requirement.

Text-only accessibility groups and their label frames could shrink to the longest word. The repair gives groups the existing 248-unit Inspector content width and wrapping frames their bounded width; narrower constraints still clamp them. IDs, text, accessibility groups, focus, user/agent commands and circuit state remain unchanged. No SDK, solver, model, format or feature expansion is introduced.

The final mounted regression fails six subcases against the original production layout and passes with the repair: RC, NMOS, PMOS and CMOS on initial render/repaint, plus 152/248-unit rail bounds. Text wraps within 236-pixel frames rather than a tall narrow column, and the entire service snapshot stays exact. The initial theme error, unsuccessful group-only repair and overly strict natural-text-width assertion are retained separately. The new native smoke guard still needs copied-bundle qualification.

Full default lint/race/vet/mounted quality passes in **229.253 s**, freezing **659 application Go files and three existing tool sources**. The binding inventory changes the two layout-source hashes and line locations only; all 400 non-line sites, 164 commands, 132 routed commands and aliases are identical. [Source proof](../docs/validation/inspector-width/README.md).

At this source checkpoint, /183 remains the qualified private review and /184 packaging is pending. Owner walkthrough remains Not tried yet, P0 stays 17 implemented /10 partial, and broader gates remain open. No promotion or new physical input/latency qualification is inferred.

## Native guard correction

The attempted /184 native smoke exited 1 because the new check compared naturally sized text allocations with group width. All eleven labels were complete, untruncated and within their text boxes, but frame geometry was not recorded. This is retained as an unqualified failed candidate, without a release receipt. The corrected guard measures the named wrapping frames and checks text containment separately. Only native smoke source changes; layout, commands and binding ledger stay exact.

Final full default quality passes again in **233.191 s**, freezing the same 659 application and three tool sources. [Final source proof](../docs/validation/inspector-native-frame/README.md) includes the earlier width proof and exact failed candidate/diagnostic. A clean /185 replacement is next; /183 remains the qualified review. No native-fix acceptance is inferred from source checks alone.
