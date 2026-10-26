# Application icon qualification

ELab **0.196.0 /273** packages the existing drawing-improvement runtime with an
original circuit-E/amber-LED icon. [Open app](../../../build/ELab-0.196.0-273-Icon/ELab.app).
[Summary](summary.json), [package identity](package.json),
[workflow/build guide](../../application-icon.md).

## Dependency and artwork

The icon feature is in Vellum's packager. Twelve unmodified files are pinned
from latest public main `4cf0176`, whose release metadata says v1.0.0. GitHub's
published release list still has the older alpha.3 prerelease. No runtime SDK
upgrade is implied: compiled Vellum remains the independently pinned elab9
runtime with ELab's existing input compatibility fixes. The sibling source and
verified SDK caches were read/preserved, not changed.
[Tool pin](packager.lock.json).

The original SVG and 1024×1024 RGBA PNG are reviewed at small/large sizes.
Vellum generates all ten standard/Retina ICNS representations, which decode to
their expected sizes. The signed manifest records both PNG and ICNS hashes;
CFBundleIconFile points to the embedded AppIcon.icns. macOS NSWorkspace's actual
file-icon lookup resolves the inspected ELab artwork. That is resource lookup,
not a Finder/Dock screenshot or proof of refresh for a previously cached icon.
[Size review](size-review.png), [macOS file-icon result](file-icon.png),
[representations](representations.json), [artwork hashes](artwork.json).

## Source and runtime checks

All **77 ELab Python tests** and **seven unmodified upstream icon tests** pass.
Source mutation, corrupt/invalid artwork, symlinks, unpinned imports and post-sign
icon/plist tampering have explicit regressions. The new wrapper verifies every
pinned tool before isolated execution. [ELab tests](python-tests.txt),
[upstream tests](upstream-icon-tests.txt).

Packaging source is `62b18c85`; the executable's compiled source remains
`5fd90873`. The /273 executable is **byte-identical to /272 after signatures are
removed from disposable copies**, SHA-256
`63cb9bcf03f1044dc42e21dc83aec36a188f00fe9120ab3dffb23e10b8ce3bcb`.
The matching agent is unchanged. All Go/module/Objective-C source inputs are
unchanged, so previous complete source/electrical qualification is carried for
those exact inputs. No command, component, model or solver-limit change.
The upstream manifest's sdk_release identifies the packager metadata, not the
independent compiled runtime. [Before inputs](before.json),
[ad-hoc signature](signature.txt).

The exact copied /273 app passes full native smoke and all four public agent
adapters with development-file and outbound-IP denial. Fixture-only policy also
denies shared framework geometry writes. All **102 external conformance
processes exit 0 normally**. Copied bundle, source bundle, agent and controller
remain exact. No security settings are changed. [Runtime receipt](runtime-report.json.gz),
[public adapters](public-report.json.gz), [process exits](process-exits.json.gz).

All 126 adjacent example files match /272; prior 62 paused/clean exact-graph
headless opens are carried for the identical matching agent rather than claimed
as a fresh rerun. The thirteen earlier release/SDK archive identities and /272
bundle executable remain exact. No owner project/session command was sent.
Prior native-input checks 3/6/9/11, participant/clean-account acceptance and
public signing/notarization remain separate. This is a development candidate;
the previously qualified downloadable release is not promoted or replaced.
