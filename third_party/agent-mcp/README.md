# ELab runtime dependency notices

Verbatim upstream license, notice, author and patent files for the pinned
runtime modules used by the standard local MCP adapter and schematic renderer.
The SDK license records its Apache/MIT licensing transition; preserve the
complete supplied text. Gonum and Go retain their separate notice files.

`notices.json` pins module versions, Go module sums, files and ownership.
`python3 -B scripts/agent_notices.py` verifies this tree against go.mod/go.sum.
The web release tools include the notices for each linked module. This is a
source/notice inventory, not an additional rights grant.
