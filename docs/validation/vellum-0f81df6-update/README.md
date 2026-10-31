# Vellum update on exp-ui · October 31, 2026

The runtime SDK and macOS packager now use public main at
[`0f81df6`](https://github.com/dshills/vellum/commit/0f81df6e6a4f9698fe3358710132a14987ad1f37).
`v1.0.0` remains the latest source tag. ELab’s [runtime lock](../../../third_party/vellum/sdk.lock.json)
identifies the private compatibility build; the [packager lock](../../../third_party/vellum/packager.lock.json)
identifies 14 unmodified upstream files. See [discovery](discovery.json) and
[internal review](internal-review.md) for the selection and conflict resolution.

Applying the stored patch to the upstream Git tree reproduces the exact derived
tree (`cc4bc24e1cc5db60c6e478733b3a5d96a124ed78`). All 9,056 SDK files match the
archive manifest; the packaged desktop executable resolves all four module
replacements into the new cache. The command inventory comparison changed only
the hash of the toolbar styling file. The experimental app is built separately
from the earlier private release.

The old SDK passed the toolbar height test. With the new SDK, Examples, Sweeps
and Fit rendered at 27 pixels instead of the required minimum of 28. ELab now
gives header text actions the same minimum content height as icon actions. The
existing toolbar test passes in all four themes without changing its assertions.

| Verification | Outcome |
| --- | --- |
| SDK extraction, desktop/CLI build, package creation and strict ad-hoc signature | Pass. The bundle's macOS minimum is 27.0, matching both its executable and the previous experimental executable. |
| Ordinary ELab desktop tests; SDK/packager Python integration tests | Pass. |
| Focused mounted ELab controls | All nine top-level tests pass, including toolbar geometry, popup painting and callbacks across agent edits. |
| Full offscreen ELab desktop tests | 574 top-level tests pass; the drawer-edge test fails in all four themes. It also fails on the original branch source with the previous SDK. |
| Native packaged smoke (before the toolbar correction) | Stops at the missing or disabled `scope-trace-focus` selector. The preserved /302 app stops at the same stage. |
| Vellum checks | Stable API compatibility and the portable race suite pass. The rebased button callback tests and native menu highlight tests pass. The native media fixture rejects its silent WAV on both SDKs, including on the new SDK outside the sandbox. ELab does not use that media control. |

Rebuild with `sh scripts/build.sh`. The [local experimental app](../../../build/ELab-0.204.0-303-Vellum/ELab.app)
uses the toolbar correction; no release was promoted or uploaded. See
[verification data](verification.json) for artifact identities, commands and test scope.
