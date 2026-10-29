# ELab first-use study packet

Protocol draft 1 · October 19, 2026 · **Not run; no participant observations.**

This makes the [PRD's formative test](../07-product-requirements.md#formative-hobbyist-test) runnable. Its six proposed gates and ten-person sample are unchanged. It is independent of the owner's [four-task review](../35-first-version-review.md); the owner walkthrough remains Not tried yet. Preparing this packet does not pass either gate or authorize contacting people.

Use the [participant cards](participant-tasks.md), a fresh [session form](session-form.md) for each consenting participant, and the blank [summary](results-summary.md). The [RC fixture](fixtures/RC-no-traces.elab) deliberately has no selected plot or cursor, so loading an example does not silently satisfy the plot-discovery task. Its [preflight](fixtures/preflight.json) is engineering evidence, not a user session.

## Decision and measures

The earlier [research plan](../06-method-and-next-research.md) suggested 8–12 interviews across a wider set of electronics/PCB jobs. This packet follows the later ELab PRD’s specific ten-hobbyist study and does not reuse competitor-research observations as ELab participant results.

The decision is whether the core hobbyist workflow is ready for broader MVP acceptance and which observed obstacles need correction. The primary measure remains **unassisted completion of draw → observe → adjust → measure → recover → save**, reported by experience group. The PRD gives no numeric threshold for that composite; do not invent one.

| Code | Existing proposed gate | Operational scoring |
|---|---|---|
| G1 · First useful circuit | At least 8/10 overall **and** 4/5 newcomers, within five minutes without facilitator intervention | From blank, produce an accepted LED current and change it by editing resistance. Use the time of the changed accepted reading, not first placement. All three conditions must occur by 300 s, with help level 0. Record first-current time separately. |
| G2 · Meaningful experiment | At least 8/10 correctly predict or explain why increasing resistance reduces LED current | Score the first explanation after T1, before teaching. It must identify the correct direction and a relevant reason at fixed supply; reciting a supplied answer does not count. Keep the verbatim response. |
| G3 · First waveform | At least 8/10 plot capacitor voltage and enter a cursor time without being told where the plot action is | The voltage trace was absent initially; the participant adds it, obtains time-varying samples, and types/commits a time associated with a visible point. Record separately whether the plot location was disclosed. |
| G4 · Recoverability | At least 8/10 repair an intentionally disconnected wire and undo an accidental deletion without losing work | Two distinct faults, two recorded outcomes. A repaired wire may use any successful strategy; the deletion must be reversed through Undo. Values and intended connectivity survive. |
| G5 · Result interpretation | At least 9/10 distinguish Running, Paused and a stale/failed result | Three prepared conditions, all three correctly explained. This round uses **stale**, not a claimed failed-solver fixture. Observe actual state before presenting each case. |
| G6 · Assumptions and ownership | At least 8/10 identify a relevant model limitation and save/reopen without missing dependencies | The participant identifies one applicable limitation, saves the intended experiment and reopens the same values/connectivity without retrieving models elsewhere. All three are required. |

For G1 and G3, apply the PRD's explicit help restrictions above. For the other gates, retain both observed completion and unassisted completion; do not silently replace their stated thresholds with a new assistance policy. Score G2 and G5 answers before any explanation from the moderator. Report help for every task, even when it does not change that gate's literal score.

For the primary full-loop count, require successful T1 construction/current change, T3 voltage plotting/cursor entry, T4 wire repair/deletion Undo and T6 save/reopen, with help level 0 in all of those task segments. This count has no five-minute limit beyond G1's separate gate. G2/G5 and the model-limitation outcome remain separate understanding gates; a mechanical full loop does not waive them.

Two guardrails outrank aggregate counts: investigate every instance where an invalid result is presented or treated as current, and every observed loss of saved work or corrupted connectivity. Distinguish participant misunderstanding from an actual application-presentation defect, but investigate both. An 8/10 or 9/10 count does not erase such an event.

## Before enrolling anyone

1. The owner reviews the protocol, safe study setup, consent wording/storage arrangements and sampling interpretation below. No recruitment has been performed.
2. Rehearse with the owner or a colleague. Mark this **pilot**, outside the ten-person cohort; a repeat run is not fresh first use.
3. Freeze one app/agent build, protocol revision and device/input setup. This draft's fixture was checked with clean **0.148.0 /165**, source `0e75875`, SDK `b22478c`. Record actual receipt/native hashes in every session. If repairs change the build, preserve the earlier results and report the new round separately.
4. Use only study-created projects in an owner-approved test account/session. Normal framework geometry uses shared account settings; the [separate-account request](../../docs/validation/startup-handoff-preflight-165/README.md) is unanswered. Do not use an ordinary launch in the owner's profile to imitate isolation.
5. Use a consistent Mac, display/window, theme and input setup for the main comparison. Record any accommodation or deviation; this study does not replace keyboard/VoiceOver or other-device qualification.
6. Decide who can access notes, how long they are kept, and how withdrawal is handled before consent is sought. Keep contact details/consent administration separate from coded observations. No real participant records belong in this source repository by default.

Use goals that do not reveal control locations, and rehearse a consistent discussion guide. Ask for thoughts without supplying a solution. These method choices follow the [GOV.UK moderated-testing guide](https://www.gov.uk/service-manual/user-research/using-moderated-usability-testing); the circuit tasks, scoring rules and order below are ELab-specific decisions.

Obtain understandable consent before notes or recordings, explain the intended use, and make stopping possible. Recording is optional; manual notes suffice. Agree on storage and withdrawal handling before collecting anything. See [consent guidance](https://www.gov.uk/service-manual/user-research/getting-users-consent-for-research) and [notes/recording guidance](https://www.gov.uk/service-manual/user-research/taking-notes-and-recording-user-research-sessions), reviewed October 19, 2026. This packet supplies no legal or organizational consent policy.

## Sampling and order

Plan ten actual hobbyists: five new to schematic simulators and five with circuit-building experience. Operational assumption for owner review: “new” means no previously completed schematic simulation; the second sampling stratum requires some circuit-building experience, without adding a requirement to be an experienced simulator user. These attributes can overlap. Assign each person to one sampling stratum **before observing outcomes**, record both actual experience attributes, and disclose overlap. The 4/5 G1 denominator is the preassigned five-person newcomer stratum; also describe any additional simulator newcomers in the other stratum.

Keep T1 first for everyone, then T2 and the first save segment. Counterbalance waveform and recovery blocks; keep final model/ownership checks last. The schedule is a plan, not ten enrolled participants:

| Planned slot | Sampling stratum | Middle blocks | Status-condition order |
|---|---|---|---|
| N1 | New to simulators | Waveform → recovery | R → P → S |
| N2 | New to simulators | Recovery → waveform | P → S → R |
| N3 | New to simulators | Waveform → recovery | S → R → P |
| N4 | New to simulators | Recovery → waveform | R → S → P |
| N5 | New to simulators | Waveform → recovery | S → P → R |
| B1 | Circuit-building experience | Recovery → waveform | P → R → S |
| B2 | Circuit-building experience | Waveform → recovery | P → S → R |
| B3 | Circuit-building experience | Recovery → waveform | S → R → P |
| B4 | Circuit-building experience | Waveform → recovery | R → P → S |
| B5 | Circuit-building experience | Recovery → waveform | P → R → S |

R/P/S are moderator-only Running/Paused/Stale keys. Show the participant unlabeled Condition A/B/C in the assigned order. Do not label a case “failed” unless it actually is.

## Moderation and assistance

Suggested planning allowance: about 45 minutes, plus setup; this is not a product speed gate. Rehearse and adjust the schedule before enrollment without changing G1's 300 s target. Say that the app is being evaluated and invite the participant to describe what they expect and notice. Explain that they may request help or stop.

Read one card at a time. A verbatim repeat of the card, neutral think-aloud prompt or planned between-task fixture setup is not a hint. Record task-relevant help at its first occurrence:

| Level | Definition |
|---|---|
| 0 | No task-relevant hint or intervention; participant may use help already in the product |
| 1 | General moderator nudge that adds information beyond the card |
| 2 | Control-location, action or conceptual hint |
| 3 | Demonstration or moderator operation of part of the participant's task |

Record the strongest level **and** exact words/actions, time, help requests and whether the plot action's location was given. A request alone is not assistance if none is supplied. Give requested help rather than concealing it to improve a score; preserve the unassisted failure and continue learning from the assisted attempt. Do not reset the clock after a wrong action or omit troubleshooting time.

T1 timing starts after the complete prompt is read and control is handed over. Record first accepted current and the first accepted changed current. If success occurs after 300 s, G1 fails but later unassisted completion can still contribute to the full-loop measure. If no success occurs, leave success time blank and record stopped duration/reason. Do not encode “not observed” as zero seconds.

## Session sequence and fixture setup

1. **Introduction/background:** confirm consent and optional recording separately. Assign a code, not a name in task notes. Record simulator experience, circuit-building experience and input/accommodation needs without teaching this interface.
2. **T1/T2:** start from a genuinely blank sheet. Do not preload an LED circuit or show the expected current. Ask the explanation only after T1, before explaining resistance/current.
3. **T6 save segment:** ask the participant to keep their LED experiment for later. Record Save/Save As independently; do not save it for them and later call ownership unassisted. Preserve this file during later fixture setup.
4. **T3 waveform:** between tasks, load `fixtures/RC-no-traces.elab` into the dedicated study session. It starts running with scope visible, no selected trace/cursors/component, and a valid 10 kΩ/1 µF RC network. Its ideal 0–5 V source repeats 50 ms high/50 ms low, with 1 ms output steps and 0.02 simulated seconds per wall second, so late plot discovery still has a recurring waveform. Do not select the capacitor, point at Plot or show a completed scope. If setup differs, stop and record a fixture deviation rather than scoring discovery on an already plotted trace.
5. **T4 recovery:** use the participant's saved LED. If waveform came first, ask them to reopen that file themselves and score the reopen segment before setup continues. With consent, introduce one wire disconnection through the native UI, then hand control back. After repair, introduce a separate accidental component deletion through the native UI; ask for recovery. Native setup avoids creating “Undo agent” history that would change this task. Record exactly what was removed and the participant's strategy. Preserve their saved file and values.
6. If recovery came first, have the participant save the intended repaired circuit before the waveform block, then reopen it themselves afterward. Never discard their unsaved changes during fixture setup.
7. **T5 status:** use separate study data, not an unsaved participant document. For each condition, reset to the valid RC fixture and add `v:C1` for a visible trace. Running: allow accepted samples to advance. Paused: pause after a valid trace. Stale: pause a valid run, then change C1 from 1 µF to 2 µF without restarting. Check the displayed/structured state is actually stale before asking. Restore the baseline before another case; don't infer truth from the scenario name. The packaged-agent [state preflight](fixtures/state-preflight.json.gz) validates preparation, not human interpretation or visible UI acceptance.
8. **T6 limitation/final ownership:** return to their experiment. Ask for one relevant limitation and confirm the saved/reopened circuit retains intended values/connectivity without fetching dependencies. Have them Save As into study storage if further changes were made. Never overwrite a personal project.
9. **Debrief:** record confidence and satisfaction separately from task success, plus the most confusing moment and desired improvement. Do not teach an answer until its first response has been captured. Thank them and confirm the agreed handling of notes/recordings.

Moderator answer key: higher resistance at fixed supply reduces LED current; “about half” after 1 kΩ → 2 kΩ is acceptable, but exact linear scaling is not required for this diode model. A relevant limitation could be that the LED is generic, has no calibrated brightness/thermal/rating behavior, or does not establish physical-hardware accuracy. Running advances accepted time, Paused holds a valid accepted state, and Stale readings cannot describe the edited circuit as current. Preserve the participant's words rather than merely checking an answer box.

## Counting and release review

Use raw observed outcomes, never the fixture preflight, synthetic tests or the owner's pilot, for the ten-person counts. Record completed, stopped, not attempted and technical-blocked tasks distinctly. A task not performed is not a success. Do not drop a product failure, slower participant or assisted session to improve a denominator. Record withdrawals and technical interruptions under the agreed consent policy; document any replacement and its reason without outcome-based substitution.

Until ten eligible sessions and the planned five/five strata are available, mark the gate decision **incomplete**, not pass. Report observed counts and missingness without presenting empty cells as 0/10 failures or fabricated successes. Preserve original observations if a later fix or retest occurs; a previously exposed participant is not a new first-use observation. This is a formative sample, not a population preference/ranking estimate.

The owner reviews the completed summary, issues and corrections. Meeting these six proposed gates does not establish complete agent coverage, numerical fidelity, installation, accessibility or visible-p95 performance. Manual task timing is not input-event-to-pixel latency.


## Check the packet

```sh
python3 -B research/first-use-study/verify-packet.py
```

This read-only check binds the packet to the unchanged PRD study section, pinned historical review receipt and archived source/SDK ledger. It verifies the archived empty-plot fixture and recorded status preparations, and reports zero measured participants with a null study decision. It neither runs the old binaries nor qualifies the current application. Current-release workflows are checked separately with the matching packaged agent; no historical observation is relabeled as a new participant session.
