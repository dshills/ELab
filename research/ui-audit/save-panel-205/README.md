# Native Save panel observations

October 23, 2026. Isolated /205 native-code-equivalent test bundle, blank
schematic and disposable `/private/tmp/elab-value-guidance-205/save-panel-fixtures`
directory. The CUA controls operate the actual AppKit panel. The normal owner
ELab session and original files remain untouched.

- [Default name](01-default-name.jpg): suggested `Untitled` saves as exactly
  `Untitled.elab`.
- [Compound name](02-compound-name.jpg): selecting the entire current name and
  typing `Timer.v2 µ.elab` saves exactly that name.
- [Command-A selection](03-select-all-basename.jpg): after Command-A the blue
  selection still covers only `Explicit`, leaving `.elab` outside it. AX selection
  of the full name succeeds, demonstrating an actionable keyboard-selection
  difference. The captured text records the selected basename.

Explicit `Explicit.elab` and intentional `Intentional.elab.elab` saves also keep
their exact names. Five file/agent checks and retained fixtures are recorded in
[the receipt](../../../docs/validation/save-panel-205/report.json). The native
panel returned `/tmp` URLs for the requested `/private/tmp` destination. No
post-picker suffix stripping was applied. Standard panel editing shortcuts still
need a controlled repair/check; these observations do not establish a full
keyboard or screen-reader Save workflow.
