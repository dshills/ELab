# Application icon

ELab has an original full-color icon: a circuit-shaped E, resistor and amber LED
on a blue instrument tile. The editable [SVG](../assets/app-icon.svg) and
1024×1024 [PNG](../assets/app-icon.png) live in `assets/`.

The latest Vellum source provides this feature through its **packager**. ELab
pins the unmodified packaging tools at
[`4cf0176`](https://github.com/dshills/vellum/commit/4cf01763c1fd5501648ea19b4bb406fe62446319),
whose release metadata says v1.0.0. The icon option was introduced in
[`3fadc8f`](https://github.com/dshills/vellum/commit/3fadc8fcd3db7aa4b1adf6739a00cd4f56ca393d).
GitHub's published release list still reports alpha.3; the selected tool is from
the newer public main source. The first icon-only /273 update kept the
independent elab9 UI runtime. The
subsequent [runtime upgrade](vellum-refresh.md) now selects the same public v1
source with ELab compatibility fixes. The icon and shared commands are retained.

## Build an app with its icon

After compiling with `sh scripts/build.sh`, run:

```sh
python3 -B scripts/package_app.py --binary build/elab \
  --output build/ELab-with-icon/ELab.app --version 0.196.0 --build-version 273
```

The wrapper verifies every pinned packaging file before executing the upstream
tool in an isolated Python interpreter. It supplies `assets/app-icon.png` by
default and requests an ad-hoc signature. `--icon` accepts another 1024×1024 PNG
or a valid ICNS. Existing output paths are refused. An optional `--notice-bundle`
is forwarded unchanged; public distribution/notarization is separate.

Vellum creates the standard 16, 32, 128, 256 and 512 point representations at
1×/2×, installs `Contents/Resources/AppIcon.icns`, sets `CFBundleIconFile`, and
records source/output icon hashes in `APP_MANIFEST.json`. It verifies the icon
and plist before and after signing and publication.

Launch the packaged `.app` to use its icon. Standalone binaries and `go run`
have no bundle icon. Finder/Dock cache refresh is OS-controlled; a new bundle
path is used rather than modifying the signed /272 app that the owner is trying.
The packager's `sdk_release` field describes its packaging metadata; the runtime
is independently identified by `third_party/vellum/sdk.lock.json`. The /273 icon
evidence carries its original elab9 runtime; the later runtime update is separate.

## Verification

All 77 ELab Python tests and all seven unmodified upstream icon tests pass.
These include PNG/ICNS decoding, source mutation, invalid artwork, hash changes,
and icon/plist tampering around signing. Packaging-file integrity regressions
cover changed helpers, extra imports and symlinks. Runtime Go/module/native
sources for the first icon-only /273 update were unchanged from /272; its prior
qualification applies to those exact inputs. Later runtime checks are recorded
in the [Vellum refresh](validation/vellum-v1-runtime/README.md). The exact copied
/273 app passes native smoke and all four public agent adapters;
all 102 external processes exit normally. Ten representations and macOS file-icon
lookup resolve the artwork. [Bundle checks and limits](validation/application-icon/README.md).
