# NPN recorder baseline profile

October 20, 2026. This data-only profile preserves six exact regions from the already inspected original NPN annotation: accepted 0 V drive, cutoff current/voltage/state, live header and result identity. The resistor-loaded fixture has no collector LED. It neither changes application code nor introduces a new decoder or capture method.

The unchanged Go PNG tool admits all twenty preserved original baselines and refuses twenty original fresh results plus one separately inspected explicit stale result. All 246 region hashes agree with independent Pillow decoding. Source archive/journal/frame hashes and original decisions are retained; temporary replay copies are removed normally. No original archive, annotation, cohort or application source is altered.

Run with Python and Pillow:

```sh
python3 -B docs/validation/npn-baseline-profile/verify.py
```

Replay reads original local archives, writes only temporary PNG copies, repeats the Go/Pillow decisions and cleans those copies. It opens no app, records no pixels and sends no input. It requires the pinned existing Go tool and original archives; missing evidence must be restored, not regenerated. All 659 current application Go hashes and canonical remain exact. No full source gate is invalidated by this new JSON profile.

This qualifies saved-image decisions only. Native/current-build admission and any timing cohort still require their own declaration and actual producer/recording validation. [Profile](profile.json), [cases](cases.json), [report](report.json), [artifacts](artifacts.json), [checkpoint](checkpoint.json).
