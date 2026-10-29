# Internal review: discoverable docked drawers

The owner clarified that the side panels should behave like drawers, then chose
moving the canvas aside rather than overlaying it. This follow-up corrects the
presentation/interaction of the preceding sidebar implementation. No service,
component, numerical model, solver limit, dependency or public command changes.

The old /286 toolbar toggles did close/reopen panels during bounded native
pointer checks, and the original layout was restored. That observation does not
invalidate the owner's report: there were no direct Close controls on the panels,
no clearly located edge tabs, and the divider's minimum prevented dragging shut.
No unsupported event-engine repair or claim of an established hardware cause was
made for the reported inability to collapse.

Both drawer controls now live at their respective workspace edges above scrolling
content. An open drawer has a title and labelled Close action with an outward
arrow. A closed drawer leaves a labelled inward-arrow tab. Both actions retain
the original `parts-toggle`/`inspector-toggle` identities and dispatch the same
`workspace.panels.set` command with captured guards. The document toolbar no
longer repeats them. A fixed-height control row keeps navigation in place while
simulation runs; closed forms consume no horizontal space.

Dividers still use the managed SplitPane. Their range now extends close enough
to the outside edge to request a close; fractions stay strictly inside 0..1 as
required by the pinned SDK. Crossing half the minimum open width dispatches the
existing visibility flag, retaining the originating preferred width. It does not
store an invalid narrow width. Home closes Parts, End closes Inspector, and a
removed divider returns focus to its persistent edge tab. Normal resizing,
opposite-panel constraints, guarded request origin and SDK contracts remain.

New retained tests exercise both Close controls, label-based reopening, both
sides of pointer/keyboard closing, preferred widths, focus, all four themes,
minimum-window positioning and multi-move drags past the slide-time threshold.
An intervening public agent edit refuses the old closing gesture; a fresh one
works. Existing canvas substitution scope is unchanged: the manual fixture
replaces only unsupported canvas paint, retaining real managed controls/input.
Multi-move resizing passed before this change too; no speculative continuation
or framework modification was introduced.

The first implementation attempt used numeric endpoints 0 and 1, which the
SDK correctly refused. Expressed the close range with one logical unit remaining
instead; no SDK validation was weakened. A rendered assertion then found that
an intrinsically sized header placed the right control beside the left one.
Explicit horizontal growth fixes its measured edge alignment. Initial failing
receipts are retained alongside passing results.

The native preview visibly shows the new edge controls and real scrollers.
Its model changed independently while on screen, so it is not treated as a
controlled electrical/interaction trial. Preserve any intervening owner work;
normal test close must refuse dirty work. Full regression and candidate results
are recorded in README.md. Existing native OS-queue/hardware, participant,
clean-account and signing gates retain their distinct status. Review is internal;
no Prism or external review provider was used.
