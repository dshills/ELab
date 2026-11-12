import {test, expect} from '@playwright/test';

test('move a circuit part to a child sheet and follow its ports after reopening', async ({page}) => {
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto('/');
  await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
  await page.getByRole('button', {name:'Pause', exact:true}).click();

  const currentSheet = page.getByRole('combobox', {name:'Current sheet'});
  const manager = page.getByRole('dialog', {name:'Manage sheets'});
  const showManager = () => page.getByRole('button', {name:'Manage sheets', exact:true}).click();
  await showManager();
  await manager.getByRole('textbox', {name:'Child sheet name'}).fill('Filter');
  await manager.getByRole('button', {name:'Add child sheet'}).click();
  await expect(currentSheet.locator('option:checked')).toHaveText('Main / Filter');
  await expect(page.locator('[data-object="C1"]')).toHaveCount(0);
  const childID = await currentSheet.inputValue();

  await currentSheet.selectOption('');
  await page.locator('[data-object="C1"]').press('Enter');
  await expect(page.locator('.selected-heading h2')).toHaveText('C1');
  await showManager();
  await manager.getByRole('combobox', {name:'Destination sheet'}).selectOption(childID);
  await manager.getByRole('button', {name:'Move selection'}).click();
  await expect(currentSheet).toHaveValue(childID);
  await expect(page.locator('[data-object="C1"]')).toBeVisible();
  await expect(page.locator('[data-object="R1"]')).toHaveCount(0);
  const lead = page.getByRole('button', {name:'C1.1 to Main', exact:true});
  await expect(lead).toBeVisible();
  await lead.press('Enter');
  await expect(currentSheet).toHaveValue('');
  await expect(page.locator('[data-object="R1"]')).toBeVisible();

  await currentSheet.selectOption(childID);
  await page.locator('#save-button').click();
  const save = page.getByRole('dialog', {name:'Save project', exact:true});
  const name = 'Two sheet RC ' + test.info().project.name;
  await save.getByRole('textbox', {name:'Project name'}).fill(name);
  await save.getByRole('button', {name:'Save project', exact:true}).click();
  await expect(save).not.toBeVisible();
  await page.reload();
  await expect(page.locator('#document-name')).toContainText(name);
  await expect(currentSheet).toHaveValue(childID);
  await expect(lead).toBeVisible();
  const state = (await (await page.request.get('/api/state')).json()).state;
  expect(state.document.format_version).toBe(6);
  expect(state.wires.filter(wire => wire.port)).toHaveLength(2);

  await currentSheet.selectOption('');
  await page.locator('[data-trace-location^="C1."]').first().click();
  await expect(currentSheet).toHaveValue(childID);
  await expect(page.locator('[data-object="C1"]')).toBeFocused();

  await page.getByRole('button', {name:'Find in project', exact:true}).click();
  const find = page.getByRole('dialog', {name:'Find in project'});
  await find.getByRole('searchbox').fill('C1.1');
  await expect(find.getByText('Port · Main / Filter', {exact:true})).toBeVisible();
  await find.getByRole('searchbox').fill('R1');
  await find.getByRole('button', {name:/^R1 · Resistor/}).click();
  await expect(currentSheet).toHaveValue('');
  await expect(page.locator('[data-object="R1"]')).toBeFocused();
  expect(errors).toEqual([]);
});

test('editing a part preserves unrelated schematic nodes', async ({page}) => {
  await page.goto('/');
  await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
  const resistor = page.locator('[data-object="R1"]');
  await resistor.evaluate(el => {el.dataset.retainedForTest = 'yes';});
  await page.getByRole('textbox', {name:'Capacitance', exact:true}).fill('2.2 uF');
  await page.getByRole('button', {name:'Apply', exact:true}).click();
  await expect(page.getByRole('textbox', {name:'Capacitance', exact:true})).toHaveValue(/2\.2/);
  await expect(resistor).toHaveAttribute('data-retained-for-test', 'yes');
});
