# Vellum public-source dependency update

October 21, 2026. ELab source builds now use public [dshills/vellum](https://github.com/dshills/vellum) commit `e58d40581fe21d1de53fd54cab1bcf6acf11c163`, with the required existing ELab compatibility fixes carried explicitly. No ELab window was launched or controlled, and the owner's /189 app and preserved /193 candidate remain unchanged.

The upstream repository has no release tags at inspection time. Its README prescribes prepared Cogent/typesetting sources and local module replacements for native consumers. ELab therefore retains its verified, self-contained SDK workflow instead of dropping those required prepared dependencies. The driver requirement names public pseudo-version `v0.0.0-20261021143700-e58d40581fe2`; the root requirement remains `v0.0.0` under upstream's requirement/module selection, with both replaced by the same exact SDK.

An unpatched public-source consumer compile fails because `ControlStateKey`, `SupportsControlStateKey` and `SettleNumberField` are absent. The [tracked compatibility patch](../../../third_party/vellum/patches/elab-public-e58d405-compat.patch) carries 22 source/test files forward from the previously qualified private integration: text/control ownership, numeric settlement, held slider/button callback context, legacy text acknowledgments and partial-scroll accessibility publication. One NumberField merge overlap retains both the public pointer-focus binding and the settlement epoch. All other merges are clean. The patch matches clean local derived commit `96e22bf25809148b9a28169baf1b24cba72af475`, based directly on the public commit; it is not represented as an upstream commit.

The upstream exporter produces `vellum-e58d405-elab1.tar.gz`: 20,374,442 bytes, SHA-256 `2930c7cbd513e08294572189bf50e8fa4f084d05871107d07d49da6404a8fd57`, with 8,408 hashed SDK files. Schema 2 of the ELab lock records repository/base, derived commit, patch checksum, complete archive/manifest hashes and six bootstrap files. New verifier imports require `native_linkers.py`. Tests refuse wrong repository/commit identity, unrecorded derived source and changed compatibility patches before any Go/bootstrap command.

Validation passes:

- Ten focused preparation/integrity tests and the full 67-test Python suite.
- Framework UI ownership race tests and driver offscreen ownership, numeric-settlement, held-input, legacy-boundary and scroll-publication race regressions.
- The no-window Cocoa legacy-handoff race fixtures. These exercise synthetic native bridge objects, not physical OS typing in an ELab window.
- ELab native consumer compilation and an isolated development executable build.
- Full `scripts/check.sh`: requirement coverage, lint/goimports/Staticcheck, Python tests, root and desktop race tests/vet, configured mounted offscreen race tests and offscreen vet. Desktop runs report 169.846 s and 142.501 s; these are package test durations, not visible performance measurements.
- Post-test verification of the new SDK and unchanged application Go sources, plus original-pin verification of the previous b22478c archive and all 7,458 cached files. Canonical, owner and candidate binary hashes remain exact.

Initial outcomes remain in the raw logs: Go first requested module tidying; the unpatched compile then reported the missing APIs. One framework UI invocation placed `-C` after another flag and exited 2 before tests; the corrected command passes. Existing AppKit deprecation and duplicate-library warnings remain visible. These are not hidden or relabeled as passing application tests.

`report.json` records the pins, checks, preservation and limitations. `raw.tar.gz` retains build/export/test logs, original dependency files, commit provenance and final input hashes; `raw-manifest.json` identifies every decompressed entry. The SDK archive and compatibility patch are separately tracked under `third_party/vellum`. No binary is duplicated in the raw evidence.

All 670 application Go files remain unchanged. This is a source dependency migration, with the SDK exporter still marking its artifact private/unreleased. Existing package receipts retain their original dependency scope. A future replacement package needs affected native interaction and copied-runtime acceptance with the new pin; this record does not qualify visible input, owner workflows, clean-account startup, public distribution, promotion or the full MVP.
