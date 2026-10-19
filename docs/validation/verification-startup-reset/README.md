# Project-startup reset restores saved coil conditions

The production source packaged in /167 retains stored `inductor_initials` when Use project startup clears its other verification overrides. A project with saved L1 startup −20 mA and a separate +30 mA Verify scenario still captures +30 mA after the reset. Both RL and RLC semantic production controls reproduce the error; their completed first samples are retained in the failure output.

The repair constructs shared default verification settings and preserves the accepted duration. This clears all scenario startup/input/sampling overrides while allowing the saved circuit’s startup values to supply the next capture. It uses the existing `verification.settings.set` command; no schema, model, format or SDK change.

Final tests pass for RL and RLC captures against independent public-agent settings and complete sample pages. RL rows also match the independent exponential-current equation. Saved component startup values, live time/physics and circuit history remain unchanged. A retained-widget RLC test types unfinished `8 ms` duration, clicks reset once, then captures the saved −20 mA startup through the actual field-settlement/event-router path. A deferred reset refuses a newer agent scenario and a fresh reset then clears it while preserving the agent’s duration.

The final suite also replays against the original /167 action helper through a Go overlay: four leaf cases fail. Full source quality passes in 213.058 s, including lint, 54 Python checks, root/native race/vet and registered mounted tests. General development native smoke passes in 40.913 s, exits 0 and leaves no owned process. These records are bound by the checkpoint. Retained-widget/semantic source observations are distinct from physical OS input and clean copied-runtime qualification.

Preserved /167 remains affected by this stored-coil-scenario edge case; its app, archive and canonical /120 remain unchanged. A clean replacement is pending. Owner walkthrough remains Not tried yet; acceptance and the other broader gates are unchanged.
