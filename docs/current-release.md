# Current unsigned alpha

**ELab 0.203.0 /290** is available as a [GitHub prerelease](https://github.com/dshills/ELab/releases/tag/v0.203.0-alpha.1).

- [Download for macOS Apple Silicon](https://github.com/dshills/ELab/releases/download/v0.203.0-alpha.1/ELab-0.203.0-macos-arm64.zip).
- [Start guide](https://github.com/dshills/ELab/releases/download/v0.203.0-alpha.1/START-HERE.md) and [checksums](https://github.com/dshills/ELab/releases/download/v0.203.0-alpha.1/SHA256SUMS.txt).
- ZIP: 46,407,811 bytes; SHA-256 `2416f13dd0f494367f45c7991cc75bd1cb3e69acc1ae1c9971ea8165f9bc05cf`.
- Clean binary source: `f5c600d07e97a00ee1b967ce91c472235752841b` (the release tag).

This release simplifies Inspect, focuses each Simulate form with Advanced settings
and a fixed primary action, and consolidates measurement selection and guidance.
It retains 61 component kinds, 87 curated editable examples and all 229 shared
user/AI-agent commands. The only new command in this batch controls presentation
disclosures. No component, electrical model, solver limit or Vellum update is
included. See the [workspace guide](workspace-layout.md).

Extract and keep the entire folder together. It contains ELab.app, a matching
elab-agent, examples, notices, START-HERE and the [self-guided first-use kit](first-use-test-kit.md)
with a separate expected-result key. This is an unsigned alpha with an ad-hoc
integrity seal, without Developer ID signing or notarization. Follow the included
start guide for macOS opening instructions; do not disable security globally.

Generate standard local MCP configuration from the matching agent's final path:

```sh
./elab-agent --mcp-config
```

See [agent setup](agent-setup.md) for explicit desktop endpoints, guards and
lifecycle. ELab itself needs no provider account/key. Opening/closing Details or
changing workspace layout does not change the circuit or accepted settings.

[Qualification evidence](validation/first-use-20261029/README.md) records complete
local portable/ordinary/rendered race and vet checks, lint, exact copied native
GPU smoke, four runtime adapters, both MCP transports, all extracted example
graphs, 470 archive payload files with exact bytes/modes and strict seals, normal
cleanup of 102 external runtime processes, and 53 preserved prior package/SDK
identities. Uploaded assets were downloaded and matched before publication.
The final smoke-controller-only selector correction has explicit separate scope
and exact-package native qualification; earlier failed trials remain recorded.

Models remain educational approximations. Participant outcomes, historical native
OS queue/hardware checks 3/6/9/11, clean-account installation and public signing
remain separate. Software-route evidence is not physical-input acceptance.

For a checkout with local artifacts, the release folder is
`build/ELab-0.203.0-290-Alpha/` and its ZIP is under `build/releases/`.
Prior /285 and other qualified packages are preserved. The owner review session
opened from /288 uses the same normal UI code; later build numbers bind the final
test-controller/source metadata. That session and owner work are left open.
