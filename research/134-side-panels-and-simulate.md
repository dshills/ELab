# Side panels and a focused Simulate inspector

Implemented the first two changes from the [UI simplification proposal](133-ui-simplification.md):
independently collapsible/resizable left and right panels, and a chooser showing
one existing Simulate tool at a time. No component, model or solver limit changed.

## What changed

The document toolbar now has Parts and Inspector visibility buttons with selected
states and descriptive accessible names/tooltips. Both panels start visible;
either can be collapsed without stopping simulation. Drag either inside divider
to resize it. Focused dividers support arrow keys and Home/End. Collapse retains
preferred width, while reopening restores keyboard access to the panel.

On small windows the displayed widths narrow to protect drawing tools and canvas.
A user can widen one panel by reducing the other visible panel. These layout
choices are session state, survive document replacement and do not dirty a
schematic or get saved into its file. They reset on a new application launch.

Simulate now has six destinations above its scrolling form: Live controls,
Recorded runs, Recorded spectrum, DC/AC/noise sweeps, Component tolerances and
Circuit details. Only the selected destination's controls are mounted. Switching
does not launch/cancel analysis or discard accepted settings/results. Each newly
selected form starts at its top; ordinary updates preserve its scrolling. The
live Scope/Spectrum dock and contextual print/sweep controls remain available.

## Shared commands

All previous 226 public commands remain. Two new guarded commands expose these
same presentation actions to agents:

- `workspace.panels.set`: atomically set visibility and preferred widths for both
  sides; inspect `workspace_panels`.
- `workspace.simulation.set`: choose the task without changing inspector/panel
  visibility; inspect `simulation_tool`.

[Workspace reference](../docs/workspace-layout.md) documents complete parameters,
bounds, task names and lifecycle. Standard discovery exposes the schemas and
contracts. Ordinary discovery remains inside its existing 64 KiB catalog budget;
waveform input schemas reuse local registry definitions to accommodate the new
commands without increasing a transport bound or changing numeric outputs.

## Verification and review

New tests cover both panel toggles, width retention and constraints, keyboard
targets, all tool destinations, existing-control reachability, stale callbacks,
an actual queued divider callback across an agent edit, exact retries, atomic
invalid requests, document/file isolation and electrical work with panels hidden.
Retained layout checks cover four themes at minimum and larger window sizes.
Existing smoke/electrical/parser/timing tests navigate the real chooser.

Owned native sessions exercised panel collapse/reopen, resizing, tool choice,
wheel scrolling and scrollbar dragging. They exposed redundant outer horizontal
scrollbars, corrected in the local native-chrome adapter while preserving the
actual managed scroller. The RC test remained paused and clean with unchanged
electrical revision/run ID. Owned sessions closed normally.

The [validation record](../docs/validation/side-panels-20261029/README.md) records
final regression outcomes, source/package boundaries and native screenshots.
[Internal review](../docs/validation/side-panels-20261029/internal-review.md)
records decisions, corrections and evidence limits. No external review service
was used. The preserved qualified /285 release is unchanged; this UI development
candidate does not close historical native/human/install/signing gates.
