# Release the browser app

The default downloads are standalone ZIPs for Linux x64, macOS (Apple Silicon and
Intel), and Windows x64. Each includes the local ELab host, its embedded browser
interface, and a short start guide. The [run guide](run-web.md) explains how to use
them. Existing published alphas are preserved.

Dispatch **Qualify web alpha** from `main` with the next version and numeric
build. The default run stages and tests the downloads. Set **publish** only
when you want to create a GitHub prerelease. The workflow:

1. Runs the portable quality gates on Linux, builds all four standalone hosts
   with Go, and verifies each ZIP's exact files and hashes.
2. Tests the exact extracted Linux host in Chromium, Firefox and WebKit,
   including the multi-sheet and large-circuit tests.
3. Starts the extracted host on each target. It verifies the version and source
   revision, runs without Go or Node, downloads a schematic PDF and PNG, saves an
   edit, backs up and restores the library, and verifies recovery after restart.
4. When publication is requested, rechecks that all four runtime receipts match
   the staged ZIPs and that GitHub `main` still matches the qualified source.
   It creates a draft, downloads and compares every asset, reruns the Linux host
   from the downloaded ZIP, and then publishes. Any failure leaves an existing
   draft unpublished. The pipeline never replaces an existing tag.

The optional **macos_launcher** input builds and checks an Apple Silicon
**ELab Web.app** ZIP. It is a separate workflow artifact and is not a prerequisite
for the browser downloads. Local Developer ID signing, notarization and
Gatekeeper verification are covered by the [Mac distribution guide](macos-distribution.md).
The standalone downloads are unsigned for distribution; the release does not
assert clean account installation or human usability acceptance.

For a local rehearsal on a supported target, install the locked `browser/`
dependencies and the three Playwright browsers. From a clean checkout:

```sh
python3 -B scripts/release_web.py stage --candidate \
  --version 0.209.0-alpha.1 --build 316 --output build/release-rehearsal
```

Staging checks the current machine and generates ZIPs for all four targets.
`--quality` can reuse a passing portable receipt with the same commit, complete
gate plan and exact source hashes. The output directory must be new.
`--candidate` allows a clean development branch and forbids publication.

To finish a release outside Actions, run `stage` without `--candidate` on clean
`main`. Use `verify-runtime --directory <stage> --target <target> --report <file>`
on each remaining target with the same source and stage files. Place those
reports under the stage's `qualification/` as `runtime-<target>.json`. Run
`publish --directory <stage>` to create and verify a draft; add `--make-public`
to publish after those checks.

Runtime launch replies contain private control credentials. They are stored in
disposable directories. Workflow artifacts contain only the distribution
files and selected reports and logs.
