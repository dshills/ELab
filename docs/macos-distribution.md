# Signed and notarized macOS distribution

The private ad-hoc packaging path remains available. Developer ID distribution
adds an explicit notarization path that requires clean committed source and an
installed **Developer ID Application** identity. An Apple Development identity
does not meet this requirement.

## Local setup

In Xcode Settings → Accounts, select the enrolled team and choose Manage
Certificates → + → Developer ID Application. The account needs the appropriate
Developer ID certificate role. Keep the certificate and its private key in
Keychain; do not export them into the repository.

Store notarization credentials interactively in your own Terminal:

```sh
xcrun notarytool store-credentials "ELab-notary"
```

Use its prompts for the Apple ID, Team ID and app-specific password. Packaging
accepts only a Keychain profile name, not passwords or private-key arguments.
Inspect available signing identities with:

```sh
security find-identity -v -p codesigning
```

## Build

Use the exact installed identity name or its certificate fingerprint:

```sh
python3 -B scripts/package_web.py \
  --output 'build/signed-alpha-313/ELab Web.app' \
  --version 0.206.0-alpha.2 --build-version 313 \
  --sign-identity 'Developer ID Application: YOUR NAME (TEAMID)' \
  --notary-profile ELab-notary \
  --archive build/signed-alpha-313/ELab-Web-0.206.0-alpha.2-macos-arm64.zip
```

If the signing identity or notary profile uses a separate Keychain, pass its path
with `--sign-keychain` or `--notary-keychain`, respectively. Paths are passed as
process arguments; they do not change the system Keychain search list.

The packager signs both executables with hardened runtime and a secure timestamp,
seals the app, submits a temporary ZIP to Apple, requires an Accepted result,
staples and validates the ticket, and checks Gatekeeper. On systems with
`syspolicy_check`, its distribution preflight must also pass. The final ZIP and
file-hash receipt are created after stapling. No existing output is overwritten.
The external package receipt records the Apple submission ID and verified status;
the embedded metadata records that notarization was requested before sealing.

Submission waiting is bounded to 20 minutes. A timeout or refusal is a failed
build, not a notarized package. Apple may continue processing a timed-out request;
use `notarytool history --keychain-profile ELab-notary` to inspect it. This tool
does not change macOS trust settings or remove quarantine attributes.

## Verify the exact archive

Use a pinned digest from the package/release record, then extract and check the
shipping bytes in an isolated directory:

```sh
python3 -B scripts/check_installation.py \
  --archive build/signed-alpha-313/ELab-Web-0.206.0-alpha.2-macos-arm64.zip \
  --sha256 'EXPECTED_ZIP_SHA256' \
  --artifacts build/signed-alpha-313-installation
```

The check refuses unsafe/boundedness-violating archive entries, verifies the
shipping identity, signatures, matching signing teams, secure timestamps,
hardened runtime, ticket and system policy, then exercises a fresh copied
runtime without Go/Node in PATH. Private launch credentials remain temporary.

This check runs under the current OS account. Clean-account/Finder acceptance
still requires downloading the exact ZIP in a separate test account, opening
the app through Finder with normal Gatekeeper settings, editing and saving a
disposable circuit, quitting/reopening, and verifying that edit. Record the OS,
ZIP digest and observations; do not label that gate passed from a command-line
or same-account runtime check. Test libraries must be separate from owner work.

References: [Developer ID certificates](https://developer.apple.com/help/account/certificates/create-developer-id-certificates),
[Apple notarization workflow](https://developer.apple.com/documentation/security/customizing-the-notarization-workflow),
[notarization issues](https://developer.apple.com/documentation/security/resolving-common-notarization-issues).
