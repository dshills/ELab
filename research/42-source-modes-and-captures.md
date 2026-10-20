# Source modes, restart and captured experiments

October 19, 2026 · exact private review **0.166.0 /183**. Twelve headless cases pass without a new application defect or code/version change. This strengthens the existing hobbyist workflow; owner walkthrough and broader MVP acceptance remain open.

| Circuit and starting state | Modes checked | Independent evidence |
|---|---|---|
| 10 kΩ /1 µF RC, initially 2 V | Auto, Charge, Discharge | Exact exponential propagation, source levels, resistor current and capacitor current balance |
| Pulse-driven NPN LED | Auto, High, Low | Source levels, base resistor law, declared emitter polarity and Ib + Ic − Ie balance; bounded on/off currents |
| 1 kΩ /100 µF LED fade, initially 2 V | Auto, High, Low | Source levels, resistor law, capacitor/LED current balance and independently bisected LED law |
| Capacitor-timed NPN flasher, saved 0.1 V/0 V startup | Auto, High, Low | Independent final graph current balance; full live/capture agreement and both capacitor transition/startup checks. No independent complete trajectory or period claim |

Each case runs for 100 ms with 1 ms output spacing. Verify explicitly captures the original mode from startup. The live circuit then queues a different mode while paused, applies it with one step and restarts. Verifying the original scenario again yields identical rows/statistics even while the live mode differs. Saving and opening in a fresh process restores startup at zero time in Auto, retaining both captures exactly.

The final set checks **1,224 captured rows**, **6,423 live channel values in exact timestamp order**, **2,456 independent row-law conditions**, **33 independently assembled final node balances**, **nine complete final result objects** and **twelve capacitor-channel continuity checks**. Maximum RC voltage discrepancy is 1.333e-15 V; maximum tested current-balance discrepancy is 9.033e-11 A, within the declared 1e-9 A bound. Twelve earlier-capture exports stay byte-identical after mode changes/restart; all 24 kept captures reopen and export exactly. These numerical observations are limited to the declared fixtures and do not certify hardware.

The final run has 24 normal joined/ready agent exits and 519 requests. Discovery and two reference-code mistakes remain preserved separately: the first equation incorrectly adds an outward emitter current, and the first offline flasher analyzer uses `id` instead of `transistor_id`. Across all declared runs, 32 owned processes close normally and 678 requests retain their replies. These mistakes are not product failures or extra passing cohort members.

[Full evidence and read-only replay](../docs/validation/mode-lifecycle-review-183/README.md), [current review](35-first-version-review.md), [delivery plan](34-first-version-plan.md). All 658 application Go hashes, sealed /183, SDK and canonical app remain unchanged. No new native input, visible latency, broad performance/accessibility/platform qualification, printing, privacy grant, owner-project edit, promotion or participant study is performed. P0 remains **17 implemented /10 partial**.
