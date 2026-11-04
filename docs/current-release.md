# Current web alpha

**ELab Web 0.207.0 alpha 1, build 313**, for Apple Silicon macOS.
[Download the ZIP](https://github.com/dshills/ELab/releases/download/v0.207.0-alpha.1/ELab-Web-0.207.0-alpha.1-macos-arm64.zip) · [Start guide](https://github.com/dshills/ELab/releases/download/v0.207.0-alpha.1/START-HERE.md) · [Checksums](https://github.com/dshills/ELab/releases/download/v0.207.0-alpha.1/SHA256SUMS).

Extract **ELab Web.app** and open it. The launcher starts the local engine and
opens your default browser; Go and Node are not required. Choose **Learn circuits
& shortcuts** for guided LED, capacitor and four-MOSFET H-bridge activities.
Connected group editing, actionable circuit checks, improved keyboard access,
library backups and private diagnostic reports are included. Existing web
projects remain in `~/Library/Application Support/ELab/web`.

The ZIP is 11,856,229 bytes. SHA-256:
`04090425038f8ef53299bb30fbc25d5ce3d9dc082fb8daea250f538e73197342`.
The public tag identifies clean `main` source `2ca9bc8d6643a672336af21ca08a392084cab3b2`.

The full local macOS gates passed: lint, 75 Python tests, 32 frontend checks,
complete Go race tests, vet and package checks. Browser qualification passed 46
journeys with two planned skips and no unexpected or flaky results. The exact
GitHub-downloaded app passed LED editing, group undo, H-bridge reversal, exact
preservation of all 50 recorded rows through save/reopen and export/import,
private diagnostic reports and library backup. All six downloaded assets match
their qualified bytes and all five payload checksums pass.
[Qualification and publication evidence](validation/web-alpha-313-20261104/README.md).

At publication, all five [exact-source hosted CI jobs](https://github.com/dshills/ELab/actions/runs/41878077636) were queued;
these pending results are explicitly separate from the passing local gates.

This private alpha is **ad-hoc signed and not notarized**. Gatekeeper can block
first launch; follow the start guide's per-app opening instructions. The exact
ZIP is tested on Apple Silicon macOS 27.2. macOS 13 is the configured build target,
not a verified minimum. Intel, clean-account/Finder installation, native
VoiceOver and whole-app accessibility conformance are not qualified.

[Release notes](https://github.com/dshills/ELab/releases/tag/v0.207.0-alpha.1) ·
[Previous web alpha](https://github.com/dshills/ELab/releases/tag/v0.206.0-alpha.1) ·
[Preserved native H-bridge alpha](https://github.com/dshills/ELab/releases/tag/v0.205.0-alpha.1).
