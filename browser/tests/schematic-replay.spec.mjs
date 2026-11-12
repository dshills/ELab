import {test, expect} from '@playwright/test';
import {readFile} from 'node:fs/promises';
import {pathToFileURL} from 'node:url';

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

test('share a captured H-bridge replay and download its circuit offline', async ({page, browser}, testInfo) => {
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
 await page.getByRole('button', {name: 'Replay on schematic'}).click();
 const replay = page.getByRole('dialog', {name: 'Schematic replay'});
 const position = replay.getByRole('slider', {name: 'Replay sample position'});
 await expect(replay.locator('[data-replay-reading]')).toHaveText(/RL current -/);
 await position.fill(await position.getAttribute('max'));
 const lastReading = await replay.locator('[data-replay-reading]').textContent();
 const note = 'Watch the reversal <img src=x onerror="alert(1)">';
 await replay.getByLabel('Note for the recipient').fill(note);
 await replay.getByRole('checkbox', {name: /Include an editable/}).check();
 const [download] = await Promise.all([
  page.waitForEvent('download'),
  replay.getByRole('button', {name: 'Download offline replay'}).click(),
 ]);
 const path = testInfo.outputPath('shared-replay.html');
 await download.saveAs(path);

 const context = await browser.newContext({offline: true});
 try {
  const local = await context.newPage();
  const remoteRequests = [];
  local.on('request', request => {if (/^https?:/.test(request.url())) remoteRequests.push(request.url());});
  await local.goto(pathToFileURL(path).href);
  await expect(local.getByText(note, {exact: true})).toBeVisible();
  await expect(local.locator('[data-replay-reading]')).toHaveText(lastReading);
  await local.getByRole('slider', {name: 'Replay sample position'}).fill('0');
  await expect(local.locator('[data-replay-reading]')).toHaveText(/RL current -.*A · RL\.2 → RL\.1/);
  const [project] = await Promise.all([
   local.waitForEvent('download'),
   local.getByRole('button', {name: 'Download editable .elab project'}).click(),
  ]);
  const projectPath = testInfo.outputPath('captured.elab');
  await project.saveAs(projectPath);
  const data = JSON.parse(await readFile(projectPath, 'utf8'));
  expect(data.format).toBe('elab.schematic');
  expect(data.paused).toBe(true);
  expect(data.graph.components.some(part => part.id === 'B_MNB')).toBe(true);
  expect(remoteRequests).toEqual([]);
 } finally {
  await context.close();
 }
});
