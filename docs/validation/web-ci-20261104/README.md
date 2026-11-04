# Hosted CI follow-up — November 4, 2026

The published alpha313 source run
[41878077636](https://github.com/dshills/ELab/actions/runs/41878077636)
reported an uncaught WebKit fetch error during the save/reload/export journey.
The retained trace places the error between `page.reload()` and its committed
navigation, with `fetchJSON → refresh → poll` on the stack. Project value/name
and export assertions completed; the strict no-page-errors assertion failed.
The local macOS release qualification remains evidence for its recorded scope,
not a claim that this later hosted WebKit failure had passed.

## Fix and regression

Read-only polling now owns an AbortController. It aborts before unload while the
old document is active, suspends timers on pagehide and resumes once on a
persisted pageshow. A cancelled navigation continues polling the current page.
Abandoned responses cannot change connection status or apply stale presentation
state. Real network errors still report disconnection. Mutating commands and
explicit edit guards are unchanged; no global error suppression or origin-policy
relaxation was introduced.

The existing save/reload/export journey now holds an actual state read during
reload and observes its explicit abort across navigation. Its new assertion fails
against the pre-fix embedded host. Thirty repeated fixed journeys passed (ten per
engine), then the complete browser suite passed 46 journeys with two planned
non-Chromium load skips and no unexpected/flaky results. Lint, all 36 frontend
checks and `go test -race . -count=1 -timeout=10m` passed. Four lifecycle checks
cover abort, no duplicate restoration chain, stale completion, real errors,
hidden boot and continued reads after cancelled navigation.

The macOS job in
[41874617722](https://github.com/dshills/ELab/actions/runs/41874617722)
was cancelled before setup, with no runner assigned. Its check annotations state
“The job was not acquired by Runner of type hosted even after multiple attempts”
and identify macOS arm64 capacity constraints. The first fallback used `macos-15-intel`, which acquired a runner, but its full
quality run exposed two production-speed numerical fixtures exceeding the fixed
100 ms CPU limit: the 555 audio render and NPN-coil capture. The workbench race
package passed, and no data races were reported. These Intel results are retained
in [attempt 2](https://github.com/dshills/ELab/actions/runs/41881658292/attempts/2),
not represented as passing qualification. The private alpha's distribution target
is Apple Silicon; Intel runtime remains unqualified.

Routine quality now uses the current `macos-26` Apple Silicon image, retaining the
release architecture while moving away from the older macOS 15 pool. GitHub
[documents this label as arm64](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).
The full macOS gate plan, both numerical fixtures, their assertions and the
100 ms/512-attempt solver bounds remain unchanged. The separate Apple Silicon
release workflow and local release gates are unchanged.

Firefox's attempt-2 log confirms all 15 tests passed with one planned skip in
49.6 seconds. Fetching Ubuntu packages from `azure.archive.ubuntu.com` consumed
nearly 19 minutes; the 20-minute job deadline cancelled it immediately after the
tests passed. Browser jobs now have a 30-minute total limit with a separate
20-minute install limit, preserving time for tests and artifact cleanup. Individual
browser test deadlines, assertions, zero retries and suite coverage are unchanged.

Internal review covered early cancellation order (WebKit cancels before pagehide),
cancelled navigation, back/forward restoration, delayed responses, single-chain
ownership, strict error assertions, command isolation and runner architecture.
Lifecycle guidance: [pagehide](https://developer.mozilla.org/en-US/docs/Web/API/Window/pagehide_event),
[pageshow](https://developer.mozilla.org/en-US/docs/Web/API/Window/pageshow_event),
and [fetch cancellation](https://developer.mozilla.org/en-US/docs/Web/API/Window/fetch).
No external code review provider was used. Existing published alpha assets and
tags are preserved; the application fix is for a subsequent build.

The two Intel-budget failures were rechecked locally on Apple Silicon with their
original race/production-speed reference paths and assertions: the audio fixture
passed in 3.615 seconds and the coil fixture passed in 7.250 seconds. The
[failed Intel quality receipt](intel-quality.json) is retained.

## Final verification

All five jobs in [fresh run 41888150836](https://github.com/dshills/ELab/actions/runs/41888150836) passed on clean
`main` source `e5136106d1d49cb13e979f0ad9337bccd012f21a`. [Hosted receipt](hosted-ci.json).

- [Apple Silicon macOS 26.6.2](hosted-macos-quality.json): lint, 75 Python tests,
  36 frontend checks, complete Go race suite, vet and package/runtime checks passed.
- [Linux](hosted-linux-quality.json): the complete portable quality plan passed.
- [Hosted browsers](hosted-browsers.json): Chromium 16 passed; Firefox and WebKit
  each 15 passed with one planned load-test skip. Total 46 passed, two planned
  skips and no failed assertions.
- [Local macOS 27.2 arm64](local-quality.json): the same complete macOS gate plan
  passed with unchanged source inputs. [Local browser summary](local-browsers.json).

The initial overlapping dispatch was superseded; the Intel/slow-mirror attempt
remains historical failed/cancelled evidence. No old runs, release tags or release
assets were deleted. This follow-up documentation changes no qualified application,
test or workflow input, and skips an unnecessary new CI run.
