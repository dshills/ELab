# ELab Web private-alpha 2 candidate

Version `0.1.0-alpha.2`, build `2`, macOS arm64, built from clean committed source
`6dfb9146f8c0d8fe8e901c7865b65b311ee192dd`. This is a prepared private candidate;
it does not replace the preserved alpha archives, merge main or publish a release.

Local artifacts:

- App: `build/web-alpha-2/ELab Web.app`
- ZIP: `build/web-alpha-2/ELab-Web-0.1.0-alpha.2-macos-arm64.zip`
- ZIP bytes: 10,384,528
- ZIP SHA-256: `2c9959ce7bfa54a6b1e3d264d625e007131d623be1f63df3baebca457d8dcfbc`

## Verification and internal review

- [Package receipt](package.json): clean source, build/version, source inputs,
  licenses and hashes of the final sealed executables and bundle files.
- [Archive check](archive-qualification.json): packager ZIP inventory/content
  checks, then an independent OS `ditto` extraction. All 12 app files retain
  their exact bytes and modes; strict signature and linked-library checks pass.
- [Copied runtime](runtime-qualification.json): executable and embedded assets
  work outside the source directory with Go/Node absent from PATH. Normal
  control Quit checkpoints/exits; the acknowledged 22 kΩ edit survives restart.
- [H-bridge round trip](hbridge-roundtrip.json): the actual Chrome-exported native
  project imports and exports through the copied packaged backend with no Go or
  Node in PATH. All 26 components, four discrete MOSFETs, native graph, device
  models, transient settings and one retained recording with all 45 exact stored
  rows remain unchanged. The private fixture host exits normally afterward.

Internal review checked the shipping bundle identifier/version, absence of the
GUI fixture's test-data-directory override, final sealed file checksums, copied
runtime independence, private host credentials, isolated data libraries and
preservation of existing packages. No external review service was used.

The [desktop/quality record](../web-overnight-20261102/README.md) covers installed
Chrome/Safari behavior, keyboard/accessibility corrections, pointer resizing,
H-bridge reversal and the full local native quality plan. The later change at
this package commit only hardens test initialization; it does not alter the
runtime. Its contention test passes 20 consecutive race-enabled repetitions,
the full web race suite and root/native lint.

## Scope and remaining gates

Ad-hoc signed private candidate, with the existing macOS 13 build target. No
Developer ID/notarization, clean-account acceptance or minimum-OS runtime test is
claimed. The updated shipping launcher GUI was not launched during this offscreen
candidate check; the launcher source itself is unchanged from the earlier
launcher qualification. Installed-browser checks used the disposable development
host, and the copied packaged runtime was verified separately as described above.

[Hosted qualification](https://github.com/dshills/ELab/actions/runs/41442136866)
reruns the corrected test setup at the exact package source. Its three browser
jobs and all seven Linux portable gates have passed; the
[original Linux receipt](portable-quality.json) confirms unchanged source inputs.
The macOS job remains in progress at this record. Confirm its final status before
promotion. VoiceOver speech/navigation and
owner usability evaluation remain separate from automated/AX-tree checks.

When the owner returned, desktop work stopped and engineering continued
offscreen. The incomplete Safari native-picker reimport is not claimed as a
passed GUI journey; the exported file's packaged API round trip is independently
verified in the receipt above.
