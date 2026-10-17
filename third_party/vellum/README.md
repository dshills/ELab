# Pinned private Vellum SDK

ELab source builds consume `vellum-aa9d023.tar.gz`, a private SDK exported from clean Vellum commit `aa9d023243a6bded514c3d9696c08a9c917119c4`. This is the isolated native mouse-position fix on framework base `77da1dd8784b5ef699f8f7a0f07f7aa1bf0e62d4`. It includes the prepared upstream dependencies. The adjacent Vellum project remains independent and can continue changing.

`sdk.lock.json` pins the complete archive, manifest, source identity and three verifier scripts. `python3 scripts/prepare-vellum.py` verifies the archive before executing the pinned verifier, checks every member, and publishes a complete SDK under `build/.vellum-sdk-aa9d023`. All four native module replacements must resolve there. Cache reuse rechecks the exact file inventory and every checksum. An altered or unknown cache is refused and preserved; move it aside to retain edits before preparing a fresh copy. Never edit the build cache to develop framework changes.

The archive carries `LICENSE`, `THIRD_PARTY_NOTICES/`, its inventory, `SDK_MANIFEST.json` and `PRIVATE_BUILD.txt`. Preserve those files. The exporter labels this a private SDK; it is not a public framework release or evidence of platform/distribution acceptance. Its internal version `v0.1.0-alpha.2` alone does not identify the snapshot; use the lock's commit and SHA-256.

Use `python3 -B` when running SDK Python utilities directly so they do not create bytecode files inside the verified cache. Normal ELab build/check scripts handle preparation. Portable headless Go commands remain independent of the SDK.

To update, qualify a clean isolated framework commit, export a new private SDK with its release script, review its source and prepared dependency differences, and update the archive, lock and four `desktop/go.mod` replacements together. Rebuild and run the native consumer checks before accepting the change. The [mouse patch and exact reconstruction hashes](../../docs/validation/native-mouse-sdk-sources.json) retain the local fix; its numbered dependency patch targets the recorded base and must not be applied blindly to a different framework revision.

The separate [packaged application baseline](../../docs/vellum-baseline-current.json) records the last qualified app bundle. This source pin does not change that bundle or its release status.
