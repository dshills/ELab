# Internal dependency review

Scope: update on `exp-ui`; no merge or push. Reviewed locally without Prism or
an external code-review provider.

## Source and compatibility

Official public main is `0f81df6e6a4f9698fe3358710132a14987ad1f37`, dated
October 30, 2026. Its release metadata still reads v1.0.0. The published v1.0.0
source tag points at an earlier commit; this update follows public main, matching
the preceding source-pin policy. The sibling Vellum checkout and verified SDK
caches are not used as editable sources.

Rebased the cumulative ELab compatibility patch in an isolated clone. Four
conflicting files required manual reconciliation: `controls.go`, `ui/node.go`,
the native dependency recipe, and its preparation test. Other overlapping
`driver.go` and `number_field.go` hunks preserve the newer upstream code.

- Buttons retain upstream's `ActivateModified` modifier mapping and precedence.
  Both callbacks are retained for admitted pointer/keyboard input; genuinely
  programmatic activation retains its intentional latest-callback contract.
  The prior ten callback-origin scenarios now run for both callback forms.
- Both existing state keys and upstream's new activation type remain. Validation
  still refuses unsupported state-key ownership and placement combinations.
- Upstream's exposure-repaint stage 108 remains ahead of the six ELab native
  compatibility stages, now 109–114. All six patch bodies and original/output
  hash records are byte-identical to their previously qualified counterparts.
  The new exposure patch touches neither their inputs nor outputs.
- Preparation continues verifying every ordered input, patch and output hash;
  no trust check, solver limit, project format or public ELab command was relaxed.

## Packager

The exact upstream packager files are pinned separately. Its new
`macos_deployment.py` import and tests are included in the complete file map.
Deployment declarations are inspected from the executable and verified in the
bundle; no deployment target is lowered. Optional document registration remains
unused. The original icon and ad-hoc-signing behavior are retained.

## Known SDK test limitation

`TestNativeMediaReadAdmission` rejects a silent WAV in the new SDK, including
three isolated trials. The same test fails in the prior SDK in this environment;
both test and Objective-C decoder implementation are byte-identical. ELab does
not instantiate Vellum's native media view. This is an observed pre-existing
failure with an undetermined cause, not a passing test or a new-update regression.
No media test or implementation was weakened or changed to hide it.

The first SDK Python attempt was restricted by the execution sandbox (process
inspection and native icon conversion). The authorized unsandboxed rerun passes.
A catalog-controller setup used unsupported command names, then corrected them
to the advertised `capabilities.list` and `parts.list`. Those setup failures are
not product results. The old media control's initial root-module command matched
no tests; only the explicit driver-module rerun establishes its failure.

The previous experiment bundle and the new executable both declare macOS 27.0
in their Mach-O load commands; the new packager also records that minimum in
the app plist. See [ELab integration results](README.md) for the build, toolbar
regression, package verification and failures reproduced with the old dependency.
