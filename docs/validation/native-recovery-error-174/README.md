# Native automatic storage failure and retry on /174

October 19, 2026. **The actual /174 window retains its autosave warning through
a native resistor edit and clears it after an automatic complete retry.** The
original manual project, previous good capsule and complete kept recording
survive. No application repair or new package was needed.

One isolated owned session uses a synthetic RC project, private recovery folder
and ordinary two-second autosave interval. The manual source has a 10 kΩ
resistor and one 200 ms, two-probe recording. A subsequent dirty 20 kΩ circuit
produces the first automatic capsule. The controller moves only that owned
folder and temporarily places an owned regular file at its configured path.

| Stage | Observed native and structured outcome |
|---|---|
| Healthy | Fresh paused 20 kΩ RC response at 10 ms; complete automatic capsule exists. |
| Storage unavailable | Visible `Autosave needs attention` warning and the same detailed `not a directory` error returned by `recovery.error`. The full detail is visible after scrolling the recovery inspector. No circuit/history/physics/document change is caused by the failure publication. |
| Native edit during failure | Private `30k` typing leaves the entire accepted snapshot unchanged. One pointer Apply commits 30 kΩ and one circuit-history entry. Warning/detail remain visible; the old simulation trace is explicitly stale. |
| Private storage repaired | Ordinary automatic retry writes the complete current 30 kΩ document without Scan/Checkpoint. Warning and detailed error disappear; Last snapshot advances. Document remains dirty, and retained stale physics/history remain unchanged. |
| Save and close | A separate final project is saved; completed shutdown joins workers and removes the owned process/endpoint. No discard or forced exit occurs. |

The retry changes only graph content relative to the original manual document;
view, pause state, probes, initial conditions, models/settings and the complete
recording remain exact. Both old and new capsule hashes are present before final
manual Save/Close. Their original bytes are also retained outside the live
recovery store. Normal manual-save cleanup may retire live recovery entries;
that is separate from their proven preservation during the fault and retry.

Failure warning removal changes the canvas allocation. The resulting native
viewport report advances the command revision once; this is explicitly recorded.
Only viewport, recovery and observation/command counters differ after retry.
Graph, document dirty state, simulation, run/electrical identity, scope and
history do not change. Autosave does not pretend the explicit project was saved.

## Agent parity and stored data

One fresh exact packaged agent replays the 20→30 kΩ paused edit. Complete graph,
transient history/events and scope match the native before/after stages,
excluding independent run/electrical IDs only, with rel 1e-9 / abs 1e-12 numeric
tolerances. Its final saved project is **byte-identical** to the native final
project. Two more independent agent processes reopen the untouched manual
source at 10 kΩ and the final project at 30 kΩ, both clean/paused at live time
zero. Each retrieves all **209 original rows** through pages of 200 and 9, with
exact original metadata/data. All three headless processes close normally at 0.

Independent piecewise RC equations check every stored voltage/current row,
including all three records at each of four ideal source edges. Before/after
currents use the known input phase while capacitor voltage remains continuous.
The live 20 kΩ trace is checked against its own 20 ms time constant. The retained
recording still belongs to its original 10 kΩ capture; changing the live graph
does not rewrite or relabel its numerical provenance.

The original [prelaunch plan](producers/plan.json), action reconstruction, public
snapshots, target-window images/AX, manual/final projects, complete capsules,
reopen pages and exact producers are preserved. Stored JSON/text is losslessly
compressed; the manifest verifies both stored and original sizes/hashes. Images
are unmodified. Some scroll-position images contain an owned divider tooltip;
it does not obscure the warning or detailed error examined here.

## Boundaries and preservation

The launcher PID becomes the exact signed /174 native executable under its
existing offline sandbox, with a private endpoint and default recovery/appearance
setup disabled. Recovery is configured only for this private folder through
public commands. No ELab app was running before launch. The geometry-denial
canary and exact before/after shared hashes/modes confirm owner settings are
unchanged. The review/copy, canonical /120 and manual source hashes also match.
`open -W` exits 0; LaunchServices does not expose the native executable's own
exit status. The terminal exec handle and saved completed shutdown confirm
normal cleanup. No printer, display recording, other app, owner project or
privacy setting is touched.

This is one cooperating-directory fault and one automatic retry, not a new
five-second/p95 performance cohort, power-loss test, all-filesystem guarantee,
screen-reader run or novice-comprehension study. The raw filesystem wording is
visible; comprehension of it is untested. The broader FILE-02/Recovery and
accessibility/agent/environment acceptance scopes remain separate. There is no
promotion or owner acceptance.

Verify hashes, native/public state, capsule/project preservation, complete
paging, equations and normal closure without launching an application:

```sh
python3 -B docs/validation/native-recovery-error-174/verify.py
```
