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
and identify macOS arm64 capacity constraints. Routine quality now explicitly
uses `macos-15-intel`, a supported
[GitHub runner label](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).
The full macOS gate plan is unchanged. Apple Silicon shipping qualification
remains on arm64: local macOS release gates and the separate `release-web.yml`
workflow still target Apple Silicon. Intel CI is not evidence that the published
Apple Silicon ZIP runs on Intel.

Internal review covered early cancellation order (WebKit cancels before pagehide),
cancelled navigation, back/forward restoration, delayed responses, single-chain
ownership, strict error assertions, command isolation and runner architecture.
Lifecycle guidance: [pagehide](https://developer.mozilla.org/en-US/docs/Web/API/Window/pagehide_event),
[pageshow](https://developer.mozilla.org/en-US/docs/Web/API/Window/pageshow_event),
and [fetch cancellation](https://developer.mozilla.org/en-US/docs/Web/API/Window/fetch).
No external code review provider was used. Existing published alpha assets and
tags are preserved; the application fix is for a subsequent build.

Fresh hosted results will be recorded after the corrected main revision completes.
