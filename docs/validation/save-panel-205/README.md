# Native Save filename baseline /205

October 23, 2026. [Results](report.json) identify the packaged /205 agent and a
separately identified native bundle with identical unsigned compiled code.
Save defaults, explicit suffixes, a compound Unicode name and an intentionally
repeated suffix were created through the native panel, then cleanly reloaded
through the matching public agent. An agent-requested extensionless path was
preserved exactly. All five blank fixtures are retained here with hashes.
No owner document was opened or changed; the owned test app closed through its
private endpoint, removed it and disappeared from CUA's running-app inventory.

[Native images](../../../research/ui-audit/save-panel-205/README.md) show a
remaining keyboard issue: Command-A did not extend the initial basename
selection to the full name. Native AX full-name selection worked. The app has
no Edit menu; the native panel uses AppKit text responders. The cause and repair
still require controlled validation. No path-normalization change is justified:
an intentional `.elab.elab` name must remain intact.

The first verification script assumed the panel returned `/private/tmp`; AppKit
returned its `/tmp` alias. The corrected check compares that native destination
by resolved path, while the direct agent path still requires exact string
preservation. This was a fixture correction, not an application repair. The
initial CUA click following a full-tree capture used a retired index and did
nothing; fresh indices were read before saving. Command-A/type sequencing also
initially appended to the basename; AX whole-name selection controlled the later
filename tests. None of those attempts is relabeled a keyboard-save pass.
