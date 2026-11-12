import {test,expect} from '@playwright/test';
import {readFileSync,writeFileSync,mkdirSync} from 'node:fs';
import {resolve,dirname} from 'node:path';
import {execFileSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
const root=fileURLToPath(new URL('../../',import.meta.url));

const quantile=(values,q)=>values.length?[...values].sort((a,b)=>a-b)[Math.min(values.length-1,Math.ceil(values.length*q)-1)]:null;
const snapshot=async page=>(await (await page.request.get('/api/state')).json()).state;
function paddedFixture(path,count){const doc=JSON.parse(readFileSync(resolve(root,path),'utf8')),g=doc.graph,ground=g.components.find(p=>p.kind==='ground');let n=0;while(g.components.length<count){const id='JX'+(++n);g.components.push({id,kind:'junction',name:'Junction',model:'junction.v1',terminals:['node'],position:{x:-1800+((n-1)%12)*300,y:1200+Math.floor((n-1)/12)*240},value:0,rotation:0});g.wires.push({id:'WX'+n,from:id+'.node',to:ground.id+'.reference',connected:true});}if(path.includes('hundred-node'))doc.scope_probes={ids:['v:V1','i:V1','v:P1.wiper','v:P25.wiper','v:P50.wiper','v:P1.1']};doc.paused=true;return Buffer.from(JSON.stringify(doc));}

test('capacity editing, group dragging and live scope remain measurable under CPU load',async({page,browserName},info)=>{
 test.skip(browserName!=='chromium','CDP CPU throttling measurement is Chromium-specific');test.setTimeout(180_000);
 const errors=[];page.on('pageerror',e=>errors.push(e.message));await page.goto('/');await expect(page.locator('#connection-status')).toHaveText('Local engine connected');
 const cdp=await page.context().newCDPSession(page);await cdp.send('Emulation.setCPUThrottlingRate',{rate:4});
 await page.evaluate(()=>{
  const blob=new Blob([`setInterval(()=>{const end=performance.now()+10;while(performance.now()<end){}},20)`],{type:'text/javascript'});window.loadWorker=new Worker(URL.createObjectURL(blob));
  window.perfRun={keys:[],pointers:[],frames:[],replacements:0,pending:null,lastTime:''};
  const finish=()=>{const p=window.perfRun.pending;if(!p?.start||p.finishing)return;const shape=[...document.querySelectorAll('#components [data-object]')].find(el=>el.dataset.object===p.id)?.querySelector('.part-shape');const values=/translate\(([-\d.]+) ([-\d.]+)\)/.exec(shape?.getAttribute('transform')||'');const parent=/translate\(([-\d.]+) ([-\d.]+)\)/.exec(shape?.parentElement.getAttribute('transform')||'');if(values&&+values[1]+(parent?+parent[1]:0)===p.x&&+values[2]+(parent?+parent[2]:0)===p.y){p.finishing=true;requestAnimationFrame(()=>requestAnimationFrame(()=>{window.perfRun.keys.push(performance.now()-p.start);window.perfRun.pending=null;}));}};
  const observer=new MutationObserver(records=>{for(const r of records)if(r.target.id==='components'&&r.type==='childList')window.perfRun.replacements++;const text=document.getElementById('time-label')?.textContent;if(text!==window.perfRun.lastTime){window.perfRun.lastTime=text;window.perfRun.frames.push(performance.now());}finish();});observer.observe(document.getElementById('app'),{subtree:true,childList:true,attributes:true,characterData:true});
  document.addEventListener('keydown',e=>{if(e.key.startsWith('Arrow')&&window.perfRun.pending)window.perfRun.pending.start=performance.now();},true);
  document.addEventListener('pointermove',e=>{if(!e.buttons)return;const start=performance.now();requestAnimationFrame(()=>window.perfRun.pointers.push(performance.now()-start));},true);
 });
 const report={passed:false,source_commit:execFileSync('git',['rev-parse','HEAD'],{encoding:'utf8',cwd:root}).trim(),scope:'Local headless Chromium, 4x main-thread CPU slowdown plus a 50%-duty worker. Synthetic event to accepted SVG update plus two frames; not field INP or physical event-to-pixel evidence.',cases:[]};
 const nodes='docs/validation/desktop-capacity-artifacts/hundred-node.elab', switching='docs/validation/live-clock-grid/capacity-pulse.elab';
 try{for(const [fixture,count] of [[nodes,100],[switching,100],[switching,200]]){
  const native=paddedFixture(fixture,count);let s=await snapshot(page);const query=new URLSearchParams({revision:s.command_revision,generation:s.document.generation,discard:'true',name:'Responsiveness fixture.elab'});
  const imported=await page.request.post('/api/upload?'+query,{headers:{'Content-Type':'application/octet-stream'},data:native});expect(imported.ok()).toBeTruthy();expect((await imported.json()).error).toBeUndefined();
  await expect(page.locator('#graph-counts')).toContainText(`${count} components`);s=await snapshot(page);expect(s.components.length).toBe(count);expect(s.simulated_component_count).toBeLessThanOrEqual(s.limits.simulated_components);
  const id=s.components[0].id;await expect(page.locator(`#components [data-object="${id}"]`)).toBeVisible();await page.locator('#schematic').focus();await page.keyboard.press('Control+a');await expect(page.locator('#selection-count')).toHaveText(`${count} components selected`);
  await page.evaluate(()=>{window.perfRun.keys=[];window.perfRun.pointers=[];window.perfRun.replacements=0;});
  for(let n=0;n<8;n++){s=await snapshot(page);const p=s.components.find(p=>p.id===id).position,dx=n%2?-10:10;await page.evaluate(p=>{window.perfRun.pending=p;},{id,x:p.x+dx,y:p.y});await page.locator('#schematic').press(dx>0?'ArrowRight':'ArrowLeft');await expect.poll(()=>page.evaluate(()=>window.perfRun.keys.length)).toBe(n+1);}
  const target=await page.locator(`[data-object="${id}"] .component-hit`).boundingBox();expect(target).toBeTruthy();
  await page.mouse.move(target.x+target.width/2,target.y+target.height/2);await page.mouse.down();await page.mouse.move(target.x+target.width/2+40,target.y+target.height/2+20,{steps:20});await page.mouse.up();
  await expect.poll(async()=>(await snapshot(page)).command_revision).toBeGreaterThan(s.command_revision);
  if(await page.getByRole('button',{name:'Resume',exact:true}).isVisible())await page.getByRole('button',{name:'Resume',exact:true}).click();await expect(page.getByRole('button',{name:'Pause',exact:true})).toBeVisible();
  await page.evaluate(()=>{window.perfRun.frames=[];window.perfRun.replacements=0;performance.clearResourceTimings();});
  await page.waitForTimeout(2000);s=await snapshot(page);
  const measured=await page.evaluate(()=>({...window.perfRun,pending:undefined,resources:performance.getEntriesByType('resource').filter(e=>e.name.includes('/api/state')&&e.decodedBodySize>0).map(e=>({bytes:e.decodedBodySize,duration_ms:e.duration}))}));
  let paused=await snapshot(page);const pause=await page.request.post('/api/command',{data:{version:1,id:'perf-pause-'+report.cases.length,key:'perf-pause-'+report.cases.length,command:'simulation.pause',params:{},expected_command_revision:paused.command_revision,expected_document_generation:paused.document.generation}});expect((await pause.json()).error).toBeUndefined();
  await page.request.post('/api/command',{data:{version:1,id:'perf-settle',command:'simulation.wait',params:{}}});
  const full=await page.request.get('/api/state'),fullData=await full.json(),conditional=await page.request.get('/api/state?'+new URLSearchParams({scene_token:fullData.scene_token,instance:fullData.instance_id})),conditionalData=await conditional.json();
  const transfer={full_bytes:(await full.body()).length,conditional_bytes:(await conditional.body()).length,same_revision:fullData.state.revision===conditionalData.state.revision,static_geometry_omitted:conditionalData.scene===undefined};
  expect(transfer.same_revision).toBeTruthy();expect(transfer.static_geometry_omitted).toBeTruthy();expect(transfer.conditional_bytes).toBeLessThan(transfer.full_bytes);
  const gaps=measured.frames.slice(1).map((t,i)=>t-measured.frames[i]);
  report.cases.push({fixture,drawing_objects:s.components.length,simulated_parts:s.simulated_component_count,electrical_nodes:s.nets.length,traces:s.scope.traces.length,status:s.status,scene_transfer:transfer,accepted_edit_ms:{samples:measured.keys.length,p50:quantile(measured.keys,.5),p95:quantile(measured.keys,.95)},drag_frame_ms:{samples:measured.pointers.length,p50:quantile(measured.pointers,.5),p95:quantile(measured.pointers,.95)},live:{scope_frames:measured.frames.length,interval_p95_ms:quantile(gaps,.95),component_root_replacements:measured.replacements,snapshot_bytes_p50:quantile(measured.resources.map(r=>r.bytes),.5),snapshot_duration_p95_ms:quantile(measured.resources.map(r=>r.duration_ms),.95)}});
  expect(measured.keys).toHaveLength(8);expect(measured.pointers.length).toBeGreaterThan(0);expect(measured.frames.length).toBeGreaterThan(1);expect(s.status).not.toBe('error');
 }expect(errors).toEqual([]);report.passed=true;}finally{await page.evaluate(()=>window.loadWorker?.terminate());await cdp.send('Emulation.setCPUThrottlingRate',{rate:1});const path=process.env.ELAB_PERF_REPORT||info.outputPath('responsiveness.json');mkdirSync(dirname(path),{recursive:true});writeFileSync(path,JSON.stringify(report,null,2)+'\n');await info.attach('responsiveness',{body:Buffer.from(JSON.stringify(report,null,2)),contentType:'application/json'});}
 expect(errors).toEqual([]);
});
