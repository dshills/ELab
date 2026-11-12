import {test, expect} from '@playwright/test';

test('dual H-bridge opens, reverses each load and saves in the browser', async ({page}) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto('/');
  await expect(page.locator('#connection-status')).toHaveText('Local engine connected');

  await page.locator('#examples-button').click();
  await page.locator('#example-search').fill('Two MOSFET H-bridges');
  await page.locator('[data-example="dual-hbridge"]').click();
  await expect(page.locator('[data-object="B_MNB"]')).toBeVisible();
  await page.getByRole('button', {name: 'Pause', exact: true}).click();
  await expect(page.getByRole('button', {name: 'Resume', exact: true})).toBeVisible();

  const state = async () => (await (await page.request.get('/api/state')).json()).state;
  const signs = async () => {
    const result = (await state()).mixed;
    return !result || result.stale ? [] : ['RL', 'B_RL'].map(id =>
      Math.sign(result.resistors.find(load => load.id === id)?.voltage_v));
  };
  await page.getByRole('button', {name: 'Restart simulation', exact: true}).click();
  await expect.poll(signs).toEqual([-1, 1]);
  await expect.poll(async () => (await state()).mixed?.mosfets?.length).toBe(8);

  await page.getByRole('tab', {name: 'Simulate', exact: true}).click();
  const duration = page.getByRole('textbox', {name: 'Step duration (s)', exact: true});
  const advance = page.getByRole('button', {name: 'Advance simulation', exact: true});
  for (const [interval, elapsed, expectedSigns] of [['22 ms', '0.022', [1, -1]], ['11 ms', '0.033', [1, 1]]]) {
    await duration.fill(interval);
    await advance.click();
    await expect(page.locator('#time-label')).toHaveText(`t = ${elapsed} s`);
    await expect.poll(signs).toEqual(expectedSigns);
  }

  await page.locator('#save-button').click();
  const save = page.getByRole('dialog', {name: 'Save project', exact: true});
  const name = 'Dual bridge ' + test.info().project.name;
  await save.getByRole('textbox', {name: 'Project name'}).fill(name);
  await save.getByRole('button', {name: 'Save project', exact: true}).click();
  await expect(save).not.toBeVisible();
  await page.reload();
  await expect(page.locator('#document-name')).toContainText(name);
  await expect(page.locator('[data-object="B_MNB"]')).toBeVisible();
  await expect.poll(async () => (await state()).mixed?.mosfets?.length).toBe(8);
  expect(errors).toEqual([]);
});
