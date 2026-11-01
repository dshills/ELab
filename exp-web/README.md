# ELab Web

ELab's local browser workbench uses the same Go service, solver, models,
validation, history and native project format as the desktop app. This is the
primary UI development direction on `exp-web`; the native client remains intact.

## Run

For the private macOS alpha, open **ELab Web.app**. It starts the local host and
opens your default browser. Its window shows status and provides Open ELab,
projects-folder/log access and Quit. Closing a browser tab keeps ELab running;
closing the launcher or Command-Q saves and stops it. A failed checkpoint leaves
ELab open so you can repair storage or export your work.

The app reuses a compatible host already using the same library. It selects a
free local port if the preferred port is occupied. If an older CLI server owns
the library, close that server once before opening the app; its projects remain
in place. The native ELab app remains a separate application.

Build a self-contained macOS app and optional private-alpha ZIP:

```sh
python3 -B scripts/package_web.py --output 'build/web-alpha/ELab Web.app' \
  --archive build/web-alpha/ELab-Web-0.1.0-alpha.1-macos-arm64.zip
```

Runtime users need neither Go nor Node. Building requires the Go toolchain and
Xcode command-line tools. Packaging refuses existing outputs and records source,
licenses, signatures and checksums. See the
[launcher qualification](../docs/validation/web-launcher-alpha-20261101/README.md)
for verification scope and signing/platform limits.

For development, run from the repository root, using the Go version in `go.mod`:

```sh
go run ./exp-web
```

Open <http://localhost:8788>. Stop with Control-C; autosaved work survives server
restart. No Node packages, CDN, account or frontend build is required to run it.
To build a standalone executable with all frontend files embedded:

```sh
go build -o build/elab-web ./exp-web
./build/elab-web
```

`--dev` serves `exp-web/static/` directly for development. `--addr localhost:8789`
selects another port. Only loopback addresses are accepted. `--data-dir /path/to/data`
selects a different persistent library; one server may write a library at a time.

The default data directory is the platform's user configuration directory plus
`ELab/web` (on macOS, `~/Library/Application Support/ELab/web`). Keep this directory
when upgrading the executable. Avoid placing real work under disposable `build/`
or temporary directories.

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

There are at most 16 active browser workspaces. An idle workspace's engine closes
after two hours; its files remain and are restored on return. Active pages poll.

## Workbench

Search all 61 component kinds or open one of the 88 examples. Select a part to edit
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
PDF/PNG use the native macOS renderer. **Print schematic** opens a portable browser
preview and the browser's Print / Save PDF dialog; it submits nothing automatically.

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
go test -race ./exp-web
go vet ./exp-web
node scripts/check-web.mjs
```

Node 22+ is used only for frontend validation, with built-in modules and no npm
installation. The shared portable/native quality plans run these frontend checks,
and their input signatures include JS, HTML and CSS. The GitHub workflow runs on
`exp-web`, `main`, PRs and the existing Codex branch pattern.

See [PARITY.md](PARITY.md) for the feature matrix, verification evidence, internal
review and platform differences. This is a local-server web app, not a browser-only
WASM application or an authenticated hosted multi-user service.
