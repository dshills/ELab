# Inspector tooltip retirement qualification

October 25, 2026. See [implementation notes](../../../research/115-inspector-tooltip-retirement.md).
The source checkpoint pins the new elab4 compatibility SDK and records the
reproduced failure and passing focused regression. The complete application
checks and /256 package/native receipts are appended at closeout.

The change closes only native TooltipStage popups originating in a retiring
managed subtree. It does not dispatch numeric Change, keyboard, mouse or other
ELab commands. The core popup-stack inspection lock is released before exact
stage deletion; no menu or retained sibling tooltip is dismissed. Reconciliation
moves surviving descendants before retiring obsolete ancestors.

The regression uses routed MouseMove and a controlled hover clock. Before the
fix, both visible bounds and arrow tooltips remain after removal. Afterward,
visible/pending cases, same-ID fresh owners and subsequent normal help pass.
The second regression verifies retained help/menu identities, same-ID owner
replacement and zero numeric Change callbacks.

The whole exported archive is checked against its manifest. The prepared SDK
verifies the ordered stage recipe, including new exact-input/output stage 111.
All 8,735 exported files are pinned; comparison reports only the new lifecycle
seam, binding, regression/recipe files and regenerated SDK documentation.
Previous /255 and qualified alpha hashes remain in preserved.json.

No ELab application Go source, public command, component model, project schema,
simulation limit or established electrical assertion is changed. The upstream
remains f9dee54, with compatibility source 2c349fc. Exact source/archive hashes
are in sdk.lock.json.
