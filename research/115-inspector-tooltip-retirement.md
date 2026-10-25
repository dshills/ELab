# Inspector tooltip retirement

October 25, 2026. Fixes the lingering PWM numeric-bounds popup observed during
[widget migration qualification](../docs/validation/vellum-widget-adoption/README.md).

A numeric field’s bounds and arrow help use separate native tooltip scenes.
Removing or replacing the inspector deletes their originating widgets without a
pointer event. The old tooltip can therefore outlive its control; relying on the
widget’s LongHovered state is insufficient because ordinary numeric help does
not necessarily set that state.

The compatibility SDK adds an owner-scoped CloseTooltips seam. Before Vellum
retires a subtree, it finds tooltip stages whose context is inside that subtree,
releases the popup-stack inspection lock, and closes those exact stages. Retained
descendants have already been moved by reconciliation. Menus, other tooltips and
user value/command callbacks are left intact. Pending hover events cannot revive
the deleted widget.

The regression reproduces the original failure before the fix. Routed pointer
hover plus the controlled clock checks visible bounds/arrow help, pending hover,
removal, fresh same-ID owners and valid help on replacement controls. A separate
case checks that deleting an unrelated sibling preserves the existing tooltip
and menu; replacing the numeric owner clears only its tooltip and sends no
numeric Change callback.

ELab’s application sources, 182 shared commands, 53 component kinds, electrical
models, accepted domains, persistence and simulation/test work limits are unchanged.
Only the SDK pin and module replacement paths change. The upstream remains
f9dee54; the complete patch and all 8,735 exported files are pinned and verified.
The previous /255 app and qualified unsigned alpha remain preserved.

See [qualification](../docs/validation/inspector-tooltip-retirement/README.md) for
completed checks, package identity, native observations and exact test scope.
