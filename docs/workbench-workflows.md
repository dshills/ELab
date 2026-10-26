# Draw, understand and experiment

The existing 53 component kinds and electrical models are unchanged. These
workflow additions use the same semantic service for the desktop and agents.

## Drawing

Choose a component and move onto the sheet to see its snapped symbol before
placing it. Wire mode highlights the closest unambiguous terminal; a dashed
pending route becomes an ordinary wire only after a valid destination click.
Overlapping/equidistant pins require zooming or the existing named controls.
Crossings alone never connect. Escape cancels unfinished drawing.

With the canvas focused: **W** Wire, **V** Select, **H** Pan and **R** rotate the
selected component. Moving/rotating connected parts keeps terminal identities
and wires; a completed drag or rotation remains one ordinary Undo operation.
Text editors retain their own typing/shortcuts.

`drawing.preview {"position":{"x":100,"y":100}}` reads the current tool/kind
and zoom. Its result includes a snapped ghost, terminal, pending path, validity,
message and existing command/params. The ghost's `Preview` ID is illustrative;
no component/wire ID is reserved. To commit, dispatch the returned command with
the result's `command_revision` and `document_generation` as envelope guards.
The normal command/retry key requirements still apply. Occupied/ambiguous
candidates are not commits. Invalid query shapes/coordinates are rejected.

## Understanding a component

Select a component and read **Understand this component** in Inspect. Existing
value and geometry edits stay ahead of the explanatory section. Accepted current,
previous stale results and unavailable readings are explicitly distinguished.
Facts combine connectivity, signed readings, related diagnostics and the
published educational model assumptions; they do not diagnose an unknown solver
failure or certify physical hardware.

`component.explain {"id":"R1"}` returns the same structured explanation. The
selected component's explanation is also in `project.inspect.component_explanation`.

## Plotting and experimenting

**Plot live** reveals the live scope, highlights the selected source and keeps
the other requested traces. With six slots full, remove a trace explicitly;
plotting never silently replaces one. Plotting an existing trace focuses it.
From recorded view it returns to live without deleting the capture or changing
its rows/cursors. It preserves paused state and does not run a new simulation.
Agents use `scope.probe.plot {"id":"v:C1"}`. Existing explicit probe selection,
scope windows, cursors and recorded measurements retain their commands.

The Simulate inspector now includes the active example's question, suggested
experiment, original expected observation and model limits. **Your changes**
compares components, wires and transient settings with the original template;
Undo updates that comparison. Template expectations remain labeled as template
expectations when the circuit is edited. Guidance remains available in Document.
Agents read `project.inspect.example.changes` and the existing example metadata.
This comparison does not run or fabricate a baseline measurement.

## Reviewing agent or user edits

`edit.batch.preview` accepts the same `{name, edits}` as `edit.batch`. It validates
against a disposable graph and returns exact before/after differences and commit
guards without editing, reserving IDs or solving. Electrical success is separate
from command admission. The existing batch allowlist and child quantity grammar
apply unchanged.

`edit.proposal.set` stages that batch for shared review. The inspector shows its
supplied purpose/name, submitting actor and changes; existing affected parts and
wires are highlighted. UI actions use these same commands:

- **Apply changes:** `edit.proposal.apply {"id":"proposal-N"}`.
- **Refresh preview:** send `edit.proposal.set` with the batch again and review
  the new differences/identity.
- **Dismiss proposal:** `edit.proposal.dismiss {"id":"proposal-N"}`.

A graph/settings change or lifetime ID allocation makes a proposal stale. Apply
then refuses until a fresh preview is reviewed. Panning, selecting and live
samples do not invalidate an unchanged source. Document replacement clears the
proposal. An actual applied content change is one named Undo operation owned by
the applying actor; ordinary cross-actor history acknowledgment remains intact.
Proposals are session-only and are not saved as pending instructions in projects.
Direct `edit.batch` remains available for explicitly authorized immediate work.

These are engineering-verified workflows. The first-use participant study and
the four previously documented native-input acceptance checks remain separate.
