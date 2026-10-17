# Disable autosave when storage is unavailable

**Disable autosave** remains available when the currently configured recovery folder cannot be opened or scanned. It changes scheduling and the interval in memory, reuses the existing store/lease and preserves the last observed error, snapshot list and selection. The native status distinguishes **Autosave is off · last recovery error** from an active failed attempt. Disabling does not claim the folder was checked or repaired.

The same rule applies to both public aliases:

```json
{"command":"recovery.configure","params":{"directory":"/the/current/configured/recovery/path","enabled":false,"interval_s":2}}
{"command":"recovery.configure.start","params":{"directory":"/the/current/configured/recovery/path","enabled":false,"interval_s":"2 s"}}
```

Use `snapshot.recovery.directory`, with the ordinary revision/document/retry guards. The waiting command joins work; the start form returns queued work to inspect. Successful same-store disable reports `recovery_configuration_applied:true` and `recovery_catalog_applied:false`. Filtered `capabilities.list` explains this in `x-existing-store-disable`; input time units and the 1–60 s bounds remain.

This exception applies only to disabling the already configured canonical path. New/different stores still perform ordinary creation/private-directory checks and scanning. Enabling uses normal preparation/scanning and can fail while storage is unavailable; a failed enable keeps autosave disabled. Invalid paths/intervals, cancellation, busy-work policy, actor attribution and retry deduplication remain. A canceled disable changes neither scheduling nor the last-known catalog/error.

Disabling stops new automatic captures; an already captured checkpoint can finish under the existing retention rules. Its late status cannot replace the disabled configuration. Existing recovery data is not explicitly discarded by the toggle. Restoring the folder while autosave stays off does not restart writes; use Enable explicitly. A successful scan on enable is distinct from a subsequent complete checkpoint of dirty work. Manual Refresh remains the separate way to update catalog observations while disabled.

The [public fault workflow](../scripts/recovery-failure-demo.py) with `--exercise-disable` verifies both aliases, preserved last-known error/catalog, refusal to enable broken storage, a three-second hold with restored available storage and no new file changes, then explicit enable and complete recorded RC capture. Controlled core tests separately release a captured writer after disable, protect cancellation/admission and verify discovery. A native callback test exercises the actual toggle, off/last-error wording, retained detail and successful re-enable. Descriptor/callback evidence does not establish full physical input, screen-reader or pixel/readability acceptance.

[Qualification receipt](validation/recovery-disable-validation.json) retains the original two-alias refusal, final source/binary identities, native/public checks and synthetic fixtures. No application command, project/recovery format, numerical model/method, component/work limit or canonical app bundle changes. General storage-device and remaining release gates stay separate.

On the recorded pilot workstation, both public transports pass disable, refused enable while broken, a three-second repaired-storage hold with no writes, and explicit enable into a complete latest recorded RC checkpoint. Full quality, 45 native smoke stages and all four existing numerical adapters pass. The private 0.113.0 /130 candidate is not promoted; complete physical input/accessibility and broader release gates remain open.
