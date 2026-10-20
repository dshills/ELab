# Paced shutdown observation in the example client

The reusable Python example client now observes an exact shutdown job with a
50 ms interval and a 15 s cooperative budget. It returns either completed or
failed metadata, preserving failures for an explicit caller decision. It never
acknowledges, cancels the accepted close, or owns client/process cleanup.

The sealed /176 native controller made excessive immediate `events.poll`
requests after closure. That transcript and package remain unchanged. This was
an ad-hoc controller defect: the existing shutdown demo already slept between
inspections. The new helper consolidates that behavior and documents that a
closed event response does not provide backoff or establish exit readiness.

- All **65 Python tests pass**, including 11 focused tests for pacing, identity,
  cancellation, budget limits, transport failure and agent-binary selection.
- Removing the sole sleep in an isolated copy makes the request-count regression
  fail at 20 requests. The normal fake-clock case makes four requests over a
  0.2 s budget. These are deterministic pacing tests, not measured UI latency.
- The new CLI flag initially passed `None` through to prefilled client factories.
  The preserved failing test led to omission of the keyword when unspecified.
  Final tests verify both omission and explicit override.
- Final-source success/failure demos run against the exact existing /176 agent:
  **55 raw requests**, no event polling, both agent exits 0. Failure uses two
  inspection requests (one helper read and one later comparison), then a separate
  deliberate acknowledgement. Saved 10 kΩ source, retained 20 kΩ recovery and
  current 30 kΩ graph remain distinct; protected failure-fixture bytes survive.

The budget/cancellation checks occur between synchronous RPCs; they cannot
interrupt blocked transport IO. A successful started close can exit before a
later inspection, so use waiting `app.close` when its final reply is needed.
EOF propagates and does not certify completion. Python 3.10 syntax is accepted;
execution used the installed Python 3.15, not a multi-version runtime matrix.

`report.json` records the scope and release pins. `sources/` retains final and
negative-control inputs; `checks/` keeps successes and expected failures;
`live/` preserves protocol, process reports and actual fixture bytes. Earlier
pre-correction explicit-agent runs are identified in the report but do not
qualify final source. The live producer and bridge scripts are archived text,
not instructions to rerun or alter historical evidence.

Run `python3 -B docs/validation/shutdown-client-wait-176/verify.py` from the
repository for read-only integrity and protocol replay. The manifest excludes
itself and its checksum checkpoint. Current-source checks intentionally fail
after a future legitimate application change; the captured protocol stays
historical evidence. No native window, app release, SDK/model/format change or
new timing/owner/full-MVP acceptance is claimed. All 650 application Go sources,
358 packaged runtime inputs and pinned artifacts match /176 at this checkpoint.
