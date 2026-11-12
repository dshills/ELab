# ELab Web

ELab's local browser workbench uses the same Go service, solver, models,
validation, history and native project format as the desktop app. This is the
primary ELab interface. The former Vellum UI is archived in Git; see
[restoration instructions](native-archive.md).

## Run

Start ELab and open [localhost:8788](http://localhost:8788) in your browser.
See [Run ELab](run-web.md) for downloads, building from source, and the optional
Mac launcher.

For development, run from the repository root, using the Go version in `go.mod`:

```sh
go run .
```

Open <http://localhost:8788>. This address serves the styles and connects the
browser to the simulation engine. Opening `static/index.html` from disk shows
unstyled markup and leaves the engine disconnected. No Node packages, CDN,
account or frontend build is required. Stop the host with Control-C. Autosaved
work survives a server restart.
To build a standalone executable with all frontend files embedded:

```sh
go build -o build/elab .
./build/elab
```

`--dev` serves `static/` directly for development. `--addr localhost:8789`
selects another port. Only loopback addresses are accepted. `--data-dir /path/to/data`
selects a different persistent library; one server may write a library at a time.

The default data directory is the platform's user configuration directory plus
`ELab/web` (on macOS, `~/Library/Application Support/ELab/web`). Keep this directory
when upgrading the executable. Avoid placing real work under disposable `build/`
or temporary directories.

## Draw across sheets

Sheets divide a drawing into smaller views of the same circuit. Choose **Explore
examples**, then open **RC charge and discharge** to try moving its capacitor
to a separate sheet.

1. On **Main**, open **Manage sheets**, enter `Filter` as the **Child sheet name**,
   and choose **Add child sheet**. ELab opens **Main / Filter**.
2. Use the **Sheet** selector to return to **Main**, then select capacitor `C1`.
   Open **Manage sheets**, set **Destination sheet** to **Main / Filter**, and
   choose **Move selection**.
3. ELab opens **Filter**. The existing connections appear as named ports. Select
   a port on the drawing or focus it and press Enter to return to **Main**.
4. Save the project. Reopening it restores the sheets, connections, and selected
   sheet. The existing scope traces still measure the same capacitor.

To make a new connection, open **Manage sheets** on either sheet. Enter a **Port
name**, choose a **Local terminal** and an **Other sheet terminal** on its parent
or child, then choose **Connect port**. Ordinary net labels connect within a
sheet. Ground is shared across the project.

**Find in project** searches part IDs, labels, terminals and signals. Choose a
result to open its sheet and center the drawing on the match. The arrow next to
a live scope trace goes to its source. Circuit checks can open the sheet with the
affected part or terminal.

You can add up to 24 child sheets. All sheets share the same project-wide
simulation and drawing limits. A project save includes every sheet. Schematic
SVG, PDF and PNG export the current sheet with its named connections.
[Circuit capacity](circuit-capacity.md) · [Project format](project-format.md)

### Reuse a sheet

Open **Manage sheets** on the parent where you want the copy. Pick a **Source
sheet** that has no children of its own, enter a **Linked sheet name**, and choose
**Place linked sheet**. The source remains part of the circuit. The copy has its
own components and measurements. Use **Connect port** to wire it to its parent.

To update the copies, open the source, make your edits, then choose **Manage
sheets → Apply to linked sheets**. The update carries over the source layout and
internal wiring. Components whose settings you edited locally keep those settings.
Existing external connections stay in place. If an update would remove a terminal
still wired on a copy, ELab identifies the wire. Disconnect or reroute it, then
apply the update again.

## Save and recover

- Edits checkpoint locally before successful command replies. Completed background
  results are also checked every two seconds. Autosave preserves the unsaved indicator.
- **Save project** creates or updates a named project; **Document → Save a copy**
  creates an independent project. Saved projects are visible in other browser workspaces.
- Each browser profile keeps a persistent workspace cookie; its tabs share that
  workspace. Different profiles have independent live circuits and a shared library.
- Reopening after server loss restores the checkpoint as unsaved work. **Projects →
  Recovery history** includes earlier replacement/save copies and unsaved workspaces
  from other or lost cookies. Recovery creates a copy, preserving the source.
- **Move to Trash** is reversible. Automatic history keeps the newest 50 copies;
  saved projects and workspace checkpoints are not removed by that retention rule.
- An external/concurrent saved-project change is refused; reopen its current version
  or save a copy. Storage errors appear in a persistent banner, and `.elab` download
  remains available to rescue the editor's current contents.
- **Open** imports a native `.elab` file. **Download .elab copy** exports it without
  clearing unsaved state. Imports are limited to 8 MiB, matching the native reader.

**Projects → Back up library** downloads saved projects, Trash, history and
workspace checkpoints with kept results. **Restore library backup…** previews and
adds validated copies while preserving the current circuit and existing data.
See [library backups](library-backups.md) for scope and size limits. Restored
history is retained outside automatic history rotation.

There are at most 16 active browser workspaces. An idle workspace's engine closes
after two hours; its files remain and are restored on return. Active pages poll.

## Workbench

Search all 61 component kinds or open one of the 89 examples. Select a part to edit
engineering-unit values, startup/loss parameters, device models and live readings.
**Simulate** exposes sampling, playback and source controls. **Workbench tools**
(the gear above the instruments) contains recording playback, interval statistics,
threshold timing, step response, comparison, FFT/THD, sweeps, tolerances, circuit
blocks, display cues, pinned measurements, audio, imports and exports. Native
schemas provide labeled forms; JSON is optional in the command browser.

The Scope source picker switches between the live circuit and completed captures.
Recorded runs have Play/Pause, previous/next stored sample, a keyboard-accessible
scrubber, exact index/time seeking and playback speed under **Position & speed**.
The current captured values and sample marker come from exact stored rows. **Keep
in project** controls saved membership; opening an existing capture leaves that
choice alone. New captures are kept automatically as before.

**Traces & interval** selects live/captured probes, sets precise windows and A/B
cursor times, and shows sample statistics. Captured runs also support time-weighted
RMS, AC RMS, mean and peak-to-peak measurements over an explicit interval. Time
entries accept units such as `20 ms`. Source changes in another tab disable stale
forms. The schematic and component inspector continue showing the live circuit.

**Record** accepts duration/sample units, source input and a default capacitor
startup override. Zero sample interval follows the project; DC circuits capture a
single operating point with native default measurements. Invalid entries stay
visible for correction. Active capture progress and cancellation appear above
the Scope. Space plays/pauses a captured Scope; the schematic toolbar controls the
live circuit.

The Scope shows A/B cursor values and reference traces. Spectrum and Sweep plot native results. Retained sweep traces include AC
gain and phase. Tools can keep captures/sweeps in project files, choose prior results,
and export their complete JSON or CSV data.

Audio is rendered independently, loaded into the browser, and played only through
**Play preview** or the browser's media controls. Electrical changes invalidate it.
**Export** includes schematic SVG/PDF/PNG, scope and analysis CSV/JSON, and WAV.
Use **Download pdf** or **Download png** in Tools → Export on any supported host.
**Print schematic** opens a browser preview and Print / Save PDF; it submits nothing automatically.

Document offers Light, Dark, System and Apollo appearance. Panels resize by drag or
keyboard; narrow screens have Parts and Inspector toggles. System reduced motion
is respected alongside the saved native display preference.

Shortcuts: V select, W wire, H pan, F fit, R rotate, Delete remove, Space pause/resume,
Command/Ctrl-Z undo, Command/Ctrl-Shift-Z redo, Command/Ctrl-K commands,
Command/Ctrl-S save, Command/Ctrl-O open, `/` find a part, `?` help. Arrow keys move a
selected part by 10 units (Shift: 100); Enter places a chosen part or connects a
focused terminal. Browser text editing retains its normal shortcuts.

## Qualification

```sh
go test -race .
go vet .
node scripts/check-web.mjs
```

Node 22+ is used only for frontend validation, with built-in modules and no npm
installation for those logic checks. The shared portable/macOS quality plans run these frontend checks,
and their input signatures include JS, HTML and CSS. The GitHub workflow runs on
`exp-web`, `main`, PRs and the existing Codex branch pattern.

Separate [browser qualification](../browser/README.md) uses locked development-only
dependencies for Chromium, Firefox and WebKit CI journeys. See the
[promotion gates](validation/web-release-confidence-20261101/README.md)
for hosted evidence, manual accessibility and pilot requirements.

See [feature parity](web-parity.md) for the feature matrix, verification evidence, internal
review and platform differences. This is a local-server web app, not a browser-only
WASM application or an authenticated hosted multi-user service.

## Agents in this workspace

Choose **Connect agent** or the document menu's **Agent connection** to explicitly
enable a private local MCP connection to the visible workspace. Copy the generated
client configuration; the installed backend provides the bridge. Agent edits use
the same guards, autosave and history. Undo of another actor's edit requires
confirmation. Disconnect revokes access; restarting ELab requires enabling and
reconnecting again. See [agent setup](agent-setup.md) for scope and retry behavior.

### Compare a Change

Select a part with an editable value and choose **Compare a Change** in the
Inspector. Enter a trial value and choose a voltage or current. ELab runs the
original and trial circuits from startup with the same simulation settings.
The chart overlays their sampled curves. When both curves reach the displayed
level, the panel shows an estimated first crossing time for each.

**Keep trial value** applies one undoable edit. **Discard trial** closes the
comparison. If an edit changes the project while the comparison is open,
reopen the panel before applying the trial.

### Replay on the schematic

Choose a completed recording in **Scope source**, then choose **Replay on
schematic**. The player opens the captured circuit and its exact stored samples.
Choose a measurement and drag the slider, click the waveform, or press **Play**.
A signed current with two captured terminal references shows an arrow in the
measured direction. The sheet picker also works for recordings of projects with
multiple sheets. Replay uses only captured channels and keeps its own position.

### Troubleshoot a circuit

Open **Why isn’t this working?** from the document panel or the circuit checks.
The guide flags missing ground, open connections, and backward LED wiring when
it can identify the cause. For common single-supply circuits, choose **Try on a
copy** to test the suggested wiring and see its startup reading. Your drawing
stays open with its original wiring.

Use the virtual voltmeter to read the voltage between two terminals at startup.
A negative reading means the red probe is at a lower voltage than the black
probe. If the circuit cannot solve, the meter explains why it has no reading.

### Save an experiment test

Open **Experiment tests** from the instrument toolbar. Choose a voltage or
current, a condition, and a duration. For example, set **C1 voltage** to rise to
**3 V** within **20 ms** in the RC example. Run the test to see pass or fail,
the measured samples that determine the result, and the waveform. Current
reversal looks for samples on opposite sides of zero.

**Save test** keeps the definition with your project. After changing a part or
reopening the project, run the test again against the current circuit’s startup
and sampling settings. The tests use sampled values and can miss changes that
happen entirely between samples. Saving a test keeps its definition. Run it to
regenerate the evidence.
