# Item 4: maintained qualification and CI

`scripts/quality.py` is the shared portable/native gate controller. It records
terminal results, owned-process interruption cleanup and source-input identity.
`scripts/release_qualification.py` assembles clean matching app/agent/controller
builds, checks curated example graphs and isolates native/public runtime tests,
then verifies exact archive payload bytes/modes, OS extraction and strict seals.
`docs/release-tooling.md` documents the supported invocations. Existing outputs
are refused, and earlier qualified packages are not replaced.

The Ubuntu and macOS workflow definitions call those same gate plans with
read-only repository access, explicit toolchains, pinned official actions and
retained receipts. Hosted jobs have not been pushed or executed by this work.

Initial/follow-up Prism reviews are complete Anthropic Claude Opus 5.5 reviews.
Resolved findings include SIGTERM cleanup, false-green report publication before
input verification, failure journals for unexpected errors, preserved cache and
Go flags, manifest-derived executable pinning, post-failure journal errors,
changed-package controls and interrupted running-step reporting.
The follow-up manifest-pin medium finding assumes that the package manifest is
not verified first: qualify calls read_package before runtime, checks all payload
bytes/modes, passes the verified executable pin, and rechecks the package after
runtime. The runtime helper independently compares the sealed executable. No
self-derived replacement pin was introduced. Existing GOFLAGS are deliberately
retained and policy flags appended; an actual Go control confirms last flag wins.

The initial complete race run failed six numerical reference tests at the
unchanged wall guard under instrumentation. All six passed together in an
isolated race run. Their original assertions now also run in production-speed
child test processes under the full race parent; this follows the existing
capacity-reference testing approach and does not change any simulation limit.
Worker, ownership, controlled-clock and other tests retain race instrumentation.
The targeted six-child control passes. Prism reviewed this harness separately.
Its rebuild-per-test medium finding is accepted as bounded isolation overhead:
there are six callers, not eight; the complete targeted control took 14.272 s
including builds, against a 30-minute gate deadline. Each test owns and cleans its
binary. A shared global binary/TestMain would add cleanup/lifecycle coupling.

The final reviewed tool check/archive against an owned copy of /279 passes:
87 saved/current examples, four public adapters with 102 normal external exits,
geometry-protected copied native runtime, 436-file archive, OS extraction and
strict seals. This exercises the tool against preserved known binaries; it does
not relabel /279 as containing newer source commands. A redundant archive
invocation was correctly refused because its output already existed; the first
successful receipt is preserved. The 93 Python controls pass, including actual
SIGTERM reaping. Native OS-input checks 3/6/9/11 remain separate and open.

The complete portable production-reference run passed workbench in 331.182 s,
but sandbox-denied local sockets failed conformance, test-spooler and transport
packages. This receipt remains failed; the authorized local-socket package
reruns are recorded separately. The harness follow-up forwards -short, derives
its deadline from the parent with reporting margin, and logs actual limit
constants. Gate build flags are inherited through GOFLAGS; arbitrary explicit
parent-only tags/CPU flags are not a supported gate configuration.

The numerical follow-up timeout finding is repaired: remaining child time is
checked after building, its own timeout precedes context cancellation, and both
commands use bounded pipe WaitDelay. Those are test-runner safeguards only.
Targeted final six-reference checks use the repaired helper.
