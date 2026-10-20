# Inspector native frame guard correction

October 19, 2026. This checkpoint retains the full [original width-repair proof](prior-width/README.md), including six original mounted failures, final passing regressions and the original full quality run. It also preserves the **unqualified /184 candidate** and its native smoke exit 1. No release receipt or ZIP was issued for /184.

All eleven /184 native labels preserved complete source text with no truncation or shaped overflow. The new guard nevertheless failed because it compared naturally sized text-box widths (212–234 pixels) with the 252-pixel group. It did not record the wrapping frame. This is a measurement error; that run does not establish the frame geometry. The corrected guard measures the named passive frames, requires at least 90% of group width and no overflow, then separately verifies natural text containment. Only the native smoke source changes from the initial width-repair commit; layout, commands, simulations and the binding ledger remain exact.

The final default lint/race/vet/mounted run freezes the same 659 application and three existing tool Go files; elapsed time is in report.json. The original six mounted regressions still run through the default gate. A clean replacement remains pending. /183 is the qualified review, /184 stays a failed candidate, and all historical failures keep their scopes.

Run `python3 -B docs/validation/inspector-native-frame/verify.py` for read-only replay. It checks hashes, the retained width evidence, failed guard and candidate, exact final source quality, unchanged routes/SDK/canonical. It opens no application and refuses optimized Python. The prior verifier is historical and is not run against this changed smoke source.

No new native fix qualification, physical input, latency, owner acceptance, promotion or full-MVP completion is inferred. [Report](report.json), [manifest](manifest.json), [checkpoint](checkpoint.json).
