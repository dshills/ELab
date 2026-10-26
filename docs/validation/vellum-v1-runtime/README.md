# Vellum v1 runtime qualification

October 26, 2026. Owner requested the latest UI runtime, with existing ELab
behavior and shared user/AI commands preserved. The selected public main source
is [4cf0176](https://github.com/dshills/vellum/commit/4cf01763c1fd5501648ea19b4bb406fe62446319),
whose release metadata is v1.0.0. GitHub's published release collection still
lists the older alpha.3 prerelease. The icon packager already uses the same
newer public source. This is a private, pinned SDK integration.

## Runtime source and compatibility

Clean derived source is `19c9ebc37ed9f06daa824e024b331caf9700f4b0`. The cumulative
compatibility patch retains ELab's existing text/control ownership, admitted and
held input callbacks, numeric settlement/stepping, scrolling, cursors, tooltips,
dividers and window backgrounds. Existing intentional selector, pure collection,
Switch/ToggleButton and programmatic contracts are preserved.

Three source conflicts are resolved: button press capture coexists with the new
NativeContent mount; disposal retains both help-tag and existing tooltip
retirement; dependency/test metadata retains prose patch 107 and shifts six
compatibility patches to 108–113. All native preparation input/output hashes
verify, including the contiguous list of 113 core patches. None of the six
compatibility input sets overlaps prose's output set.

All **8,862 SDK files** match their manifest hashes; the manifest is pinned separately. The lock pins the
upstream/derived commits, cumulative patch, archive, manifest and six bootstrap
files. All four module replacements point into a separate verified SDK cache;
prior caches and archives are preserved. [SDK pin](sdk.lock.json),
[archive comparison](sdk-comparison.json), [preparation](sdk-prepare.txt).

## Complete framework checks

Fresh race suites pass **505 portable /561 ordinary driver /1,028 rendered
driver** top-level tests (**1,259 /797 /1,976** including subtests), with no
skipped tests. All framework vets pass. The nested API tool's own **10 /36**
race tests and vet pass; the stable v1 API comparison reports **no incompatible
changes**, without replacing or weakening its upstream baseline. All 90 SDK
Python tests and 34 native preparation tests pass.
[Counts](framework-counts.json), [steps](report.json),
[stable API comparison](api-check.txt).

SDK Python tests include native icon conversion and invalid-artwork refusal.
Expected failure-path diagnostic text is retained in the log; its test verdict
is successful. Go source/driver/geometry evidence does not establish native OS
queue or hardware-rate acceptance.

## Retained setup corrections

The initial migration script confuses stage records with patch ordinals and
assumes every stage contains a patch. Asset-only stages disprove both assumptions.
The corrected recipe selects explicit patch ordinals; native preparation and its
full test both verify every input/output and the 1–113 sequence. The initial
conflicted-manifest invocation is retained as setup evidence, not a product bug.
[Apply record](compat-apply.txt), [setup-only failure](prepare-conflicted-setup.txt).

The first API command is invoked from the parent module, which excludes the
nested tool module. Running from tools/apicompat succeeds, as do the tool's own
tests and vet. The erroneous command is excluded from product/API claims.
[Initial command](api-check-module-setup.txt), [successful command](api-check.txt).

## Consumer and package scope

ELab's complete source, electrical, ordinary/rendered UI and packaged checks
follow below. No new component, electrical model, solver limit or app command is
introduced. Original native-input acceptance checks 3/6/9/11, participant and
separate-account acceptance, and public signing/notarization remain separate.
The existing owner app and projects are not driven by these tests. The sibling
Vellum checkout, original compatibility checkout and verified SDK caches are
never edited; new work uses an isolated owned source tree and cache. No push,
upload, security-setting change or heartbeat restart is authorized or performed.

## Complete ELab source checks

All fresh **880 portable /347 ordinary desktop /484 rendered** top-level race
tests pass (**1,963 /847 /1,518** including subtests), with no skipped tests or
cached test results. Lint/imports, all three vets, 77 Python tests, PRD audit and
build pass. Every frozen executable/test/module input remains exact during the
run. Existing tests and assertions are retained; no app-source, model, command
or solver-limit adjustment is needed after the SDK compatibility merge.
[Counts](application-counts.json), [steps](application-report.json),
[source freeze](application-frozen-inputs.json).

Final clean-source package and exact copied native evidence follow after the
source checkpoint. Original /273 and all qualified release bytes are preserved.

Fresh retained-driver GPU captures verify proposal/inspector geometry and routed
Apply in Light/Dark/System/Apollo. Light and Apollo are visually inspected;
these are owned rendered scenes, not desktop screenshots or native queue proof.
[Visual comparison](visual-comparison.json), [Light](visual/light-proposal.png),
[Apollo](visual/apollo-proposal.png), [focused check](visual-check.txt).

## Runnable candidate /274

**ELab 0.196.1 /274** is available at
[the updated app](../../../build/ELab-0.196.1-274-Vellum/ELab.app), with its matching
agent, original icon, updated SDK notices and all 62 existing examples. Clean
binary source is `500e39541b3d8d612046a3e5b04a4bd9e42c2aef`; final documentation
is a later checkpoint. [Package identities](candidate274/package.json),
[summary](candidate274/summary.json), [final seals](candidate274/final-signatures.txt).

The exact copied app passes full native smoke and all four public adapters with
development-file/outbound-IP denial and fixture-only shared-geometry write denial.
All **102 external conformance processes exit 0 normally**. The copied bundle,
source bundle, matching agent and controller remain exact.
[Runtime receipt](candidate274/runtime-report.json.gz),
[public adapters](candidate274/public-report.json.gz),
[normal process exits](candidate274/external-process-exits.json.gz).

The **entire command/schema and component catalogs are identical** to the
pre-update baseline: **194 commands /53 component kinds**. Existing drawing,
explanation, plotting, experiment and proposal walkthroughs pass through the
copied native service and matching headless agent. The feature fixture closes
with exit 0 and endpoint removal. These are shared-command workflows, not
physical mouse/keyboard or native OS queue acceptance.
[Catalog comparison](candidate274/catalog-comparison.json),
[native walkthrough](candidate274/native-feature-new-workflows.json),
[headless walkthrough](candidate274/headless-features.json).

All **62 examples** reopen paused and clean with exact graphs, and all 62
matching agents exit 0. All **126 packaged example files** match the earlier
qualified ZIP. All **1,383 frozen application inputs** and **19 protected prior
archive/executable identities** remain exact. Current and previous SDK caches
are reverified against their complete manifests. The four representative theme
captures are byte-identical to the previous runtime, with actual routed Apply
working in each theme. No app code, test assertion, model or solver limit is
changed to accommodate the upgrade.
[Examples](candidate274/examples.json),
[preservation](candidate274/preservation-after.json).

The owned native fixtures are closed normally; no owner app/project command is
sent. The previously qualified download is preserved, and /274 remains a
separate development candidate. No new endurance, participant, clean-account,
public signing/notarization or original native 3/6/9/11 acceptance is claimed.
No push, upload, security weakening or heartbeat restart occurs.
