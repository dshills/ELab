# Switch and wiper command behavior

Filtered `capabilities.list {"command":"switch.set"}` and `capabilities.list {"command":"potentiometer.wiper.set"}` describe their target inventories, value bounds, state requirements, effects, errors and completion. The native inspector uses the same concise explanations. The ordinary catalog and numeric output schemas are unchanged.

| Control | Fresh network transient, running or paused | DC, stale or incomplete drawing |
| --- | --- | --- |
| `switch.set {id,closed}` | Save desired state immediately. The old applied circuit advances one accepted sample before the pending switch group applies. Paused changes wait for Step/Resume; `simulation.wait` alone leaves them pending. | Ordinary saved startup edit; running changes recalculate, paused readings remain stale until Restart/Resume. |
| `potentiometer.wiper.set {id,position}` | Admit background `input_work` at the current simulated time. Commit the wiper and its event only after a successful solve, preserving capacitor voltage and the previously applied switches. | Ordinary saved startup edit under the same recalculation/paused-stale policy. |

`closed` is a required Boolean. `position` is a finite fraction from 0 to 1; 0.5 means 50%. Target IDs come from current components of the matching kind. IDs are exact case-sensitive identifiers up to 64 ASCII characters. Unknown/wrong-kind targets, missing/null or malformed fields and out-of-range values refuse before mutation.

Switch snapshots distinguish desired, applied and pending state. Repeated desired values are no-ops, and returning to the applied Boolean coalesces a pending change away. A failed/canceled physical advance preserves the committed time/charge and the saved desired state remains pending. Restart uses that desired state as startup. Save and Verify also capture desired startup, while the live symbol/readings follow applied state.

A wiper admission can return before calculation finishes. Require the matching `input_work.id`, `command`, `requested_wiper` and completed status before a dependent Step. Only the latest input job is retained; these IDs are separate from retained verification operations. Cancel through `simulation.work.cancel` with the exact active ID, or Pause. Failure, cancellation and supersession preserve committed values and history. The five-second cooperative deadline remains. Setting the already committed position cancels a pending wiper request for that same potentiometer; otherwise an unchanged value preserves physical state/history.

While wiper work is pending, the inspector shows the requested percentage but labels the electrical reading **Committed · wiper change pending**. The values still describe the committed circuit. Stale readings retain their Stale label. Native percentage controls use float32; recommitting an unchanged represented value now leaves a more precise stored fraction intact rather than creating a rounding edit. This does not claim unrestricted precision for newly typed native values or complete typing/IME qualification.

Both controls preserve run/electrical identity during live actuation; actuation revision tracks changes separately. Direct switch configuration is undoable immediately. Wiper configuration/history commits on successful publication. Atomic `edit.batch` and undo/redo use ordinary schematic/startup rules rather than live actuation. Saved wiper values are committed values, not an unfinished request; Verify starts from captured configuration rather than replaying live movement history.

The native builders now live in `desktop/actuation_controls.go`; field/action IDs and dispatcher routes remain. Tests cover shared hints, requested/committed/stale labels, rendered revision guards, exact stored-value preservation, endpoint/malformed input, pending-switch Wait semantics and controlled wiper supersession without partial publication. Existing blocked-job tests cover failed staging, Save during staging, cancellation, geometry merging and ownership.

The public reference constructs a switch/potentiometer/capacitor circuit, exercises queued and same-time changes, checks the Thevenin RC response, tests batch startup/history and fresh-process save/open, and exports without changing source or physical state:

```sh
python3 scripts/actuation-contracts-demo.py \
  --output build/actuation-report.json --artifacts build/actuation-assets
```

Use fresh output paths and an owned blank session. Optional `--connect` closes that owned native window unless `--keep-open` succeeds. Full physical input/accessibility, visible latency, broad circuit/device and release acceptance remain separate gates.

Packaged checkpoint: **0.99.0 /114**. [Final receipt](validation/actuation-contracts-validation.json), [headless workflow](validation/actuation-contracts-headless.json), [native workflow](validation/actuation-contracts-native.json), [lossless discovery witness](validation/actuation-contracts-discovery.json), [adapter conformance](validation/actuation-contracts-conformance.json), and [bounded visual review](validation/actuation-contracts-visual-review.json).

The visual session showed the circuit and successful AX object navigation, but automated wheel/scrollbar attempts did not expose the lower controls. Physical unchanged-value commits and the newly added hint layout are not qualified by that session. The exact-value repair is covered at the real native field callback; no full mounted typing/scrolling claim follows. Existing model/kernel equations and quotas are unchanged.

The 0.100.0 inspector pass moves actuation controls ahead of whole-network explanations. In a mounted paused RC fixture, Tab/Return on the unchanged displayed wiper percentage preserved the precise agent-set fraction and physical/history state; callback tests separately cover an emitted unchanged event. Tab/Space also coalesced a pending switch back to its applied state without advancing time. These are bounded physical attempts, not full typing/accessibility acceptance. Long explanatory labels still clip in the narrow rail. See [current-run review](../research/ui-audit/keyboard-inspector/README.md).
