import {test, expect} from '@playwright/test';

test('run and save a threshold test with the circuit', async ({page}) => {
 await page.goto('/');
 await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 const open = page.getByRole('button', {name: 'Experiment tests', exact: true});
 await open.click();
 const dialog = page.getByRole('dialog', {name: 'Experiment tests'});
 await dialog.getByLabel('Test name').fill('Capacitor reaches 3 V');
 await dialog.getByLabel('Measurement').selectOption('v:C1');
 await dialog.getByRole('textbox', {name: 'Threshold', exact: true}).fill('3 V');
 await dialog.getByLabel('Duration').fill('20 ms');
 await dialog.getByRole('button', {name: 'Run test', exact: true}).click();
 await expect(dialog.getByRole('status')).toHaveText(/^Pass · Capacitor reaches 3 V/);
 await expect(dialog.getByRole('img', {name: /Threshold versus time/})).toBeVisible();
 await dialog.getByRole('button', {name: 'Save test', exact: true}).click();
 await expect(dialog.getByRole('status')).toHaveText('Test saved in this project.');
 await dialog.getByRole('button', {name: 'Close experiment tests'}).click();
 await page.reload();
 await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 await open.click();
 await dialog.getByRole('button', {name: 'Run Capacitor reaches 3 V'}).click();
 await expect(dialog.getByRole('status')).toHaveText(/^Pass · Capacitor reaches 3 V/);
});
