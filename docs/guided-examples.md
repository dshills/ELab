# Guided editable examples

The component tray identifies the current example and shows its question, template expectation, a suggested experiment and one model limitation. The catalog has 18 templates covering LED, split-rail supply, RC/high-pass, RL/RLC/flyback, NPN/manual switching, NMOS/PMOS/CMOS, op-amp follower/inverting amplifier, pulse blinking, flasher, LED fade and rectifier experiments. All guidance is available through the same immutable `examples.list` catalog to users and agents.

Loading an example creates an editable copy and starts a fresh run, including when reloading an unchanged, paused template. The template is its clean starting point: opening, running or switching between untouched examples does not mark the document dirty or require a Save/Discard decision. Value, position, experiment-setting, display/pin and kept-result edits still use ordinary save protection. Undoing all such edits back to the template clears the dirty flag; Reset also establishes a clean template. The title area and footer show the catalog example title until the copy is saved, then show its filename. The legacy LED loader follows this rule too; its previously missing restart could clear results without starting a replacement run. Changing its circuit geometry, topology, values or experiment settings labels it **Based on this example · circuit/experiment edited**. Display settings, pins, live time and navigation do not pretend the template physics changed. The displayed expectation explicitly describes the template, not a guarantee about an edited circuit.

**Reset [current example title]** restores that template rather than always loading the LED. It preserves the current pause preference, starts fresh at time zero/from template startup, and clears inherited physical/model state. Default experiment settings, positions, wires and view are centralized with the template. The built-in template is never edited by changing its copy. Reset detaches the current file path; a saved user project is not overwritten. If graph, preferences or kept results are unsaved, reset requires the same Save / Discard / Cancel decision as other replacements. One undo restores the edited graph and its prior example identity.

## Shared commands

| Command | Parameters | Behavior |
|---|---|---|
| `examples.list` | `{}` | Read the current immutable catalog of IDs, titles, questions, expectations, suggested experiments and limitations |
| `example.load` | `{"id":"rc","discard_changes":true}` | Load a known template as a running editable copy; discard remains explicit |
| `example.current.reset` | `{"expected_example_id":"rc","expected_document_generation":7,"discard_changes":true}` | Restore current template with optional identity/generation guards and unsaved protection |

Legacy `example.reset` remains an alias for loading LED, while `example.rc`, `example.npn`, `example.npn-pulse`, `example.flasher` and `example.rectifier` retain their named loads. `example.switch-npn` loads the manual transistor circuit, also available as `example.load {"id":"switch-npn"}`. All use the centralized definitions and loader-to-template mapping for fresh-run decisions. `simulation.restart` still restarts current **edited** physics; it does not replace the graph with a template.

`snapshot.example` is absent for a blank/unassociated document; otherwise it contains `info` from the catalog and `modified`. This association follows new/load/undo/redo/reset. A native rendered reset button captures the displayed example ID and document generation, and pending Save/Discard retains those guards. An intervening user/agent document replacement refuses a stale reset rather than acting on the replacement. Reset without an association is unavailable; unknown IDs and extra input fields refuse.

## Saved origin

Version-4 documents may carry optional `example_id`. It must be a known ID, with no null/empty/unknown values. It is origin metadata, not trusted graph/model/measurement data: an edited or externally authored graph can claim a known origin, but comparison with the actual immutable graph/settings marks it edited. A reader never makes an unsupported circuit valid because of this ID. Files without origin continue to load with no inferred association. Save/recovery carry the identity, and fresh reopening starts physical simulation from saved startup conditions rather than restoring old time/charge.

The origin participates in the complete saved presentation/provenance baseline and history. Older readers lacking this optional version-4 extension refuse it instead of dropping it. Legacy versions 1–3 remain readable without newer extensions. The legacy compatibility tests deliberately remove newly added origin metadata when synthesizing an old-format fixture; validation was not weakened to permit a new extension in an old version.

## Evidence

```sh
./scripts/check.sh
./scripts/build.sh
./build/elab --smoke
python3 scripts/examples-demo.py --output docs/validation/headless-examples.json \
  --artifacts docs/validation/example-artifacts
# Owned development session only: replaces its circuit.
python3 scripts/examples-demo.py --connect build/examples.endpoint.json \
  --output docs/validation/desktop-examples.json \
  --artifacts docs/validation/desktop-example-artifacts
```

Core/public checks cover all 18 running templates, immutable guidance, edited labeling, rejected reset purity, exact template restoration, fresh paused startup, undo identity, save/reopen and stale example/generation guards. A regression checks repeated loading of every catalog template and all seven legacy loaders. The public demo also checks fresh run/electrical/document identities and available results for each legacy reload. Native tests and actual smoke cover guidance controls, reset/cancel/discard decisions and stale rendered-button refusal. Existing electrical reference tests remain authoritative for the templates' model accuracy. These checks establish engineering behavior, not novice comprehension, participant outcomes or complete keyboard/screen-reader qualification.
