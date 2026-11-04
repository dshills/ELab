import {test,expect} from '@playwright/test';
const snapshot=async page=>(await (await page.request.get('/api/state')).json()).state;
test('circuit checks identify a disconnected terminal and guide a real repair',async({page})=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/');await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 const original=await snapshot(page),wire=original.wires[0];
 await page.locator(`[data-object="${wire.id}"]`).press('Enter');await page.keyboard.press('Delete');
 const panel=page.getByRole('complementary',{name:'Circuit checks'});await expect(panel).toBeVisible();await expect(panel).toContainText('Connect');
 const broken=await snapshot(page),help=broken.diagnostic_presentation.items.find(v=>v.terminal_refs?.length);expect(help).toBeTruthy();
 const ref=help.terminal_refs[0];await expect(page.locator(`[data-pin="${ref}"] .pin-hit`)).toHaveClass(/diagnostic-target/);
 await panel.getByText('Technical details',{exact:true}).first().click();expect((await snapshot(page)).command_revision).toBe(broken.command_revision);
 await panel.getByRole('button',{name:`Connect ${ref}`,exact:true}).click();
 const destination=ref===wire.from?wire.to:wire.from;await page.locator(`[data-pin="${destination}"] .pin-hit`).press('Enter');
 await expect(panel).not.toBeVisible();await expect(page.locator('#schematic .diagnostic-target')).toHaveCount(0);
 await expect.poll(async()=>(await snapshot(page)).wires.length).toBe(original.wires.length);expect(errors).toEqual([]);
});
test('invalid setting help stays local to the request and clears after correction',async({page})=>{
 await page.goto('/');await expect(page.locator('#connection-status')).toHaveText('Local engine connected');await page.locator('[data-object="R1"]').press('Enter');
 const before=await snapshot(page);const resistance=page.getByRole('textbox',{name:'Resistance',exact:true});await resistance.fill('not a resistor value');await page.getByRole('button',{name:'Apply',exact:true}).click();
 const panel=page.getByRole('complementary',{name:'Circuit checks'});await expect(panel).toContainText('R1 needs a valid resistance');
 expect((await snapshot(page)).components.find(p=>p.id==='R1').value).toBe(before.components.find(p=>p.id==='R1').value);
 await expect(page.locator('[data-object="R1"]')).toHaveClass(/diagnostic-target/);
 await resistance.fill('22k');await page.getByRole('button',{name:'Apply',exact:true}).click();await expect(panel).not.toBeVisible();await expect(page.locator('#schematic .diagnostic-target')).toHaveCount(0);
});
