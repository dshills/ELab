import {test, expect} from '@playwright/test';

test('try and discard a capacitor value, then apply it as a normal edit', async ({page}) => {
 await page.goto('/');
 const value = page.getByRole('textbox', {name: 'Capacitance', exact: true});
 await expect(value).toBeVisible();
 const original = await value.inputValue();
 const launch = page.getByRole('button', {name: 'Compare a Change', exact: true});
 const dialog = page.getByRole('dialog', {name: 'Compare a Change'});
 const trial = dialog.getByRole('textbox', {name: 'Trial value'});
 const keep = dialog.getByRole('button', {name: 'Keep trial value'});
 async function compare() {
  await launch.click();
  await trial.fill('2.2 uF');
  await dialog.getByRole('button', {name: 'Run comparison'}).click();
  await expect(dialog.getByRole('status')).toHaveText('Comparison ready.');
  await expect(dialog.getByRole('img', {name: 'Original and Trial versus time'})).toBeVisible();
  await expect(dialog.getByText(/first crossing moved from/)).toBeVisible();
 }
 await compare();
 await dialog.getByRole('button', {name: 'Discard trial'}).click();
 await expect(value).toHaveValue(original);
 await compare();
 await keep.click();
 await expect(dialog).not.toBeVisible();
 await expect(value).toHaveValue(/2\.2/);
 await page.locator('#schematic').focus();
 await page.keyboard.press('Control+z');
 await expect(value).toHaveValue(original);
});
