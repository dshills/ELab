ELab **0.207.0 alpha 1 (build 313)** brings guided learning, connected group
editing and clearer circuit feedback to the web workbench.

Download **ELab-Web-0.207.0-alpha.1-macos-arm64.zip**, extract it and open
**ELab Web.app**. It starts the local engine and opens your default browser.
No development tools, account or internet connection are needed to use ELab.
[Start guide](https://github.com/dshills/ELab/releases/download/v0.207.0-alpha.1/START-HERE.md).

- **Learn one step at a time:** LED, capacitor and four-MOSFET H-bridge activities
  explain units, signed readings and what to change without assuming electronics
  experience.
- **Edit connected groups:** box/Shift selection, move, copy, paste and duplicate
  preserve internal wires and undo together. Faster selection/delete/drag input
  keeps the intended target.
- **Understand circuit checks:** associated parts and terminals are highlighted,
  with explanations and controls to locate a problem or begin a repair.
- **Keep the workspace responsive:** conditional scene updates and stable drawing
  nodes reduce update work; live controls and seven-segment displays keep updating.
- **Use the keyboard:** placement previews, construction/wiring, inspector and
  instrument navigation, dialogs, recorded samples and analysis have expanded
  keyboard and automated accessibility coverage.
- **Keep your library:** back up projects, history, Trash and unsaved workspaces;
  preview a restore and add copies while preserving existing work.
- **Work with a local agent:** enable MCP access to the visible browser workspace;
  agent edits appear on screen, and Disconnect invalidates the connection.
- **Identify and report a build:** the status-bar version opens About ELab and an
  explicit diagnostic preview/copy/download. Reports exclude project contents,
  recordings, paths, URLs, credentials and raw errors. Nothing is uploaded.

The editable four-MOSFET H-bridge remains included. Record `45 ms` at `1 ms`
to capture both load directions, then save and reopen the kept recording.
Existing web projects remain in `~/Library/Application Support/ELab/web`.
Earlier releases and the native-source archive are preserved.

**Qualification:** the complete local macOS quality plan passed (race, vet, lint and tooling), with **46 browser journeys passed and two planned non-Chromium load skips**. The exact archived backend passed inventory/signature checks, version/build/source identity, recovery, restart and backup/restore checks. Headless Chromium against the shipping binary verified LED learning, group editing, four-MOSFET reversal, exact preservation of all **50 recorded H-bridge rows** through save/reopen and export/import, private diagnostic reports and library download. [Hosted source CI](https://github.com/dshills/ELab/actions/runs/41878077636) at qualification: Web browser (chromium): queued, Web browser (webkit): queued, macos: queued, Web browser (firefox): queued, portable: queued. Local results do not replace uncompleted hosted results. Detailed scope and observations are in `QUALIFICATION.json`.

This private alpha is **ad-hoc signed and not notarized**. macOS can block first
launch; use the start guide and Apple's per-app opening guidance if you choose
to trust this download. The ZIP is tested on Apple Silicon macOS 27.2. macOS 13
is the configured build target, not a verified minimum. Intel, clean-account
installation, native VoiceOver and whole-app accessibility conformance are not
qualified by these checks.

Verify the release assets with `SHA256SUMS`. `release.json`, `PACKAGE.json` and
`QUALIFICATION.json` record source, hashes, toolchains and observed checks.
Built from clean `main`: `2ca9bc8d6643a672336af21ca08a392084cab3b2`.
