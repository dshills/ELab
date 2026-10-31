# Current unsigned alpha

**ELab 0.205.0 alpha 1, build 311**, runs on Apple Silicon with macOS 27.2 or later.
[Download the ZIP](https://github.com/dshills/ELab/releases/download/v0.205.0-alpha.1/ELab-0.205.0-macos-arm64.zip) · [Start guide](https://github.com/dshills/ELab/releases/download/v0.205.0-alpha.1/START-HERE.md) · [Checksums](https://github.com/dshills/ELab/releases/download/v0.205.0-alpha.1/SHA256SUMS.txt).

Open **Examples → Power & relays → Four-MOSFET H-bridge**. Pause, restart and step across 20 ms to reverse the 100 Ω load from about −4.725 V to +4.725 V. Record 45 ms to capture the reversal and return. The example is also included as `Examples/discrete-hbridge.elab`. [H-bridge guide](hbridge.md#four-mosfet-bridge).

Extract the whole folder to keep the app, matching agent, 88 editable examples and schematics, notices and first-use kit together. To connect a local MCP client, run `./elab-agent --mcp-config` from its final location. [Agent setup](agent-setup.md).

The ZIP is 46,822,732 bytes. Its SHA-256 is `b238ec256eae30778e435e0c7af56141d4f7bec1a6a0e1c1453b9bc18696d973`. The downloaded native app reopened all 50 recorded H-bridge samples after a save. This alpha has an ad-hoc integrity signature. Gatekeeper can block the first launch. Follow the per-app instructions in the start guide. [Qualification and installation results](validation/hbridge-alpha-311/README.md).

[GitHub release notes](https://github.com/dshills/ELab/releases/tag/v0.205.0-alpha.1) · [Previous native workspace alpha](https://github.com/dshills/ELab/releases/tag/v0.204.0-alpha.1).

All local target gates passed. Both hosted jobs failed in the portable race suite. All 26 Linux failures and 16 of the 17 macOS failures also appeared in the pre-H-bridge baseline. The additional hosted macOS failure was an MCP stale-document guard assertion. [Hosted checks and comparison](validation/hbridge-alpha-311/README.md#hosted-checks).
