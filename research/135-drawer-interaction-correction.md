# Drawer interaction correction — October 29, 2026

The owner expected drawers that open, close and resize, and explicitly chose
**move the canvas aside** over an overlay. The preceding /286 implementation
provided toolbar-based collapsible sidebars, with weak discoverability and no
drag-to-close affordance.

Drawer controls now sit at the left/right workspace edges. Open drawers have
visible Close buttons; closed drawers leave labelled Parts/Inspector tabs.
The document toolbar no longer carries those controls. Inside edges resize;
dragging toward the outside edge closes the drawer and preserves its preferred
width. Focused dividers also close with Home (Parts) or End (Inspector), returning
focus to the edge tab. Controls remain above scrolling forms.

The same `workspace.panels.set` command and unchanged 228-command catalog serve
users and agents. The Simulate chooser and all existing tools remain. No service,
component, model, limit or dependency changes. Choices are still session-only.
This implements docked, explicitly controlled drawers; it introduces no automatic
opening/closing on selection or live simulation updates.

[Validation and candidate](../docs/validation/drawers-20261029/README.md) records
full regression checks and the local candidate. [Internal review](../docs/validation/drawers-20261029/internal-review.md)
records the original native observations, corrections and scope distinctions.
