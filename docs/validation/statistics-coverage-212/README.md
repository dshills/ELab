# Recorded statistics coverage /212

October 23, 2026. [Receipt](report.json), [source inputs](source-inputs.json),
[repair notes](../../../research/89-recorded-statistics-coverage.md).

The unchanged /211 packaged agent accepts an imported constant 5 V capture
whose final timestamp precedes its nominal duration within existing import
roundoff. A partially unsupported tiny interval reports 2.5 V mean and
3.5355339059327378 V RMS. Only the final timestamp was changed in the fixture.

The shared statistics path now refuses intervals outside actual saved rows,
preserving prior cached results and the full snapshot. The desktop defaults
to the saved endpoint; external agents can discover it through existing APIs.
Import compatibility, timestamps, commands and formats remain unchanged.

Backend race regressions pass in 1.755 s; desktop callback race regressions
pass in 1.913 s. Covered intervals as narrow as one floating-point step retain
5 V mean/RMS and zero AC RMS. The complete source gate passes lint/staticcheck,
67 Python tests, root/desktop race and vet, mounted race and offscreen vet.
Desktop race takes 177.986 s; mounted race takes 159.929 s. All 879
captured inputs match. [Raw checks and producers](raw.tar.gz) include the
baseline failures, exact changed sources, logs and [member hashes](raw-manifest.json).

No new graphical application is launched for this check. The SDK and /205
download remain unchanged. Source/headless checks do not qualify native
acceptance of the changed measurement default or accumulated UI fixes.
Standing desktop/app authorization is saved in AGENTS.md; no new permission
or owner review is requested, and no unattended launch prompt is queued.


The [clean /212 candidate receipt](candidate-report.json) identifies source
`7be54298ee72d69901246ddd81c7ce50359ac251` and both exact binary hashes. All three Go binaries
report that revision with `vcs.modified=false`; strict ad-hoc bundle validation
passes. Its matching packaged agent refuses eight unsupported query/set cases,
passes four covered constant-source measurements (including a one-step-wide
interval), discovers the endpoint through public row reads, and preserves rows
and metadata through export/reimport. The agent closes normally with joined
workers and exit code 0. The signed bundle is unchanged by these checks.

The candidate is held for [native acceptance](native-acceptance-pending.json).
No graphical launch, download replacement, release promotion or publication
occurred. Previous /211 binaries and the /205 ZIP remain exact.
