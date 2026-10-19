# Native capacitor and inductor startup input

October 19, 2026. One isolated owned session of the exact copied **0.157.0 /174**
app exercised private startup entry followed by a single pointer click on
Restart or Reset. A fifth case placed one strict agent edit inside a held Reset
click call. The [prelaunch plan](producers/plan.json), original target-window
images/AX, accepted snapshots and producer sources are preserved.

| Case | Result |
|---|---|
| Capacitor, private 3 V → Restart | Typing leaves accepted startup 2 V and the 5 ms result unchanged. One Restart click commits 3 V and restarts at t=0, capacitor voltage 3 V. Command revision 18→20; history tip belongs to the user. |
| Capacitor, private 4 V → Use experiment initial voltage | One click restores inherited startup, displayed as 0 V, with the previous 5 ms trace visibly stale. Command revision 23→25. |
| Inductor, private 30 mA → Restart | Typing leaves accepted startup −20 mA and the 5 ms result unchanged. One Restart click starts at t=0 and +30 mA. Command revision 28→30; history tip belongs to the user. |
| Inductor, private 40 mA → Reset startup to zero | One click clears explicit startup, displays 0 A and truthfully marks the previous result stale. Command revision 33→35. |
| Private 30 mA, held Reset plus agent 40 mA | The single agent request guards revision 38/document identity, completes at revision 39 inside the CUA click call, and remains the final accepted 40 mA. No later user command overwrites it. The retained result stays stale; its sample is unchanged. |

All five private-entry snapshots are exactly equal to their preceding accepted
snapshots, including revisions, graph, physics, document and history. Screenshots
show the private entries; the legacy capacitor stepper's AX value continues to
report its accepted value while text is private. Inductor AX reports its private
text. This distinction is preserved rather than using AX alone as commit proof.

One fresh packaged headless agent independently replays each case. All **ten**
before/after stages match complete components/wires, transient history/events,
scope and mixed results where present. Only independent run/electrical IDs are
excluded; numeric comparison uses rel 1e-9 / abs 1e-12. Final history-tip names,
actor attribution and command deltas are checked separately. The ordinary Reset
replay includes the private draft followed by default, but intermediate native
history entries were not individually undone. No broader history-stack claim
is made. The fresh agent ends in completed joined shutdown and process exit 0.

Independent equations check every retained row for the 10 kΩ/1 µF RC and
100 Ω/1 H RL fixtures while their 5 V pulse is high. RC uses
`V(t)=5+(V₀−5)exp(−t/0.01)`; RL uses
`I(t)=0.05+(I₀−0.05)exp(−t/0.01)`. Branch voltage/current and stored energy
also match. The largest errors are 9.137e-11 V, 9.137e-13 A and 6.892e-15 J.
Stale rows are checked against their original accepted startup, not the newly
changed graph startup. This is a simple-fixture reference check, not general
device, topology or timing qualification.

The held case records CUA invocation/return and public-request timestamps, not
OS mouse-down/up. No public old Reset request or visible conflict was observed,
so the evidence establishes preservation across that call without attributing
the outcome to a particular rejection layer. Separate source-mounted schedules
retain their original guard scope. Exact physical interleaving, blur, keyboard
variations, invalid-entry correction, reader/IME and full binding acceptance
remain open. There is no latency cohort or owner-acceptance claim.

## Isolation and evidence

The exact launcher PID becomes the pinned copied native executable with a
private endpoint, recovery/appearance disabled and the existing offline sandbox.
No ELab app was running before launch. The inherited launcher bundle name still
mentions the earlier slider test; its display name, actual executable, PID and
endpoint identify this startup session. No alternate app was controlled.

Normal cleanup saves `Final-Startup-174.elab` separately, calls `app.close`
without discard and observes joined workers, finalized recovery, completed
shutdown, absent owned PID and removed endpoint. `open -W` exits 0; LaunchServices
does not expose the native process's own exit status. Both baselines, copied
bundle/review, canonical /120 and shared geometry hashes/modes stay unchanged.
The geometry denial canary and expected denial messages in stderr remain saved.
No forced exit, other-app interaction, display recording, printer job or
privacy/settings change occurs. The producer's status remains `ready_for_cua`
after cleanup; its explicit end/closure fields and terminal exec handle prove
normal completion.

`native/` and `reference/` preserve exact output bytes, with JSON/text compressed
losslessly. `manifest.json` checks stored and original hashes/sizes. Images are
unmodified. `action-record.json` reconstructs actual CUA calls after execution;
it is not a timestamped OS event log. `producers/` holds the historical launch,
control, strict interleaving and public replay code. These sources retain their
original private paths and are not an unattended rerun recipe. No app, SDK,
command, model or package changed; EDIT-03/SIM-02/AGENT-01 remain partial.

The repository check found that the preceding slider evidence accidentally
stored a standalone source copy as `potentiometer.go`. A reproduced root-package
build failure is retained under `repo-hygiene/`. Renaming it to `.go.txt` keeps
the original bytes and hashes while excluding it from Go package discovery.
That earlier manifest's storage path is corrected; its original bytes are also
retained here. This is an evidence-storage repair, not an app repair. Root
package discovery, tests and vet are checked after the correction; unchanged
native suites are not repeated.

Verify preserved evidence and equations without launching an app or reading
owner settings:

```sh
python3 -B docs/validation/native-startup-input-174/verify.py
```
