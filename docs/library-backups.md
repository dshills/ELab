# Back up and restore the ELab library

Open **Projects → Back up library** to download a portable ZIP. Keep a copy on
another disk or in your own backup storage. Autosave on this Mac does not protect
against losing the Mac or its disk.

The backup includes:

- All named projects, including Trash and projects imported by earlier restores.
- Recovery history, including retained history imported by earlier restores.
- Autosaved workspaces from active, inactive and lost browser cookies.
- Each document's embedded models, kept completed recordings and kept completed
  sweeps, plus other preferences/results supported by the native `.elab` format.
- Dormant projects/history from the older per-workspace storage layout. Identical
  migrated copies are deduplicated; older differing versions are preserved.

Active workspaces are checkpointed under the library and workspace locks before
reading the snapshot. A checkpoint/read/checksum/project-validation failure stops
the backup with an error. It never silently produces a partial library backup.
Only published project revisions are included; obsolete unpublished/scratch files
are not library entries.

Agent credentials, MCP endpoint files, host-control files, logs, undo stacks,
browser cookies/layout/theme settings and unfinished or unkept analysis are not
included. A backup cannot resume an in-flight solve. The ZIP is not encrypted;
store it with the same care as your circuit projects.

## Restore copies

Choose **Projects → Restore library backup…** and select the ZIP. ELab validates
its manifest, file list, sizes, checksums and every native project before showing
the counts. Cancel leaves the library unchanged. **Restore copies** adds the
validated batch without overwriting any existing project or the open circuit.

Saved projects keep their names and Trash status, with new local identities.
History and all backed-up workspaces appear in **Recovery history**; choose
**Recover a copy** to open unsaved work and then save it under a name. Workspace
copies are independent of browser cookies and old saved-project associations.
Restored history is retained outside the newest-50 automatic-history rotation.
Importing the same backup twice deliberately adds another set of copies.

Restore writes into a private staging directory, then publishes the entire batch
with one atomic directory rename. Invalid archives and ordinary staging/publication
failures publish no copies. Existing data stays in place. A process/disk failure
can leave an unpublished `.restore-*` staging directory; it is not indexed as a
library and the original backup remains usable. Backups and restore batches are
local files, not a remote synchronization service.

## Format and bounds

Version 1 uses `manifest.json` (`format: "elab-library"`) and an exact numbered
list of native project payloads. The manifest records source kind/identity,
metadata, byte size and SHA-256 for each payload. Workspace source identities
are one-way derived identifiers; browser-cookie values are not exported. Additional/duplicate files,
path substitutions, links, unsupported versions and invalid project/result/model
data are refused. Incoming ZIP paths are never used as extraction destinations.

Only one library transfer runs at a time, bounding aggregate ZIP/validation memory.
Limits are 256 records, 64 MiB ZIP, 128 MiB total project payloads, 1 MiB manifest
and the engine's 8 MiB limit per native project. A record is one saved project,
history entry or workspace, so these counts include multiple versions of a circuit.
An over-limit library fails explicitly; individual `.elab` project exports remain
available. A digest detects corruption, not the identity or trustworthiness of a
backup's author. Restore adds copies and never restores authorizations.
