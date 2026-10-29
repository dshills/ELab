# Internal review

Reviewed the final diff, node identities/order, shared command routes, field
ownership/settlement, refusal paths, optional control states, native screenshots,
source inventory and test evidence. No external provider or Prism was used.

## Findings addressed

1. **Primary actions below optional settings.** Retained actual inspector nodes
   had zero/unlaid bounds at short height when thermal/loss controls were enabled.
   Extracted the existing basic value editor, preserving its field/apply IDs,
   then placed it and Plot controls before detailed settings. All startup controls
   keep their top-level component ownership bindings. The first full desktop
   suite caught op-amp readout ordering; restored readout-before-plot without
   weakening the existing assertion.
2. **Checkbox clipping.** Native Apollo evidence and actual-font managed widths
   reproduced overlong transistor/op-amp/motor captions and mismatched accessible
   names for passive losses. Enabled-state expansion additionally reproduced
   Early effect, leakage, transformer polarity and current/load captions. Kept
   concise names within available width and moved longer explanations to Help;
   callbacks, parameters and control identities remain unchanged.
3. **Plot rejected after own draft commit.** Actual retained-driver input changed
   command revision 8→9 when focus loss committed the draft, then refused Plot
   with the old revision. Reused valueRelatedAction rather than fetching an
   arbitrary newer snapshot. Tests compare against public commands and preserve
   graph/live physics; agent/document interleavings cannot adopt stale input.
   The native single-click control additionally passed on the RC fixture.
4. **Misleading explanations.** Plot-help claimed silent replacement at six
   traces, contrary to the service refusal. Corrected it. Native custom-profile
   controls showed nonzero capacitances beside an instantaneous/default-model
   explanation. Display text now names saved profile/temperature settings and
   labels the default limitations as base assumptions. It does not change model
   laws, public explanation/query contracts or numerical capacity.

## Fixture corrections and limits

The initial caption fixture reserved too much glyph space and attempted to read
unlaid motor geometry; those were fixture errors, excluded from product claims.
The retained reproductions use the corrected 20px glyph/gap reservation and a
checkbox-only mount of production nodes with the actual root theme. The short
rail test now rejects empty rectangles. A replacement-document test initially
failed because the example clears selection; it now explicitly reselects the
same ID through public commands before delivering the held release, establishing
cross-document retirement despite a reused component ID.

Native controller setup initially used an unknown leakage field, then omitted a
required winding field; both were explicit schema refusals. It also attempted to
step immediately after model configuration; the app correctly refused until an
explicit restart established a valid paused transient. Corrected the controller
using the documented parameter fields and restart sequence. None is reported as
a product failure. No alternate input/capture mechanism, security change or
owner file was used.

No unresolved regression was found within this batch's scope. This is an internal
engineering review, not independent usability acceptance or full native queue/
hardware-rate qualification. Broader capacity, model import and hardware handoff
remain separate ordered work in research/131-inspector-usability.md.
