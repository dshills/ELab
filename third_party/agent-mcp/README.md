# Agent MCP dependency notices

Verbatim upstream license, notice, author and patent files for the pinned runtime
modules added by the standard local MCP adapter. The SDK license records its
Apache/MIT licensing transition; preserve the complete supplied text. Gonum and
the desktop runtime retain their separate existing notice bundles.

`notices.json` pins module versions, Go module sums, files and ownership.
`python3 -B scripts/agent_notices.py` verifies this tree against go.mod/go.sum.
The release tool includes these notices beside the agent and in the app notice
bundle. This is a source/notice inventory, not an additional rights grant.
