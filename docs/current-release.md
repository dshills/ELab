# Current web alpha

**ELab Web 0.207.0 alpha 2, build 314**, for Apple Silicon macOS.
[Download the ZIP](https://github.com/dshills/ELab/releases/download/v0.207.0-alpha.2/ELab-Web-0.207.0-alpha.2-macos-arm64.zip) · [Start guide](https://github.com/dshills/ELab/releases/download/v0.207.0-alpha.2/START-HERE.md) · [Checksums](https://github.com/dshills/ELab/releases/download/v0.207.0-alpha.2/SHA256SUMS).

This update includes the WebKit reload fix: old-page state reads are cancelled
before navigation, polling resumes after a reload or Back/Forward restore, and
cancelled navigation keeps the current page updating. Editing commands and the
solver's existing work limits retain their safeguards.

Extract **ELab Web.app** and open it. The launcher starts the local engine and
opens your default browser; Go and Node are not required. Guided LED, capacitor
and four-MOSFET H-bridge activities, connected group editing, diagnostics, library
backups and private support reports remain included. Existing web projects stay
in `~/Library/Application Support/ELab/web`.

Build 314 predates the large-circuit changes on `main`. Multi-sheet projects,
linked sheets, the higher simulation limits and the dual H-bridge example are
available from the current source checkout. [Run from source](web-workbench.md#run)
· [Draw across sheets](web-workbench.md#draw-across-sheets).

The ZIP is 11,856,976 bytes. SHA-256:
`7b02492bd646390deee41e40cf08560786b63185ed686c452be42227d3c3aa6d`.
The public tag identifies clean `main` source `6c0d66e3cafcedb0228e5db978fb6c4f6089d5b8`.

Fresh macOS gates passed on that source: lint, 75 Python tests, 36 frontend checks,
complete Go race tests, vet and package/runtime checks. All 46 browser journeys
passed with two planned skips and no unexpected or flaky results. The exact
GitHub-downloaded shipping app passed six in-flight reload checks across Chromium
and WebKit, plus LED editing, group undo, H-bridge reversal and exact preservation
of all 50 recorded rows through save/reopen and export/import. All six downloaded
assets match their qualified bytes and all five payload checksums pass.
[Qualification and publication evidence](validation/web-alpha-314-20261108/README.md).

[Hosted CI](https://github.com/dshills/ELab/actions/runs/41888150836) passed all five jobs on
`e5136106d1d49cb13e979f0ad9337bccd012f21a`. Its application, test and workflow input hashes
exactly match this release commit; the intervening commit contains documentation
only. Fresh local qualification and downloaded-binary checks use the release
commit itself.

This private alpha is **ad-hoc signed and not notarized**. Gatekeeper can block
first launch; follow the start guide's per-app opening instructions. The exact
ZIP is tested on Apple Silicon macOS 27.2. macOS 13 is the configured build target,
not a verified minimum. Intel, clean-account/Finder installation, native
file-picker interaction, VoiceOver and whole-app accessibility conformance remain
unqualified by these automated checks.

[Release notes](https://github.com/dshills/ELab/releases/tag/v0.207.0-alpha.2) ·
[Previous web alpha](https://github.com/dshills/ELab/releases/tag/v0.207.0-alpha.1) ·
[Preserved native H-bridge alpha](https://github.com/dshills/ELab/releases/tag/v0.205.0-alpha.1).
