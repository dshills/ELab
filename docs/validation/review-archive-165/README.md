# Private /165 ZIP and local macOS extraction rehearsal

The sealed review is packaged at `build/ELab-0.148.0-165-review.zip` with a companion `.zip.sha256` checksum. Its 46,163,819 bytes contain exactly the 788 sealed files plus their release manifest. macOS `ditto` extracts all 789 files with identical bytes and Unix modes, including both executables, into a Unicode path with spaces. The source review and its original receipt remain unchanged.

The exact extracted native app and agent run from an empty working directory under the byte-identical denial policy previously qualified for /165. The original policy controls retain their original run scope; this rehearsal does not repeat their canaries. Both adapters load and save the three included examples; six fresh extracted-agent sessions reopen the saved files with graph, pause preference, results and scope parity. Every original extracted file still matches the sealed manifest afterward. The owned native retry closes normally at exit 0, its endpoint is removed, and both owned test PIDs are absent.

The first runtime producer incorrectly required every example to be paused, although the DC LED example explicitly stores `paused:false`; the RC and flasher examples store `true`. That failed report and terminated-host exit −15 are retained. The corrected producer checks each actual saved pause preference and runs in a fresh owned directory; it passes without changing the app, examples or tolerances. All 54 Python tests pass, including rejection of modified executable bytes, lost executable permission, symlink, duplicate, extra and noncanonical/traversal archive entries before extraction.

This is direct executable startup and public-command evidence on the existing workstation. It does not test Finder Archive Utility, LaunchServices/document association, quarantine/Gatekeeper, signing/notarization, default user configuration, a clean account/OS, another machine or physical UI input. Recovery and appearance persistence are disabled for these owned sessions. No user project, settings or app were modified, and the private alpha is not promoted. General installation/distribution and owner acceptance remain open.

The large reports, synthetic projects and native logs are losslessly compressed. Endpoint credentials are excluded. `checkpoint.json` pins the archive, existing qualification receipt, helper source and every proof file. Run the read-only verifier from the source repository:

```sh
python3 -B research/ui-audit/first-version/verify-review-archive.py
```
