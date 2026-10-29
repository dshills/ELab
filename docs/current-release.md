# Current local unsigned release

**ELab 0.201.0 /285**, clean binary source `70e113d5f2767e12963fe47337f35a7200f10a9e`.
This build includes the inspector usability repairs, Scope/Spectrum tools and
standard local MCP agent setup. It is a local macOS Apple Silicon build with an
ad-hoc integrity signature; it has no Developer ID signature or notarization.
No GitHub download asset was published by this qualification.

From a checkout containing the built artifacts:

- App/agent/examples: `build/ELab-0.201.0-285-Agents/`.
- Archive: `build/releases/ELab-0.201.0-macos-arm64.zip` (46,309,442 bytes).
- SHA-256: `f61ef990202f395955d92aef9489e8e9ad49ce968574d0fc76a136f3c4bcbc65`.

Open `ELab.app` in that directory. The adjacent `elab-agent` matches the app's
source, and `Examples` contains all 87 curated editable projects. Move the whole
extracted folder to keep the supplied guides/notices and agent together.

For standard agent setup, run the matching executable from its final location:

```sh
./build/ELab-0.201.0-285-Agents/elab-agent --mcp-config
```

Use the printed configuration in your MCP client. It starts an owned headless
circuit; see the [agent setup guide](agent-setup.md) for explicit desktop connection,
guards and lifecycle. All 226 existing commands use the same underlying service;
the raw JSON adapters remain available.

[Agent/setup qualification](validation/agent-setup-20261029/README.md) records
protocol/electrical tests, copied native runtime and four existing adapters,
all examples, exact archive validation and preserved older artifacts.
[Usability qualification](validation/usability-release-20261029/README.md) records
the complete preceding UI/rendered gates and hobbyist workflows; final desktop
source and used dependency versions are byte-identical to that release.

Participant outcomes, four historical native queue/hardware checks, clean-account
installation and public signing remain separate. This is engineering-qualified
unsigned software with those recorded limits. No participant acceptance or native
OS-input result was inferred from software-route tests.
