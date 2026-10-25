# Vellum widget migrations

October 25, 2026. Implements the first five recommendations from
[the adoption review](113-vellum-widget-adoption-review.md).

- Component search now uses Vellum SearchBox with a search symbol, hint and Clear.
  `part-search` remains the field's stable keyboard/automation ID;
  `part-search-clear` is the new UI affordance. Both use `parts.search.set`.
  Return still commits the complete query and moves to matching results; Clear
  returns focus to the entry. Common/categories and schematic symbols remain.
- Inspector fields use FormField metadata with existing visible labels and editor
  IDs. Group surfaces are flat, with the previous four-point label gap. Actual
  rejected component-value guidance is associated with the value field. Global
  feedback uses InlineNotice. Electrical validation, corrected drafts and their
  continuation/Undo paths remain in ELab's shared service and existing handlers.
- Inspector NumberFields use Vellum's compact up/down stepper. Bounds, decimal
  steps, exact edits and logical-owner keys remain. Engineering-unit text entries
  stay text entries. Wheel input scrolls its containing panel.
- A scoped semantic Palette supplies popup, field, line, focus and accent colors.
  The macOS preference observer updates System appearance and accent. Explicit
  Light/Dark/Apollo remain selected; increased contrast and reduced motion apply
  without editing documents or saved settings. Moving current dots become static
  under system Reduce Motion, and custom canvas paints invalidate on appearance
  changes even when no simulation/document revision changes. The palettes use
  opaque surfaces; no OS appearance settings are changed.
- Sweeps and Fit are in a bounded secondary action toolbar with Vellum overflow.
  More appears only when the actions do not fit. Drawing modes, Cancel and
  Pause/Resume remain directly accessible. An open More popup retires on any
  intervening command revision, preventing it from borrowing an agent edit's
  refreshed callbacks. The More menu stays framework-managed
  so its hidden-item routing and accessibility filter remain correct; existing
  File/Examples/choice menus remain AppKit menus. No toolbar customization or
  preference persistence was added.

All changes retain the existing application command set and component models.
UI proposals continue to use revision/document admission, history and validation;
old SearchBox Clear and overflow callbacks refuse intervening agent changes.
Native menu guards remain in place. The upstream pin remains f9dee54. Compatibility SDK elab3 extends the existing
decimal stepping fix to compact controls: displayed decimals round once to the
native float32, compact bounds clamp, inline bounds wrap and typed off-grid
values keep their precision. The preceding SDK/archive and /254 package remain
preserved. Electrical equations and work limits are unchanged.

Complete lint, tooling, root/desktop/rendered race and vet commands and results,
packaged example replay and bounded native observations are recorded in
[validation](../docs/validation/vellum-widget-adoption/README.md).

No thumbnail examples chooser, native titlebar migration, new electronics model,
new solver work allowance or project-format change is included in this migration.

The /255 package is available at
[ELab.app](../build/ELab-0.194.0-255-Widgets/ELab.app), with its matching agent.
The bounded native check confirmed search/Clear focus, compact pointer stepping,
wheel value preservation, refused-value guidance and corrected unit entry.
A lingering PWM bounds tooltip after changing inspectors remains a separately
recorded native issue; see the validation notes. System changes and constrained
More-menu routing are covered by synthetic/routed tests rather than OS settings
changes or native window resizing.
