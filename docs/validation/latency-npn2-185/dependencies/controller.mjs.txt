// Pure admission gate: no capture, filesystem, application, or input APIs.
const digest = value => typeof value === 'string' && /^[0-9a-f]{64}$/.test(value);
const require = (condition, message) => { if (!condition) throw new Error(message); };

export function validateAdmission(receipt, expected) {
  require(expected && typeof expected.trial_id === 'string' && /^[a-z0-9-]{1,64}$/.test(expected.trial_id), 'Invalid expected trial');
  require(typeof expected.attempt_id === 'string' && /^[a-zA-Z0-9-]{16,80}$/.test(expected.attempt_id), 'Invalid expected attempt');
  require(Number.isSafeInteger(expected.frame_index) && expected.frame_index > 0 && expected.frame_index <= 300, 'Invalid expected frame');
  for (const key of ['png_sha256', 'tool_sha256', 'profile_sha256']) require(digest(expected[key]), 'Invalid expected ' + key);
  require(expected.png === 'frame-' + expected.png_sha256 + '.png', 'Invalid expected PNG');
  require(Number.isSafeInteger(expected.width) && expected.width > 0 && Number.isSafeInteger(expected.height) && expected.height > 0, 'Invalid expected dimensions');
  require(Array.isArray(expected.regions) && expected.regions.length > 0 && new Set(expected.regions).size === expected.regions.length && expected.regions.every(x => typeof x === 'string' && x.length > 0), 'Invalid expected regions');
  require(receipt && receipt.version === 1 && receipt.admitted === true && receipt.tool_exit_code === 0, 'Baseline was not admitted');
  for (const key of ['trial_id', 'attempt_id', 'frame_index', 'png', 'png_sha256', 'tool_sha256', 'profile_sha256']) require(receipt[key] === expected[key], 'Admission binding differs: ' + key);
  const result = receipt.result;
  require(result && result.version === 1 && result.admitted === true && result.code === 'admitted', 'Tool refused baseline');
  require(result.profile_sha256 === expected.profile_sha256 && result.image_sha256 === expected.png_sha256, 'Tool data identity differs');
  require(result.width === expected.width && result.height === expected.height, 'Tool dimensions differ');
  require(Array.isArray(result.regions) && result.regions.length === expected.regions.length && result.regions.every((r, i) => r.name === expected.regions[i] && r.matches === true && digest(r.rgb_sha256)), 'Baseline regions differ');
  return true;
}

// The caller supplies the existing CUA operation and marker/ack validation. An
// invalid receipt cannot enter either callback. Full-recording visibility and
// causal timing analysis remain necessary after a valid initial baseline.
export async function withAdmittedBaseline(receipt, expected, requestAndValidateMarker, apply) {
  validateAdmission(receipt, expected);
  require(typeof requestAndValidateMarker === 'function' && typeof apply === 'function', 'Missing controller operations');
  await requestAndValidateMarker();
  return await apply();
}
