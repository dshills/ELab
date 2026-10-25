# Vellum widget adoption review

October 25, 2026. Recommendation review; no application or dependency changes.

ELab should adopt selected new controls, starting with search, field feedback and
compact numeric stepping. Keep the schematic canvas, symbol list, existing
commands and document workflow. A wholesale replacement of the inspector,
toolbar or component browser would bring more migration risk than immediate
benefit.

## Reviewed versions and evidence

ELab currently pins upstream `f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40`
(`v0.1.0-personal.1`), with compatibility source
`218c0dec8cfb52de4d63b91e0701f340396b7d84`. Compared with the preceding
`e58d405` SDK, the current SDK contains 38 additional non-test Go implementation
files under `ui/`, as well as changes to existing primitives and native services.
That count describes source files, not 38 independent widgets.

GitHub's personal macOS branch was checked during this review. Its head is
[`4986897`](https://github.com/dshills/vellum/commit/4986897237b782b5f35357f84b79d267edab939e),
three commits beyond ELab's pin:

- `7d37a11`: portable Windows permission assertion in a gallery test.
- `f1b6b5e`: durable Notes example with guarded files and recovery.
- `4986897`: omit unused palette/motion fields to preserve serialized theme and
  replay identities.

These add no widgets. The last change is relevant to a later dependency refresh,
but does not prevent using the controls already in ELab's SDK. This review leaves
the qualified pin and package intact. The complete
[upstream comparison](https://github.com/dshills/vellum/compare/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40...4986897237b782b5f35357f84b79d267edab939e)
identifies the exact newer changes.

This is a source/API and integration review. It uses the existing
[ELab update qualification](../docs/validation/vellum-f9dee54-update/README.md),
including the 557 passing mounted tests/subtests and separately scoped native
check. No new gallery launch, physical interaction study, benchmark or widget
integration is claimed here. Upstream specifications describe capabilities;
recommendations below are engineering judgments about ELab.

## Recommended first five migrations, in order

| Order | Adopt | ELab use | Benefit and migration condition |
| --- | --- | --- | --- |
| 1 | `ui.SearchBox` | Component search | A recognizable search symbol, hint and explicit Clear action. Keep global component search and the current Return-to-results workflow. |
| 2 | `ui.FormField` and `ui.InlineNotice` | Inspector labels, units, help and errors; operation feedback | Replace repeated label/error presentation while preserving each editor's ID, callback, draft and electrical validation. Start with one small inspector section. |
| 3 | `NumberButtons: ui.NumberButtonsStepper` | Numeric inspector fields | Compact right-hand up/down arrows give more room to the number and unit than the current inline minus/plus controls. Keep decimal steps, ranges and wheel scrolling. |
| 4 | `ui.Palette`, scoped `Node.Theme`, and `cogent.Driver.OnSystemAppearance` | Existing Light, Dark, System and Apollo modes | Put popup, field and focus colors on shared semantic tokens. Let System mode observe OS appearance/accessibility preferences through the new notification path. Preserve explicit theme choices. |
| 5 | `ui.WithToolbarOverflow` and toolbar space primitives | Carefully isolated action-only groups | Make secondary actions reachable through More when width is tight. Retain essential drawing and run controls outside overflow. Mixed toolbars need to be split first. |

### 1. SearchBox

The current [component palette](../desktop/component_palette.go) builds a label
and ordinary TextField. SearchBox supplies the missing search/clear affordances
without adding a search engine, timer or background service. Its edits use the
existing field commit semantics; it does **not** provide filtering on every
keystroke. Do not describe this migration as live autocomplete.

Use the existing `parts.search.set` command for both typing and Clear. Retain
`parts.category.set` behavior, including clearing search when a category changes.
Keep the current selection/placement commands and schematic symbols.

Its generated input ID is `ui.SearchInputID(root)`, rather than a caller-selected
leaf ID. Update keyboard-focus mappings and UI automation references explicitly;
do not silently break `part-search` targets. Retain the owner-held component,
and preserve ELab's revision guards around its callback. Match its three-child
layout to the narrow library rail before replacing the current field.

Source: [SearchBox implementation](https://github.com/dshills/vellum/blob/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40/ui/search_box.go).

### 2. FormField and InlineNotice

ELab currently assembles field captions and wrappers in
[inspector_style.go](../desktop/inspector_style.go), and has specialized correction
logic for timing/value errors. FormField keeps the supplied leaf ID and callbacks,
combines label/help/error metadata, and adds no extra focus stop. InlineNotice
provides a consistent message and optional application-owned action.

Use these for presentation only. Keep the existing command validators, rejected
draft handling, correction continuation, save/recovery logic and operation IDs.
Neither component performs electrical validation or guarantees a screen-reader
live announcement. FormField uses a FormGroup whose default borders/padding need
explicit ELab styling; otherwise every field could become an oversized boxed
panel. InlineNotice's row layout also needs wrapping/containment checks for long
unit or solver messages.

Sources: [FormField](https://github.com/dshills/vellum/blob/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40/ui/form_field.go),
[InlineNotice](https://github.com/dshills/vellum/blob/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40/ui/inline_notice.go).

### 3. Compact numeric stepper

This is the smallest migration: it is a NumberField presentation option, with
the same accepted value, numeric bounds and Change callback. Try PWM frequency
and duty first, then capacitor startup voltage and sensor controls.

Preserve `ControlStateKey`, decimal stepping, minimum/maximum admission and exact
typed values. Scroll wheels must continue scrolling their containing panel,
rather than changing the number. An engineering-unit text entry such as `4.7k`
must stay a unit-aware text entry; the numeric stepper is not its replacement.

Sources: [NumberField option](https://github.com/dshills/vellum/blob/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40/ui/controls.go),
[retention/stepper regression](https://github.com/dshills/vellum/blob/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40/drivers/cogent/number_field_offscreen_test.go).

### 4. Semantic themes and system preferences

The new framework has app-local semantic colors for window, panel, field, popup,
ink, line and accent. It also publishes macOS dark appearance, accent, increased
contrast, reduced motion and reduced transparency. ELab currently resolves its
own palette using `matcolor.SchemeIsDark` and does not opt into this new callback.

Bridge preferences into ELab's existing appearance state on the application
owner. Enabling `FollowSystemAppearance` alone would not recolor ELab's many
explicit node styles. Map those styles and the driver/theme palette together.
Use `appearance.theme.set` for explicit choices; platform preference observation
is environmental input, not an alternate document editor. Contrast/motion policy
should remain meaningful in explicit themes, while Apollo retains its selected
colors and instrument typography.

This can eventually reduce some private theme adapters, but the public palette
does not expose every scrollbar/private-widget detail. Keep required adapters
until equivalent visible behavior is proven. Preserve the focus-border allowance
introduced in /254; avoid a second spacing regression.

Sources: [semantic palette](https://github.com/dshills/vellum/blob/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40/ui/palette.go),
[system appearance implementation](https://github.com/dshills/vellum/blob/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40/drivers/cogent/system_appearance.go).

### 5. Action overflow

The helper fits a visible prefix and exposes remaining actions in a More menu.
Its contract is horizontal toolbars containing 1–64 plain Buttons, Separators or
toolbar spaces. It refuses editors, nested groups, MenuButtons, placement sources
and per-item context menus.

ELab's current header, drawing tools and scope controls contain combinations of
menus, labels, nested groups and editors. Applying overflow to those containers
unchanged would fail validation. First isolate a short secondary action bar.
Keep Select/Wire/Pan, Cancel and Pause/Resume directly accessible. Do not add
customization/persistence as part of this small layout change.

ELab's AppKit menu conversion must explicitly accommodate any new overflow menu;
its current native-choice transport is flat. The framework's retained controls
and hidden-item accessibility behavior need an ELab integration test after that
conversion, not just a passing framework fixture.

Source: [overflow contract](https://github.com/dshills/vellum/blob/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40/ui/toolbar_overflow.go).

## Useful next, after the first five

| Capability | Decision for ELab |
| --- | --- |
| `CollectionView` | Strong candidate for a separate examples chooser with schematic thumbnails, names and short descriptions. Keep the dense component symbol list in the narrow rail. Examples selection must still run the guarded example-load command and unsaved-change flow. |
| Cascading menus | Group the long Examples list by circuit family. Useful alongside a future chooser. Preserve actual AppKit menus: ELab's current native adapter serializes flat items and does not transport Vellum branches, icons or shortcut metadata. |
| `ui.Inspector` | Pilot only in a wider panel. Default 112-point right-aligned labels plus editor and gaps crowd the current rail; it has no automatic stacked-label fallback. Its existing editor IDs and retained disclosures are valuable. It is not a drop-in replacement for the present polished vertical form. |
| `ComboBox` | Potentially combine existing value presets with exact engineering-unit entry. The literal unit parser and command must remain authoritative. Maximum 64 suggestions; no live filtering/completion. Its generated field ID and missing direct logical-owner-key option require focused draft/selection migration work. |
| `Sheet`, `Popover`, `Alerts()` | Use sheets for bounded configuration tasks, popovers for short contextual details, native alerts for appropriate confirmations. Keep critical electrical errors inline and file pickers on the existing native path. Managed Sheet/Popover are not AppKit panels. Preserve cancellation/default-action guards and headless command results. |
| `RangeSlider` | Useful as an optional overview selector for a fixed scope interval, alongside exact text fields. It uses float32 values and allows equal endpoints; map relative time and enforce ELab's strict interval rules. Do not replace precise sample-snapped cursor entry. |
| `LevelIndicator` | A passive battery charge/depletion overview could complement numeric live readings. Its warning thresholds trigger at high values: do not accidentally mark a full battery as low charge. Use no thresholds or an explicitly labeled depletion mapping. Never invent component ratings. |
| `SettingsPane` | Suitable once existing display/appearance/simulation preferences merit their own settings surface. Do not move frequent experiment controls away from the circuit. Retain inactive page drafts and expose the same settings commands. |
| `Sidebar` / `OutlineView` | Useful when ELab has hierarchical projects or enough example categories for navigation. Current single-circuit documents do not need a project tree. Keep Common/category/search simple. |
| Native `TitlebarToolbar` | Visually promising, but defer production migration. It accepts an immutable catalog for one standard-chrome window opening, with 1–16 items and initial disabled flags. ELab needs dynamic Undo/Redo and Pause/Resume state; no public retained item-update API was found. Callback guards alone would not provide accurate visible enablement. |

The native titlebar implementation is genuinely AppKit-owned; most other new
widgets compose existing managed controls. These two paths should not be described
as equivalent native wrappers. See the
[titlebar API and restrictions](https://github.com/dshills/vellum/blob/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40/drivers/cogent/titlebar_toolbar.go).

## Defer for lack of a current workflow

| Group | Widgets/capabilities and reason |
| --- | --- |
| File-oriented navigation | DocumentList, ColumnBrowser, PathControl/Breadcrumb and Quick Look are useful for a project/recent-files browser, but not prerequisites for editing one circuit. Reuse existing Save/Open and recovery. |
| Rich decoration | ColorWell/ColorPicker, native Colors/Fonts panels, FontPicker, GradientWell, ImageWell, Image and IconPicker can serve future annotations or appearance customization. Adding those editors now would expand product scope. |
| Other form/navigation controls | TokenField, SecureField, DateTimePicker/Calendar, RuleEditor and FindBar have no immediate circuit workflow. FindBar is text find/replace, not a schematic object-query engine. |
| Extra interaction | AngleDial is continuous wrapped rotation; ELab currently uses discrete schematic rotations. ShortcutRecorder needs a shortcut-policy feature first. Multi-selection lists are useful only when a real multi-selection command exists. |
| Alternative button/navigation composition | SplitButton, SegmentedControl, headerless Tabs and ToolbarCustomizer are available, but do not replace the recently repaired tabs/drawing modes just for visual novelty. Assess keyboard semantics and retained scope before migration. Customization would add preference commands and persistence. |

The [upstream control inventory](https://github.com/dshills/vellum/blob/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40/docs/controls.md)
and [composition contracts](https://github.com/dshills/vellum/blob/f9dee54fdb2a9e73a38fdd3fb67c8871dc0f6c40/specs/mac-composition-widgets.md)
describe these options. Existing canvas, split panes, disclosures, lists, basic
buttons, sliders and numerical controls remain appropriate for ELab.

## Shared user/agent contract and acceptance

Widgets propose intents; ELab's service accepts them. Keep this rule for every
migration. A component callback must invoke the same command an agent would use,
with the same revision/document checks, validation, history and persistence.
Never put electrical defaults, corrections or direct schematic mutations inside
a widget-only callback. Existing native menu guards must survive migration.

Retain owner-held composites on the application owner. Fresh descriptors are
expected, fresh logical owners on every render are not. Generated widget IDs
must be mapped deliberately in keyboard focus and UI automation. Changing native
identity while a rejected draft is active needs an explicit migration decision.

For each small migration, require:

1. Equivalent UI and agent accepted/rejected results, including Undo and saved
   circuit values. Presentation-only actions need not create document edits.
2. Exact typed units, decimal numeric steps, rejected drafts and focus preserved
   through agent edits, selection changes and new documents; stale callbacks refused.
3. Mouse/wheel/thumb/keyboard checks at minimum and ordinary window sizes, with
   complete borders, labels, tabs and popups in Light, Dark, System and Apollo.
4. System-appearance changes preserve the explicit theme policy and user values;
   do not change OS settings to implement or qualify the feature.
5. Relevant lint/race/rendered checks and a bounded owned native check. Reuse
   previous passing electrical evidence unless an electrical path changes.

Start with one SearchBox migration. Follow with one field/notice pilot and a
small numeric-stepper pilot before converting broad inspector families.
