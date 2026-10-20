# Actionable circuit diagnostics

Known affected components have an outline and **!** marker; known terminals have a larger ring; known wires have a thicker highlight. The Inspector's **Circuit guidance** supplies readable explanation and next action, with **Technical details** as an explicit disclosure. The source object/terminal/wire IDs are also structured in the snapshot. Markers do not select, move or alter objects. A temporary wire gesture continues while the disclosure opens, and correction/undo removes obsolete markers.

Compiler diagnostics retain their specific connection/model explanation. Solver failures use a brief summary with the raw numerical reason kept in the technical section. Failed manual steps explain that committed time and charge were preserved; a failure from an older run is identified as previous work. Source-sampling warnings identify the actual source and point to the existing resolution action. Unknown solver causes receive no invented location.

The existing `diagnostics` and `source_warnings` arrays remain available. `snapshot.diagnostic_presentation` adds `details_visible` and `items`, each with source, code, severity, summary, next action, technical details, `object_ids`, `terminal_refs` and `wire_ids`. Current net IDs resolve to their actual members/routes. A malformed or missing identity is not silently substituted with the editing selection. Known LED shorts now include their actual part identity, and specialized LED solve failures use the compiled LED ID rather than assuming `D1`.

## Rejected request context

Applicable failed command responses include an optional `help` object with the same structure. It describes related **request intent**, rather than pretending those parts caused a solver failure. Valid supplied object/terminal references can be highlighted; unknown IDs are not fabricated. The original error code/message remain unchanged. Producing this context does not change the graph, physical state, actor, document, history or service revisions.

Native request context is local to that response and scoped to its command revision. Opening its technical disclosure retains the explanation. An intervening user/agent command clears obsolete local context; numerical publication alone does not. Requests from an external agent expose the same help to that agent without mutating global diagnostics or another client's failed-request display. This preserves the failed-mutation atomicity contract.

Rejected `recording.seek` positions explain the selected capture's actual whole-index and time ranges, with `ms`/`us` unit guidance. A single-sample DC capture instead instructs index `0` or time `0 s`. Original error codes and detailed validation messages remain available, and the guidance adds no circuit highlights or state changes. Native time-field refusal displays this same response help; correcting the entry clears it and permits the requested recorded step. This source follow-up is not included in the unchanged /189 alpha.

[Source validation](validation/recorded-position-guidance/report.json) preserves the original generic-message failures and the passing full repository check. Ten public refusal cases cover RC/DC syntax, fractional indices, bounds and ambiguous positions; the native view/callback test checks identical agent help, unchanged refused state, and correction followed by Previous. This adds no new packaged-window or full accessibility acceptance.

## Shared command and state semantics

`diagnostics.details.set {"visible":true}` is the same command used by the native disclosure; `false` closes it. It requires normal revision/retry admission and changes session presentation only. It preserves physical time/state, editing selection, unfinished wires, undo/history and dirty flags. New/load/example/recovery replacement clears disclosure state. No diagnostic interaction launches a modal during an incomplete circuit gesture.

Core tests cover open-terminal/short/known net/wire associations, conservative unknown handling, actual renamed LED solve failure identity, rejection purity, ownership, disclosure invariants and reset. Native tests verify shared controls, preserved wiring, failed-value explanations and automatic clearing after an agent correction. Built/packaged GPU smoke rebuilds resources, checks actual diagnostic pixels, completes the connection and verifies those pixels disappear. Public stdio/private-socket workflows use the same paths:

```sh
./scripts/check.sh
./scripts/build.sh
./build/elab --smoke
python3 scripts/diagnostics-demo.py --output docs/validation/headless-diagnostics.json \
  --artifacts docs/validation/diagnostic-artifacts
# Owned development session only: replaces its circuit.
python3 scripts/diagnostics-demo.py --connect build/diagnostics.endpoint.json \
  --output docs/validation/desktop-diagnostics.json \
  --artifacts docs/validation/desktop-diagnostic-artifacts
```

These engineering checks do not establish novice discoverability, complete keyboard/focus or screen-reader qualification. No unsupported topology is made solvable by diagnostic presentation; existing models and numerical/work limits remain unchanged.

General transient work limits now distinguish attempt exhaustion from elapsed-time exhaustion, with safe recovery actions and unchanged caps. See [integration limits](integration-limits.md).

Failed captured Verify operations now expose the same explanation/action/technical structure through `operation.failure_help`, including in snapshot operation summaries. The Verify inspector displays it with the existing shared technical disclosure. These failures describe the captured input; they do not add guessed current-object markers or replace live diagnostics. Later source changes are labeled as an earlier captured experiment. See [Verify failure guidance](verification.md).
