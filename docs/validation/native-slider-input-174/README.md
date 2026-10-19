# Native potentiometer input on /174

October 19, 2026. One owned isolated 0.157.0 /174 app session exercised an
actual CUA pointer drag while paused, a drag while running, native Undo and
Pause, and one held-click attempt containing an external agent edit. The exact
copied signed native and agent hashes match the preserved release receipt.
The circuit is a synthetic 0–5 V, 10 Hz pulse driving a 10 kΩ potentiometer
and 1 µF capacitor. No production source, SDK, model, command or package changed.

| Case | Observed result |
|---|---|
| Paused drag, 50% → 75.1012% | One user command/history entry. Time remains 5 ms and capacitor voltage remains exactly 2.161685253823 V. Completed input work publishes fresh results. |
| Native Cmd+Z | Restores 50% and the original graph; one redo entry records the user adjustment. Retained results are explicitly stale. |
| Running drag, then native Pause | One user adjustment at simulated 680 ms; Pause at 691 ms. Run/electrical identities stay unchanged. All three retained rows at the actuation instant have identical capacitor voltage while current changes. Final readings are fresh. |
| Held click plus agent edit | The strict single-attempt agent request, guarded at revision 18/document generation 4, commits 80% at revision 19. It completes between CUA call invocation and return. The final visible control is 80%, history belongs to the agent, and no late user command overwrites it. Charge and time stay unchanged. |

Five native stages match one fresh packaged headless replay: complete
components, wires, mixed results, transient history/events and scope fields,
excluding independent run/electrical identities only. Numeric comparison uses
the existing rel 1e-9 / abs 1e-12 tolerances. The headless process exits normally
at 0. The running case compares all 512 retained samples, not just its endpoint.

An independent piecewise Thevenin solution checks the pulse edges and wiper
change with continuous charge. It uses the declared track model's two 1 Ω end
resistances, rather than an ideal zero-end-resistance substitute. Across the
retained native sample sets the largest voltage error is 24.8628 µV, within the
existing 2 mV envelope. Duplicate/overlapping snapshots are not independent trials.

The held-click record establishes controller/agent call order and the final
accepted state. It does **not** timestamp OS mouse-down/up or establish an exact
physical interleaving. A click held at a new position is distinct from moving a
held thumb. Window blur was not attempted: the available CUA interface exposes
atomic drag/click operations without a separate down/move/up sequence, and no
reliable deactivation sequence was established. Document replacement during a
physical gesture, keyboard slider operation, reader/IME and broad input
qualification remain open. Earlier retained-widget guard tests keep their own
scope. EDIT-03 and SIM-02 remain partial; this is not a latency cohort or owner
acceptance.

## Preservation and cleanup

The launcher execs the exact copied /174 native binary under its existing
offline sandbox, with a private endpoint and recovery/appearance disabled.
Its PID/executable are checked before semantic control. A denial canary confirms
shared geometry cannot be read or written; framework stderr includes the
expected denied geometry writes. Exact shared geometry hashes/modes, copied
bundle/review, original baseline and canonical /120 are unchanged.

Cleanup saves `Final-Pot-174.elab` separately, closes through public `app.close`
without discard, and observes joined workers, finalized recovery, completed
shutdown, absent owned PID and removed endpoint. `open -W` exits 0; the native
process's own exit code is unavailable through LaunchServices. The original
producer report retains its `ready_for_cua` status after cleanup; its explicit
end/closure fields and the original exec handle establish completion. No forced
exit, other-app control, display recording, printer job or settings grant occurs.

The setup's default Swift cache permission refusal and ambiguous CUA bundle-ID
binding refusal are recorded in `action-record.json`; the successful retry uses
a build-local cache and the exact observed running app URL. Neither refusal was
bypassed. One oversized CUA result was inspected from its already-saved files
without repeating input. These are setup/reporting limitations, not app defects.

## Evidence and replay

`native/` preserves original snapshots, projects, AX text, target-window images,
strict request/response, controller timing and cleanup. `reference/` preserves
the fresh agent replay. JSON/text gzip files decode to exact original bytes;
the manifest checks both stored and original sizes/hashes. Images are unmodified.
`action-record.json` is a post-run reconstruction of actual CUA calls, not a
timestamped OS event log. `producers/` holds the exact session, control,
interleaving, replay, launcher and sandbox sources; these retain their historical
paths and are not an unattended rerun recipe.

Verify stored evidence, parity, history, continuity, RC law and cleanup without
launching an app or reading owner settings:

```sh
python3 -B docs/validation/native-slider-input-174/verify.py
```
