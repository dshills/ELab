# Native workspace alpha 0.204.0, build 309

The October 31 prerelease runs on Apple Silicon with macOS 27.2 or later.
The tag resolves to clean source `d9b1181e57471a81d3bca83608ddc8a15af7f2e3`.
The published ZIP contains 470 files and is 46,797,797 bytes. Its SHA-256 is
`6d11a73b69625ace0507a13627bcd8328c1554a554a6df873f98005d800511d0`.
[GitHub release and assets](https://github.com/dshills/ELab/releases/tag/v0.204.0-alpha.1).

## What passed

- The full ordinary and offscreen desktop race suites and both vet runs passed on
  the final source. The rendered suite caught a one-pixel shift of the Pause
  button on an earlier candidate. The title column now takes the available space
  without moving the action buttons. The portable race and vet results carry
  forward from identical portable inputs. The inventory, Python tests, SDK
  preparation, PRD audit and lint passed.
- The packaged agent verified all 87 bundled examples. The inventory held 61 component kinds and 229 commands. Exact
  copied executables passed native smoke, the public suites on four adapters and
  both MCP transports. Extraction preserved archive contents, modes and seals.
  [Qualification record](qualification.json).
- The ZIP, start guide and checksum file downloaded from the GitHub release draft
  matched the originals byte for byte. The same assets were then published.
  From a fresh extraction directory on the existing account, the matching agent
  opened the bundled RC project in the desktop app. It was paused and clean,
  all three MCP tools were available and the app exited normally.
  [Installation record](installation.json).

## Open acceptance work

| Check | Observation |
| --- | --- |
| Save As and Undo (3) | The owner selected and replaced the full filename with ⌘A, then restored it with ⌘Z. [Physical input](physical-input.md). |
| Divider (6) | The owner reported a successful drag. Native automation resized the schematic and restored it. The physical slow and fast rates still need to be captured. [Physical input](physical-input.md). |
| Queued Return (9 and 11) | Ten native fields discarded a stale Return after the agent committed its edit and accepted a fresh correction. The instrumented run used build 307 source. [Event sequences](../native-workspace-307/queue/README.md). |
| First installation | The existing account passed the extracted-app rehearsal. Gatekeeper rejected the ad-hoc app on explicit assessment. A browser download with quarantine and Finder launch in a fresh standard account or VM remain to be observed. [Installation record](installation.json). |
| First-use study | The kit and moderator instructions are ready. No first-time hobbyist sessions have been observed. A common obstacle can be identified after those sessions. [Participant kit](../../first-use-test-kit.md) · [Maintainer key](../../first-use-maintainer-key.md). |
