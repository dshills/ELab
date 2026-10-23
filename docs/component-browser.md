# Component browser

ELab keeps the existing 18 simulated component kinds. The browser starts in
**Common**, with DC source, resistor, red LED, capacitor, ground and SPST switch.
Use the category control to browse Sources, Passives, Semiconductors,
Amplifiers / ICs, Switches or Connections. **All components** shows the full set.
Each category displays its part count; each row shows the actual schematic
symbol beside the component name. Common and the six families fit without scrolling. All components uses a
scrollable list with its category and search controls kept above it.

Type a name or hobbyist alias in **Search all components**, then press Return.
Search always covers the whole catalog, regardless of the selected category.
For example, `battery` finds the DC source and `MOSFET` finds both MOSFETs even
from Common. Case and surrounding whitespace do not matter. Multiple words
must all match the component's searchable metadata. Clear the query and press
Return to return to the selected category. Choosing a category clears search.

Return from search focuses its results when there are matches. Use arrows,
Home/End, page keys or an initial letter in the list, and Return/Space to choose
a component. Click the sheet to place it, or use the existing exact position
controls. Browsing and searching do not change the current placement kind.
An empty result list explains how to try another name or alias.

## Shared agent interface

- `parts.list` returns all 18 kinds, regardless of the browser. Each part also
  includes its stable `category` ID and `common` boolean, plus the existing
  aliases, terminals, value domain, model and limitations.
- `project.inspect` includes `part_category`, `part_categories` (IDs, names and
  whole-catalog counts), `part_search` and effective `matching_parts`.
- `parts.category.set` takes `{"category":"passives"}`. Accepted IDs are
  `common`, `all`, `sources`, `passives`, `semiconductors`, `amplifiers`,
  `switches` and `connections`. It clears the query atomically. Unknown IDs
  fail without changing state. The capability schema publishes the enum.
- `parts.search.set` takes `{"query":"MOSFET"}`. A nonempty query searches all
  categories; an empty/whitespace-only query restores the selected group.
- `drawing.tool.set` with `{"tool":"place","kind":"nmos"}` still selects
  placement directly. Agents need not manipulate the browser to place a part.
  `component.place` and all editing/simulation commands retain their behavior.

Browser state is session-only, independent of schematic saving and undo/redo.
These commands use the normal revision/document guards and retry semantics.
They preserve the graph, electrical run, undo history and document dirty flag.
Category/search changes from an external agent are reflected in the native UI.
No file-format, model or component-set changes are introduced.
