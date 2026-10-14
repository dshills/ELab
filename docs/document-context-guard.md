# Guarding rendered actions across document changes

Mutations may include top-level `expected_document_generation`, using `snapshot.document.generation` from the same observation as their target and values. They still require `expected_revision` or `expected_command_revision` and a retry key:

```json
{
  "version": 1,
  "id": "set-resistance",
  "command": "component.set",
  "params": {"id": "R1", "value": "2k"},
  "expected_command_revision": 12,
  "expected_document_generation": 3,
  "key": "set-resistance-1"
}
```

All supplied guards must match at mutation admission. A mismatched generation returns `conflict` before any circuit/history/file/physical effects. It prevents a saved/imported document with the same component ID from becoming the target of an old rendered action. This matters when asynchronous project opening publishes a replacement after admission without another command revision. Revision alone cannot distinguish that case.

The generation guard is optional and does not replace a revision guard. Queries retain their existing read-only behavior. Existing clients that omit it work as before. It is separate from the similarly named parameter-level generation guard used by replacement/file commands; an envelope guard protects any mutation's context, while parameter guards belong to the particular operation.

Retry lookup remains before guard evaluation. A successful key returns its original independent response even after document replacement or changed guard values; it does not reapply the operation. Inspect current state before continuing from a duplicate historical response. Different command/parameters with the same key still refuse as key reuse.

Native ordinary action buttons capture command revision and document generation from the rendered view and use these shared admission rules. Continuous solver samples do not alter command revision or document identity and therefore do not invalidate those buttons by themselves. This change does not claim complete raw-field/menu/gesture coverage; the [native inventory](../research/13-native-command-inventory.md) lists the remaining binding work.
