# /165 startup isolation and repository handoff preflight

Two concrete README errors were corrected. The repository advertised affected /161 instead of qualified /165, and its development packaging command invoked an obsolete SDK cache (`ade66eb`) rather than the verified current pin (`b22478c`). The current review verifier now rejects the old README advertisement and requires the packaging example to resolve the verified SDK dynamically. The corrected packaging snippet successfully builds a development-only 0.0.0 /1 bundle, substituting only a clean /165 input binary and new temporary output path. The copied executable hash and bundle metadata match; the entire SDK is reverified afterward. This is command/documentation verification, not a new app build or release qualification. The sealed /165 review and ZIP remain unchanged.

Ordinary startup cannot be isolated completely with ELab's current recovery/appearance flags. Those paths support overrides, but the framework's normal window geometry uses `user.Current().HomeDir/Library/Cogent Core/window-geometry-0.3.6` with `.json`, `.lst` and `.lck` files. Smoke mode disables the saver; ordinary mode does not. The source hashes and path facts are retained in `startup-paths.json`. No ordinary launch in the current account or reassignment of HOME was used to imitate a clean account. A separate disposable macOS account/VM is required to test actual defaults without affecting the owner's configuration; availability was requested and no answer/approval is inferred.

A read-only `security find-identity -v -p codesigning` query found zero valid code-signing identities in the current account. The exact /165 app has an ad-hoc signature with no team identifier or signing authority. No identity names/fingerprints, private key material, signing operation or notarization request are stored/performed. This preflight does not qualify signing, Gatekeeper, installation or public distribution. A suitable authorized identity and notarization access remain external prerequisites for that path.

The four-task owner walkthrough remains Not tried yet. VoiceOver permission, actual visible-p95 observations and real hobbyist participants also remain open; these gates are not replaced with more functional reruns. No app source, API, model, SDK pin, saved format, PRD status or promoted bundle changes. P0 remains 17 implemented /10 partial. The current private alpha and evidence are ready for the owner to review.

The old README is retained losslessly as compressed historical data, not as current instructions. The packaging output is explicitly a new development test artifact and was not launched. Endpoint credentials and signing identities are absent. Verify with:

```sh
python3 -B research/ui-audit/first-version/verify-startup-handoff-preflight.py
```
