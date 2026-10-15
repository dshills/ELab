# Native dropdown repair — October 14, 2026

The product owner still found menus strange and boxy after the initial menu repair. Inspection identified two implementations: AppKit header menus and custom Vellum Inspector selector popups. Header menu installation also delayed presentation. All macOS action menus and selectors now present AppKit `NSMenu` choices directly, preserving Vellum layout, controls and original selection callbacks. Vellum’s service installs the application menu bar once the native window is available.

AppKit supplies rounded material surfaces, row spacing, hover/keyboard selection, menu scrolling and checkmarks. Current object/probe/theme choices are checked. The trigger remains a Vellum button with its semantic name and current choice in accessible help; popup rows use AppKit accessibility. Other platforms retain their Vellum selectors and menus.

A menu captures the command revision used to construct its descriptor. Opening or applying an outdated menu is refused, including when an agent changes state during menu tracking. The captured revision also reaches atomic shared-command admission, closing the gap between predicate and dispatch. Live sample revisions do not invalidate it. Disabled and unlisted choices are refused. Existing callbacks retain shared commands, unsaved-change prompts, actor-aware history guards and unit/value parsing. Cancellation releases the bounded native callback admission. Presentation runs asynchronously on the native main thread; selections return through the Vellum owner.

## Capture ledger

These are unedited native screenshots at 1184 × 854 pixels. The early candidates are retained as evidence, not relabeled as final captures.

| Capture | Observed behavior |
|---|---|
| [01 Examples candidate](01-examples-menu.png) | Actual AppKit popup, rounded material and compact rows |
| [02 Object candidate](02-object-menu.png) | Native object/wire choices, current-choice checkmark; RC was selected through the Examples menu |
| [03 Theme candidate](03-theme-menu.png) | Light checked; resistor selected through the object popup |
| [04 Final Theme](04-final-theme-menu.png) | Checked current choice; complete compact header and recorded scope |
| [05 Final object](05-final-object-menu.png) | Native object/wire choices over the paused recorded RC |
| [06 Final Examples](06-final-examples-menu.png) | Five actual native example choices; Escape dismissed without changing the document |
| [07 Final File](07-final-file-menu.png) | Native New/Open/Save/Save As actions; Escape dismissed without opening a file dialog |
| [08 Exact packaged dark Theme](08-qualified-dark-theme-menu.png) | Final Vellum checked-action metadata produces the native Dark checkmark; recorded cursor/time retained |
| [09 Exact packaged live scope](09-qualified-live-scope.png) | Entire live control row, compact trace row, cursors, meters and footer visible |
| [10 Exact packaged object menu](10-qualified-object-menu.png) | Native rounded object choices over the full RC in the collapsed dark layout; R1 selection verified |

Keyboard type-ahead selected R1, and Return committed it; an additional observation/Return was needed during automation. This proves the observed scenario, not exhaustive keyboard or assistive-technology acceptance. The [scope layout ledger](../scope-collapse/README.md) records compact expanded/collapsed/restored views. The [validation receipt](../../../docs/validation/native-menu-validation.json) records tests and package evidence.

The header now uses tighter horizontal action padding and short actor captions. Theme stays visible after agent edits. Small-zoom schematic labels/cards, long-menu navigation, all controls with screen readers/IME, and non-macOS appearance remain separate qualification work. Native menus follow the OS appearance independently of the circuit palette.

## October 15 final menu spot check

An isolated 0.54.0 / 69 app (`org.dshills.elab.menufinalaudit`) was inspected through native CUA. Fresh inline screenshots at 1184 × 854 showed:

1. File: rounded AppKit menu, all five labels fit; Escape dismissed without opening a dialog.
2. Examples: all seven examples fit, including Rectifier / smoothing; Escape dismissed.
3. Theme: Light/Dark/System fit, Light carries a checkmark; Escape dismissed.

Visible header controls fit without clipping. Native menu material follows OS appearance independently of the circuit theme. These are current visual observations, not a complete keyboard/screen-reader/resize audit. The screenshot API displayed the captures inline but provided no documented local-save operation, so no new screenshot files or full screenshot audit are claimed. The earlier saved capture ledger above remains historical evidence. The owned session was closed through `app.close`; existing user windows were preserved. Electronics-focused implementation resumed after this check. Schematic output/printing requirements are in the PRD's owner addition; printer discovery/submission and actual device tests remain pending.
