# Build information and diagnostic reports

The workbench status bar shows the running app version. Open that control for
About ELab: version/build number, full source revision, modified-source flag,
OS/architecture and Go runtime. Development builds say Development; unavailable
VCS identity says unknown. Packaged backends embed version/build in the executable.
Package qualification compares the running backend's identity with the sealed
package receipt, rather than trusting a nearby metadata file to label the UI.

Choose **Create report preview** to capture a report. Nothing is uploaded. The
preview is a frozen JSON snapshot that can be refreshed, downloaded or copied.
Closing the dialog clears it. A late response from a closed dialog cannot fill a
newly opened preview. Failed requests disable download/copy and offer a retry.
Clipboard refusal leaves manual copy/download available.

Included: app build/source, timestamp, browser family/version/platform, viewport
and pixel ratio, bounded solver status and job counts, circuit object/wire counts,
known circuit-check codes, unsaved/storage-attention booleans and published limits.
Unknown status/code strings become `other`; raw user-agent strings are not kept.

Excluded: project names, component IDs/values/settings, circuit graphs, wire
references, recordings/waveforms, filesystem paths, current URLs, workspace and
host identifiers, cookies, agent/control credentials, raw errors and toolchain
paths. The backend uses an explicit typed scalar projection; it never serializes
a whole snapshot, workspace or error object. Technical details shown in the
ordinary circuit-check panel are not automatically included in this report.

The build endpoint does not allocate a workspace. Both endpoints use the existing
origin/method policy and no-store responses. Reports do not edit the circuit or
change history. The user decides whether and where to share the downloaded file;
ELab supplies no automatic sending mechanism.

Validation includes hostile private strings in snapshots/workspace metadata,
unknown codes, bounded output, read-only revisions, source fallback, copied
shipping-binary identity and browser journeys covering explicit preview, exact
file bytes, failure/retry, clipboard fallback, closed-dialog late responses,
keyboard dismissal/focus return and light/dark accessibility.
