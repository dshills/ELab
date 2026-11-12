# Current web alpha

**ELab Web 0.208.0 alpha 1, build 315**, for Apple Silicon macOS.
[Download the ZIP](https://github.com/dshills/ELab/releases/download/v0.208.0-alpha.1/ELab-Web-0.208.0-alpha.1-macos-arm64.zip) · [Start guide](https://github.com/dshills/ELab/releases/download/v0.208.0-alpha.1/START-HERE.md) · [Checksums](https://github.com/dshills/ELab/releases/download/v0.208.0-alpha.1/SHA256SUMS).

Build 315 adds multi-sheet schematics with named ports, linked sheets with
independent wiring and settings, and search across sheets. The simulator supports
up to 128 parts, 200 electrical nodes including ground, and eight FETs. Open
**Two MOSFET H-bridges on one supply** to see two loads reverse independently.
[Sheet guide](web-workbench.md#draw-across-sheets) · [Dual H-bridge walkthrough](hbridge.md#two-mosfet-h-bridges-on-one-supply) · [Circuit capacity](circuit-capacity.md).

Extract **ELab Web.app** and follow the start guide for first launch. The
launcher starts the engine and opens your browser. Before upgrading, use
**Projects → Back up library** to make a portable backup, then close the older
launcher normally. The app stores web projects in
`~/Library/Application Support/ELab/web`. This build reads earlier `.elab`
files. Projects with child sheets require version 6 support.

The release tag points to `6137058afdddc2df19da07515fd021ce12ab56c9` on `main`.
The ZIP is 11,971,953 bytes with SHA-256
`6a0ae6911cca5cbdbd783580f4826270cf11c8a9b85073daa884aa8194a73821`.

The macOS quality plan and archive runtime checks passed on Apple Silicon
macOS 27.2. Chromium, Firefox, and WebKit passed 58 browser checks with two
planned skips. The downloaded app passed all 12 large-circuit browser checks.
The dual H-bridge recording covered all four direction combinations and
retained all 53 samples across six traces through save, reopen, and export.
All five payload checksums passed after publication. All five GitHub Actions
jobs passed, including the hosted macOS engine, web, and package checks.
[Publication-time qualification receipt](https://github.com/dshills/ELab/releases/download/v0.208.0-alpha.1/QUALIFICATION.json) · [Completed hosted CI run](https://github.com/dshills/ELab/actions/runs/43235023151).

On Apple Silicon macOS 27.2, Finder copied the exact published app and strict
signature and file checks passed. Gatekeeper rejected the ad-hoc-signed bundle
with assessments enabled. First launch of the published bundle remains
unverified. The package targets macOS 13 and has no notarization ticket.

The build 314 → 315 upgrade passed with re-signed test launchers and an isolated
library. Build 315 reopened both the named save and the separate autosave from
build 314. The two-sheet dual H-bridge saved and reopened with its shared VDD
port and all 53 recorded samples across six traces. Recovery restored the
renamed child sheet and its 180 Ω load as a separate project. The original
150 Ω save remained byte-for-byte intact. [Desktop validation report](validation/web-alpha-315-20261112/README.md)

[Release notes](https://github.com/dshills/ELab/releases/tag/v0.208.0-alpha.1) ·
[Previous web alpha, build 314](https://github.com/dshills/ELab/releases/tag/v0.207.0-alpha.2) ·
[Build 314 evidence](validation/web-alpha-314-20261108/README.md).
