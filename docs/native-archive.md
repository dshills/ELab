# Native interface archive

On November 3, 2026 the owner chose the web interface as ELab's primary UI.
The complete pre-migration repository is preserved in the annotated Git tag
[`archive/native-ui-2026-11-03`](https://github.com/dshills/ELab/tree/archive/native-ui-2026-11-03).
The tag was pushed before native-only tracked files were removed.

The archive contains `desktop/`, the Vellum SDK/packager pins, native UI audit and
frame-probe tools, old design experiments, native packaging/release scripts and
the original documentation. Restore the whole snapshot when reproducing those
builds; copying `desktop/` alone omits its pinned SDK and tooling.

```sh
git fetch origin tag archive/native-ui-2026-11-03
git worktree add --detach ../ELab-native archive/native-ui-2026-11-03
```

Then use that worktree's README and scripts. The command creates a separate
checkout and leaves the current web checkout intact. Existing native release
packages and their evidence were not rebuilt, overwritten or removed.

The shared Go `workbench`, solver, agent/MCP adapters, transport, project format
and useful platform render/audio/print services remain maintained. The small
Swift macOS launcher also remains: it starts the web host rather than drawing a
native circuit interface. Browser dependencies are test-only; Node and Vellum
are not runtime requirements.

`docs/validation/` and `research/` preserve historical evidence at its original
paths and bytes. Native UI instructions and reproduction helpers there apply to
the archived revision, not the current tree. Native-only root scripts/tools and
their unit tests were retired with the interface. Shared numerical, agent,
project/recovery and browser coverage remains in the active quality plan.

No project schema or library migration is required. The web library stays at
its existing platform configuration path plus `ELab/web`. If an older CLI host
owns that library, close it normally before starting the new host; keep the data
folder. Ignored local caches left by native source removal were preserved under
`build/retired-native-local/` rather than deleted.
