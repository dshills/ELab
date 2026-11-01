import {test,expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test.beforeEach(async ({page})=>{
 const errors=[];
 page.on('pageerror',error=>errors.push(error.message));
 page._elabErrors=errors;
 await page.goto('/');
 await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 await expect(page.locator('#component-value')).toBeVisible();
});
test.afterEach(async ({page})=>{expect(page._elabErrors).toEqual([]);});

test('keyboard tabs, dialog naming, focus return and text composition',async ({page})=>{
 const scope=page.getByRole('tab',{name:'Scope',exact:true});
 await scope.focus();
 await page.keyboard.press('ArrowRight');
 await expect(page.getByRole('tab',{name:'Spectrum',exact:true})).toBeFocused();
 await expect(page.getByRole('tab',{name:'Spectrum',exact:true})).toHaveAttribute('aria-selected','true');
 await page.keyboard.press('End');
 await expect(page.getByRole('tab',{name:'Sweep',exact:true})).toBeFocused();
 await page.keyboard.press('Home');
 await expect(scope).toBeFocused();
 await page.getByRole('button',{name:'Projects',exact:true}).click();
 const dialog=page.getByRole('dialog',{name:'Projects & recovery'});
 await expect(dialog).toBeVisible();
 await dialog.getByRole('tab',{name:'Saved projects'}).focus();
 await page.keyboard.press('ArrowRight');
 await expect(dialog.getByRole('tab',{name:'Recovery history'})).toBeFocused();
 await page.keyboard.press('Escape');
 await expect(dialog).not.toBeVisible();
 await expect(page.getByRole('button',{name:'Projects',exact:true})).toBeFocused();
 await page.locator('#schematic').dispatchEvent('keydown',{key:'w',isComposing:true,bubbles:true});
 await expect(page.locator('[data-tool="select"]')).toHaveClass(/active/);
});

test('edit, undo, save, reload and export a native project',async ({page})=>{
 const value=page.getByRole('textbox',{name:'Capacitance',exact:true});
 await value.fill('2.2 uF');
 await page.getByRole('button',{name:'Apply',exact:true}).click();
 await expect(value).toHaveValue(/2\.2/);
 await page.locator('#schematic').focus();
 await page.keyboard.press('Control+z');
 await expect(value).not.toHaveValue(/2\.2/);
 await value.fill('2.2 uF');
 await page.getByRole('button',{name:'Apply',exact:true}).click();
 await page.getByRole('button',{name:'Save project',exact:true}).click();
 const save=page.getByRole('dialog',{name:'Save project',exact:true});
 const name='Browser qualification '+test.info().project.name;
 await save.getByRole('textbox',{name:'Project name'}).fill(name);
 await save.getByRole('button',{name:'Save project',exact:true}).click();
 await expect(save).not.toBeVisible();
 await page.reload();
 await expect(value).toHaveValue(/2\.2/);
 await expect(page.locator('#document-name')).toContainText(name);
 await page.getByRole('tab',{name:'Document',exact:true}).click();
 const download=page.waitForEvent('download');
 await page.getByRole('button',{name:'Download .elab copy'}).click();
 expect((await download).suggestedFilename()).toMatch(/\.elab$/);
});

test('capture, exact keyboard playback and retained analysis',async ({page})=>{
 await page.getByRole('button',{name:'Record',exact:true}).click();
 const dialog=page.getByRole('dialog',{name:'Record an experiment'});
 await dialog.locator('[name="duration"]').fill('20 ms');
 await dialog.locator('[name="step"]').fill('1 ms');
 await dialog.getByRole('button',{name:'Start recording'}).click();
 await expect(page.locator('#recording-position')).toBeVisible();
 await page.locator('#recording-position').focus();
 await page.keyboard.press('Home');
 await expect(page.locator('#recording-position')).toHaveValue('0');
 await page.keyboard.press('ArrowRight');
 await expect(page.locator('#recording-position')).toHaveValue('1');
 await expect(page.locator('#recording-position-label')).toContainText('· 1 /');
 await expect(page.locator('#recording-keep')).toHaveAttribute('aria-pressed','true');
 await page.getByRole('button',{name:'Analyze recording'}).click();
 const analysis=page.getByRole('dialog',{name:'Recording analysis',exact:true});
 await expect(analysis).toBeVisible();
 await analysis.getByRole('tab',{name:'Compare',exact:true}).focus();
 await page.keyboard.press('End');
 await expect(analysis.getByRole('tab',{name:'Step response',exact:true})).toBeFocused();
});

test('narrow layout exposes parts and inspector without losing keyboard controls',async ({page})=>{
 await page.setViewportSize({width:600,height:900});
 await page.getByRole('button',{name:'Show parts',exact:true}).click();
 await page.getByRole('searchbox',{name:'Find a component'}).fill('resistor');
 const part=page.getByRole('button',{name:'Resistor',exact:true});
 await part.focus();
 await page.keyboard.press('Enter');
 await expect(page.locator('#schematic')).toHaveClass(/tool-place/);
 await expect(page.locator('#schematic')).toBeFocused();
 const count=await page.locator('#graph-counts').textContent();
 await page.keyboard.press('Enter');
 await expect(page.locator('#graph-counts')).not.toHaveText(count);
 await page.getByRole('button',{name:'Show inspector',exact:true}).click();
 await expect(page.getByRole('tab',{name:'Simulate',exact:true})).toBeVisible();
 await expect(page.locator('#app')).toHaveJSProperty('scrollWidth',600);
});

test('workbench and modal have no serious or critical automated accessibility violations',async ({page},testInfo)=>{
 for(const modal of [false,true]) {
  if(modal)await page.getByRole('button',{name:'Projects',exact:true}).click();
  const result=await new AxeBuilder({page}).withTags(['wcag2a','wcag2aa','wcag21a','wcag21aa']).analyze();
  await testInfo.attach(modal?'projects-accessibility':'workbench-accessibility',{body:JSON.stringify(result.violations,null,2),contentType:'application/json'});
  expect.soft(result.violations.filter(item=>['serious','critical'].includes(item.impact))).toEqual([]);
 }
});
