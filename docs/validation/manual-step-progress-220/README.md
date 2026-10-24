# Manual-step planned sample counts /220

October 24, 2026. [Receipt](report.json), [source inputs](source-inputs.json),
[raw evidence](raw.tar.gz), [member hashes](raw-manifest.json),
[notes](../../../research/99-manual-step-progress.md).

Three /219 public runs complete with misleading 1/2 or 3/4 counts. Three source
regressions fail at admission. The repair counts the same bounded absolute-time
targets the worker already uses, including its unchanged sub-picosecond final
tail rule. Target calculation is shared; equations, limits and physical timing
remain unchanged. Seven source cases include genuine partial intervals, a
nonzero origin, an adjacent endpoint and the 1,000-sample boundary. They check
counts, exact end time, analytical RC voltage and protected source/history.

The first full gate fails when the existing direct flasher numerical test hits
the production 100 ms work budget after 27 attempts. That path does not invoke
manual stepping. The log is preserved; no scheduler cause is claimed. The
numerical test now uses the existing test clock while retaining its attempt
cap. Production budget code and dedicated time/attempt enforcement tests are
unchanged. Combined focused race checks pass in 2.398 s.

The final full gate passes lint/staticcheck, 67 Python tests, root/desktop race
and vet, configured mounted race and offscreen vet. Desktop race: 187.764 s;
mounted race: 168.829 s. All 890 source/check inputs match. The earlier plot
routing regression is included. The SDK and 126 production desktop Go sources
are unchanged. Initial test-fixture command/stale-run mistakes are retained
separately. /219 and /205 stay preserved; clean packaging follows.

The [clean /220 candidate and matching agent](candidate-report.json) pass
eight complete-count cases, exact physical comparison with five /219 traces,
sixteen analytical RC readings and unchanged over-bound refusal. All owned
workers join and exit 0. The candidate remains held for the unchanged
[native acceptance list](native-acceptance-pending.json); /205 stays current.
