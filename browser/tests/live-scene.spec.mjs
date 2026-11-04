import {test,expect} from '@playwright/test';
test('live counter updates display segments without replacing the focused drawing node',async({page})=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/');await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 const snapshot=async()=>(await (await page.request.get('/api/state')).json()).state;let s=await snapshot();
 const result=await page.request.post('/api/command',{data:{version:1,id:'live-display',key:'live-display',command:'example.load',params:{id:'display-counter',discard_changes:true},expected_command_revision:s.command_revision,expected_document_generation:s.document.generation}});expect((await result.json()).error).toBeUndefined();
 const display=page.locator('[data-object="DS1"]');await expect(display).toBeVisible();await display.evaluate(el=>el.dataset.identity='preserved');await display.focus();
 s=await snapshot();const initial=await display.locator('.display-segment.lit').count();
 await expect.poll(()=>display.locator('.display-segment.lit').count(),{timeout:15000}).not.toBe(initial);
 await expect(display).toHaveAttribute('data-identity','preserved');await expect(display).toBeFocused();expect((await snapshot()).command_revision).toBe(s.command_revision);expect(errors).toEqual([]);
});
