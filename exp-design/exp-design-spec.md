
# SPEC — ELab "Native" interface

| Field | Value |
|---|---|
| Status | Draft 1 |
| Date | 2026-10-30 |
| Scope | Visual and interaction redesign of the ELab · Live electronics editor to the "Native" direction |
| Design reference | Canvas "Schematic App Designs", rows "Applied to ELab" and "ELab · Native — the remaining screens" (https://claude.ai/artifact/Uh5UqKcUU8yoQ1m8nZGdV2) |
| Normative words | MUST and MUST NOT mark requirements. Anything else is descriptive. Optional behaviour is written out as "optional" with its consequence. |

## 0. Context and assumptions

0.1 ELab is a browser-delivered web application (the current build runs in a browser tab). This spec describes a theme and layout refactor of that application. It does not require packaging as a macOS binary.

0.2 Every control present in the current ELab interface (listed in §15) MUST remain reachable after the redesign. This spec changes where and how controls appear; it does not remove capabilities.

0.3 Simulation, netlist, measurement and sweep semantics are out of scope. Where this spec names a value (for example "sample interval"), the existing engine's definition of that value applies.

0.4 The design reference shows window chrome (traffic lights) because the artboards depict a Mac window. §5.2 states when that chrome is and is not rendered.

## 1. Goals and non-goals

1.1 Goals
- G1. The editor reads as a current macOS document application: full-height sidebar, unified toolbar with a document title, inspector on the right, tool controls floating over the canvas, one accent colour.
- G2. The schematic is the most visually prominent content on screen; chrome is neutral grey and white.
- G3. Live readings are visible where the thing they measure is: probe pills on the sheet, pinned readings in the inspector.
- G4. Every state the current app has (live, paused, sweeping, wire-in-progress, warning) has a defined appearance.

1.2 Non-goals
- N1. No change to simulation algorithms, file format, or example content.
- N2. No new measurement types beyond those listed in §15.
- N3. No touch or phone layout. Minimum supported viewport is defined in §5.4.
- N4. No custom window chrome in the browser build.

## 2. Definitions

| Term | Meaning |
|---|---|
| Sheet | The schematic drawing surface (currently labelled "canvas" in the UI). |
| Dock | The bottom panel holding Scope, Spectrum and Sweep. |
| Inspector | The right panel with Inspect, Simulate and Document tabs. |
| Parts sidebar | The left panel listing components and holding Connect by name. |
| Tool pill | The floating control at the bottom centre of the sheet holding Select, Wire and Pan. |
| Probe pill | A rounded badge drawn on the sheet showing a live value for one node or part. |
| Pinned reading | A live value listed in the inspector that persists until removed. |
| Primary action | The single filled-accent button in the toolbar. Its label depends on run state (§6.3). |
| Run state | One of: live, paused, sweeping. |
| Unit | One sheet grid unit, as the engine defines it today (the status line currently reports "10-unit snap"). |

2.1 Abbreviations and identifiers used in this document

| Short form | Expansion |
|---|---|
| G1…G4, N1…N4 | Goal and non-goal identifiers in §1 |
| AC1…AC14 | Acceptance criterion identifiers in §16 |
| OD1…OD4 | Open decision identifiers in §17 |
| CSS | Cascading Style Sheets |
| WCAG AA | Web Content Accessibility Guidelines, conformance level AA |
| ARIA | Accessible Rich Internet Applications attributes |
| SI | Système international unit prefixes (p, n, µ, m, k, M, G) |
| PDF, PNG, CSV | Portable Document Format, Portable Network Graphics, comma-separated values |
| SVG, DOM | Scalable Vector Graphics, Document Object Model |
| OS | Operating system |
| RLC | Resistor–inductor–capacitor network (the current engine's name) |
| VIN, U1, SP1, VP, VN | Reference designators in the "Amplifier driving a loudspeaker" example: input source, op-amp, loudspeaker, positive supply, negative supply |
| JU, JF, JB, VCC, VEE, GND | Net names in that example |
| ⌘, ⌥, ⇧, ⌫ | Command, Option, Shift and Delete keys |
| L* | Lightness in the CIELAB colour space |

## 3. Colour tokens

3.0 Implementation constraint (intentional, not a behavioural requirement). Colours MUST be defined once as CSS custom properties on `:root` and referenced everywhere by token name; components MUST NOT hard-code a colour literal. This constraint exists so that dark mode (§3.3) and future theming are a token swap. It is verified by inspection in AC2, separately from the behavioural criteria.

3.1 The tokens below are the complete palette.

3.2 Light appearance (default)

| Token | Value | Use |
|---|---|---|
| `--bg-window` | `#FFFFFF` | Sheet background |
| `--bg-sidebar` | `#ECECEF` | Parts sidebar |
| `--bg-panel` | `#F6F6F8` | Toolbar, inspector, dock |
| `--bg-control` | `#FFFFFF` | Text fields, push buttons, popups |
| `--bg-segment` | `#E3E3E7` | Segmented control track |
| `--bg-floating` | `rgba(255,255,255,0.94)` | Tool pill, zoom pill, notices |
| `--separator` | `rgba(0,0,0,0.09)` | Panel borders |
| `--hairline` | `rgba(0,0,0,0.12)` | Control borders, dividers |
| `--text` | `#1D1D1F` | Primary text, schematic ink |
| `--text-secondary` | `#5C5C62` | Field labels, secondary values |
| `--text-tertiary` | `#68686E` | Hints, section headers |
| `--text-disabled` | `#B0B0B5` | Disabled controls |
| `--accent` | `#0A84FF` | Selection, primary action, focused tab, links |
| `--accent-tint` | `rgba(10,132,255,0.12)` | Pressed toolbar toggles, tag chips |
| `--grid-dot` | `#D6D6DB` | Sheet grid dots |
| `--destructive` | `#D70015` | Delete, dangling-end marker, error callouts |
| `--warning` | `#FF9F0A` | Notices, paused indicator |
| `--ok` | `#34C759` | Live indicator |
| `--trace-vin` | `#D98B00` | Scope trace for VIN (was "amber") |
| `--trace-voltage` | `#1E8A7A` | Scope trace for SP1 voltage (was "teal") |
| `--trace-current` | `#E0533A` | Scope trace for SP1 current (was "coral") |
| `--net-a` | `#0A84FF` | Net label colour 1 |
| `--net-b` | `#9B3FD9` | Net label colour 2 |
| `--net-c` | `#D97706` | Net label colour 3 |
| `--net-d` | `#8E8E93` | Ground / reference nets |

3.3 Dark appearance. Dark mode is part of the first release of this redesign. The application MUST honour `prefers-color-scheme: dark` with these overrides. Tokens not listed keep their light value.

| Token | Dark value |
|---|---|
| `--bg-window` | `#1E1E20` |
| `--bg-sidebar` | `#26262A` |
| `--bg-panel` | `#2A2A2E` |
| `--bg-control` | `#3A3A3F` |
| `--bg-segment` | `#3A3A3F` |
| `--bg-floating` | `rgba(44,44,48,0.94)` |
| `--separator` | `rgba(255,255,255,0.10)` |
| `--hairline` | `rgba(255,255,255,0.14)` |
| `--text` | `#F2F2F5` |
| `--text-secondary` | `#A1A1A6` |
| `--text-tertiary` | `#9A9AA0` |
| `--text-disabled` | `#5C5C64` |
| `--grid-dot` | `#3A3A3F` |
| `--trace-vin` | `#FFB340` |
| `--trace-voltage` | `#3FC1B0` |
| `--trace-current` | `#FF7A6B` |

3.3a Translucent fills and shadows. These are tokens as well; later sections refer to them by name. They are identical in light and dark appearance unless listed below.

| Token | Value | Use |
|---|---|---|
| `--fill-quiet` | `rgba(0,0,0,0.055)` | Search field fill |
| `--fill-cancel` | `rgba(0,0,0,0.06)` | Cancel button in the tool pill |
| `--fill-hover` | `rgba(0,0,0,0.085)` | Focused or hovered list row, pressed Examples button |
| `--accent-wash` | `rgba(10,132,255,0.07)` | Selection fill, cursor span tint |
| `--accent-halo` | `rgba(10,132,255,0.14)` | Snap halo fill |
| `--warning-wash` | `rgba(255,159,10,0.12)` | Simulate callout fill |
| `--destructive-wash` | `rgba(215,0,21,0.08)` | Net inspector dangling-end callout |
| `--note-fill` | `rgba(255,249,196,0.9)` | Sheet note |
| `--bg-menu` | `rgba(246,246,248,0.98)` | Menus and popovers |
| `--shadow-floating` | `0 10px 28px rgba(0,0,0,0.14)` | Tool pill |
| `--shadow-small` | `0 6px 18px rgba(0,0,0,0.10)` | Zoom pill, notices |
| `--shadow-popover` | `0 14px 40px rgba(0,0,0,0.20)` | Popovers |
| `--shadow-menu` | `0 14px 40px rgba(0,0,0,0.22)` | Menus |
| `--shadow-control` | `0 1px 2px rgba(0,0,0,0.14)` | Selected segment |

Dark values: `--fill-quiet` `rgba(255,255,255,0.07)`, `--fill-cancel` `rgba(255,255,255,0.08)`, `--fill-hover` `rgba(255,255,255,0.10)`, `--note-fill` `rgba(120,110,40,0.5)`, `--bg-menu` `rgba(44,44,48,0.98)`.

3.4 Contrast. Every text token except `--text-disabled` MUST meet a 4.5:1 contrast ratio (WCAG AA) against every background token it is used on: `--bg-window`, `--bg-sidebar`, `--bg-panel`, `--bg-control` and `--bg-floating`. The values in 3.2 and 3.3 satisfy this (lowest pair: `--text-tertiary` on `--bg-sidebar`, 4.7:1 light, 5.1:1 dark). If a token value is changed, the contrast requirement wins and the value MUST be adjusted until it passes. Disabled text is exempt, as WCAG exempts it. The design reference artboards used a lighter grey for hints; this table supersedes them.

3.5 Trace colours MUST differ from one another in lightness as well as hue so they remain distinguishable in greyscale. The three values in 3.2 satisfy this; replacements MUST keep a lightness gap of at least 15 L* between any two traces.

## 4. Typography and iconography

4.1 Font stack: `-apple-system, BlinkMacSystemFont, "SF Pro Text", "Helvetica Neue", Helvetica, Arial, sans-serif`. Numeric readings, coordinates, net names and cursor values use `ui-monospace, "SF Mono", Menlo, monospace`.

4.2 Sizes

| Role | Size / weight |
|---|---|
| Document title (toolbar) | 13 px / 600 |
| Toolbar subtitle, hints, section headers | 11 px / 400 (headers 600) |
| Body, field values, list rows | 13 px / 400 |
| Inspector object name | 17 px / 600 |
| Schematic reference designators | 12 px / 600 |
| Schematic values and net names | 11 px / 400 (net names 600) |
| Probe pill value | 11 px / 600, monospace |
| Axis labels | 10 px, monospace |

4.3 Icons are stroke icons at 18 × 18 px, 1.5–1.6 px stroke, round caps and joins, drawn in `currentColor`. Emoji MUST NOT be used as icons. Every icon-only button MUST carry an accessible name.

## Requirements

Sections 5 through 14 are the functional and behavioural requirements of this specification. Each numbered statement containing MUST or MUST NOT is a requirement; §16 gives the acceptance tests.

## 5. Window layout

5.1 Regions, left to right and top to bottom:

```
┌──────────┬──────────────────────────────────────────┬───────────┐
│ Parts    │ Toolbar (52)                             │           │
│ sidebar  ├──────────────────────────────────────────┤ Inspector │
│ (240)    │ Sheet (fills)                            │ (280)     │
│          ├──────────────────────────────────────────┤           │
│          │ Dock (250, collapsible)                  │           │
└──────────┴──────────────────────────────────────────┴───────────┘
```

Widths and heights are CSS pixels. The sidebar spans the full window height; the toolbar spans sheet + dock width; the inspector spans toolbar + sheet + dock height.

5.2 Window chrome
- In the browser build, no traffic lights or title bar are drawn. The sidebar's top 52 px row contains only the "Hide sidebar" button, right-aligned.
- If ELab is later wrapped as a desktop app with a hidden-inset title bar, the top-left 78 × 52 px of the sidebar MUST be reserved as a drag region with no controls. This is the only chrome concession.

5.3 Panel visibility
- Sidebar, inspector and dock are each independently hideable. Toggles: sidebar button at the sidebar's top row and at the toolbar's left edge when hidden; inspector button at the toolbar's right edge; dock via "Hide scope" in the dock header and "Show scope" in the toolbar when hidden.
- When the sidebar is hidden the toolbar extends to the window's left edge and the sidebar toggle appears as its first control.
- Visibility of all three panels MUST persist per browser (local storage) and restore on load. A stored preference takes precedence over the 1100–1280 px rule in §5.4; that rule applies only when no preference is stored. The 1024–1099 px rule in §5.4 is the one exception: there the sidebar and inspector start hidden regardless of preference, and the stored preference is left unchanged for the next load at a wider size. If local storage cannot be read or written, the defaults apply (all three shown, subject to §5.4) and no error is shown to the user.

5.4 Minimum viewport
- Preferred viewport: width ≥ 1100 px, height ≥ 700 px. Supported minimum: 1024 × 700 px (last bullet).
- Between 1100 and 1280 px wide, the sidebar auto-hides on load when no visibility preference is stored (§5.3). The user is still able to show it; showing it MUST NOT push the inspector below 280 px.
- The lowest supported width is 1024 px. From 1024 px to 1099 px the sidebar and inspector are hidden on load regardless of stored preference, and the following MUST remain available and fully visible: every toolbar control of §6.1 (the title truncates first), the tool pill, the zoom pill, the sheet, the dock with all header controls, the Examples menu and the Sweep popover; the sidebar and inspector MUST still open from their toggles, overlaying the sheet rather than shrinking it below 480 px. Horizontal page scroll MUST NOT occur at any supported width. Behaviour below 1024 px is unspecified.

5.5 Resizing
- The dock height is user-resizable between 180 and 420 px by dragging its top edge. The sidebar and inspector widths are fixed.

## 6. Toolbar

6.1 Height 52 px, background `--bg-panel`, 1 px `--separator` below. Contents, left to right:
1. Undo, Redo (icon buttons, 30 × 28). Redo uses `--text-disabled` when the redo stack is empty.
2. Examples (text button with chevron; opens the menu in §11.1).
3. Title block, centred in the remaining space: document title (one line, truncated with an ellipsis at 420 px) over a status subtitle.
4. Sweeps (text button; opens the popover in §11.2). Shows the pressed style (`--accent` text on `--accent-tint`) while a sweep is running or the popover is open.
5. Fit (text toggle; pressed style when the sheet is fitted to its contents).
6. Primary action (§6.3).
7. Share (icon), Inspector toggle (icon; pressed style while the inspector is shown). Share copies the document's URL to the clipboard and shows the notice "Link copied" (§8.10) for 3 seconds. If the clipboard write is refused, the notice instead shows the URL in a selectable text field with the sentence "Copy this link".

6.2 Status subtitle. 11 px, `--text-tertiary`, preceded by a 6 px dot:

| Run state | Dot colour | Text |
|---|---|---|
| live | `--ok` | `Live · {engine} · t = {t} · run {n}` |
| paused | `--warning` | `Paused at t = {t} · {engine} · run {n}` |
| sweeping | `--accent` | `Sweeping {source} {quantity} · point {k} of {N} · {current value}` |

`{t}` is formatted per §12 with 3 decimals in seconds.

6.3 Primary action. Exactly one filled button (background `--accent`, white 600 text, 7 px radius, 28 px tall) is present at all times:

| Run state | Label | Icon | Effect |
|---|---|---|---|
| live | Pause | pause | Pauses the run; state → paused |
| paused | Resume | play | Continues the run; state → live |
| sweeping | Stop sweep | stop | Cancels the sweep, restores the swept value, state → live |

Within the toolbar the primary action is the only filled-accent control. (The active tool in the tool pill, §8.4, also uses the filled accent style; it is outside the toolbar.)

6.4 The File menu and Save button of the current UI move to: File → the Examples menu's lower section and the Document tab (§9.4); Save → autosave with status in the Document tab. A keyboard shortcut ⌘S MUST still trigger an immediate save.

6.5 Autosave contract
- A document change is any of: part added, moved, rotated, deleted or revalued; wire added, moved or deleted; net renamed or recoloured; a Document tab field edited; a sheet note added, edited or removed; a pinned reading added or removed.
- The first change after a save starts a 1 second debounce timer; further changes restart it. The document MUST be written no later than 5 seconds after the first unsaved change regardless of further edits, and immediately on ⌘S, on `visibilitychange` to hidden, and on `pagehide`.
- At most one write is in flight at a time; when it completes, if the document has changed since, a new write starts immediately with the latest revision. Stored content therefore always moves forward: a write for revision r MUST NOT replace stored content of a revision greater than r. Every save request carries the document revision number current when it started. Save completions apply in revision order: a completion for revision r updates the saved status only if r is at least the highest revision already reported, and "Saved" is shown only when the saved revision equals the current revision; otherwise the status remains "Saving…" or "Not saved" as appropriate. Document › Storage shows "Saved {relative time}" after a successful write of the current revision and "Saving…" while a write is in flight.
- If a write fails (storage unavailable, quota exceeded, or the storage target rejects the write): the Storage row shows "Not saved — {reason}" in `--destructive`; the toolbar title gains the suffix " — Edited"; a sheet notice (§8.10) reads "Changes are not being saved." with the actions "Retry" and "Download copy" (which downloads the document as a file); the unsaved document stays in memory; the application retries every 10 seconds and on the next change; and leaving the page prompts with the browser's unsaved-changes dialog. The failure state (suffix, row text, notice, retries) clears only when a write of the *current* revision succeeds; a late success for an older revision leaves it in place and retries continue.

## 7. Parts sidebar

7.1 Background `--bg-sidebar`, width 240, right border `--separator`. Content, top to bottom:
1. Top row (52 px): hide-sidebar button, right-aligned (and the drag region of §5.2 when applicable).
2. Search field: 26 px tall, 7 px radius, `--fill-quiet` fill, magnifier icon, placeholder "Name or alias". Typing filters the list live.
3. Scope row: a popup button labelled with the current set (default "Common") and a right-aligned match count ("6 matches").
4. Component list: one row per component (36 × 26 symbol thumbnail on `--bg-control` with a 1 px hairline, then the name). Row height 36, radius 7. The keyboard-focused row uses `--fill-hover`.
5. Hint text (11 px, tertiary): "Arrow keys choose · Tab places on the sheet · Return drops at the cursor".
6. Footer (pinned to bottom, 1 px separator above): "Connect by name…" push button; counts line "{components} components · {wires} wires" left, storage location right.

7.2 Keyboard behaviour is unchanged from the current app: ↑/↓ move the focused row, Tab enters placement mode, Return drops at the cursor position.

7.3 Component thumbnails MUST be the same symbol drawing used on the sheet, rendered at thumbnail scale, so the list and the sheet never disagree about a symbol.

## 8. Sheet

8.1 Background `--bg-window` with a dot grid: 1 px dots of `--grid-dot` every 20 px, offset 10 px. The Document tab (§9.4) switches the grid between dots, lines and off; the default is dots.

8.2 Schematic drawing
- Ink `--text`, 1.7 px strokes, round caps and joins.
- Junction dots radius 3.6 px.
- Reference designators 12 px/600 above-left of the symbol; values 11 px `--text-secondary` below or beside.
- Net labels 11 px/600 in the net's colour (§8.6).

8.3 Selection
- A selected part shows a rectangle inset 4 px outside its symbol bounds: 1.6 px `--accent` stroke, `--accent-wash` fill, and eight 8 × 8 white handles with 1.3 px `--accent` stroke at corners and edge midpoints.
- A selected wire renders at 2.2 px in `--accent`.
- The current red selection rectangle MUST NOT be used; red is reserved for errors (§8.7).

8.4 Tool pill
- Floating, bottom centre, 16 px above the sheet's bottom edge (above the dock when shown). Background `--bg-floating`, radius 13, 1 px hairline ring and `--shadow-floating`.
- Contains Select, Wire, Pan as 36 × 36 icon buttons (radius 9). The active tool is filled `--accent` with a white icon. Exactly one tool is active.
- A hairline divider follows, then a context hint (11 px tertiary). Hint text per tool:
  - Select: "R rotate · W wire · ⌫ remove"
  - Wire: "Click to place a corner · Return finishes · Esc cancels", followed by a Cancel button (26 px, `--fill-cancel` fill).
  - Pan: "Drag to pan · double-click to fit"
- While a sweep runs, editing is disabled and the hint reads "Editing is paused while a sweep runs".
- Shortcuts: V Select, W Wire, Space (held) Pan, Esc cancels an in-progress wire (discarding its placed segments, §8.8) and returns to Select.

8.5 Zoom pill
- Bottom right, 16 px inset. Contains −, the percentage (tabular numerals, 40 px min width), +. The + and − buttons move to the next value in the fixed sequence 25, 33, 50, 67, 75, 90, 100, 110, 125, 150, 175, 200, 250, 300, 400 %; from an intermediate zoom reached by scrolling, + goes to the next higher value in the sequence and − to the next lower. At 400 % + is disabled; at 25 % − is disabled. Scroll-wheel zoom continues to work on the sheet and is clamped to the same 25–400 % range.

8.6 Net colours. Each named net is assigned one of `--net-a`…`--net-c` on creation (round-robin); ground and reference nets use `--net-d`. A net's colour is shown on its label, in the sidebar's Connect by name list, and in the net inspector, where the user changes it (§9.3). The three colours differ in lightness as well as hue.

8.7 Dangling wire ends
- A wire end that touches no pin, junction or wire is drawn with a 10 × 10 white square with a 1.6 px `--destructive` stroke.
- When such a wire is selected, a 20 px tall rounded label beside the end reads "Dangling end · 0 A flows".
- Dangling ends also surface in the net inspector (§9.3).

8.8 Wire mode
- Clicking a pin or wire starts a wire; the committed segments render in `--accent` at 2.2 px; the segment following the cursor renders 2 px dashed (5/4).
- When the cursor is within 10 px of a pin, the pin shows a 9 px radius snap halo (`--accent-halo` fill, 1.6 px `--accent` stroke) and a 22 px tall accent tag above it reading "Snap · {part}.{pin} → {net}" (or "Snap · {part}.{pin}" when no net exists yet). Clicking commits to the snapped point.
- Each click adds a corner; segments between placed corners are *placed* but not yet document content. Return finishes the wire: every placed segment becomes document content and the dangling cursor segment is dropped. Esc cancels the wire: every placed segment and the cursor segment are discarded and the document is unchanged. Clicking on a pin, junction or existing wire finishes the wire at that point exactly as Return does.

8.9 Probe pills
- A probe pill is a 22 px tall capsule anchored 10 px above its node with a 1.6 px leader line, filled with the trace colour of its quantity (VIN → `--trace-vin`, voltage → `--trace-voltage`, current → `--trace-current`, other → `--accent`), white 11 px/600 monospace text, value formatted per §12.
- A pill is created by "Plot …" actions (it appears together with the dock trace) and by "Pin to sheet" (it appears together with the pinned reading). It updates at the dock's refresh rate. Hovering a pill reveals a 16 px × button at its right end; activating it (or focusing the pill and pressing ⌫) removes the pill together with what created it: the dock trace for a Plot pill, the pinned reading for a Pin pill. Removing the trace or the reading from the dock or inspector removes the pill the same way.
- Pills MUST NOT overlap one another. Pills are laid out in creation order. Each pill starts 10 px above its anchor; if its rendered bounds, expanded by 4 px, intersect any earlier pill's bounds, it moves up in 26 px steps until they do not. The leader line stretches to the final position. Pills are positioned in sheet coordinates and pan and zoom with the sheet (their text size stays 11 px). If moving up would place a pill's top above the sheet's top edge at the current view, that pill is placed 10 px below its anchor instead and then moves down in 26 px steps until clear. Removing a pill re-runs the layout for the pills created after it.

8.10 Notices
- A sheet notice is a 30 px tall floating capsule at the top centre of the sheet, 14 px from the top: warning icon in `--warning`, one sentence, and an optional action button (22 px, `--accent-tint` fill, `--accent` 600 text).
- At most one notice is visible at a time, chosen by the precedence rule below. The current two-line sample-step message becomes: "{source}'s {step} sample step hides waveform detail." with action "Use {recommended}". The notice is shown when the engine's `recommendedOutputInterval()` (§8.12) returns a value and the current output sample interval is greater than it. The action sets the output sample interval to `{recommended}` and the notice clears because its condition no longer holds.

8.12 Engine interface for the sample-step condition. The engine MUST expose `recommendedOutputInterval(): number | null`, returning seconds: the largest output sample interval that still samples the fastest periodic source at least 32 times per period, or `null` when there is no periodic source or the engine cannot compute it. `{source}` and `{f}` in notices come from the same call's companion `fastestSource(): {ref: string, frequency: number} | null`. When either call returns `null`, the notice of §8.10 and the callout of §9.5 are not shown and the Output field is styled normally.
- Notices are of two kinds. A *condition notice* (sample step §8.10, unsaved changes §6.5) is shown while its condition holds and clears only when the condition no longer holds. A *transient notice* (Share §6.1) is shown for 3 seconds and then clears on its own. A transient notice temporarily covers a condition notice; when the transient notice clears, the condition notice reappears if its condition still holds. Among condition notices, unsaved changes takes precedence over sample step. Among transient notices, one with an action (export failure) takes precedence over one without (Share): a Share notice displaced by an export failure is discarded; an export-failure notice is never displaced by Share and is shown after a Share notice that is already visible clears. Two notices with actions queue in order of occurrence.

8.11 Sheet notes (optional feature, see OD3). If shipped, a sheet note is a rectangular note (`--note-fill` fill, 8 px radius, 12 px/600 title, 11 px body) placed on the sheet from the Document tab. Notes are document content and MUST be saved with the file. If not shipped, the Document tab has no note control and nothing else in this spec changes.

## 9. Inspector

9.1 Width 280, background `--bg-panel`, left border `--separator`. A tab strip (Inspect · Simulate · Document), styled as a segmented control, sits in a 10 px inset at the top; for accessibility it is a `tablist` (§13.4), not a radio group. The inspector scrolls vertically if its content exceeds the height; the bottom action row (where specified) stays pinned.

9.2 Inspect — component (one part selected)
1. Header: 44 × 44 symbol thumbnail, object name (17 px/600), one-line description "{type} · {key values}" in tertiary.
2. Live row: "Live · {node}" left, value right (monospace). Below it a one-line plain-language status in tertiary (for example "Configured educational model").
3. Action grid, 2 × 2 push buttons: Plot voltage, Plot current, Rotate, Delete (Delete in `--destructive`).
4. Disclosure rows: Details, Choose another object (unchanged behaviour).
5. Measure section (§9.6).
6. Pinned readings (§9.7), pinned to the bottom.

9.3 Inspect — net or wire (a wire selected)
1. Header: wire glyph thumbnail, net name, "Net · {wires} wire(s) · {connections} connection(s)".
2. One callout per dangling end on the net, listed in the order of the connections list (item 4), each with `--destructive-wash` fill, an error icon, one sentence naming the end's sheet coordinate and the nearest unconnected pin within 60 units of that end (if any), and a button "Extend to {part}.{pin}" that adds the straight or single-corner segment from that end to that pin. If no pin is within 60 units the button reads "Delete open end" and removes the segment that owns that end.
3. Name field (renaming updates every label on that net), colour swatches (the four net colours plus ink), Label checkbox (show/hide the net label).
4. Connections list: one row per attached pin "{part} · {pin}" with its sheet coordinate right-aligned; open ends listed in `--destructive`.
5. Action grid: Plot voltage, Highlight net, Split at corner, Delete wire.

9.4 Document tab
- Title field; Notes multi-line field; Tags (chips in `--accent-tint`, "+ Add").
- Sheet: Grid segmented (Off · Dots · Lines), Snap field in units, "Show net names" checkbox, Units segmented (SI prefixes · Scientific; drives §12).
- Storage: Location, Saved (relative time, "autosave on"), Based on ("Example · modified" when a loaded example has been edited). Buttons: Move to … (storage target), Reset to example (confirmation required).
- Export, 2 × 2: Schematic PDF, Schematic PNG, Netlist, Scope CSV. Each export produces a file download named `{document title}.{ext}`. If generating the file or starting the download fails, a transient notice (§8.10) reads "Export failed — {reason}" with the action "Try again", shown for 5 seconds instead of 3; the document and any unsaved-changes recovery actions (§6.5) are unaffected. The "Download copy" action of §6.5 follows the same failure rule.

9.5 Simulate tab
1. Engine popup (current: "RLC network transient").
2. Quality segmented Auto · High · Low, with one line of explanation under it.
3. Time step field (solver), Output field (sample interval). When `recommendedOutputInterval()` (§8.12) returns a value and Output exceeds it, the Output field border turns `--warning` and a callout (`--warning-wash` fill) follows with the sentence "{source} at {f} needs {recommended} or less to keep its shape in the scope." and a button "Fit to fastest source". This callout and the sheet notice of §8.10 are the same condition; applying either clears both.
4. Run for: segmented Live · 0.5 s · 2 s · … (custom field).
5. Checkboxes: "Restart when a part or value changes", "Keep the last 4 runs for comparison", "Warn when a source clips a supply".
6. "This run" summary: run number and state, simulated time, samples, real time per step, amplifier status.
7. Bottom row: Step +1 sample (enabled only when paused), Restart.

9.6 Measure section (inside Inspect)
- Source popup ("{part} · component" or a node), Quantity popup (voltage node pairs and currents valid for that source), Sample field (ms) with the hint "live follows t".
- One sentence: "Plot adds a scope trace. Pin adds a reading to the sheet beside the part."
- Buttons: Plot live, Pin to sheet.

9.7 Pinned readings
- Section header "Pinned readings"; one row per reading: description left, monospace value right. Hovering a row reveals a remove button. Rows are also removable by removing the matching probe pill.
- Disclosure "More measurements & circuit overview" opens the existing overview.

## 10. Dock

10.1 Height 250 by default (§5.5), background `--bg-panel`, top border `--separator`. Layout: header row (40 px), plot card (fills; white, 8 px radius, hairline ring, 12 px side margins), footer row (34 px).

10.2 Header
- Segmented control Scope · Spectrum · Sweep. The Sweep tab is present only while a sweep is running or a sweep result exists in the session.
- A status phrase in tertiary: Scope "U1 powered · window {ms}"; Spectrum "Last {ms} · Hann window · {points} points"; Sweep "{quantity} vs {parameter} · {settle} settle per point".
- Right-aligned controls: Scope → sample segmented Auto · High · Low, Step +1 (enabled when paused), Restart. Spectrum → span popup, units segmented dB · Linear, Step +1, Restart. Sweep → progress bar (120 × 6 px) with "{k} of {N}", Compare with run {n−1}, Export CSV.
- "Hide scope" sits at the far right of the header on every tab.

10.3 Scope plot
- Left axis volts, right axis milliamps, 5 labelled gridlines each, zero line in `--hairline` drawn at 1.5× its alpha (0.18). Time range labelled at the footer ("{t0} → {t1}").
- Traces 1.5 px in the §3 trace colours, drawn at the plot's pixel resolution. The plot MUST show new samples within one display frame (16.7 ms at 60 Hz) of the engine producing them.

10.4 Cursors
- Two cursors, A and B. Each is a 1 px dashed `--accent` vertical with a 22 × 14 accent tag at the top. Click on the plot places the selected cursor; dragging a tag moves it.
- Selection: A is selected when the dock opens. Placing A selects B; placing B selects A. Clicking a cursor's tag or its footer value selects that cursor; the keys A and B select by name while the dock has focus. The selected cursor's tag and footer value are filled `--accent`; the other's are outlined. "Clear cursors" removes both and selects A.
- When both are placed, the span between them is tinted `--accent-wash` and a readout tag beside B shows "Δ {ms} · {Hz}", where Δ = |t_B − t_A| (always non-negative, independent of which cursor is further right) and `{Hz}` is 1 / Δ. When Δ is 0.00 ms the tag reads "Δ 0.00 ms · — Hz".
- Footer shows A and B values in monospace with their tags and a "Clear cursors" button. The current free-text cursor fields are replaced by these; typing a value is still possible by double-clicking a cursor value.

10.5 Spectrum plot
- Log frequency axis 20 Hz – 20 kHz by default (span popup offers 20 Hz–2 kHz, 20 Hz–20 kHz, 20 Hz–200 kHz). dB axis 0 to −80 relative to the strongest bin; Linear shows the engine's units.
- Cursor A snaps to the nearest spectral peak when placed within 8 px of one and its tag reads "{f} · {dB}".

10.6 Sweep plot
- X axis follows the sweep's spacing (log or linear); Y is the measured quantity. Measured points are 4 px radius white circles with a 1.8 px trace-colour ring joined by a 1.6 px line; pending points are 3 px hollow grey circles on the baseline. A dashed accent cursor marks one measured point: the point whose value is farthest from the mean of all measured points so far (a dip or a peak); ties go to the lowest parameter value. Its tag reads "{parameter} · {value} · {min|max}". The cursor updates as points arrive.

10.7 Footer (all tabs): legend chips (8 px dot + name + optional value), then cursor or range info, then right-aligned buttons (Inspect trace…, Export CSV, or Clear cursors as specified above).

10.8 Readouts. The current three-column readout strip below the dock is removed; its contents appear as pinned readings (§9.7) and probe pills (§8.9). On first launch after the upgrade, every reading that was in the strip MUST be migrated to a pinned reading so nothing disappears.

10.9 Status line. The current bottom status line ("17 components · 16 wires", "89% · 10-unit snap · local workspace") is removed. Components and wires appear in the sidebar footer (§7.1); zoom in the zoom pill (§8.5); snap in the Document tab (§9.4); storage in the Document tab Storage section.

## 11. Menus and popovers

11.1 Examples menu
- Anchored under the Examples button: 290 px wide, 10 px radius, `--bg-menu` fill, hairline ring and `--shadow-menu`, 5 px inner padding.
- Section header "Open an example"; one 30 px row per example with a checkmark column (shown for the currently loaded example), name, and part count right-aligned in tertiary. The current example's row is filled `--accent` with white text when the menu is open.
- Divider, then "Open in a new window" (⌥ hint) and "Browse all examples…".
- Choosing an example while the document has unsaved changes MUST prompt before replacing it.

11.2 Sweep popover
- Anchored under the Sweeps button, 300 px wide, `--bg-menu` fill, `--shadow-popover`, with a 12 px caret. Fields: Vary (popup of sweepable source parameters), From, to, Points, Log · Linear segmented, Measure (popup of measurable quantities), Settle (ms).
- Buttons: Run (becomes Stop while running), Save as preset….
- While a sweep runs: the swept source shows a 34 px radius dashed accent ring and a tag "{param} = {value}" on the sheet; the parameter's inspector field is disabled with accent styling and the hint "Controlled by the sweep. Returns to {original} when the sweep ends."

11.3 Connect by name popover
- Anchored to the sidebar's Connect by name button, 280 px wide, `--bg-menu` fill, `--shadow-popover`, caret on its left edge. One sentence of explanation; Pin popup (defaults to the selected pin if one is selected); Net list box (one row per net: colour dot, monospace name, short description); Cancel and Connect buttons.
- Connect creates a net label at the pin and joins it to the chosen net. The result is a normal net connection and MUST be reflected in §9.3's connections list.

## 12. Number formatting

12.1 With Units = SI prefixes (default), values are shown with an SI prefix and exactly 3 significant figures, trailing zeros kept: `−1.15 µV`, `7.90 mA`, `1.10 kHz`, `270 ms`. Values whose magnitude is below 1 pV, 1 pA or 1 ps display as `0`. The prefix set is p, n, µ, m, (none), k, M, G.

12.2 With Units = Scientific, values display as the engine returns them (for example `-1.1537e-06 V`).

12.3 Exceptions to 12.1, applied regardless of the Units setting: simulated time in the toolbar subtitle uses seconds with 3 decimals (`0.270 s`); the dock time range uses seconds with 4 decimals (`0.2189 s`); cursor positions and the cursor delta use milliseconds with 2 decimals (`233.40 ms`, `Δ 2.50 ms`); the frequency derived from the cursor delta uses 4 significant figures (`400.0 Hz`).

12.4 A readout MUST NOT change width more than once per second. Tabular numerals (`font-variant-numeric: tabular-nums`) are the expected means; any technique that meets the requirement is acceptable.

## 13. Keyboard and accessibility

13.1 Every interactive element is a native `button`, `a`, `input`, `select` or `textarea`, or carries the matching ARIA role with keyboard handling. Tab order follows visual order: sidebar, toolbar, sheet, dock, inspector.

13.2 Shortcuts (all unchanged unless listed as new):

| Key | Action |
|---|---|
| V | Select tool (new) |
| W | Wire tool |
| Space (held) | Pan |
| R | Rotate selection |
| ⌫ / Delete | Remove selection |
| Esc | Cancel wire / clear selection |
| ⌘S | Save now |
| ⌘Z / ⇧⌘Z | Undo / Redo |
| ⌘1 / ⌘2 / ⌘3 | Inspector tab Inspect / Simulate / Document (new) |
| ⌥⌘1 / ⌥⌘2 | Toggle sidebar / inspector (new) |
| P | Pause / Resume (new); ignored while a sweep is running |

13.2a Shortcut scope. Single-key shortcuts (V, W, R, P, A, B, Space, ⌫, Delete, Return, Esc) are active only while focus is on the sheet, the tool pill, the dock plot, or the document body. While focus is in any text field, text area, search field or editable cell, those keys perform their normal text-editing function and MUST NOT trigger a shortcut; Esc in a field blurs the field. The component list (§7.2) handles ↑, ↓, Tab and Return itself while it or its search field has focus; these are list behaviours, not global shortcuts, and are unaffected by this rule. Command-key shortcuts (⌘S, ⌘Z, ⇧⌘Z, ⌘1–3, ⌥⌘1–2) are active everywhere.

13.3 Focus rings use a 2 px `--accent` outline with 2 px offset and MUST be visible on every control, including segmented control segments and list rows.

13.4 Segmented controls that choose a value (Quality, Run for, Grid, Units, sample rate, dB · Linear, Log · Linear) use `role="radiogroup"` with `role="radio"` segments. Controls that switch visible content (inspector Inspect · Simulate · Document, dock Scope · Spectrum · Sweep) use `role="tablist"`/`tab` even though they share the segmented visual style; the component list uses `role="listbox"`/`option`. Live values use `aria-live="polite"` only in the inspector's live row, not on every probe pill.

13.5 Colour is never the only carrier of meaning: dangling ends have a shape and a label; traces have legend names; warnings have an icon and text.

## 14. Motion and performance

14.1 Panel show/hide animates width or height over 200 ms with an ease-out curve; popovers and menus fade and scale from 0.96 over 120 ms. With `prefers-reduced-motion: reduce`, all durations are 0.

14.2 Benchmark fixture: the document `bench/amplifier-x6.elab`, to be added to the repository with this spec, consisting of six copies of the "Amplifier driving a loudspeaker" example placed in a 3 × 2 grid sharing one VP and one VN (6 × 7 parts + 2 supplies + 6 grounds = 50 components), with the dock plotting VIN of copy 1, SP1 voltage of copy 1 and SP1 current of copy 1. Conditions: run state live, window 1440 × 900, in the current stable Safari and Chrome on a MacBook Air (M1, 8 GB) on mains power with a 60 Hz display, measured for 30 seconds with the browser's frame timing tools. Pass: mean frame rate ≥ 57 fps and no more than 1 % of frames longer than 32 ms. Implementation note, non-normative: rendering each trace as one path or on a canvas element meets this; one DOM node per sample does not.

14.3 Any popover, menu or inspector tab MUST be interactive (its first control accepts input) within 100 ms of the activating click or key press, measured on the §14.2 environment. The 120 ms fade of §14.1 continues after that point and is not part of the measurement.

## 15. Parity with the current interface

Every item below exists in the current UI and MUST have the stated home after the redesign.

| Current control | New home |
|---|---|
| File ▾ | Examples menu lower section ("Browse all examples…") and Document tab Storage/Export |
| Save | Autosave + ⌘S; status in Document › Storage |
| Examples | Toolbar, unchanged position (§6.1) |
| Undo / Redo | Toolbar left (§6.1) |
| Settings ▾ | Simulate tab (§9.5) and Document tab (§9.4) |
| "Live · RLC network transient" | Toolbar subtitle (§6.2) |
| Pause | Primary action (§6.3) |
| Parts › Close | Sidebar hide button (§7.1) |
| Components, Common · 6 ▾ | Sidebar scope row (§7.1) |
| Search all components, Name or alias | Sidebar search (§7.1) |
| Component list | Sidebar list (§7.1) |
| "6 matches · arrows choose · Tab to position" | Sidebar count + hint (§7.1) |
| Connect by name | Sidebar footer button + popover (§11.3) |
| Select / Wire / Pan / Cancel | Tool pill (§8.4) |
| Sweeps | Toolbar + popover (§11.2) |
| Fit | Toolbar toggle (§6.1) |
| Sheet hints line | Tool pill hint (§8.4) |
| Sample-step info lines | Sheet notice (§8.10) + Simulate callout (§9.5) |
| Inspector › Close | Toolbar inspector toggle (§6.1) |
| Inspect / Simulate / Document tabs | Inspector segmented control (§9.1) |
| SP1 header, live value, model text | Inspect header + live row (§9.2) |
| Plot voltage / Plot current / Rotate / Delete | Inspect action grid (§9.2) |
| Details, Choose another object | Inspect disclosures (§9.2) |
| Measure section and SP1 · component ▾ | Measure section (§9.6) |
| Plot live / Pin | Measure buttons (§9.6) |
| More measurements & circuit overview | Pinned readings disclosure (§9.7) |
| Scope / Spectrum tabs | Dock segmented control (§10.2) |
| Hide scope | Dock header (§10.2) |
| t = …, U1 powered · run n | Toolbar subtitle (§6.2) and dock status phrase (§10.2) |
| Auto / High / Low | Dock header (§10.2) |
| Step +1 sample, Restart | Dock header and Simulate tab (§10.2, §9.5) |
| Scope plot and axes | Dock plot (§10.3) |
| Export CSV, Inspect trace… ▾ | Dock footer (§10.7) |
| Legend line | Dock footer legend chips (§10.7) |
| Cursor A / B fields and × | Dock cursors (§10.4) |
| Three readout columns | Pinned readings + probe pills (§10.8) |
| Status line | Distributed per §10.9 |

## 16. Acceptance criteria

Each criterion is pass/fail.

- AC1. With browser local storage cleared, loading the "Amplifier driving a loudspeaker" example at 1440 × 900 produces the layout of §5.1 with the sidebar, inspector and dock shown, matching the design reference artboard "ELab · Native" within 4 px for every region boundary.
- AC2. Implementation-conformance inspection for the constraint in §3.1: searching the page's stylesheets finds no colour literal outside the token definitions of §3.
- AC3. Switching the OS appearance to dark re-themes the application without reload, and every text token other than `--text-disabled` passes 4.5:1 against each background token it is used on, exactly as §3.4 defines.
- AC4. Pressing Pause, then Resume, then starting a sweep, cycles the primary action through Pause → Resume → Stop sweep with the subtitle texts of §6.2; at no time are two filled-accent controls visible in the toolbar.
- AC5. Hiding each panel, reloading the page, and observing that the same panels are hidden.
- AC6. In the "Amplifier driving a loudspeaker" example, run state live, VIN at 0.05 Vpk and 400 Hz, with the JU wire shortened to end 40 units before SP1.positive: the open end shows the §8.7 marker; selecting the wire shows the §8.7 label and the §9.3 callout; pressing "Extend to SP1.positive" produces a wire that reaches the pin, the connections list shows SP1 · positive, and within 10 ms of simulated time the SP1 current trace has a peak amplitude of at least 5 mA.
- AC7. Setting Output to 100 µs with a 400 Hz source shows both the sheet notice and the Simulate callout; pressing either action sets Output to the engine's recommended value and clears both.
- AC8. Placing cursors A and B 2.50 ms apart shows "Δ 2.50 ms · 400.0 Hz".
- AC9. Every control in §15's left column is reachable in at most two clicks from the default layout.
- AC10. Upgrading a document that had three readouts in the old strip shows three pinned readings with the same descriptions and values after the upgrade.
- AC11. Tabbing from the page start visits every interactive control in the order of §13.1 with a visible focus ring, and all shortcuts of §13.2 work.
- AC12. With Units = SI prefixes, −1.1537e-06 V renders as "−1.15 µV"; with Units = Scientific it renders unchanged.
- AC13. At 1024 px wide with sidebar and inspector hidden, there is no horizontal scrollbar and the sheet, tool pill, zoom pill and dock are all visible.
- AC14. A sweep of 12 points fills the Sweep plot progressively, with pending points drawn hollow on the baseline, and Stop sweep restores the source's original value.

## 17. Open decisions

- OD2. Which storage targets "Move to …" offers (local only today).
- OD3. Whether sheet notes (§8.11) are in scope for the first release.
- OD4. Confirm that 32 samples per period is the right threshold for `recommendedOutputInterval()` (§8.12). The current app recommends 7.8125e-05 s for a 400 Hz source, which is exactly 32 samples per period, so the default follows the existing behaviour.
