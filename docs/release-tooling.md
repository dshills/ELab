# Quality and unsigned release tooling

Run the maintained gate plans from any working directory:

```sh
python3 scripts/quality.py --profile portable --artifacts build/quality-portable
python3 scripts/quality.py --profile native --artifacts build/quality-native
python3 scripts/quality.py --profile native --dry-run
```

Portable runs trusted SDK preparation, Python tests, PRD/inventory verification,
complete portable race tests and vet. Native adds lint/imports, complete desktop
and rendered race suites and both vets. Go test-runner timeouts are 30 minutes;
`--command-timeout` bounds each owned command separately. Simulator accuracy,
integration/work and resource limits do not change. Six timing-sensitive
numerical references run their original assertions in fresh ordinary child test
binaries when the parent uses -race; the rest of the suite stays instrumented.
The children inherit GOFLAGS (including any supplied build tags), preserve
-short, and finish before the parent deadline. Explicit extra parent-only build
tags/CPU flags are not forwarded; these gate plans do not use them. Native requires macOS;
the process-group runner supports POSIX hosts. `scripts/check.sh` delegates to
the native plan with a fresh output directory, overridable through
ELAB_QUALITY_PROFILE and ELAB_QUALITY_ARTIFACTS.

Output directories must be new. Atomic report.json publication stays unpassed
until every gate and source-input verification completes. SIGTERM/SIGHUP/SIGINT
use the same owned-process cleanup path. report.json records each attempted command,
terminal state and elapsed time, including failures/timeouts/interruptions.
The runner stops on the first failed gate. It signals only a newly created
process group on timeout/interruption. inputs.json hashes non-ignored source,
test, module, script, workflow and pinned dependency inputs; a change during the
run makes the final report fail. ELAB_GOCACHE overrides are preserved; CI sets GOCACHE before setup-go so the
restored cache is actually used. Existing GOFLAGS are retained with bounded
parallelism and -mod=readonly appended. The source commit is recorded separately;
working-tree tests need not claim a clean binary build.

`.github/workflows/quality.yml` calls the same portable plan on Ubuntu 24.04 and
native plan on [macOS 15 arm64](https://docs.github.com/en/actions/reference/runners/github-hosted-runners), with read-only repository permissions, no persisted Git
credentials, bounded jobs and retained test receipts. Go 1.27.3 matches the local
qualified toolchain; Python 3.13 is pinned by minor version. Actions are pinned
to verified source commits: [checkout v4.2.2](https://github.com/actions/checkout/releases/tag/v4.2.2),
[setup-go v5.5.0](https://github.com/actions/setup-go/releases/tag/v5.5.0),
[setup-python v5.6.0](https://github.com/actions/setup-python/releases/tag/v5.6.0),
[upload-artifact v4.6.2](https://github.com/actions/upload-artifact/releases/tag/v4.6.2).
These are deliberate pins, not claims of latest releases. No workflow has been
pushed or remotely executed by this work; hosted-runner behavior remains to be
observed when the owner pushes.

## Assemble a candidate

```sh
python3 scripts/release_qualification.py assemble \
  --version 0.200.2 --build 280 \
  --examples build/ELab-0.200.1-279-Reliability/Examples \
  --licenses build/ELab-0.200.1-279-Reliability/Licenses \
  --output build/ELab-0.200.2-280-Readiness --work build/assembly-280
```

Assembly requires clean committed source, explicit curated examples/notices and
new non-overlapping outputs. It builds matching agent/native/conformance binaries,
uses the pinned icon/notice packager, verifies all three clean VCS identities and
writes ELAB_PACKAGE.json with every payload file's SHA-256 and mode. Curated
examples must match the current built-in IDs/graphs; update that input corpus
when intentional example changes occur. Explicit notices are inherited inputs,
not an automatic license/rights clearance for future dependency changes.

## Qualify and archive

```sh
python3 scripts/release_qualification.py check \
  --package build/ELab-0.200.2-280-Readiness \
  --artifacts /private/tmp/elab-qualification-280 \
  --runtime --controller build/assembly-280/elab-conformance
python3 scripts/release_qualification.py archive \
  --package build/ELab-0.200.2-280-Readiness \
  --qualification /private/tmp/elab-qualification-280/qualification.json \
  --archive build/releases/ELab-0.200.2-macos-arm64.zip \
  --artifacts build/archive-280
```

Check reopens every example paused and clean, compares saved and current built-in
graphs, observes component/command counts and normal agent exit. `--runtime`
adds copied native smoke plus four public adapters under development-file and
outbound-IP denial. A fixture-only denial protects shared window geometry.
The conformance controller must match the same clean source. Without --runtime,
the report explicitly records that runtime checks were not performed.

Archive requires a successful qualification tied to the exact manifest. It
refuses changed, missing, extra, link or mode-modified payloads and existing ZIP
paths, verifies exact ZIP membership/bytes/modes, OS extraction and strict ad-hoc
seals. Qualification artifacts stay outside the package. Errors preserve partial
owned outputs and logs for inspection; they never replace an earlier candidate.
These tools neither push/upload releases nor satisfy native OS-input, participant,
clean-account, Developer ID or notarization acceptance gates.
