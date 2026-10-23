# Actionable component-value refusals

October 23, 2026. First overnight quality item; the qualified /204 package is
preserved. This addresses the EDIT-03/HELP-01 workflow without changing models,
bounds, parsing, commands, error codes or accepted circuit values.

## Reproduction

In the /204 keyboard walkthrough, applying 0 to a resistor retained the valid
2 kΩ circuit and 1.477 mA reading, but displayed only “That request could not be
applied.” [Native baseline](ui-audit/first-version-completion-204/02-invalid-value.jpg).
The existing structured error already knew the resistance range; the generic
request presentation hid that information from the prominent feedback.

Focused baseline regressions reproduce generic feedback for all eight numeric
component kinds and miss the actual sine-offset/pulse-low constraints. The
original failures remain in `build/value-guidance-205-tools/before.txt`.

## Repair

For a strictly decoded `component.set` request targeting a known numeric part,
request help first confirms that the same pure value validator rejects the value.
The response then names the component and parameter, shows its published range
and accepted unit syntax, and states the prior accepted value. Sine amplitude
help incorporates the current offset; finite pulse high-level help incorporates
the current low level. The service's existing lock gives this help the same
accepted target/settings context as the rejection.

For example, a rejected R1 value now starts with “R1 needs a valid resistance.”
The action gives 10–100000 Ω, the supported k/K/M spellings and the previous
accepted value. The original technical error remains unchanged in `details`.
Known object highlighting and the original error code remain available.

Malformed payloads, unknown targets and unrelated refusal causes retain generic
help. No numeric cause is invented for them. Failed requests are not newly
retained as accepted idempotent mutations: a stale retry still conflicts, the
original help remains immutable, and a fresh refusal describes the new accepted
value after another actor's edit.

## Checks

- Focused root race checks cover all eight numeric domains, bad syntax,
  representability/range failures, current waveform coupling, unchanged full
  state and technical errors, malformed requests and actor/retry context.
- Native binding/view checks compare the exact help with the agent response,
  preserve the invalid draft and prior circuit, recover on the first corrected
  Apply, and undo the correction as one edit. Existing disclosure, recording
  position guidance and value-unit tests also pass.
- Static native-command inventory is unchanged: 166 commands, 134 routes,
  404 binding sites and 75 inventoried sources. Counts are not a complete
  dynamic-conformance claim.
- The full `scripts/check.sh` gate passed with serialized Go builds: lint and
  staticcheck, 67 Python tests, complete root/desktop race tests and vet, and the
  required mounted-control race tests and offscreen vet. The desktop race run
  took 174.194 seconds; the mounted run took 145.407 seconds. Exit code: 0.

The affected packaged/native checks are tracked in
`build/value-guidance-205-tools/`. Their final receipt must identify the exact
source and binary before a new package is promoted. This does not close the
remaining IME, screen-reader, every-binding or participant-study requirements.
