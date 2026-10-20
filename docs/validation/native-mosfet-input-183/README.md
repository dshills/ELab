# Native MOSFET signs and controls · /183

Ten accepted native electrical states and three fresh saved-file reopens pass on the exact qualified **0.166.0 /183** app and agent. This adds bounded pointer/keyboard and electrical evidence; it does not qualify all native bindings or establish a new application defect. No application source, model, SDK, tolerance, budget, format or version changes. One PMOS documentation sentence is corrected to match the current UI.

| Native case | Observed accepted result |
|---|---|
| NMOS example, Restart / Low + Step / High + Step | Zero-time triode, cutoff at 1 ms, triode at 2 ms. Drain/source current is positive; gate current is zero. |
| PMOS example, Restart / Low + Step / High + Step | Zero-time cutoff, on at 1 ms, cutoff at 2 ms. On-state VGS, VDS and drain/source current are negative; LED current and dissipation remain positive. |
| CMOS example, Restart / Low + Step / High + Step | High input gives nearly zero output; Low gives 4.9986 V output. The two devices retain opposite signed drain currents. |
| CMOS VIN high changed to 2.5 V, Apply once, Restart | Private draft leaves the complete accepted snapshot exact. Apply changes the value once and labels old data stale; Restart produces fresh zero-time readings. Output is 2.380952 V; PMOS/NMOS carry approximately −52.619/+52.381 mA, both with positive dissipation. |

The examples were loaded through native menus from an owned blank app; graphs were not seeded through an agent. Source High/Low and single-sample steps used native controls. The VIN edit used field focus, Cmd+A and typing, then one named Apply and Restart. Three separate `.elab` files were saved through **public commands**, not native Save panels; each opens clean with the exact graph and zero-time startup in a fresh packaged agent. Source files remain unchanged.

A matching agent reproduces all **ten complete graph/result/transient/scope states**, with only local run/electrical identities excluded from nested numerical comparison (rel 1e-9 /abs 1e-12). Independent scalar bisections use the published square-law channel, body-junction and LED parameters; separate explicit-wire node sums and V×I calculations check current and power balance. Maximum load-line voltage difference is **8.305e-14 V**, current difference **4.272e-16 A**, independently assembled KCL residual **2.981e-15 A**, and power-balance residual **5.703e-15 W**. These are fixture observations, not hardware or general topology guarantees. No intrinsic gate charge, heating or real part identity is modeled.

## Input observations retained separately

Two named Pause attempts after example loading did not establish Pause. Raising the owned window and coordinate Pause succeeded; those earlier attempts remain unqualified. The window was enlarged from **1053×802** to **5054×1410** to inspect readouts, then restored; this does not qualify ordinary startup, minimum-size readability or a display-mode transition. A scroll attempted immediately after restoring geometry returned CUA `windowNotFoundAtPosition` (−10005); a refreshed observation recovered the same owned window. Dense/narrow Inspector text remains a layout limitation.

The first VIN selector attempt combined an AX item click with Return. Although the intermediate AX tree still showed the selected menu item, the later app was Running instead of Paused (the earlier accepted sample was 2 ms; the retained public observation is 7.107 s). That outcome is preserved as unexpected and unqualified; the input/dispatch timing and cause are unestablished. It is **not** evidence of a successful paused selection.

A separately declared control uses pointer Resume/Pause, a **pointer-opened selector, Home/four Down keys, then Return**. It selects MN while preserving Pause, the exact 0.46 s time and complete graph/results/trace/history. This control passes; it does not relabel the earlier mixed AX-item/Return outcome or establish a framework defect/repair. The post-run `actions.json` reconstructs CUA calls and outcomes; it is not an OS event journal.

## Model presentation and evidence

`docs/pmos.md` previously said the native Inspector/probes displayed MOSFET power. The native summary actually shows VGS, VDS, ID, IG and region; voltage/current probes retain their signed references. MOSFET power is present as `mixed.mosfets[].power_w` in structured results. The documentation now states this precisely; no new UI feature is inferred.

The first agent-reference attempt used the captured-record key `led_model` against a live snapshot, which publishes `model`, and raised a local KeyError. Its exact producer, output, full protocol and normal joined/ready agent exit remain retained. The corrected final reference uses six normal joined/ready processes and 56 requests. Sixteen read/save observer bridge connections exit 0 with 22 retained requests; these close their connection, not the app. The controller connection's separate bridge exit status was not retained and is not included in those counts.

The owned native host receives completed joined/recovery-finalized/ready shutdown; its PID disappears and endpoint is removed. `open -W` exits 0; **the native host's own exact exit status is unavailable** because LaunchServices owns it. The controller's `ready_for_cua` field is historical launch status; its terminal exec result and final closure fields establish cleanup. No forced termination is used. Shared window settings, all 658 application Go inputs, copied/sealed /183, SDK and canonical /120 remain unchanged. No owner project, display recording, privacy grant, printer operation or promotion occurs.

[Report](report.json), [manifest](manifest.json), [checkpoint](checkpoint.json), [qualified app](../recorded-dc-review-183/README.md). Original CUA pixels are retained without modification; JSON/text/protocols are compressed losslessly. The initial package preflight incorrectly assumed PNG; CUA returned JPEG bytes. Its failure/producer/partial-file inventory are retained. Original temporary .png names remain unchanged there; tracked images use .jpg with source-path mapping and identical bytes, without conversion. Producers are stored as text, with failed revisions separate.

Run `python3 -B docs/validation/native-mosfet-input-183/verify.py` for read-only replay. It starts no app/process, performs no UI/settings access, and refuses optimized Python. It verifies source/release/artifact pins, private/stale/fresh value semantics, ten agent matches and independent equations, fresh files, native/bridge cleanup and the distinct menu outcomes. Native timing/blur/IME/reader, all-binding/performance/platform, owner walkthrough and actual study remain open. P0 stays **17 implemented /10 partial**.
