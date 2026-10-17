# Recovery ownership that cannot be verified

A valid recovery capsule with an identified foreign session stays protected when that session's lease is unreadable, malformed, oversized, linked, nonregular, mismatched or has an invalid PID. Those cases no longer mean “inactive.” The snapshot remains available for inspection and **Restore as a new copy**, while foreign dismissal refuses with `unavailable` and a false removal outcome.

This repairs a reproduced data-loss guard: a malformed lease previously made a still-running owner's capsule appear inactive and allowed another session to remove it. Tests reproduce the old deletion and now require unchanged capsule/source bytes. Ownership is checked both during preparation and again at the removal boundary, including a lease that becomes unverifiable after preparation.

Candidate snapshots and inspection/read results now expose typed `owner_status`:

| Value | Meaning |
| --- | --- |
| `active` | Valid lease identifies a process observed active; Unix EPERM remains conservatively active |
| `inactive` | Lease is absent under the existing cooperating-session protocol, or a valid process probe confirms termination |
| `unknown` | Existing lease or process liveness cannot be validated |

For compatibility, `active_session` is true for both active and unknown owners: it remains a conservative protection flag. Use `owner_status` to explain the distinction. `current_session` still identifies this store's own random namespace; a session can dismiss its own snapshots. The status is an observation at scan/read time, not a credential or a guarantee that a process remains alive later. Cached read-job results remain historical captures.

The native recovery choice says **owner not verified**, shows a short explanation, keeps copy restoration available and disables foreign dismissal. The shared dispatcher enforces the same decision for agents and rechecks ownership immediately before removal. A changed UI/catalog or a stale recovery choice cannot bypass the service guard.

A normally closing owner removes its lease, and a subsequent refresh can identify its retained snapshot as inactive. A readable lease repaired while its process is still running returns to active. For a damaged abandoned lease, keep or restore the capsule, then verify the owner has stopped before manually repairing/moving the lease or recovery files. An unverifiable owner is not evidence that deleting its data is safe.

Existing source/capsule formats, integrity/hash checks, current-session dismissal and explicitly hash-confirmed damaged-capsule handling remain. An unidentifiable damaged capsule is a separate case; the app does not infer a foreign session from missing identity data. These checks support ordinary cooperating application/filesystem use, not a hostile-process atomic conditional unlink or power-loss guarantee. Missing leases retain the existing graceful-close meaning; a process intentionally removing another live lease is outside that protocol.

On Windows, the current process can be identified; another present lease remains unknown/protected until platform liveness is qualified. This does not claim Windows runtime acceptance. Unix dead-process/crash behavior is checked separately.

[Validation evidence](validation/recovery-ownership-validation.json) retains the failing regression, real separate-process active/unknown/repaired/inactive transitions, copy restoration, source/capsule preservation and existing SIGKILL recovery. The safety repair itself does not count as performance qualification. Subsequent [repeated timing evidence](recovery-timing.md) covers named paused/advancing fixtures on the pilot workstation.
