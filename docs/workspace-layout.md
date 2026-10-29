# Adjustable schematic and scope workspace

When a live or recorded scope is expanded, drag the divider between the schematic and scope to choose how much space each gets. The default assigns 65% of the central workspace to the schematic. The divider accepts 35–85%, leaving both panes available. With divider focus, Up/Down changes the share by two percentage points and Home/End selects the limits. The accessible divider exposes the same accepted value.

Scope contents fill the available width and scroll vertically when their controls do not fit. Wheel input over the plot scrolls its containing scope pane; it never zooms the trace. Collapsing the scope removes the divider and keeps the chosen fraction for the next expansion. Print preview and DC sweep hide the source/scope split while retaining its preference.

## Shared command

```json
{"command":"workspace.layout.set","params":{"schematic_fraction":0.8}}
```

This fragment omits the ordinary request envelope, mutation guards and retry key. `schematic_fraction` is a required finite number from 0.35 to 0.85. Missing/null/text/out-of-range fractions and unknown fields refuse before mutation. `project.inspect.workspace_layout` exposes the accepted fraction; detailed command discovery explains effects, guards, errors and completion.

The preference is session-only. Example load/reset, New, file open and recovery retain it; schematic files do not import another session's layout. It changes no graph, simulation/run/time/charge, history, dirty flag, scope visibility, probes, cursors or recorded playback. The native divider uses this command with captured command revision and document generation, so a stale callback cannot overwrite a newer agent choice. Widget float32 range endpoints map to the exact declared bounds.

The command updates its preference immediately; native layout follows on a later frame. The existing `view.viewport.set` reports the resulting drawable area. [Fit mode](viewport-fitting.md) then reframes the schematic if following is enabled. Manual views remain unchanged. A headless client can report virtual canvas dimensions explicitly; the layout fraction alone does not resize an OS window or fabricate measurements.

## Evidence and limits

Tests cover layout bounds and refusal atomicity, owned/pure discovery, physical and measurement invariants, recorded cursors/playback, session/file separation, native callbacks and stale refusal. Mounted smoke checks actual canvas resizing, full-width scope allocation and scrollbar movement from the plot without changing its zoom. It preserves manual navigation and restores its own split fixture before later pixel tests. Recorded GPU assertions scroll their plot into view before demanding a fresh frame.

The public reference `scripts/workspace-layout-demo.py` runs a fresh native session first, then a headless replay using `--sizes-from`. Both inspect the same commands, measured-view transitions, recorded cursor preservation and file/session boundary. Direct native accessibility inspection sets the divider from 65% to 80% and shows the fitted NMOS example grow from about 29% to 41% at the default window. This is an improvement over the previous fixed scope allocation, not a claim that every label is readable at every size.

The fraction is a preference rather than a guaranteed minimum pixel size. Large schematics still need zooming or scope collapse for readable labels. Full responsive rail/tool layouts, complete keyboard/IME/accessibility acceptance and visible latency remain open. Callback and mounted event tests do not establish every physical pointer/scrollbar gesture.

## Inspector sections

The document workspace separates the right inspector into **Inspect**, **Simulate**, and **Document**. Navigation stays above a single scrolling form. Inspect contains selected-object editing and measurements. Simulate shows one tool at a time; see the tool chooser below. Document starts with export/print actions and page settings, followed by appearance, display and recovery. The component library remains on the left.

Users and agents select the same section:

```json
{"command":"workspace.inspector.set","params":{"section":"simulation"}}
```

Allowed `section` values are `component`, `simulation`, and `document`; they map to Inspect, Simulate, and Document respectively. The normal request envelope, mutation guards and retry key are required as described in the agent API. `project.inspect.inspector_section` reports the accepted section. Detailed discovery describes the contract, and view/preference events announce changes.

The initial section is `component`. This choice is session-only, survives document replacement, and is absent from saved schematic files. It changes no graph, electrical revision, simulation time, history, dirty state, probes or cursor measurements. Hidden sections have no mounted fields or keyboard targets. Each newly selected section and replacement document starts at the top of its form; ordinary edits and repaints preserve the current section's scroll. Print-preview and sweep workspaces retain their contextual left controls and the right-hand task navigation.

## Collapsible, resizable side panels

Parts and Inspector are independent docked drawers that move the canvas aside.
Each open drawer has a **Close** button beside its title at the workspace edge.
Closing it leaves a labelled **Parts** or **Inspector** edge tab to reopen it;
the closed form takes no horizontal space. The controls stay above scrolling
content, rather than being hidden among the document-toolbar actions.

