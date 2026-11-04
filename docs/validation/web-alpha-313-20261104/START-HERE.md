# ELab Web 0.207.0 alpha 1 — build 313

Download `ELab-Web-0.207.0-alpha.1-macos-arm64.zip` from this release and extract
it. Move **ELab Web.app** to Applications, then open it. The small launcher starts
the local engine and opens your default browser. No development tools, account or
internet connection are needed to use ELab. This download is for Apple Silicon.

This private alpha is ad-hoc signed and not notarized. macOS may block first
launch. If you choose to trust this specific download, follow Apple's
[per-app opening guidance](https://support.apple.com/en-us/102445): after the
blocked attempt, find ELab in **System Settings → Privacy & Security** and review
**Open Anyway**. Leave system-wide security protections enabled. This release is
tested on Apple Silicon macOS 27.2. macOS 13 is the configured build target,
not a verified minimum. Intel and clean-account installation are not qualified.

## Start without electronics experience

1. Choose **Learn circuits & shortcuts → Start LED activity**. Follow one step at
   a time. Changing resistance lets you compare the light's current; you do not
   need to calculate it first.
2. Try **Start capacitor activity** to watch voltage change over time, then
   **Start H-bridge activity** to explore a change in direction. A minus sign in
   a reading describes its reference direction.
3. Give useful experiments a name with **Save project**. **Projects** reopens
   them and contains recovery history and reversible Trash.

Select parts with Shift-click or a box drag on empty sheet space. Copy, paste,
duplicate and group movement keep internal connections together and undo as one
edit. Arrow keys move a selection; while placing a part, arrows move its preview
and Enter places it. Escape finishes. Keyboard skip links reach the schematic,
inspector and instruments. On macOS Safari, Option-Tab includes all controls.

**Circuit checks** explains a problem and offers controls to find the affected
part/terminal or begin a repair. It does not silently rewrite your circuit.

## Keep your work and upgrade

Edits autosave locally. Named saves belong to **Projects**. Use **Projects →
Back up library** for a portable backup of saved work, history, Trash and unsaved
workspaces with kept results. **Restore library backup** previews the contents
and restores copies while preserving existing projects. **Document → Download
.elab copy** exports one circuit, including retained recordings.

Existing web projects remain in `~/Library/Application Support/ELab/web`.
Keep that folder when upgrading. Earlier native releases and their source archive
remain available separately. Closing a browser tab leaves ELab running; use the
launcher's **Quit ELab**, or Command-Q with the launcher focused, to save and quit.
Close an older host normally before opening the new app against the same library.

## Capture the four-MOSFET bridge

Choose **Explore examples → Four-MOSFET H-bridge**, pause, then restart. The 100 Ω
load starts near −4.725 V. In **Simulate**, enter a **Step duration** of `21 ms`
and choose **Advance simulation** to see about +4.725 V. **Record** for `45 ms`
with a `1 ms` sample interval to capture both directions. Keep the recording,
save the project and reopen it to inspect the exact samples. These educational
models omit gate charge, switching loss and dead time.

## Build details and sharing a problem

The version in the status bar opens **About ELab**. **Create report preview**
collects build/browser details, counts and status. Inspect it, then choose Copy
or Download if you want to share it. It excludes project contents, values,
recordings, paths, URLs, credentials and raw error messages. Nothing is uploaded.

Download the release's checksum file and other assets into one directory:

```sh
shasum -a 256 -c SHA256SUMS
```

`release.json`, `PACKAGE.json` and `QUALIFICATION.json` identify the exact source,
archive hashes and observed checks. Automated checks do not establish native
VoiceOver or whole-app accessibility conformance.

Source: `2ca9bc8d6643a672336af21ca08a392084cab3b2` on `main`.
