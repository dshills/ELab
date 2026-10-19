# Isolated macOS runtime rehearsal

A private ELab bundle and headless agent can run from a copied location without reading the repository, Vellum checkout/prepared SDK, selected Go caches/module cache/toolchain, or Xcode developer files. Outbound IPv4/IPv6 TCP and UDP are denied while the private Unix-socket agent interface remains available. This qualifies the named binaries on the recorded workstation; it does not establish a clean OS installation, signing/notarization, other machines/platforms or a security boundary shipped by ELab.

Run the rehearsal with a matching app, agent and conformance runner:

```sh
python3 scripts/runtime-portability.py \
  --app /absolute/path/to/qualified/ELab.app \
  --agent-binary /absolute/path/to/elab-agent \
  --conformance-binary /absolute/path/to/elab-conformance \
  --artifacts /private/tmp/new-elab-runtime-results
```

The output directory must be new, its parent must exist, and it must be outside the denied development paths. The harness copies the app and agent, verifies the bundle manifest, executable hashes and every bundle file/mode, and inspects native dependencies with `otool`. A non-system native dependency refuses; this rehearsal does not invent a closure for extra libraries.

Before running ELab, a subprocess proves an allowed local read succeeds and each available development-file control fails with an actual permission error. Missing files, timeouts and connection refusal do not count as permission denial. Local IPv4/IPv6 TCP and UDP targets are first proven reachable without the policy, then the restricted subprocess must receive a permission denial for each. The generated policy also denies writes to development paths. No source/user-file write attempt is used to test that clause.

The external app and agents execute inside the generated macOS policy from an initially empty working directory. The native smoke and all public project/electrical/inductor/op-amp/storage/capacity suites construct, simulate, Verify, export, save, close and freshly reopen owned files. The in-process baseline and qualification controller remain outside the restriction and may use developer tools. The controller uses the verified SDK's process-group cleanup for stage timeout/interruption. Stage journals record failures/interruption as terminal outcomes; interrupted work does not imply successful qualification.

`GOCACHE` selects the active Go-cache read control. The project cache has a separate logical control and can resolve to the same directory. The raw report records every path, so control count is not a count of distinct directories. Missing optional caches/checkouts are omitted rather than counted as denied. A build with other runtime dependencies needs its own closure and controls.

## Evidence and limits

The [qualification receipt](validation/runtime-portability-validation.json) retains the exact app/agent/controller hashes, source freeze, original/final harnesses, generated policies, control outcomes, process logs, complete public reports, typed review and saved project/recording files. Source/caches and IP networking are unavailable to the copied external processes; macOS frameworks, fonts, services, GPU drivers and ordinary user/system caches remain available. The native renderer's 45-stage smoke and matching electrical results are runtime evidence, not full physical input/accessibility or clean-account evidence.

The test intentionally avoids installing software, modifying system settings, submitting a physical print job or replacing the canonical app. A forced process kill/power failure and blocked operating-system calls remain outside cooperative cleanup guarantees. `sandbox-exec` is a host qualification tool; it is not an application feature or a proposed distribution dependency.


## Private review ZIP rehearsal

`scripts/package_review.py` creates an exclusive-output ZIP from a release-pinned receipt, checks every archived file/hash/mode, then verifies a macOS `ditto` extraction into a Unicode path with spaces. It refuses existing output paths, extra/duplicate archive members, changed bytes/modes, links and noncanonical/traversal paths before extraction. The original review stays sealed. For example, use new output paths with the current /165 receipt:

```sh
python3 -B scripts/package_review.py \
  --receipt research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json \
  --output /private/tmp/new-elab-review.zip \
  --artifacts /private/tmp/new-elab-archive-results
```

[The /165 rehearsal](validation/review-archive-165/README.md) separately starts the extracted app/agent under the previously qualified denial policy and tests the three bundled examples and six saved-file reopens. That policy's canaries retain their original run scope. The ZIP is a private unsigned handoff; this does not qualify Finder Archive Utility, LaunchServices/document association, quarantine/Gatekeeper, normal user configuration, a clean account or installation/distribution. The sealed folder's manifest, build identity and broader open gates are unchanged.
