# Web-root migration, 2026-11-03

The owner selected the browser interface as the primary application. The
complete pre-migration source is preserved in the published Git tag
`archive/native-ui-2026-11-03` at `a0379ce04ea34d5ad81a71632a2bf8b5a24c25d9`.
Existing native/web release artifacts and project libraries are preserved.

**Final result:** the web-root source at
`9335f7f16aee48524f97ec3f7061c376113ecfc8` passes all five jobs in
[hosted qualification 41703613414](https://github.com/dshills/ELab/actions/runs/41703613414):
Linux, macOS, Chromium, Firefox and WebKit. The final documentation commit
records these receipts without changing application or test sources.

See the [CI summary](hosted-ci.json), [Linux gates](hosted-linux-final.json),
[macOS gates](hosted-macos-final.json) and
[macOS package/runtime check](hosted-macos-package-final.json).
Both platform profiles pass lint, all 62 Python and 28 frontend tests, the
complete Go race suite and vet. macOS also passes clean-source packaging,
signature/archive validation, copied-runtime execution, normal shutdown and
restart recovery. All five browser journeys pass in each of the three engines.

## Reviewed scope

- Moved the complete `exp-web` host, frontend and test sources to the root.
  The small web-host launcher is now `launcher/macos/main.swift`.
- Removed the archived Vellum UI/SDK, old design experiment, native audit/probes,
  native packaging and native-only test/release controllers from the active tree.
  Shared headless clients, agent/MCP tests and numerical/reference demos remain.
- `workbench/` production sources, `transport/`, module pins and dependency notices are unchanged.
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

The first hosted migration run passed all three browser engines. Its newly
enabled Linux lint gate found two existing Darwin-only declarations in shared
files. The CUPS socket field and Quartz path tokenizer now live behind their
platform build constraints; the unsupported print backend has no unused field.
The platform behavior and tokenizer expression are unchanged. Internal review,
macOS/Linux lint and targeted print/PDF race tests pass for this correction.

## Preflight evidence

The root host/agent/conformance build, all 28 frontend tests, 62 retained Python
tests, lint and workflow lint pass. All current guide links resolve. The
Vellum-free package preflight passes OS archive extraction, exact file/mode and
signature checks, copied execution without Go/Node in PATH, normal Quit and
recovery of an acknowledged 22 kΩ edit. That preflight used an uncommitted tree
and is not a clean-source release qualification.

## Committed-source qualification

The complete local macOS profile passes at
`44d79360b86e04fec7bb70bbf96c2b57243acfe9`, with unchanged source inputs:

- Lint, 62 Python tests and 28 frontend logic tests.
- The complete Go race suite (521.946 seconds) and vet.
- A clean-source, ad-hoc-signed package, OS ZIP extraction, exact bytes/modes
  for all 12 files, copied execution without Go/Node, normal Quit and restored
  acknowledged 22 kΩ edit after restart.

See [local quality receipt](local-quality.json) and
[local package/runtime receipt](local-package.json). The package is a temporary
qualification build, not a new distributed release. Its scope does not include
GUI launcher interaction, notarization, clean-account or minimum-OS acceptance.
The disposable development host was saved and closed normally; the pre-existing
default-library host and library were not changed.

## Hosted audio fixture correction

Run `41694653825` passed every browser job and the complete workbench,
conformance, agent and transport race suites on Linux and macOS. Both platforms
failed the browser WAV test with a failed 555 audio render. The old failure
message printed an error pointer rather than its diagnostic, so those receipts
do not establish the precise render failure cause.

The transfer test now checks a powered active buzzer under race instrumentation
and retains the original 555 WAV assertions through the existing
`RunAtProductionSpeed` numerical-reference helper. Both cases also require
nonzero PCM samples, and failures now print the audio diagnostic. No production
solver limit, timeout or audio behavior changed. The active-buzzer case passes
50 one-scheduler-CPU race repeats; both cases together pass 10 repeats, including
proof that the original 555 assertions ran in each ordinary-speed child.

Run `41698091612` then passed Linux and all three browser jobs. macOS passed
the revised browser tests and failed only
`TestFlasherLayoutPreservesTheOriginalCircuitAndTimedResponse`: its instrumented
sample reached the 100 ms CPU budget (100.033 ms measured). This is a numerical
reference comparing 601 samples of the old and new layouts, not a concurrency
test. It now uses the same production-speed helper, preserving every electrical,
voltage/current/charge and trace comparison. The neighboring layout geometry
checks and controlled-budget/concurrency tests retain race instrumentation.
Internal review confirms this correction changes only the test entry point;
production budgets and equations are unchanged.

All flasher checks pass 10 one-scheduler-CPU race-harness repeats (38.985 s),
including explicit proof that both numerical reference tests ran their original
assertions in ordinary-speed children. Lint passes. The complete web-host race
package also passed after the audio correction (12.721 s).
The final committed source is `9335f7f16aee48524f97ec3f7061c376113ecfc8`;
its [clean-source package/runtime check](local-package-final.json) passes.

No merge to main or public release is made here.
