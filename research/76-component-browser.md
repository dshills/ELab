# Component browser implementation

October 23, 2026. This implements the organization proposed in
[the component library review](75-component-library-review.md), retaining all
18 existing kinds and their simulation models.

The default Common view contains DC source, resistor, red LED, capacitor, ground
and SPST switch. The category menu also offers All components and six shallow
families: Sources (4), Passives (4), Semiconductors (5), Amplifiers / ICs (1),
Switches (1), and Connections (3). Search covers names and existing hobbyist
aliases across all categories. It commits on Return, then focuses its results.
Clearing search restores the chosen group. Choosing a group clears search.

Rows combine actual schematic geometry and names. The same drawing helper now
supplies the canvas and the browser's theme-aware vector artwork. The native
ListBox retains its keyboard navigation, focus, selection and accessibility
semantics. A small application-local presentation adapter adds artwork without
editing the pinned Vellum SDK. Common and each family fit without list scrolling.
The All view uses a bounded list with its own scrollbar, keeping category and
search above it. A full-height outer-scroll experiment did not behave reliably
at small window sizes and was not retained.

User and agent operations share `parts.category.set`, `parts.search.set` and the
existing placement commands. `project.inspect` publishes categories, selected
group, raw query and effective matches. `parts.list` always returns all 18 kinds,
with category and Common metadata. Browser state is session-only and preserves
the graph, simulation, history and document dirty flag. Direct agent placement
continues to work regardless of the visible browser filter.

See [the user/agent contract](../docs/component-browser.md) and
[validation receipt](../docs/validation/component-browser-201/report.json).
The qualifying checks and final packaged UI captures are recorded with the
receipt; preliminary native captures included blank/stale frames and an outer
scroll experiment. Those preliminary results do not qualify the final layout.
No new component models, file-format changes or expanded release acceptance are
claimed.
