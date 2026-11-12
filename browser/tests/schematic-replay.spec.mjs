import {test, expect} from '@playwright/test';

test('replay the H-bridge current reversal on its captured drawing', async ({page}) => {
 await page.goto('/');
 await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 await page.locator('#examples-button').click();
 await page.locator('#example-search').fill('Two MOSFET H-bridges');
 await page.locator('[data-example="dual-hbridge"]').click();
 await expect(page.locator('[data-object="B_MNB"]')).toBeVisible();
 await page.getByRole('button', {name: 'Record', exact: true}).click();
 const capture = page.getByRole('dialog', {name: 'Record an experiment'});
 await capture.locator('[name="duration"]').fill('22 ms');
 await capture.locator('[name="step"]').fill('1 ms');
 await capture.getByRole('button', {name: 'Start recording'}).click();
 await expect(page.getByRole('button', {name: 'Replay on schematic'})).toBeVisible();
 await page.getByRole('button', {name: 'Replay on schematic'}).click();
 const replay = page.getByRole('dialog', {name: 'Schematic replay'});
 const reading = replay.locator('[data-replay-reading]');
 const position = replay.getByRole('slider', {name: 'Replay sample position'});
 await expect(reading).toHaveText(/RL current -.*A · RL\.2 → RL\.1/);
 await position.fill(await position.getAttribute('max'));
 await expect(reading).toHaveText(/RL current [0-9].*A · RL\.1 → RL\.2/);
 await expect(replay.locator('[data-replay-arrow] path')).toHaveCount(1);
 await replay.getByRole('button', {name: 'Close replay'}).click();
 await expect(page.locator('#recording-position')).toHaveValue('0');
});
