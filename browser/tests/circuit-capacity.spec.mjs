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
  const command = async (name, params, id) => {
    const before = await state();
    const response = await page.request.post('/api/command', {data: {
      version: 1, id, key: id, command: name, params,
      expected_command_revision: before.command_revision,
      expected_document_generation: before.document.generation,
    }});
    expect((await response.json()).error).toBeUndefined();
  };
  const signs = async () => {
    const result = (await state()).mixed;
    return !result || result.stale ? [] : ['RL', 'B_RL'].map(id =>
      Math.sign(result.resistors.find(load => load.id === id)?.voltage_v));
  };
  await command('simulation.restart', {}, 'capacity-start');
  await expect.poll(signs).toEqual([-1, 1]);
  await expect.poll(async () => (await state()).mixed?.mosfets?.length).toBe(8);
  await command('simulation.step', {duration_s: .022}, 'capacity-first-edge');
  await expect.poll(signs).toEqual([1, -1]);
  await command('simulation.step', {duration_s: .011}, 'capacity-second-edge');
  await expect.poll(signs).toEqual([1, 1]);

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
