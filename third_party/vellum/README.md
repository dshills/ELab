# Pinned private Vellum SDK

ELab source builds consume `vellum-bc6b456.tar.gz`, a private SDK exported from clean isolated commit `bc6b456f0c71b0327d0c02629b109eabbbc8994c`, based on the previous `47b7153` pin. It adds the minimal partial-scroll accessibility geometry repair reproduced during first-version UI testing. The existing stage-78 mouse-position patch and all prepared dependencies are unchanged. The adjacent Vellum checkout is unchanged and remains independent. See [integration and validation](../../docs/vellum-refresh.md). The prior `vellum-47b7153.tar.gz` and older archives are retained for historical reproduction.

`sdk.lock.json` pins the complete archive, manifest, source identity and five verifier/import scripts. `python3 scripts/prepare-vellum.py` verifies the archive before executing the pinned verifier, checks every member, and publishes a complete SDK under `build/.vellum-sdk-bc6b456`. All four native module replacements must resolve there. Cache reuse rechecks the exact file inventory and every checksum. An altered or unknown cache is refused and preserved; move it aside to retain edits before preparing a fresh copy. Never edit the build cache to develop framework changes.

The archive carries `LICENSE`, `THIRD_PARTY_NOTICES/`, its inventory, `SDK_MANIFEST.json` and `PRIVATE_BUILD.txt`. Preserve those files. The exporter labels this a private SDK; it is not a public framework release or evidence of platform/distribution acceptance. Its internal version `v0.1.0-alpha.2` alone does not identify the snapshot; use the lock's commit and SHA-256.

Use `python3 -B` when running SDK Python utilities directly so they do not create bytecode files inside the verified cache. Normal ELab build/check scripts handle preparation. Portable headless Go commands remain independent of the SDK.

To update, qualify a clean isolated framework commit, export a new private SDK with its release script, review its source and prepared dependency differences, and update the archive, lock and four `desktop/go.mod` replacements together. Rebuild and run the native consumer checks before accepting the change. The [mouse patch and exact reconstruction hashes](../../docs/validation/native-mouse-sdk-sources.json) retain the local fix; its numbered dependency patch targets the recorded base and must not be applied blindly to a different framework revision.

The separate [packaged application baseline](../../docs/vellum-baseline-current.json) records the last qualified app bundle. This source pin does not change that bundle or its release status.
