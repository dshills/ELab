# Packaged project-job retry check

October 19, 2026. Frozen /173 packaged agent, SHA-256
`134f9863e2885e384035e30550adfc5fe0bea1bdee9f3d3cd42feaf802bb7de5`.

The [bounded public check](../../../scripts/release-project-jobs.py) passes six
scenarios over headless stdio with actual private temporary project files:

1. Invalid Open retains a failed job without changing committed LED source,
   document, physics, scope or history.
2. Repairing that file and retrying the exact original key retains the original
   failure and job identity; it does not read or install the repaired RC file.
3. A new key opens the repaired RC graph cleanly, leaves both source files byte
   identical and retains the historical failure for exact inspection.
4. Retrying a completed `project.load.start` returns its original queued admission;
   exact job inspection still reports completion.
5. Terminal and unknown job cancellation refuse `operation-unavailable`, preserving
   committed source and command revision.
6. Normal `app.close` reports joined workers/readiness; normal EOF exits 0 with
   empty stderr.

The final transcript has 26 requests. No native window, display capture, recovery
configuration, shared preference access, network or print submission occurs.
This release-pinned headless evidence complements the existing controlled
in-flight cancellation/publication tests. It does not qualify native controls,
transport disconnect races, every job family or platform/window teardown.
AGENT-04 and the wider PRD audit remain partial. No app/SDK/model/format or public
command changes are made.

The initial producer incorrectly refreshed its current command revision from
`project.work.inspect`, whose revision describes the job's original admission.
The following start request correctly refused a conflict (expected 7/current 8).
That failed producer/declaration/transcript is retained in `initial-client-failure`.
Its EOF cleanup returned without an additional exception, but that initial
process's exit code was not saved. The client now refreshes revisions only from
full current snapshots, as the existing agent documentation requires. The
corrected run passes; the final run additionally asserts required protected
snapshot fields and checks history availability/names. Both passing runs and
their exact hash-checked producer bytes are retained.

Replay saved decisions/provenance without launching a process:

```sh
python3 -B docs/validation/project-job-release-173/verify.py
```

Repeat the real check in a new output directory:

```sh
python3 -B scripts/release-project-jobs.py \
  --agent /private/tmp/elab-first-version-173-signed-runtime/elab-agent \
  --sha256 134f9863e2885e384035e30550adfc5fe0bea1bdee9f3d3cd42feaf802bb7de5 \
  --output build/a-new-project-job-check
```

The producer declares its hash and pinned binary before launching its owned
process, bounds responses/waits, records every request and cleans up by normal
EOF even after a failed assertion. The actual files are disposable and removed
after each run; neither original owner projects nor ELab settings are involved.