Drag the divider at a drawer's inside edge to resize it. Drag far enough toward
the outside edge to close it; its preferred open width is retained. A focused
divider accepts Left/Right; Home closes Parts and End closes Inspector, with
the opposite key widening it. Buttons announce Open/Close and continue keyboard
focus into a reopened drawer or back to its edge tab on close. In print/sweep
workspaces the left tab is labelled Print/Sweep and reopens those controls.

Panel widths and visibility are shared session state:

```json
{"command":"workspace.panels.set","params":{"parts_visible":false,"inspector_visible":true,"parts_width":280,"inspector_width":420}}
```

All four parameters are required; widths are finite logical units independent
of display density. Parts accepts 224–360; Inspector accepts 296–560. The native
layout temporarily narrows preferred widths when necessary to leave drawing
space. Inspect `workspace_panels` for the accepted preferences. Collapse retains
widths; changing documents retains these session choices. They are not saved in
schematic files or persisted across app launches, and do not mark a circuit dirty.
Existing `workspace.layout.set` keeps its original vertical scope-split contract.

## One Simulate tool at a time

The Simulate inspector has a persistent tool chooser, with just the selected
tool's form beneath it:

| `tool` | Visible controls |
| --- | --- |
| `live` | Live timing, source/LC sampling, base drive, probes and example experiments |
| `recordings` | Verify from startup, recording import/selection, playback, comparison, timing and exports |
| `spectrum` | Recorded FFT and distortion, its settings/results and exports |
| `sweeps` | Existing DC, AC response, loop/noise sweeps and retained results |
| `tolerances` | Existing component-tolerance trials and their settings/results |
| `circuit` | Network details, capacity and circuit information |

```json
{"command":"workspace.simulation.set","params":{"tool":"recordings"}}
```

Observe `simulation_tool` in `project.inspect`. The default is `live`; the choice
survives document replacement within this session. Navigation does not start a
job, change panel visibility/inspector section, or discard another tool's accepted
settings or results. Hidden forms are unmounted and absent from keyboard traversal;
selecting another tool starts at its top. The chooser remains above scrolling
content, and print/sweep views retain task navigation. Error guidance stays with
the active task. Large individual forms still scroll rather than clipping fields.

The live Scope/Spectrum instrument dock is unchanged; recorded spectrum in this
chooser remains a separate analysis of captured data. Every existing electrical,
file, analysis and instrument command still works with either side panel collapsed.
Both new commands use the ordinary shared guards, retry identities and view/
preference events. The request fragments above omit that required envelope.

## Details and Advanced sections

Inspect puts the selected part's value, main actions, live summary and common
plots first. **Details** retains model settings, initial conditions, full readings,
coordinates, duplicate and explanation controls. **Choose another object**
retains the object selector and navigation. Direct actuation such as switches,
pushbuttons, sensor inputs and potentiometer position stays in the basic view.

The disclosure choice is shared session presentation:

```json
{"command":"workspace.details.set","params":{"section":"component","visible":true}}
```

Both parameters are required. `project.inspect.workspace_details` exposes the
choices. Allowed sections are `component`, `selection`, `measurement`, `pins`,
`live`, `recordings`, `spectrum`, `sweeps`, `tolerances` and `circuit`. Choices
start closed and survive document replacement within a session; they are not
saved in schematic files and do not affect dirty state, electrical results,
history or accepted analysis settings. Normal guards and exact retry identities
apply. Hidden controls retain private editor state through collapse, but are
excluded from visible input/keyboard traversal. Changing documents retires that
private state through the existing document-generation keys. Electrical commands
remain available to agents regardless of disclosure state.

Simulate keeps essential inputs in the basic view and the chosen tool's primary
Record/Run/Analyze action above its scroller. **Advanced** retains startup and
sampling overrides, spectrum windows/intervals, sweep resolution/probes and
variation/trial settings. Closed fields use their accepted settings; opening
Advanced does not start a job. Recorded Spectrum offers Record/Open routes when
no completed recording exists. Compile guidance leaves saved-result import and
existing captured results available.

## Measurements and guidance

**Measure** has one source chooser for components, wires, terminals, nets and
voltage/current probes. Follow schematic selection restores automatic inspection;
Plot and Pin keep their existing shared commands. The bottom strip follows that
measurement and preserves signed units and stale/unavailable qualifiers. A blank
circuit shows a selection prompt. The prior named whole-circuit metrics remain
under **Circuit overview**, controlled by the `measurement` Details section.

The header reports the actual experiment state; paused does not cover up an
invalid/unsupported circuit. The canvas shows the current actionable guidance;
**All guidance & technical details** in the inspector retains every issue,
related object and technical explanation through `diagnostics.details.set`.
