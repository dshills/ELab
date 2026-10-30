# Experimental native UI implementation

Branch: `exp-ui`. Do not merge to main.

Visual truth: the four supplied 1440 × 900 PNGs and `exp-design-spec.md`.
The supplied spec is preserved verbatim.

## Native adaptation

ELab is a Go/Vellum macOS application, rather than the browser application assumed
by the draft. Implement native retained controls and shared commands; do not
replace the simulator or introduce a separate web frontend. Central Go palette
values are the equivalent of CSS tokens. Keep the real OS title bar and window
movement; do not paint imitation traffic lights.

Preserve real file saving, recovery, explicit Save/Save As, exports and dirty-file
protection. Browser URLs, localStorage, iCloud, invented solver quality settings,
and unsupported metadata must not appear as working controls. Existing advanced
features remain reachable. Do not change wire electrical semantics, frequency
analysis conventions, cursor signs, solver limits or saved file format as a
side effect of matching a mockup. Optional sheet notes are deferred.

## Work order

1. Full-height Parts / centered toolbar and sheet+dock / full-height inspector.
2. Native palette, typography, segmented navigation, field and action rhythm.
3. Floating drawing and zoom controls; compact notices; named-wire/sweep panels.
4. Scope/Spectrum/Sweep organization, inspector live controls, document actions.
5. Regression tests, internal review and same-state rendered comparison at
   1440×900 and the supported compact size; keep evidence and limitations explicit.

No merge, push, publication or replacement of qualified release packages.

## Implemented scope and deliberate differences

The native shell, menus/popovers, inspector hierarchy, tool/zoom pills, real
probe pills, typography, light/dark tokens and instrument tabs are implemented.
The existing resize/collapse commands, local Save/Save As, recovery, advanced
analysis and component editors remain reachable. No simulator or document-model
change was made beyond session layout defaults.

The draft includes capabilities that ELab does not currently expose. This UI
experiment does not add cloud storage, URL sharing, document notes/tags, selectable
solver/quality modes, automatic retention of four runs, timed-run presets,
invented warning toggles, wire-to-floating-net repair, sample-settled transient
frequency sweeps, log/relative-dB live FFT axes, or a readout-to-pin migration.
Existing terminal-to-terminal wiring, independent AC/DC sweeps, spectrum units,
source-model annotations, pin definitions and local documents remain exact.

Accordingly, visual QA compares the native composition and applicable controls;
it does not claim all fourteen draft acceptance criteria or a four-pixel clone.
Actual simulation values, timestamps, plot scaling and net geometry remain real
rather than matching fabricated artboard values. Existing examples are unchanged.
Native AppKit menus and Vellum popup scenes differ from browser-styled menus.
Keyboard focus outlines require slightly larger controls. Minimum window sizing
continues to follow the existing native constraint; a 1024px browser viewport is
not newly promised. All these differences are explicit, rather than inactive
controls that imply unavailable functionality.

See `design-qa.md` and `docs/validation/experimental-native-ui/README.md` for the
comparison artifacts, regression evidence and remaining platform acceptance limits.
