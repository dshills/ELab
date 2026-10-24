# Decimal number steps /223

[Receipt](report.json), [frozen inputs](source-inputs.json),
[raw evidence](raw.tar.gz), [member hashes](raw-manifest.json).

The /222 native playback control displays 0.031999998 after typing 0.03,
clicking plus and pressing Up. Existing binary addition has accumulated into a
different float32, so lowering text precision would hide a stored-value mismatch.
Two framework and two real live/recorded playback regressions reproduce it.
The initial live test's conflicting isolated minimum width is a retained harness
failure, corrected to fit the production maximum before the genuine baseline.

Managed steps now add the displayed value/increment as exact decimals and round
once to the existing native float32. The accepted native value and round-trip
text agree. The public command types remain unchanged; supplied high-precision
values and old saved values are not shortened or migrated. Seven driver cases
cover repeated positive/negative steps, off-grid values, page steps, both wraps
and a tight finite boundary. Both production playback cases pass with single
shared-command delivery and unchanged circuit topology.

Focused draft/wheel/precision checks and complete framework/application gates
pass. Every one of 8,439 SDK members matches its manifest; only the factory line,
helper, test and generated provenance change. Dependency, notices and bootstrap
bytes are unchanged. All 8,437 previous SDK cache members, native /222 bytes and
the /205 download are preserved. Source checks do not qualify a fresh native
candidate; changed-candidate native acceptance follows separately.


## Native affected-control results and open lifecycle issue

[Native receipt](native-report.json), [raw captures/checks](native-raw.tar.gz),
[hashes](native-raw-manifest.json), [remaining checks](native-acceptance-pending.json).

The clean 0.181.22 /223 app/agent are built from `4356b8fb`. Normal Quit and an
ordinary update/reopen retain the owned installation's identity, launcher and
sandbox, with /222 saved as backup. CUA first required refreshing its stale app
binding; that tooling state was resolved without a permission question. Tests
use a separately sealed copy whose nonempty Mach-O sections match the unchanged
original candidate. Exact original-bundle-byte launch is not claimed.

The native plus button followed by Up now reaches 0.031 then 0.032 in both live
and recorded playback. Their public snapshots contain exactly the corresponding
native float32 values, with one command per step. Both live endpoints and wraps
work. An unfinished 0.035 survives wheel input in the Simulate rail and scope
dock; complete accepted snapshots remain identical. External-agent speed set is
visibly reflected in the same native control. One initial coordinate click after
Return hit help because focus reveal had scrolled the inspector; it changed
nothing and is excluded from button qualification.

After the first restore, the owned process and endpoint disappeared unexpectedly.
The retained native log has no panic; a filename search found no ELab report in
the user's DiagnosticReports directory. This does not prove a crash or its cause.
The same installation reopened normally. Clean/private and dirty/private/scroll
restoration repeats did not reproduce the exit. The same reopened PID/executable
was live at explicit checkpoints 296.8 seconds apart; this is a bounded
observation, not general lifecycle acceptance. The issue stays open.

All 1,001 saved rows remain exact after reopening and the 2,002 independent RC
voltage references pass, maximum error below 5e-16 V. The saved fixture, 897 source
inputs, strict signatures, original candidate, /222 backup and /205 ZIP remain
unchanged. The final fixture is clean, paused at zero, Light, with original live
and recorded speeds restored through public commands. The arithmetic defect is
repaired; lifecycle and the remaining ten original native checks stay held.
No owner-review wait, new permission question, publication or release promotion.
