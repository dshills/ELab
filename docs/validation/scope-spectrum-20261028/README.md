# Scope / Spectrum tabs and live FFT — October 28, 2026

Two shared commands are added: scope.view.set and spectrum.live.set. There are
224 commands, 61 component kinds and the same 87 built-in examples. The source
inventory is updated and verified. Ordinary discovery is 65,445 bytes, below the
unchanged 65,536-byte budget. Live input schemas use references; output frequency
settings remain numeric despite unit-text input support.

The dock now has matching Scope/Spectrum tabs. Spectrum offers Live/Recorded,
the selected scope voltage/current probes, Hold, window/sample/fundamental and
harmonic controls. The existing recorded analyser remains reachable through both
the dock and inspector, including its interval settings and exports. Live state
is separate, session-only and not a saved Verify recording. Recorded exports and
paged queries retain their original captured-result target.

One lazy worker copies at most 512 accepted scope points and checks at 250 ms
intervals. It runs the unchanged recorded FFT outside Service.mu and publishes
only for matching document/electrical/run/actuation/settings/selection/window
identity. It never advances simulation or runs an electrical solve. Live FFT
uses automatic or 32–256 samples because the existing end-exclusive grid needs
N intervals; the recorded limit remains 4096 and scope history remains 512.
Snapshot reads only clone owned data. Analysis jobs own the live state/worker;
Service.mu remains the sole command/publication authority. Close cancels and
joins the worker. No component, solver, integration or transport limit changes.

## Internal review and corrections

No Prism or external code review was run. Internal review checked source-key
publication, work outside the mutex, detached result ownership, bounded history,
unchanged-data deduplication, warm-up/paused/Hold behavior, removed probes,
request retries, document replacement/shutdown and recorded-state independence.
It also checked field-context revision guards, compact discovery/schema ownership,
frequency-unit normalization, fixed-axis input and control geometry.

Corrections made during this work:

- A 512-sample live grid cannot fit 512 end-exclusive history rows. Live choices
  are bounded to 256; no history or electrical limit was increased.
- Live state/result nested copies are detached, including cloned response paths.
- Native selectors add padding/borders outside content width. Explicit outer
  slots remove the observed overlap, with pairwise-bounds assertions added.
- Inspector and dock no longer alternate revisions on one shared mesh canvas.
- Unavailable existing probes cannot trap users in Live or prevent Hold/Recorded.
- Identical numerical data still refreshes a changed source key; otherwise
  removing another selected channel could leave the frame perpetually invalid.
- A hidden scope cannot accept a retained cursor callback after a tab switch.
- Discovery initially exceeded 64 KiB. Shared schema references fix that without
  relaxing the bound. Initial failed full/discovery receipts remain retained.

An initial UI fixture used a mixed-network initializer for ideal RC; it was
corrected to wait for the actual transient. A publication test reused an earlier
restart retry key and correctly received the old cached response; a unique key
fixes that fixture. Neither is claimed as a product defect. The original FFT
functions are exact to the preceding commit (engine-identity.json).

## Tests and evidence scope

Complete ordinary portable, desktop and rendered suites pass. Focused race
suites cover live/recorded FFT, discovery, real stdio/Unix sockets and rendered
controls. Mounted four-theme tests prove in-bounds, non-overlapping and stationary
control geometry from warm-up through accepted samples and real Resume updates.
Actual routed mouse events activate tabs, Hold and Recorded; direct UI callbacks
and agent commands produce identical spectra. Held results preserve physics/time,
recorded results remain intact, retired computations cannot publish after agent
changes, and retries do not re-execute old actions.

Manual software captures were controls-only (plot omitted for geometry testing)
and intermittently blank/partial. They are not a native visibility verdict; they
helped find the initial selector overlap, while geometry/routing assertions are
the acceptance evidence here. Mesh preparation is exercised separately. These
checks do not close historical native OS-input/rate gates 3/6/9/11.

Final-source full suite logs, race results, lint/vet and inventory/PRD receipts
are compressed byte-exactly. No owner projects/settings, verified SDK caches or
older qualified release bytes were changed. No push or upload.
