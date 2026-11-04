# Repeatable web alpha releases

The maintained pipeline is `scripts/release_web.py` and the manually dispatched
“Qualify web alpha” workflow. Existing published alphas are preserved. This work
adds tooling; it does not publish a release or establish a hosted CI verdict.

On clean `main`, after installing the locked `browser/` test dependencies and
Playwright browsers:

```sh
python3 -B scripts/release_web.py stage \
  --version 0.207.0-alpha.1 --build 313 --output build/release-313
```

Use the actual next available version/build; the values above are examples.
Staging runs the macOS quality plan and all three browser engines, packages the
app, extracts the exact ZIP, verifies its file hashes/modes and signature, then
runs the copied shipping backend outside the checkout without Go/Node in PATH.
It checks restart, saved recovery and whole-library backup/restore. Source must
stay unchanged throughout. `--quality build/quality-...` reuses only a passing
receipt with the same commit, complete gate plan and exact current source hashes.
`--candidate` permits a clean development-branch rehearsal and marks it
non-publishable. Every output must be new; failed stages retain evidence.

Default packaging uses ad-hoc signing and explicitly records that notarization
is not qualified. Locally, existing `--sign-identity` and `--notary-profile`
names enable the maintained Developer ID/notary path. Provisioning remains with
the owner. The hosted workflow currently stages the private alpha without those
credentials; it does not silently claim distribution signing or a clean-account
installation result.

Review `release.json`, `SHA256SUMS` and `RELEASE-NOTES.md`, then:

```sh
python3 -B scripts/release_web.py publish --directory build/release-313
```

This creates a **draft** prerelease in `dshills/ELab`, bound to the qualified
commit. It requires clean local main and the same current GitHub main, and
refuses an existing tag. Only the ZIP, public release receipt and checksum file
are uploaded. It downloads all assets from that draft, checks their exact bytes,
then extracts and runs the downloaded shipping binary. A failed download/runtime
check leaves the draft unpublished for investigation. It never overwrites an old
release. No automatic retry changes release identity.

Adding `--make-public` publishes only after those checks and a second main/source
check. Without it the release stays a verified draft. The workflow's `publish`
input explicitly requests the same draft/download/publication sequence; its
default is false. Workflow dispatch is restricted to main, serialized, and does
not cancel a release already being qualified.

Runtime directories contain local startup/control credentials and must remain
private. The workflow uploads an explicit allowlist of distribution files and
credential-free reports, never the whole stage directory. The public receipt
contains source/version/build, artifact hashes, browser counts and bounded check
names. It records engineering evidence, not native VoiceOver, human usability,
clean-account installation or remote-CI success that was not observed.
