import {test,expect} from '@playwright/test';
const snapshot=async page=>(await (await page.request.get('/api/state')).json()).state;
const positions=s=>Object.fromEntries(s.components.filter(p=>['R1','C1'].includes(p.id)).map(p=>[p.id,p.position]));
test('multi-selection moves, copies connected parts and undoes each operation once',async({page})=>{
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/');await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 await page.locator('[data-object="R1"]').click();await page.locator('[data-object="C1"]').click({modifiers:['Shift']});
 await expect(page.locator('#selection-count')).toHaveText('2 components selected');
 const before=await snapshot(page);await page.locator('#schematic').focus();await page.keyboard.press('ArrowRight');
 await expect.poll(async()=>positions(await snapshot(page))).toEqual(Object.fromEntries(Object.entries(positions(before)).map(([id,p])=>[id,{x:p.x+10,y:p.y}])));
 await page.getByRole('button',{name:'Undo',exact:true}).click();await expect.poll(async()=>positions(await snapshot(page))).toEqual(positions(before));
 await page.getByRole('button',{name:'Copy selection',exact:true}).click();
 await page.getByRole('button',{name:'Paste selection',exact:true}).click();
 await expect.poll(async()=>(await snapshot(page)).components.length).toBe(before.components.length+2);
 const pasted=await snapshot(page);expect(pasted.wires.length).toBe(before.wires.length+1);
 await expect(page.locator('#selection-count')).toHaveText('2 components selected');
 await page.getByRole('button',{name:'Undo',exact:true}).click();await expect.poll(async()=>(await snapshot(page)).components.length).toBe(before.components.length);
 await page.getByRole('button',{name:'Redo',exact:true}).click();await expect.poll(async()=>(await snapshot(page)).components.length).toBe(before.components.length+2);
 expect(errors).toEqual([]);
});
test('box selection and group drag preserve relative positions and cancel safely',async({page},info)=>{
 await page.addInitScript(()=>{window.boxEvents=[];for(const kind of ['pointerdown','pointerup'])document.addEventListener(kind,e=>{const svg=document.getElementById('schematic');if(!svg||!e.target.closest('#schematic'))return;const box=document.querySelector('.selection-marquee');window.boxEvents.push({kind,x:e.clientX,y:e.clientY,target:e.target.closest('[data-object]')?.dataset.object||e.target.id,view:svg.getAttribute('viewBox'),canvas:svg.getBoundingClientRect().toJSON(),marquee:box?.getBoundingClientRect().toJSON(),bodies:[...document.querySelectorAll('.component-hit')].map(el=>({id:el.parentElement.dataset.object,...el.getBoundingClientRect().toJSON()}))});},true);});
 await page.goto('/');await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 const svg=page.locator('#schematic');await svg.focus();await page.keyboard.press('Control+a');await expect(page.locator('#selection-count')).toHaveText('4 components selected');
 await page.getByRole('button',{name:'Clear selection',exact:true}).click();
 const bounds=await svg.boundingBox();const parts=await page.locator('#components [data-object]').evaluateAll(nodes=>nodes.map(n=>{const b=n.getBoundingClientRect();return{x:b.x,y:b.y,r:b.right,b:b.bottom};}));
 const x0=Math.max(bounds.x+2,Math.min(...parts.map(p=>p.x))-5),y0=Math.max(bounds.y+2,Math.min(...parts.map(p=>p.y))-5);
 const x1=Math.min(bounds.x+bounds.width-2,Math.max(...parts.map(p=>p.r))+5),y1=Math.min(bounds.y+bounds.height-2,Math.max(...parts.map(p=>p.b))+5);
 await page.mouse.move(x0,y0);await page.mouse.down();await page.mouse.move(x1,y1,{steps:6});await page.mouse.up();await info.attach('marquee-events',{body:Buffer.from(JSON.stringify(await page.evaluate(()=>window.boxEvents))),contentType:'application/json'});await expect(page.locator('#selection-count')).toHaveText('4 components selected');
 const before=await snapshot(page),target=await page.locator('[data-object="R1"] .component-hit').boundingBox();
 await page.mouse.move(target.x+target.width/2,target.y+target.height/2);await page.mouse.down();await page.mouse.move(target.x+target.width/2+35,target.y+target.height/2+25,{steps:5});await page.mouse.up();
 await expect.poll(async()=>(await snapshot(page)).undo_name).toBe('Move selection');const moved=await snapshot(page);const delta={x:moved.components[0].position.x-before.components[0].position.x,y:moved.components[0].position.y-before.components[0].position.y};expect(Math.abs(delta.x)+Math.abs(delta.y)).toBeGreaterThan(0);
 for(const p of before.components){const q=moved.components.find(x=>x.id===p.id);expect(q.position).toEqual({x:p.position.x+delta.x,y:p.position.y+delta.y});}
 await page.getByRole('button',{name:'Undo',exact:true}).click();await expect.poll(async()=>positions(await snapshot(page))).toEqual(positions(before));
});
