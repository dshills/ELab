# Native walkthrough and final input checks: /195

October 21, 2026. Existing **ELab 0.178.0 /195** passes the private-alpha engineering reference workflows and declared native input classes. **Owner acceptance and release designation remain pending.** The owner answered “Not tried yet.” No new binary, framework change, installation, feature or promotion was made.

[Report](report.json), [raw manifest](raw-manifest.json), [207-file evidence archive](raw.tar.gz). Archive: 6,822,302 bytes; SHA-256 `e32770887b2da00e863cacda536da91f661e8d40a5d52cc999b099ee693946eb`.

Source: clean `8d7d646344cf8a98a9638494fa0fe4d5661e2e3a`. Native SHA-256 `b668fecefc1f09c30078bfbeaa07d80d3364f208b9db1d28445dbcd37b2420ac`; matching agent `b5dfc91eb98fff3b5176072fca868e027c11b63c24a3ecb53ab296de4253627b`. The existing ZIP remains `76da198544ebb7aff0d1fc1d9e98b1bf052e37abbc44963396ef99c8f9a2f02d`.

## Native reference workflows

| Task | Observed result |
|---|---|
| LED from blank | Native search, keyboard position/placement and four named-terminal wires create the loop at 2.917 mA. No API graph seeding. Removing a wire labels the retained reading stale; reconnecting restores a current result. Native `2k`, Apply, Undo/Redo give 1.477 → 2.917 → 1.477 mA. |
| RC | Native example, Pause/Restart/ten single-sample Steps give 3.1606027941427888 V at 10 ms for 10 kΩ/1 µF. Discharge queues while paused without changing charge; ten Steps give 1.1627207896741474 V and negative current. Restart/Charge, voltage/current probes, 0/10 ms cursors, Resume/Pause and the native CSV Save dialog work. All 40 exported rows match actual trace points; positive-time rows match the independent ideal RC law. |
| Flasher | Both collectors and LED currents are observed. Native cursors measure about 490 ms at 10 µF. The native both-capacitors preset changes to 20 µF, marks paused readings stale, and Restart/Resume produces about 975 ms. Independent saved-sample crossing arithmetic gives 487.985/488.040 ms and 976.030/975.940 ms on the two collectors: ratios 2.00012/1.99971. This calculation interpolates samples; UI cursors use nearest samples. |
| Keep/share | Six owned projects saved through the public API reopen in fresh exact packaged agents with graph, values, models, settings and probes preserved. Native Save/Open/SVG and picker/unsaved Cancel were already checked on these same bytes in the [earlier session](../inspector-candidate-195/README.md). Current native CSV export adds waveform sharing. Prior native Discard/Save-and-continue and controlled recovery evidence retain their original scope. |

Matching agents reproduce the LED result, RC endpoint/cursor readings, both complete flasher endpoint results/cursor readings, retained 1001-row RC data/timing, and one-row DC data. Existing relative `1e-9` / absolute `1e-12` tolerances are unchanged. Same-engine agreement is parity evidence, not independent physical-model accuracy. Live time, rolling history and cursors are session state and are not claimed persisted by saving.

## Remaining native input classes

- Rejected seek time corrected to accepted 8 ms → Previous selects row 7. Rejected time corrected to accepted 7 ms → Play starts playback; Pause leaves row 49. /194 covers corrected time → Next.
- CUA’s numeric setter rejected `bad` before delivery. A second owned session instead uses confirmed numeric-editor keyboard focus and actual typing: `bad` blocks Next at row 8; typing accepted `8` then Next without Return selects exact row 9. Its readings match the packaged agent. AX reports the accepted value during local rejection; the screenshot retains the framework-worded validation error.
- Accepted timing interval 0–5 ms → individually rejected start 6 ms plus pending end 20 ms → Rising arrivals commits both. The exact 6–7 ms bracket contains the independent RC half-charge time. /194 covers coupled thresholds, threshold correction/paging and agent probe interleaving.
- Verify sampling `bad` refuses Verify. Correcting to accepted `0 s`, then Verify without Return, produces 105 rows at 100 ms. A fresh exact agent produces identical rows. Both recordings reopen; original 1001 rows/live physics are unchanged. /194 covers duration correction, startup reset and all three sweep fields.
- Native DC Verify records one point, with Previous/Next/Play disabled. Seeking 1 ms with Return refuses at row 0. Keyboard navigation visibly reveals **“Use sample index 0 or time 0 s. Your circuit is unchanged.”** Correcting to 0 s works.

These are representative correction classes, not every startup override, keyboard/IME path or binding.

## Layout, isolation and preservation

Native resize clamps at the declared 1040×720 minimum; captures including window chrome/shadow measure 1053×762. Both themes also render at 1293×834. Native Tab scrolls the final scope control into view at minimum height. Hiding scope expands the readable schematic; selection, value error/correction and native menus work. The keyboard route to the exact DC guidance is retained. This is reference-control acceptance, not an exhaustive layout/accessibility audit.

All desktop input and screenshots use CUA. Public API fixture loading, observation and owned saves are identified separately. Both native copies and fresh agent subprocesses use the retained policy denying repository/SDK/toolchain/cache/shared-geometry access and outbound IP. Controllers and CUA are outside that child policy. This is copied-runtime/offline evidence, not clean-account/default installation acceptance.

Both native sessions save before normal close, join workers, finalize recovery, remove endpoints and disappear; both `open -W` controllers return 0. Seven successful fresh counterpart clients complete joined shutdown and close their transports. Initial offscreen delivery, scroll/menu, helper and cursor-focus attempts are retained and not counted. The first headless checker incorrectly assumed the saved DC LED was paused; its failed report/protocol and normal shutdown are retained. A dotted-directory archive-path mistake stopped the first freeze before an archive was written. See `walkthrough/attempt-limitations.json` and original artifacts.

All 671 Go hashes match the completed full-quality gate. The only production Go difference from /193 is the qualified inspector viewport ownership repair; solver, model, persistence and command code are unchanged. No full suite was repeated for documentation. Canonical /120, owner /189, prior /191–/194 trees, both SDK archives, current 124-file package/ZIP and original fixtures remain unchanged.

The sealed package’s notes retain their earlier checkpoint; this follow-up advances evidence without rewriting it. Minor numeric-validation wording remains feedback material. Ordinary startup/clean-account qualification is owner-deferred for this private alpha. Broader MVP performance, accessibility/IME, full bindings, study, printing settings and public distribution remain separate. No fresh PDF/PNG, margins/scale/tiling or physical-print qualification is claimed.
