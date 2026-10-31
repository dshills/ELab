# Native Return ordering on build 307 source

The remaining ordering portions of historical checks **9 and 11 passed** on
Apple Silicon macOS 27.2. Check 9 used recorded seek time. Check 11 used both
Verify fields, the DC sweep start, stop and point count, both threshold levels,
and the recorded timing interval start and end. The earlier current refusal
and correction cases are documented in the [recorded-position report](../../native-recorded-position-223/README.md).
The [uninstrumented app observations](../../native-workspace-306/native-input.md)
cover a person starting a private edit before an agent changes the accepted field.

Each trial typed the literal INVALIDQUEUE into a real, focused native field and
pressed Return through the desktop input tool. A Go build overlay added one
bounded FIFO barrier ahead of the normal Vellum field callback. The callback,
captured field value, parser and revision checks were otherwise unmodified.
The controller waited until the barrier was active **and** the Return callback
was posted, committed a new editor draft over the matching external agent's MCP
connection, then released the barrier. Application-local timestamps record entry
to the original parser and its return. The agent's command revision advanced by
one and was still the exact current revision after delivery in all ten cases.
Accepted verification settings, document, circuit, recording, scope and live
physics also matched the agent's accepted state.

The stale text was absent from the visible fields after Return. Each fresh
native correction then succeeded. Independent final readback showed a 3 ms seek,
20 ms Verify duration, 500 µs sample interval, 1–4 V sweep with seven points,
thresholds of 3 and 4 V, and a 1–450 ms timing interval. No barrier timed out.
The diagnostic window and controller exited normally. This is native driver
queue evidence from the instrumented executable. The release package has no
barrier, sentinel, receipt writer or test controller.

[Ten event sequences](observations.json) · [Final readback](fresh-state.json) ·
[Binary and SDK identities](identity.json) · [Exact overlay](diagnostic.patch.txt.gz) ·
[External controller](controller.py.txt).
