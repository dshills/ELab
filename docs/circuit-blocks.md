# Reusable circuit blocks

Open **Reusable RC circuit block** (`rc-block`). Select FILTER to edit its
published resistance/capacitance or Show internals. Hidden parts remain ordinary
components with their original IDs, electrical laws, readings and stored states.
A symbol is a drawing boundary, not an extra circuit or an averaged model.

In **Document → Circuit blocks**, enter a new ID/name, comma-separated member IDs,
named pins such as `in=R1.1,ground=C1.2,out=C1.1`, and optional numeric values such
as `resistance=R1,capacitance=C1`. Choose the symbol's grid position and Create.
Every existing external wire must meet a declared pin. Missing pins cause an
atomic refusal; crossing a box or drawing over it never creates a connection.

Select a block and **Copy block**. In a new/existing drawing, Document → Circuit
blocks → **Paste copied block**, with a fresh ID and free position. The in-app
clipboard owns an independent template, survives document changes and never
changes the operating system clipboard. It is session-only. Project files retain
complete instances; reopen a saved instance and copy it to reuse it later.
Paste copies only members/internal wires, assigning independent IDs and settings;
it never copies the old drawing's external connections. Ordinary Duplicate also
copies the entire independent block. Existing physical and drawing limits apply.

**Show internals** reveals every member for selection and normal configuration.
Select the block again from the Document list and press **Hide internals**
(`block.view.set {expanded:false}`). Named external
pins remain the connection boundary. Release into ordinary parts before adding or
deleting members or splitting internal wires; release preserves exact connections.
Deleting a whole block removes its members and incident wires as one history edit.
Moving the symbol translates its members; rotating it rotates the symbol/pin
positions. The internal drawing keeps its own component rotations.

Collapsed members do not occupy invisible positions on the main sheet. Their
internal coordinates remain independent of other collapsed blocks and outside
parts. If expansion, collapse or Release would overlap visible grid points, the
action refuses atomically; move the block to a free area first.

All instance metadata, published values and expansion state save/reopen and
participate in Undo/Redo. Grouping, release, move and visual expansion preserve
accepted electrical state. Editing an electrical parameter uses ordinary restart
or paused-stale rules. Named-pin voltage probes resolve to the actual internal
terminal, including existing retained live samples. No aggregate block-current
reading is fabricated. Wire currents remain conservative; boundary pins are
zero-injection virtual edges, just as connectivity aliases are not current sources.

The first implementation is explicitly **single-level**, with 1–32 members,
1–16 named pins and at most 32 unique numeric parameters. Names address exact
member terminals/values. Block IDs are ASCII ValidID up to 24 characters; pin and
parameter names are ValidID up to 32. Long pin captions abbreviate in the symbol;
the inspector and protocol retain full names. Ground retains the ordinary common
reference. Nested blocks and net-label members are refused so there is no hidden
scope or accidental short between copied instances. Sources, storage and every
other device still consume their original capacities.

Users and agents share:

- `block.create {id,name,position,members,ports,parameters}`; ports are objects
  `{name,terminal}`, parameters are `{name,component_id}`. Arrays are required,
  with an empty parameters array when none are exposed.
- `block.inspect {id}`: pure self-contained template query.
- `block.copy {id}`, then `block.paste {id,position}`: independent session copy.
- `block.parameter.set {id,parameter,value}`: ordinary quantity text and bounds.
- `block.view.set {id,expanded}` and `block.release {id}`.
- Ordinary move/rotate/duplicate/delete, named-terminal wire, probe, project,
  export/print and guarded atomic edit commands.

The ordinary catalog uses short summaries to retain its 64 KiB budget; filtered
`capabilities.list {command:...}` returns the original full description, required
fields, bounds and completion contract. Output schemas and fields are retained.
