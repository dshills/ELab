from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');out=root/'build/releases/ELab-0.195.4-macos-arm64'
(out/'README.md').write_text('''# ELab 0.195.4 /265 — macOS arm64

Draw a circuit, watch it simulate, edit values, measure waveforms and save your
experiment. The matching `elab-agent` exposes the same circuit and browser commands.
This package includes the current 53 component kinds and 62 editable examples;
this reliability round adds no components or electrical models.

## Run

Apple Silicon Mac, macOS 26.0 or later. Tested on this workstation's macOS 26.6.2.
No Go, Xcode, Vellum checkout, account or internet service is needed at runtime.
Verify the ZIP with the adjacent `.sha256` file, extract it, and open `ELab.app`.
Keep prior versions separately rather than replacing an app while it is open.

This is the owner-approved unsigned distribution: an ad-hoc integrity seal,
without Developer ID signing or notarization. Download quarantine/Gatekeeper
may block it; use macOS's normal trust decision yourself if necessary. This
package does not change security settings. Intel/older macOS are not qualified.

## First circuit

Choose Examples, search for LED, select LED current limiter and Open example,
or open `Examples/led.elab`. Resume the paused
file. A 5 V supply and 1 kΩ resistor give about 2.917 mA; 2 kΩ gives 1.477 mA.
Select R1, enter `2k`, commit, then Apply value. Invalid values show bounds and
leave the accepted circuit unchanged. Cmd+Z and Cmd+Shift+Z undo and redo.

To draw from scratch, search for a part, Return to its matching list, then
Return/Space to choose it. Click the canvas, or Tab to X/Y and Place at X / Y.
Connect by name offers From/To terminal menus and Connect wire. Escape/Cancel
abandons an unfinished placement or wire. Existing drawing objects stay intact.

The inspector's Inspect, Simulate and Document sections hold component edits,
analysis and output respectively. Scrolling a numeric field out of view now
cancels its help tooltip without changing its value. Themes include Light,
Dark and Apollo. The component category starts at Common; search covers all
categories. No component has been removed or added in this release.

## Experiments

The Examples browser offers Start here, topic categories, global word search,
real schematic previews and short experiments. Return commits search, arrow
keys choose and Return opens. Close/Escape returns to the toolbar. Browsing
never changes the live circuit; actual replacement protects unsaved edits.

All 62 `.elab` files in Examples are paused, editable copies with matching SVG
schematics. `Examples/README.md` explains what to observe, what to change and
where each educational model stops. Opening an untouched built-in example
starts clean; real edits are protected by Save/Discard/Cancel. Cmd+S saves your
own copy. Restart starts fresh physical state; pausing holds it for inspection.

Try RC charge/discharge and capacitor-timed NPN/555 flashers, then use named
voltage/current probes and scope cursors (`0 ms`, `10 ms`) to compare timing.
The catalog also covers logic/counters/display, sensors, power supplies,
transformers, motors/H-bridge and audio. Audio previews require explicit Play;
simulation never starts sound automatically.

## Quality repairs

Background scope/sweep cursors and object shortcuts now refuse input while the
Examples sheet is open. Deliberate public agent measurements remain available.
Printed/exported parts rows include active PWM, op-amp dynamics and sounder
settings, so shared experiments retain those configurations in human-readable
output. The schematic project files and numerical models are unchanged.

## Models and output

These are bounded educational models, not calibrated device or safety/rating
predictions. Read each component's model assumptions and each example's limits.
Observe diagnostics; stale traces are marked and are not a new solved result.
Existing solver precision, timestep and work/capacity limits are unchanged.

Save/Open preserves schematic, settings and kept completed results. Document
contains SVG/PDF/PNG export and print/page preview. Verify/sweeps publish only
completed data; save again if a kept result completes after the earlier save.
Technical source notes are in Documentation; some evidence links refer to the
repository and are not included in this distribution.

## AI agents

Run `./elab-agent` for an independent headless session, or connect to an explicitly
owned native app endpoint. Start with `capabilities.list`, `parts.list`,
`examples.list` and `project.inspect`. Mutations use the documented revision,
document and retry guards. See AGENT_PROTOCOL.md. There are no provider keys or
cloud requirements. The app and agent are from the same clean source revision.

See RELEASE_NOTES.md for exact validation scope and ROLLBACK.md before switching
versions. Owner walkthrough and a separate clean-account installation check
remain deferred by instruction; this is engineering qualification on one Mac.
''')
(out/'ROLLBACK.md').write_text('''# Keeping a previous version

Preserve the qualified ELab 0.181.4 /205 download and your known-good app.
Quit the new app normally before reopening the older copy. Never run two copies
against the same recovery directory. Keep project backups before overwriting;
use Save As for projects edited by newer releases. The newer component models
and parameters may be unsupported by /205; do not use an older app to overwrite
those projects. Saved live physical state is reinitialized on open.

The /205 ZIP is preserved unchanged with SHA-256:
`d13973f02f9c2895d353d24338ed060492a3a2200ff8a91e3606bfa867f4946b`.
The /256 and /257 development packages are also preserved locally. Installing
this package is not required to recover files saved by those packages.
''')
print('Release usage and rollback notes prepared')
