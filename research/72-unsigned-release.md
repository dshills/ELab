# ELab 0.179.0: unsigned release ready

**Current successor:** [ELab 0.181.3 /204](79-first-version-release-closeout.md).
This document preserves the original /196 release designation and its evidence.

October 22, 2026. The owner selected **“Prepare an unsigned release.”** The macOS arm64 release is prepared, checked and ready for that channel. No feature was added and no review response is required.

- [Release ZIP](../build/releases/ELab-0.179.0-macos-arm64.zip)
- [SHA-256 checksum](../build/releases/ELab-0.179.0-macos-arm64.zip.sha256)
- [Open app](../build/releases/ELab-0.179.0-macos-arm64/ELab.app)
- [Agent](../build/releases/ELab-0.179.0-macos-arm64/elab-agent), [installation and use](../build/releases/ELab-0.179.0-macos-arm64/README.md), [release notes](../build/releases/ELab-0.179.0-macos-arm64/RELEASE_NOTES.md)
- [Final readiness evidence](../docs/validation/unsigned-release-196/README.md) and [machine-readable receipt](../docs/validation/unsigned-release-196/report.json)

SHA-256: `fcb9714728bea0ff602c14f42555eb4fc55ca28e5b178b0be76c2d74cb424edf`.

## Release boundary

This is version 0.179.0, build 196, from clean source `885af1ad3c2a130092e532e8ffe966a5c6789889`. It requires Apple Silicon and has an actual binary minimum of macOS 26.0; the tested host is macOS 26.6.2 arm64. It includes a matching external agent, three editable paused examples, dependency notices, installation instructions and rollback. The app and agent bytes match the [qualified engineering alpha](71-first-engineering-release.md). No rebuild or version increment was needed for packaging changes.

The owner-selected unsigned channel does not require Developer ID or notarization. The app retains its ad-hoc integrity seal. Gatekeeper rejects a quarantined test copy as expected; no automatic downloaded-app approval is claimed. Consult [Apple's official app-opening guidance](https://support.apple.com/102445). The release contains no quarantine-removal or security-disabling script, and no such override was performed. [Developer ID signing and notarization](https://developer.apple.com/developer-id/) would be needed for a different distribution channel.

## Closed release checks

- All 127 archive members retain hashes and modes on extraction, and the extracted app passes strict ad-hoc signature verification.
- Actual extracted app startup and rendering pass from a Unicode/space-containing path. Both binaries depend only on macOS system libraries; no repository, SDK, Go toolchain, cache or outbound IP access is needed by the tested core workflows.
- LED, RC and flasher reference results, graph preservation and saving agree between the extracted native endpoint and extracted agent. Fit displays the complete imported flasher. Final test processes shut down normally and remove their endpoint; archive contents stay unchanged.
- The earlier full source gate, independent electrical checks, Save-during-analysis repair, user/agent contracts, keyboard reference tasks and fitted/tiled output checks remain valid. All 672 Go source hashes still match; no test tolerance or runtime source changed.
- Remaining cursor Return/refusal wording and filename-basename suffix behavior are recoverable, preserve accepted state and remain documented. They do not reopen the accepted core task or data-integrity gates.
- Canonical /120, owner /189, the /195 candidate and /196 engineering alpha remain preserved. No owner project or settings were replaced.

Owner review and separate clean-account/default installation remain deferred. Broad accessibility/IME, complete PRD acceptance, hobbyist study, other platforms and general physical-device/SPICE qualification are not claimed. “Ready” here means the explicitly chosen unsigned release, with those limits disclosed. No publication destination or remote is configured; the files are ready locally and have not been uploaded or pushed.
