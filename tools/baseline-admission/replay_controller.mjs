// Replay real saved decisions through fake callbacks. No UI or capture APIs.
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { withAdmittedBaseline } from './controller.mjs';

const cases = JSON.parse(await readFile(process.argv[2], 'utf8'));
let admitted = 0, refused = 0;
for (const item of cases) {
  let markers = 0, inputs = 0;
  const operation = withAdmittedBaseline(item.receipt, item.expected,
    async () => { markers++; }, async () => { inputs++; });
  if (item.should_admit) {
    await operation;
    assert.equal(markers, 1, item.name);
    assert.equal(inputs, 1, item.name);
    admitted++;
  } else {
    await assert.rejects(operation, undefined, item.name);
    assert.equal(markers, 0, item.name);
    assert.equal(inputs, 0, item.name);
    refused++;
  }
}
console.log(JSON.stringify({cases: cases.length, admitted, refused, actual_capture: false, actual_input: false}));
