# Pinned Vellum SDK from public source

ELab now consumes [dshills/vellum](https://github.com/dshills/vellum), pinned to public commit `e58d40581fe21d1de53fd54cab1bcf6acf11c163` (October 21, 2026). There are no upstream release tags at the time of this update. The public framework still documents prepared Cogent/typesetting sources and explicit local module replacements as its supported native consumer setup; replacing only the module URL is insufficient.

The public snapshot omits APIs and input fixes already used by ELab. The checked-in [compatibility patch](patches/elab-public-e58d405-compat.patch) preserves control/text ownership keys, numeric settlement, held button/slider context, legacy text acknowledgments and partial-scroll accessibility publication. It includes their existing regression tests. These changes are applied to public source in an isolated local commit, `96e22bf25809148b9a28169baf1b24cba72af475`; they are not claimed to be upstream. The one merge overlap retains both public numeric pointer-focus behavior and ELab's field-settlement epoch. No electronics feature or application command changes.

`vellum-e58d405-elab1.tar.gz` is the resulting prepared SDK. Lock schema 2 records the public repository/base commit, compatibility patch checksum, derived source commit, complete archive/manifest digests and six verifier/import scripts. `python3 scripts/prepare-vellum.py` checks provenance and archive identity before executing bootstrap code, then verifies every SDK member. All four native replacements resolve into `build/.vellum-sdk-e58d405-elab1`. Unknown or edited caches are refused and preserved.

The driver requirement names public pseudo-version `v0.0.0-20261021143700-e58d40581fe2`. The root module's `v0.0.0` requirement is retained by upstream's driver requirement and Go module selection; both modules are replaced by the same exact locked SDK. The local replacements carry the compatibility patch; a module version alone does not describe those derived bytes.

The exporter still marks SDK artifacts as private builds because public SDK/distribution qualification remains incomplete. **Public source availability is distinct from a qualified public SDK release.** MIT licensing, upstream notices, `SDK_MANIFEST.json` and `PRIVATE_BUILD.txt` remain included. Existing archives/caches, app bundles and their historical validation are unchanged. This dependency update does not qualify or replace the /189 owner app or /193 candidate.

Run SDK Python utilities with `python3 -B` to preserve cache integrity. Normal build/lint/check scripts prepare the SDK automatically. Portable headless Go commands remain independent of Vellum. [Update validation](../../docs/validation/vellum-public-source-update/README.md) records the compile incompatibility, preserved fixes and final checks.

## Previous pin: b22478c

ELab source builds consume `vellum-b22478c.tar.gz`, exported from clean isolated commit `b22478cf8cec8dda1ccdcd93d6ff6b62e2cc11de`, based on the preserved `274f5cc`. The numeric-field settlement barrier commits a complete coordinate draft before an action that does not move pointer focus. It retains originating-window FIFO ordering, numeric validation, record/window lifetime and application command/document guards. Four source/test members change; prepared dependencies, notices and all five bootstrap sources are unchanged. The preceding Cocoa/legacy handoff repair remains included. The sibling project, prior archives and prepared caches are preserved. [Source repair and qualification](../../docs/validation/number-field-placement/README.md), [integration history](../../docs/vellum-refresh.md).

The previous lock pinned the complete archive, manifest, source identity and five verifier/import scripts. Its verified cache remains under `build/.vellum-sdk-b22478c`; active module replacements now use the public-source pin above. Never edit a verified build cache to develop framework changes.

The archive carries `LICENSE`, `THIRD_PARTY_NOTICES/`, its inventory, `SDK_MANIFEST.json` and `PRIVATE_BUILD.txt`. Preserve those files. The exporter labels this a private SDK; it is not a public framework release or evidence of platform/distribution acceptance. Its internal version `v0.1.0-alpha.2` alone does not identify the snapshot; use the lock's commit and SHA-256.

Use `python3 -B` when running SDK Python utilities directly so they do not create bytecode files inside the verified cache. Normal ELab build/check scripts handle preparation. Portable headless Go commands remain independent of the SDK.

To update, select a fixed public commit, review compatibility patches against it, prepare dependencies and export a checksum-pinned SDK with the upstream release script. Update the archive, provenance lock, bootstrap set and four `desktop/go.mod` replacements together. Run source/offscreen consumer checks; native acceptance remains separate when no desktop session is available. The [historical mouse patch and reconstruction hashes](../../docs/validation/native-mouse-sdk-sources.json) target their recorded base and must not be applied blindly to a different revision.

The separate [packaged application baseline](../../docs/vellum-baseline-current.json) records the last qualified app bundle. This source pin does not change that bundle or its release status.
