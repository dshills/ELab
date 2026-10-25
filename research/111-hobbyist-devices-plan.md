# Remaining hobbyist device work

The owner requested the four remaining proposals after the H-bridge: buck
conversion/PWM control, op-amp bandwidth/slew limiting, a latched shift register,
and piezo/active-buzzer sound with explicitly controlled audio preview.

Implement in that order. Preserve existing model identities and old project
semantics, numerical accuracy, circuit capacity and per-output work limits.
Users and agents share all configuration, simulation, playback and export
commands. Keep previews opt-in. Each phase needs analytical/electrical tests,
working examples, persistence/capture integrity and native/agent parity. Finish
with quality checks, packaged replay and a separate owned desktop fixture;
preserve the qualified alpha and owner sessions. No unrelated component families.

## Implementation and qualification

All four groups and their six examples are implemented. Users and agents share
PWM configuration, optional amplifier dynamics, sound configuration, explicit
audio rendering/playback/stopping, and WAV export through the existing file
worker. Shift/latch control uses actual wired sources and switches.

The complete root, ordinary desktop and rendered desktop race suites passed,
along with vet, goimports/staticcheck, Python tests and the PRD coverage-record
audit. Numerical accuracy, circuit capacity and per-output work limits are
unchanged. See [qualification](../docs/validation/hobbyist-devices/README.md)
and the [model/command guide](../docs/hobbyist-devices.md).

A separate 0.192.0 /253 package, matching-agent replay and owned native check
follow this clean source checkpoint. The qualified alpha remains preserved.
