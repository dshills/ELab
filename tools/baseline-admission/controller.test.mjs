import assert from 'node:assert/strict';
import test from 'node:test';
import { validateAdmission, withAdmittedBaseline } from './controller.mjs';

const expected = {
  trial_id: 'rc-test', attempt_id: 'unique-offline-attempt-001', frame_index: 1,
  png: 'frame-' + 'a'.repeat(64) + '.png', png_sha256: 'a'.repeat(64),
  tool_sha256: 'b'.repeat(64), profile_sha256: 'c'.repeat(64),
  width: 1053, height: 802, regions: ['accepted-capacitor', 'live-header'],
};
const receipt = {
  version: 1, admitted: true, ...expected, tool_exit_code: 0,
  result: {
    version: 1, admitted: true, code: 'admitted', width: 1053, height: 802,
    image_sha256: expected.png_sha256, profile_sha256: expected.profile_sha256,
    regions: expected.regions.map(name => ({name, matches: true, rgb_sha256: 'd'.repeat(64)})),
  },
};

test('valid admission orders marker acknowledgment before input', async () => {
  const calls = [];
  const result = await withAdmittedBaseline(receipt, expected,
    async () => { calls.push('marker'); await Promise.resolve(); calls.push('ack'); },
    async () => { calls.push('apply'); return 'applied'; });
  assert.deepEqual(calls, ['marker', 'ack', 'apply']);
  assert.equal(result, 'applied');
});

const refusals = {
  missing: () => null,
  'false-admission': r => { r.admitted = false; },
  'truthy-admission': r => { r.admitted = 'true'; },
  'wrong-version': r => { r.version = 2; },
  'failed-process': r => { r.tool_exit_code = 2; },
  'different-trial': r => { r.trial_id = 'rc-other'; },
  'previous-attempt': r => { r.attempt_id = 'previous-offline-attempt'; },
  'different-frame': r => { r.frame_index++; },
  'different-file': r => { r.png = '../frame.png'; },
  'different-png-sha': r => { r.png_sha256 = '0'.repeat(64); },
  'different-profile': r => { r.profile_sha256 = '0'.repeat(64); },
  'different-tool': r => { r.tool_sha256 = '0'.repeat(64); },
  'missing-tool-result': r => { delete r.result; },
  'tool-refusal': r => { r.result.admitted = false; },
  'tool-wrong-code': r => { r.result.code = 'region-mismatch'; },
  'tool-wrong-version': r => { r.result.version = 2; },
  'tool-wrong-image': r => { r.result.image_sha256 = '0'.repeat(64); },
  'tool-wrong-profile': r => { r.result.profile_sha256 = '0'.repeat(64); },
  'wrong-width': r => { r.result.width++; },
  'wrong-height': r => { r.result.height++; },
  'missing-regions': r => { delete r.result.regions; },
  'extra-region': r => { r.result.regions.push(r.result.regions[0]); },
  'reordered-regions': r => { r.result.regions.reverse(); },
  'mismatched-region': r => { r.result.regions[0].matches = false; },
  'malformed-region-sha': r => { r.result.regions[0].rgb_sha256 = 'bad'; },
};
for (const [name, change] of Object.entries(refusals)) {
  test(name + ' sends neither marker nor input', async () => {
    let value = structuredClone(receipt);
    const changed = change(value);
    if (changed !== undefined) value = changed;
    let markers = 0, inputs = 0;
    await assert.rejects(withAdmittedBaseline(value, expected,
      async () => { markers++; }, async () => { inputs++; }));
    assert.equal(markers, 0);
    assert.equal(inputs, 0);
  });
}
test('marker acknowledgment failure prevents input', async () => {
  let inputs = 0;
  await assert.rejects(withAdmittedBaseline(receipt, expected,
    async () => { throw new Error('ack missing'); }, async () => { inputs++; }));
  assert.equal(inputs, 0);
});
test('incomplete expectations cannot accidentally equal absent receipt fields', () => {
  for (const key of Object.keys(expected)) {
    const incomplete = structuredClone(expected);
    delete incomplete[key];
    assert.throws(() => validateAdmission(receipt, incomplete), key);
  }
});
