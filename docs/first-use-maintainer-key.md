# First-use kit: maintainer key and engineering checks

Read after the participant tasks. This is an expected-result key, not a report
of participant outcomes. No hobbyists have been recruited or observed for this
batch; the owner requested a self-guided kit instead.

| Task | Expected result | What to learn from participant feedback |
| --- | --- | --- |
| LED | Default educational red LED, 5 V, 1k: approximately 2.9168 mA; 2k: approximately 1.477 mA. Undo restores the earlier resistance. | Can someone find parts, connect terminals, apply a value and identify current without knowing ELab? |
| RC | Curated 10k / 1 µF, 5 V charge: 3.160602794 V at 10 ms. Discharging that voltage for another 10 ms: 1.162720790 V. Record from startup captures a separate experiment; live time/settings stay unchanged. | Are tool choice, pause/restart/step and recording provenance understood? |
| Flasher | Both LED currents/collector voltages alternate. Generic models and startup matter; doubling both timing capacitances approximately doubles the settled period. Compare matching waveform edges, not startup or half-cycle to full-cycle. | Are probe names/references, scope cursors and selected components discoverable? Do controls stay still? |
| Files | Reopened saved graph matches components/wires/values; SVG/PDF contain complete drawing. Fit-to-page yields one page for the practice circuit. Preview is not a submitted print job. | Are Save, Open, Export and Print Preview distinct and recoverable? |

Record blocks and hints separately from completion. A task completed only after
reading the help is assisted, not unassisted. Preserve original feedback; group
issues by wrong turn/confusion and severity. Never convert automated engineering
checks into participant success rates or invent completion times. A few volunteer
sessions can reveal problems; they cannot establish broad usability statistics.

## Maintained engineering counterparts

These test the same mechanics and electrical values through actual desktop
bindings, retained widgets and the public agent contract. They do not establish
physical mouse/keyboard behavior or beginner discoverability.

- `TestBuildFromBlankThroughUIAndAgent`, `TestCanvasPlacementAndTerminalGestureUseSameCommands`:
  blank construction, values/history and terminal gestures using shared commands.
- `TestRCDrawingScopeAndInputControlsMatchAgentCommands`: charging/discharging and
  exact reference equation through desktop controls and the agent.
- `TestFlasherDrawingInitialConditionsAndAgentParity`: both sides and startup.
- `TestMountedRecordFromBasicFormUsesAcceptedAdvancedSettings`,
  `TestMountedAdvancedFirstClickKeepsPrivateDuration`: retained basic recording,
  private duration handoff, unchanged live experiment.
- `TestMountedSimulationPrimaryActionsStayOutsideScrollingForms`,
  `TestMountedContextReadingsStayStationaryAsCircuitAdvances`: tool controls and
  reading geometry across updates; theme coverage is explicit in each test.
- `TestFirstUsePracticeSaveReopenAndExport`: saved graph, independent reopen and
  complete SVG/PDF output using disposable files and shared file/export commands.
- `scripts/mcp-demo.py`: matching agent with generated local MCP configuration;
  builds an LED, checks exact current, records two RC runs and compares samples.
  `--socket` also verifies a separately owned socket host. No mutation auto-retry.

The final alpha's evidence lists exact commands, results, clean source and archive
identities. Historical native OS queue/hardware-rate checks, clean-account install,
participant acceptance and Developer ID/notarization remain separate.
