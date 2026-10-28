# Single-level circuit blocks

Prism initial review: Anthropic Claude Opus 5.5, complete 10/10 chunks,
41 files, no skips/truncation. Nine medium/fourteen low observations retained.

Medium dispositions:

- bda7079526ab9075: terminalHit calls TerminalAt, which already projects
  VisibleGraph. Native direct hidden-terminal and expanded-member tests cover
  this path; no second projection is needed.
- e790d0eb02d989b0 /6092ae40002039ab: history and snapshots already deep-copy
  graphs and block pointers. Parameter/view edits now also clone their graph
  before mutation for consistency; explicit Undo and ownership controls pass.
- ab437ab6a77ff6e3: ownership assertion now checks the actual source graph.
- 874b2aa01bf28528: service internals in tests are read under the mutex and
  JSON marshal errors are checked.
- af2b059752af1643: malformed standalone block SVG rendering is nil-safe.
- ed070927a3fd5897: both ordinary and per-zoom fit skip hidden annotations;
  a source at -9000 is tested inside a collapsed block. SVG also filters visible
  annotation geometry while retaining full authored metadata.
- 5433d30c57a16f33: BlockTemplate already deep-clones metadata and every member
  including nested device settings, and copies value-only wires. The pure query
  response ownership test mutates ports/members/passive settings without touching
  the service. An additional full graph clone would be redundant.
- 908fb1d8cc82df75: duplicate retries only a private position error sentinel,
  preserves the final real position cause and fails immediately for other errors.

Low dispositions:

- 42598241c720a800: malformed form pairs deliberately reach shared validation
  with empty/invalid names or targets and produce its ordinary refusal. Empty
  separators are skipped by FieldsFunc; no silent successful malformed creation.
- a765819cd21ecc44: duplicate/empty pin and parameter names are rejected before
  the graph can reach the inspector. Existing refusal/UI validation tests cover it.
- 32b69796effe1adc /b28b1852b1419a36: one BlockBodySize helper now serves routing,
  exported/native geometry, hit and selection extents with explicit lead padding.
- 8e00e306ca44ef99 /10bc585772f09702: exact Hide internals button and the
  zero-injection wording are corrected.
- d0c2a5b3fdc3c317: raw local receipts preserve the authorized SDK/repository
  identity consistently with prior evidence; no credentials are included.
  Staged diffs go only to the user-authorized Anthropic reviewer.
- e7b8ae30d1d1375d: junction refusal asserts a real matching internal wire exists.
- 843094746236eb1e: unused decode removed and errors checked.
- b82b935e06de1e2e: invalid positions/radii refuse before the visible projection.
- 795c79919a4aa708: one checked pin-position lookup supplies caption side.
- 75c3d281c37f6a36: body width is fixed at 85 half-units; only height varies.
  Caption x=79 is its declared six-unit inset, not an unknown variable width.
- 435ae3ce8fa30e8d: snapshot literal style corrected.
- d9fc7580b18d89f3: nextID is a read-only search and paste validates before all
  reservations. Duplicate now additionally chooses its ID once for all attempts.

Independent checks cover original/copy RC response, distinct instance time
constants, named voltage probes, live LED bridge currents, physical ownership,
view/parameter Undo, independent clipboard, save/reopen, invalid boundaries and
no restart for purely visual grouping. Ordinary/offscreen UI actions invoke the
same commands with stale-control guards and all themes validate.

The broad first regression found only a hard-coded 78-example expectation and
an old viewport test that included hidden members. The catalog is now79; viewport
checks explicitly inspect the visible drawing. The corrected broad race run
passes (receipt retained). Early fixture errors and compiler/test assumptions
remain under build and are excluded from passing claims. In particular the
first RC wire assertion expected unqualified implicit-ground currents; a
single-ground LED control supplies the positive independent current proof.

A true already-running grouping defect lost block-pin voltage readings; the
before receipt is retained. Probe positive/reference terminals now resolve to
canonical existing member refs, preserving retained samples without a restart.
Ordered virtual boundary edges carry no injected current and preserve conservative
wire-cut inference. Collapsed members have independent position domains; they
cannot invisibly block normal placement, while expansion/release/collapse into
an occupied visible grid point refuses atomically. Internal new-member topology
edits require Release. Full model, primitive/resource and work limits are unchanged.

These are source/routed/software tests, not native OS queue, hardware, participant
or clean-account qualification. Original native checks 3/6/9/11 remain open.

Follow-up review covers working-tree fixes relative to the exact initial staged
index (4/4 chunks,21 files, no skips/truncation). The initial-index diff digest
is retained in review-source.json. Two medium/eight low observations:

- 75132c38a5453a91: BlockBodySize explicitly returns half extents. Both hit tests
  compare abs(local) against them and selection draws corners at +/- the same
  values. The original initial outline incorrectly treated full width as a half
  extent; it is already corrected. New direct edge tests reject points one unit
  beyond each drawn selection half extent.
- 5992fdb757c4c0ec: expanded terminal resolution is now a single direct lookup
  of an ordinary member. It refuses nested/self/cyclic block targets even on an
  unvalidated disposable graph; no recursion or visited-set allocation is needed.
- 2605d920dcede9d5: research spacing corrected.
- 6b49ebd2d85117af /31b381b2394dd3c0 /8c976f1604cbfd7c: position validation now
  precomputes domains once, reports both conflicting IDs neutrally and returns
  the shared private position sentinel. Duplicate only retries that classification.
- 0e609135b302f5e2: explicit graph equality after refused expansion/release and
  after reload verifies independent positions and the saved expanded flag.
- 5a3548318461e414: the nil guard is directly inside a switch case after its
  path loop has ended. Go break exits that switch, not the enclosing component
  loop. Standalone malformed rendering remains nil-safe.
- d05698a75c4a3a1c: route projection is bounded by the existing100-object/300-wire
  drawing envelope. The prior cache key already serializes the entire graph for
  each wire. This adds a bounded projection, not a new asymptotic cache contract;
  no native frame-rate qualification is claimed. A broader cache redesign is not
  required for this feature and is not introduced speculatively.
- 5627538928aaf278: BodyBounds now matches every other part's existing padded
  route bounds; the extra fit margin is deliberately conservative. Direct body
  geometry still comes from BlockBodySize. Shared viewport/edge tests pass.

Analysis continuation after a5f09cc3 reproduced AC rejection of circuit-block
helper symbols (blocks-ac-before.txt). AC support/bias and shared DC operating
point setup now explicitly use an owned physical graph. The authored graph,
source hashes and named-pin probe metadata remain intact. Independent RC complex
transfer and DC frame checks pass, alongside loop/tolerance controls.

Prism analysis review completes1/1 chunk (3 files, no skips/truncation), two
medium/one low observations. 5e34670187e4efeb is inapplicable: PhysicalGraph always
starts with Graph.Clone, even when there are no blocks, and filters the cloned
components; it never aliases the input. 088f19374a00aed6 is inapplicable because
Graph.Compile explicitly delegates to that same physical compile and only adds
alias entries. A new equality assertion checks every physical terminal-net key,
and the DC verdict additionally reads through the returned frame and canonical
block probe, independent of a second node-map lookup. acedc17bd7ac7eaf adds exact
point/response counts to prevent a vacuous pass or an indexing panic.
