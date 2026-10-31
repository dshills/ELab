# Current unsigned alpha

**ELab 0.204.0 /309** ships the redesigned native workspace on Apple Silicon.
The embedded application and matching agent require **macOS 27.2 or later**.
[Download the alpha ZIP](https://github.com/dshills/ELab/releases/download/v0.204.0-alpha.1/ELab-0.204.0-macos-arm64.zip) · [Release notes](https://github.com/dshills/ELab/releases/tag/v0.204.0-alpha.1) · [Start guide](https://github.com/dshills/ELab/releases/download/v0.204.0-alpha.1/START-HERE.md) · [Checksums](https://github.com/dshills/ELab/releases/download/v0.204.0-alpha.1/SHA256SUMS.txt).

Use the floating **Wire** tool to connect terminals. The RC example shows Auto,
Charge and Discharge above the Scope. The schematic badge, inspector and trace
use the same simulation sample. The toolbar includes local file actions and
retains a confirmation before closing an unsaved project.

Extract the entire folder. It contains the native app and matching agent, 87
editable examples, notices and the [first-use kit](first-use-test-kit.md). For
a local MCP client, run `./elab-agent --mcp-config` from its final location.
See [agent setup](agent-setup.md) to connect to a desktop session.

The ZIP contains 470 files and is 46,797,797 bytes. SHA-256:
`6d11a73b69625ace0507a13627bcd8328c1554a554a6df873f98005d800511d0`.
The release app, agent and conformance controller were built from clean source
`d9b1181e57471a81d3bca83608ddc8a15af7f2e3`.
[Build and qualification evidence](validation/native-workspace-309/README.md).

This alpha has an ad-hoc integrity seal. Gatekeeper rejected it during assessment
on macOS 27.2. Consult the per-app opening instructions in the start guide.
Clean-account installation, first-time hobbyist observations and measured physical divider-drag rates remain open.
[Physical keyboard results](validation/native-workspace-309/physical-input.md). Native queued-Return ordering passed across ten fields in an
[instrumented app](validation/native-workspace-307/queue/README.md). The [previous 0.203.0 alpha](https://github.com/dshills/ELab/releases/tag/v0.203.0-alpha.1) remains available.
