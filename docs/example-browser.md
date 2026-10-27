# Circuit examples

Choose **Examples** in the toolbar, or **Examples → Browse examples…** in the
application menu. Start here offers six approachable circuits. Each card shows
its schematic and name; selecting it shows a larger drawing, what to observe,
what to try and the model limits. The current catalog contains **63 examples**,
including the all-discrete class-AB amplifier.

Choose a topic to narrow the collection, or enter words in Search and press
Return. Search covers every category; all words must match. Short terms such
as RC and AC match whole words, and H-bridge also matches hbridge. Clear restores
the chosen category. A new category clears search. Arrow keys choose an example;
Return, double-click or **Open example** opens it. Close or Escape dismisses the
browser and returns focus to the toolbar. Wheel and scrollbar scrolling move
the collection without changing the selected circuit or simulation.

For the amplifier, choose **Amplifiers & filters**, or search globally for
**class AB**. Its ID is `discrete-class-ab`; the [circuit guide](discrete-class-ab.md)
explains the eight transistor stages, default traces and startup condition.
Agents can use `examples.search.set {"query":"class AB"}`, then
`examples.select {"id":"discrete-class-ab"}` to preview it, followed by
`example.load {"id":"discrete-class-ab"}` to open the same template.

Browsing never replaces the document, changes its dirty status or resets the
simulation. Opening uses the existing protected `example.load` command. Real
unsaved changes still offer Save / Discard / Cancel; Cancel keeps both the
edited circuit and the browser. A successful replacement closes the browser
and focuses component Search. Browser filters are session presentation, not
saved schematic content. Examples start as clean editable templates; save your
own copy before keeping changes.

## Commands shared with agents

`project.inspect` includes `example_browser`, containing visibility, category,
query, selected ID, matching IDs and category names/counts. `examples.list`
retains its existing metadata format. These mutations use the ordinary observed
revision, document generation and retry-key contract:

| Command | Parameters | Effect |
| --- | --- | --- |
| `examples.visibility.set` | `{"visible":true}` | Open or close, preserving filters |
| `examples.category.set` | `{"category":"timers"}` | Set topic and clear search |
| `examples.search.set` | `{"query":"555 flasher"}` | Search globally; 128 UTF-8 bytes maximum |
| `examples.select` | `{"id":"555-flasher"}` | Select an ID in the matching list |
| `example.load` | Existing ID/protected replacement parameters | Actually open the template |

`example.preview {"id":"555-flasher"}` is a pure query returning metadata,
category, SVG and `source_sha256` of the exact shipped graph/settings. It needs
no mutation guard. It neither loads nor solves a circuit and performs no file
or network IO. Desktop thumbnails rasterize this same drawing. Document export
still produces the original report, including its header and parts list;
thumbnail SVG omits that report framing. Browser choices never create history
entries or alter the existing educational models and solver limits.

Callbacks retain their original command/document context. A queued selection,
open, clear or close after another agent's edit cannot borrow a newer revision.
See [agent API](agent-api.md) for discovery, conflicts, retries and protected loads.
