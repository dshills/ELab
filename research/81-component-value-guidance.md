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

The [0.181.4 /205 receipt](../docs/validation/value-guidance-205/report.json)
identifies clean source `ef3a63b05403a5ad0028d2c6af8d76faa3409ad6`, the candidate
and extracted agents, and the ZIP. All eight numeric kinds pass refusal,
correction and stale-revision checks through both agents and the native socket.
Malformed requests retain generic help without changing accepted state.

At 1053×802, CUA verified retained invalid input, readable wrapped guidance,
first-Apply correction from 1 kΩ / 2.917 mA to 2 kΩ / 1.477 mA, and one Undo.
[Images and method](ui-audit/value-guidance-205/README.md) distinguish the isolated
native test bundle from the distribution bundle. Their unsigned compiled
executables are byte-identical; signatures and bundle identities differ so that
CUA cannot accidentally route actions to the owner's session. The owned test
closed, removed its endpoint and disappeared from CUA's running-app inventory.

The [ZIP](../build/releases/ELab-0.181.4-macos-arm64.zip) passed member/mode and
Unicode/space extraction checks and strict ad-hoc signature verification. Its
SHA-256 is `d13973f02f9c2895d353d24338ed060492a3a2200ff8a91e3606bfa867f4946b`.
The /204 release remains unchanged. Nothing was publicly uploaded or pushed.
This does not close IME, screen-reader, every-binding or participant-study
requirements, and makes no additional electrical accuracy claim.
