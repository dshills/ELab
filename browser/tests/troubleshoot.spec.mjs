import {test, expect} from '@playwright/test';

test('measure startup voltage between two selected terminals', async ({page}) => {
 await page.goto('/');
 await page.locator('#command-button').click();
 const commands = page.getByRole('dialog', {name: 'Commands'});
 await commands.getByRole('searchbox').fill('Why isn');
 await commands.getByRole('button', {name: 'Why isn’t this working?'}).click();
 const dialog = page.getByRole('dialog', {name: 'Why isn’t this working?'});
 await expect(dialog.getByRole('status')).toHaveText('No connection or startup LED polarity issue found.');
 await dialog.getByLabel('Red probe').selectOption('V1.negative');
 await dialog.getByLabel('Black probe').selectOption('V1.positive');
 await dialog.getByRole('button', {name: 'Measure voltage'}).click();
 await expect(dialog.getByText(/red minus black/)).toHaveText(/^[-−]5 V/);
});
