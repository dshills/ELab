# Session colors for technical details

October 23, 2026. Continuing the UI review after the held /207 candidate. The
compact dark render has a light Technical details box with a blue heading.
The current download remains /205; /204 and /207 are preserved.

Two independent sources cause the mismatch. ELab includes Disclosure in its
light driver defaults but omits it from per-session surface assignment. Once
that surface is corrected, Vellum's private text-button heading still replaces
the controlled foreground with process-wide blue. That produces blue text on a
dark surface. Fixing the surface alone is insufficient.

ELab now applies its existing session surface/corner treatment to disclosures.
The pinned Vellum compatibility source uses the native action-button style for
the private heading, allowing it to inherit the managed foreground. Heading
identity, text escaping, callbacks, expansion ownership, key handling and child
retention are unchanged. No global palette or device preference is mutated.
Existing user and agent expansion still uses `diagnostics.details.set`.

## Focused checks

- The original ELab descriptor test fails on the dark default surface. The
  repaired descriptor test passes in both themes.
- A new Vellum manual-window test fails on process-wide blue, then passes with
  two independently colored sibling disclosures and their post-mount palette
  swap. Existing disclosure race tests pass, including pointer/keyboard,
  visibility, focus, stale callbacks and retained children.
- The full Vellum driver/module race suite and offscreen vet pass. The initial
  restricted run refused local Unix sockets in workbench tests; the rerun with
  automatically reviewed socket access passes. Both logs are retained.
- ELab's mounted consumer test fails with the old driver even after the surface
  repair: actual heading glyph pixels are still blue in both themes. With the
  new pin it passes painted surface/ink checks, physical mouse expansion and
  shared-agent collapse, preserving circuit, time, document and history.
- Actual 320×360 Vellum renders are inspected in Light and Dark using the managed
  test font. This is offscreen evidence, not a native macOS window capture.

The first consumer invocation mistakenly selected the portable root module;
offscreen-tagged historical scratch snippets under ignored build directories
failed to compile. It did not execute the intended desktop regression and is
not counted as a product failure or a passing test. The corrected invocation
runs in `desktop`, as the standard quality script does.

## Dependency identity and remaining scope

The derived SDK source is `211e3fe22aa2069721e77c607d0005f78fc0d9f0`, based on the
existing public `e58d40581fe21d1de53fd54cab1bcf6acf11c163` plus the full recorded
ELab compatibility patch. A clean exporter creates elab7; its lock, archive,
manifest, cache and four module replacements move together. The SDK verifier
passes all 8,425 members. Only `drivers/cogent/disclosure.go`, its existing test
file and `SDK_DOCUMENTATION.md` differ. Dependencies, notices and all six
bootstrap sources remain byte-identical. The old archive/cache is preserved;
the sibling Vellum project and public repository are untouched.

The native command ledger changes only `desktop/theme.go`'s source hash. Every
semantic site, alias and command/coverage set remains unchanged at 166 public
commands, 134 source routes, 404 sites and 76 production Go files.

The [final ELab source gate passes](../docs/validation/disclosure-theme-208/README.md):
lint/staticcheck, 67 Python tests, complete root/desktop race and vet, required
mounted-control race and offscreen vet. Desktop race takes 173.949 s and mounted
race 149.661 s; unchanged root results reuse valid test cache entries.
Fresh native validation of this
changed SDK and the preceding /207 guidance/file-menu changes remains pending.
No fresh launch prompt is queued: the owner's standing authorization is saved
in `AGENTS.md`, and the existing owned session remains available for applicable
unchanged-code checks. No new component, model, command or file format is added.
