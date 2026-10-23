# Technical-details palette /208

October 23, 2026. [Receipt](report.json), [source/check inputs](source-inputs.json),
[rendered progression](../../../research/ui-audit/disclosure-theme-208/README.md),
[repair notes](../../../research/85-disclosure-theme.md).

The original descriptor and private-heading regressions fail independently.
The surface-only consumer render still fails on blue glyph pixels. With both
changes, actual mounted Light/Dark surface/ink checks, physical expansion and
shared-agent collapse pass without changing physics, document or history.
The separate driver test covers independent simultaneous palettes and late
updates, alongside existing disclosure ownership/focus/keyboard checks.

The complete ELab source gate passes: 67 Python tests, root/desktop race and vet,
required mounted controls and offscreen vet. Desktop race takes 173.949 s;
mounted race takes 149.661 s. Unchanged root tests reuse valid cache entries.
The full Vellum driver/subpackage race suite and offscreen vet also pass. Its
first restricted Unix-socket failures and the successful automatically reviewed
rerun remain separate. No desktop launch or device setting change is used.

The SDK pin moves to a clean derived elab7 snapshot with only three changed
members: disclosure code, its existing test and the generated provenance note.
All dependencies, notices and six bootstrap sources are unchanged. Old caches,
archives and the qualified /204 and /205 app packages remain preserved.

[Raw archive](raw.tar.gz) and [member hashes](raw-manifest.json) preserve the
failing and passing logs, producer correction, SDK comparison/export, build
producers and source snapshots. Producers retain their original workspace and
stage paths and require an explicit reviewed relocation elsewhere. The mistaken
consumer test invocation from the portable root did not run the desktop case;
the corrected desktop runs establish the recorded failure and pass.

Source and mounted renders do not qualify a newly launched macOS binary. Native
disclosure paint/fonts/keys and the preceding /207 file-menu/guidance changes
remain pending; a candidate must stay unpromoted until those checks pass. The
current download remains /205. Standing launch/test authorization is saved in
`AGENTS.md`; no fresh launch prompt is left waiting for the owner.
