# Host background and divider grip /209

October 23, 2026. [Receipt](report.json), [source/check inputs](source-inputs.json),
[native baseline and paint probes](../../../research/ui-audit/window-background-209/README.md),
[repair notes](../../../research/86-window-background-and-divider-grip.md).

The already-running owned /206 fixture independently shows a white inset in
Dark and a divider focus click changing 0.65 to 0.66611844. Light and 0.65 are
restored afterward. No graphical application is launched for these checks.

The framework repair synchronizes each host background with its accepted
opaque root surface, preserving the existing inset and native/configured
fallback. Split resizing now applies physical movement relative to its press
position and accepted fraction. A plain focus click submits no resize.

The full framework normal/offscreen race suites and offscreen vet pass. The
complete ELab gate passes lint/staticcheck, 67 Python tests, root and desktop
race/vet, mounted UI race and offscreen vet. Desktop race takes
181.456 s; mounted race takes 156.936 s. Focused root probes check
both palettes, all four corners, unchanged geometry and accepted snapshots.
These are offscreen probes with schematic children omitted, not full native
app screenshots. Framework tests additionally exercise both divider axes and
compare actual pane movement to pointer movement.

The broad framework run also exposed a gallery metric left over from elab7's
shadow-free disclosure action. It reproduces on the unchanged elab7 SDK; the
metric correction retains the exact geometry and enablement assertions.
Earlier failures, the initial fixture compile correction and final passes are
preserved separately in [the raw archive](raw.tar.gz) with
[member hashes](raw-manifest.json). Source snapshots and producers are included;
their original absolute workspace paths require reviewed relocation elsewhere.

The elab8 archive is exported from clean derived source
`a4ae0319ec1f1ed1ec5f85aee825af893a2a4f8a`. Every member matches its manifest; only the
nine reviewed source/test/provenance members differ from elab7. Dependencies,
notices and all six bootstrap sources remain unchanged. The old archive/cache
and /205 download verify unchanged. The native command ledger is byte-identical:
166 commands, 134 source routes, 404 sites and 76 production Go files.

Fresh native acceptance of this repair and the accumulated /207 guidance/file
menus and /208 disclosure changes remains pending. No candidate is promoted by
this source record. The current download remains /205. Standing launch and
desktop authorization is saved in `AGENTS.md`; no further owner review is
required and no new launch approval prompt is left waiting.

The [clean 0.181.8 /209 candidate](candidate-report.json) builds from
`2975c3ecb5c3cfc5d80f148e2d284519c4d83fdf`, verifies its strict ad-hoc bundle signature
and passes the matching headless cursor/RC protocol with completed joined
shutdown. All 868 source/check inputs still match. It remains held for
[fresh native acceptance](native-acceptance-pending.json); no graphical launch
or new approval request was used. The /205 download is unchanged.
