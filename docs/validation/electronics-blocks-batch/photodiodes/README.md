# Photodiode evidence

Item 4 adds generic PN photocurrent/dark current/shunt/Cj, a light slider/exact
field and saved config; identical shared commands support users and agents.
Live optical input reuses the existing atomic sensor input worker and preserves
charge/time/run. Constant Cj starts at zero, consumes existing storage/part slots
and remains separate from helper metadata. Scope/wire currents, energy, models,
save/captures, optical DC sweeps, AC/noise and schematic render/export are integrated.
Two curated examples: reverse-biased load sensing and a split-rail TIA.

Independent controls: declared Shockley/photocurrent laws and derivatives, DC
load conversion, TIA gain, optical-sweep slope, closed-form reverse-biased RC
response, source/optical electrical power balance, shot-noise law, storage budgets,
live input continuity, save/capture/reopen/undo and per-reading forgery rejection.
Ordinary/offscreen all-theme inspector parity/stale input checks accompany them.

Initial outcomes retained under build: first scope hook used an undeclared energy
identifier; first canvas hook used float32 for float64 coordinates. Initial
photodiode implementation incorrectly reused the positive-series-resistance solver
with zero series resistance; electrical Jacobian/TIA tests reproduced the failure.
It now uses its own direct monotone law with continuous high-forward continuation.
A first test applied the far-reverse-current approximation to a near-zero-bias TIA;
that reference assertion now applies only to the reverse-biased load fixture.
Save/reopen caught a TIA playback/sample ratio above the existing 1000 samples/wall
second limit; example speed is corrected to 0.001, retaining all existing limits.

Initial Prism complete: 8/8 Anthropic Claude Opus 5.5 chunks, 46 files, no
skips/truncation; five medium/seven low findings. Concrete dispositions:

- bfda9f0ee3335727 /69d4f04affde23b8 /54cbdc3a9637beed: Graph.Clone calls
  cloneDeviceParameters which clones Photo. Compile owns MixedGraph via g.Clone;
  staged input graph, retained snapshots and history all own their configs.
  Publication mutates the committed graph only after saving an owned history
  snapshot. Explicit graph/worker/snapshot/undo tests now verify those boundaries.
- 953297fca95a579c: the existing constructor switch's default sets nonlinear=true
  for every non-listed passive/source kind, including photodiode. A direct network
  assertion verifies the flag; electrical current/derivative tests already pass.
- 9d808d055a95ea84: photodiodeLaw returns photocurrent as the nonnegative magnitude
  responsivity*light*1e-6, then subtracts it for terminal current. Validation bounds
  responsivity strictly positive and light nonnegative. Independent references
  explicitly assert this returned magnitude at multiple levels.
- 4fc7ceb4a0c8e62e /99756ae2f1c871af /4e653be6cb85c9db: README punctuation fixed;
  parity test checks solution presence and resolves detector config by ID.
- a15cc0e61ec423b6: separate slider/exact nodes genuinely share a handler; their
  bindings are now on separate lines for unambiguous source inventory evidence.
- 09eac88ed3071095: compile's diode list is capacity/connectivity classification;
  the MNA/result dispatch explicitly handles photodiode before ordinary diodes.
  DC/transient/AC/noise tests use the actual optical law, not a list-derived stamp.
- 2b74e3fcb49170d0: DeviceParameters is a validated one-model discriminator. Other
  active fields cannot coexist with Photo. The configuration reducer runs inside
  the common atomic draft/history/electrical invalidation path, just like other
  device configure commands; it must replace the one variant, not retain junk.
- 5a37dc43db526c4b: collocation now required only when detector Cj>0.

Follow-up Prism complete: 1/1 chunk, 7 repair/inventory files, no omissions;
one medium/two low. 2b73de20e6d87f43: graph/file validation already requires the
model before constructing a solver, so a legacy/partial document cannot enter
this path. The collocation predicate additionally guards both pointers for
standalone malformed-plan callers. 10f756ba0da288e5: cloned graph variable renamed.
b8a6e3f72f2f2bba: final ownership test uses public snapshots and checked ID lookups.

Final scoped core race passes 8.354 s (photo, discovery, batch, browser and sweep
contracts); review ownership/electrical race 1.638 s. Ordinary drawing/symbol/photo
UI controls pass 0.530 s; earlier broader offscreen sweep/theme controls 2.299 s.
Lint passes; source inventory reports 219 commands, 525 sites, 181 routes and
110 sources. These are software/routed checks, not native hardware qualification.
