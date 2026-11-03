# Web-root migration, 2026-11-03

The owner selected the browser interface as the primary application. The
complete pre-migration source is preserved in the published Git tag
`archive/native-ui-2026-11-03` at `a0379ce04ea34d5ad81a71632a2bf8b5a24c25d9`.
Existing native/web release artifacts and project libraries are preserved.

## Reviewed scope

- Moved the complete `exp-web` host, frontend and test sources to the root.
  The small web-host launcher is now `launcher/macos/main.swift`.
- Removed the archived Vellum UI/SDK, old design experiment, native audit/probes,
  native packaging and native-only test/release controllers from the active tree.
  Shared headless clients, agent/MCP tests and numerical/reference demos remain.
- `workbench/`, `transport/`, module pins and dependency notices are unchanged.
  No solver equations, numerical limits, project schema, library path, browser
  storage prefix or session/control protocol changed.
- Updated builds, browser harness paths, CI and current guides. Native UI evidence
  remains historical at its original paths; reproduction uses the archive tag.
- Web packaging uses macOS `sips`/`iconutil`, without Vellum. The macOS quality
  profile now checks a real package/extracted ZIP and its copied runtime instead
  of compiling the retired UI. All shared Go race/vet gates remain.
- Quality input signatures include browser dependency locks, the app icon and
  retained C/Objective-C platform sources as well as Go/frontend/Swift sources.

Internal review checked root/static embedding versus development serving,
package source hashes and launcher paths, headless adapter compatibility,
archive restoration, retained dependencies/data paths, test-only library
isolation, ZIP bytes/modes/signatures, and credential-free quality artifacts.
No external review provider was used.

## Preflight evidence

The root host/agent/conformance build, all 28 frontend tests, 62 retained Python
tests, lint and workflow lint pass. All current guide links resolve. The
Vellum-free package preflight passes OS archive extraction, exact file/mode and
signature checks, copied execution without Go/Node in PATH, normal Quit and
recovery of an acknowledged 22 kΩ edit. That preflight used an uncommitted tree
and is not a clean-source release qualification.

Full committed-source quality and hosted browser/platform qualification follow
this implementation commit. No merge to main or public release is made here.
