# Recording-analysis review resolution

Prism reviewed seven staged files with configured Wafer/GLM-5.3 defaults and the
owner-requested output allowance. Coverage is complete, with no excluded files,
skipped chunks or truncated input. It returned zero high, one medium and eleven
low findings. The receipt describes that pre-fix staged snapshot.

Accepted refinements:

- Explicitly treat an absent recording as stale before reading its result fields.
  The existing owner identity changes when leaving recorded mode, so the stale
  return already protected that path; the explicit check strengthens the boundary.
  Browser testing switched the source to Live in another tab: all analysis fields
  and refresh disabled, with no TypeError or console error.
- Normalize non-Error rejection values before displaying feedback.
- Assert the complete timing request, including captured probe and non-default
  direction. Cover unchecked overlay/checked statistics, capture-ID refusal for
  all actions, and final-page/previous-offset clamping.
- Replace multiple assertions on one line with one complete request assertion;
  check JSON-marshalling failures and report all missing embedded modules together.
- Quote the refresh control in prose and link the dated validation evidence and
  completed receipt instead of leaving the review-pending note.

The suggested checkbox string 'off' is outside the HTML FormData contract:
unchecked boxes are absent and checked boxes send 'on'. Both valid paths are
covered, including native JSON omitting false preferences on a browser round trip.
No change to checkbox semantics was needed.

Internal review also clarified that native pulse/cycle/transition summaries use
both directions while same-direction spacing follows its direction filter. Page
filters do not shorten the summary interval. Reopening/refreshing the dialog now
starts at the source header so the capture identity is visible.

Post-review changes were internally inspected and checked with all 23 frontend
tests/syntax checks, the complete web race suite and vet, plus the absent-source
browser check. These fixes do not claim a second external review. The full shared
race run and all-package vet passed before these presentation/test-diagnostic
refinements; the shared native runtime was unchanged.

Evidence: [frontend log](checks/frontend.txt), [web race log](checks/web-race.txt),
[combined race log](checks/shared-race.txt), [HTTP regressions](../../../exp-web/recording_analysis_test.go),
[comparison](recording-comparison-result.png), [step response](recording-step.png)
and [review receipt](recording-prism.json). Browser fixtures use disposable data
on port 8789; they do not modify the owner's persistent project library.
