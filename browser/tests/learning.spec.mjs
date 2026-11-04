import {test,expect} from '@playwright/test';
test('beginner activity changes LED current, preserves cancellation and reopens saved work',async({page})=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/');
 await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 await page.getByRole('button',{name:/Learn circuits & shortcuts/}).click();
 await page.getByRole('button',{name:'Start LED activity',exact:true}).click();
 const replace=page.getByRole('dialog',{name:'Replace this circuit?'});
 if(await replace.isVisible())await replace.getByRole('button',{name:'Replace circuit'}).click();
 await expect(page.locator('#learning-title')).toHaveText('Make a light change');
 const guide=page.locator('#learning-guide');await expect(guide.getByRole('status')).toContainText('Step 1 of 4');
 await guide.getByRole('button',{name:'Next step',exact:true}).click();await expect(guide).toContainText('milliamperes');
 const current=async()=>{const s=(await (await page.request.get('/api/state')).json()).state;return s.result?.current_a??s.mixed?.leds?.[0]?.current_a;};
 await expect.poll(current).toBeGreaterThan(0);const before=await current();
 await expect.poll(async()=>(await (await page.request.get('/api/state')).json()).state.selected).toBe('R1');
 // Reselecting the current part must not invalidate an immediately focused edit.
 await page.route('**/api/command?presentation=true',async route=>{if(route.request().postDataJSON()?.command==='component.select')await new Promise(resolve=>setTimeout(resolve,200));await route.continue();});
 await page.locator('[data-object="R1"]').press('Enter');
 await page.getByRole('textbox',{name:'Resistance',exact:true}).fill('2 kΩ');await page.getByRole('button',{name:'Apply',exact:true}).click();
 await expect.poll(current).toBeLessThan(before);
 await page.getByRole('button',{name:/Learn circuits & shortcuts/}).click();await page.getByRole('button',{name:'Start capacitor activity',exact:true}).click();
 await page.getByRole('dialog').filter({has:page.locator('#replace-message')}).getByRole('button',{name:'Keep editing',exact:true}).click();
 await expect(page.locator('#learning-title')).toHaveText('Make a light change');
 await expect(page.getByRole('textbox',{name:'Resistance',exact:true})).toHaveValue(/2/);
 await page.getByRole('button',{name:'Save project',exact:true}).click();const save=page.getByRole('dialog',{name:'Save project',exact:true});
 await save.getByRole('textbox',{name:'Project name'}).fill('Beginner LED rehearsal');await save.getByRole('button',{name:'Save project',exact:true}).click();await expect(save).not.toBeVisible();
 await page.reload();await expect(page.locator('#document-name')).toContainText('Beginner LED rehearsal');
 await expect.poll(current).toBeLessThan(before);expect(errors).toEqual([]);
});
