import {escapeHTML as esc} from './forms.mjs';
import {resultTable} from './results.mjs';
import {plotInstrument} from './instrument-plot.mjs';

export const sweepMethods = [['dc','DC / component'],['ac','AC gain & phase'],['noise','White noise'],['loop','Loop response']];
export function sweepMethod(parameter) {
 return ({frequency:'ac',noise:'noise','loop-frequency':'loop'})[parameter] || 'dc';
}
const key = target => JSON.stringify([target.component_id,target.parameter]);
export function sweepDefaults(target) {
 const method=sweepMethod(target.parameter), spectral=method!=='dc';
 const endpoints={voltage:['0 V','5 V'],resistance:['100 Ω','10 kΩ'],wiper:['0','1'],temperature:['-40','125'],light:['0','1000'],frequency:['1 Hz','100 kHz'],noise:['1 Hz','100 kHz'],'loop-frequency':['.1 Hz','100 kHz']}[target.parameter] || [String(target.minimum),String(target.maximum)];
 return {component_id:target.component_id,parameter:target.parameter,start:endpoints[0],stop:endpoints[1],points:spectral?201:41,spacing:spectral?'logarithmic':'linear'};
}
export function sweepRequest(values,target,probeIDs=[]) {
 if(!target)throw new Error('Choose an available sweep target.');
 const settings={component_id:target.component_id,parameter:target.parameter,start:values.start,stop:values.stop,spacing:values.spacing,points:Number(values.points)};
 // Omission asks the native engine to choose measurements; [] is invalid there.
 if(target.parameter==='loop-frequency')settings.probe_ids=['loop:'+target.component_id];
 else if(probeIDs.length)settings.probe_ids=probeIDs;
 return {settings};
}
export function nearestSweepPoint(points,x,log=false) {
 if(!Number.isFinite(x)||log&&x<=0)return null;
 let index=null,distance=Infinity;
 points.forEach((p,i)=>{if(!Number.isFinite(p.input)||log&&p.input<=0)return;const d=Math.abs(log?Math.log(p.input)-Math.log(x):p.input-x);if(d<distance){distance=d;index=i;}});
 return index;
}

export function sweepWorkbench(api) {
 const root=document.getElementById('analysis-body');
 let owner='',method='dc',targetKey='',pending=false,domain=null,formGuard=null,draft=false,inspected=null;
 const current=()=>api.state().dc_sweep;
 const identity=()=>JSON.stringify([api.instance(),api.state().document.generation,current()?.targets]);
 const active=()=>root.dataset.instrument==='sweep';
 function chosen(){return (current()?.targets||[]).find(t=>key(t)===targetKey);}
 function build(useSaved=true) {
  const s=current(),targets=s?.targets||[],saved=s?.settings;
  if(useSaved){method=sweepMethod(saved?.parameter);targetKey=saved?key(saved):'';}
  if(!targets.some(t=>sweepMethod(t.parameter)===method))method=sweepMethod(targets[0]?.parameter);
  const filtered=targets.filter(t=>sweepMethod(t.parameter)===method);
  if(!filtered.some(t=>key(t)===targetKey))targetKey=filtered[0]?key(filtered[0]):'';
  const target=chosen(),cfg=target?(useSaved&&saved&&key(saved)===targetKey?saved:sweepDefaults(target)):{};
  const probes=method==='loop'?[]:(api.state().scope.available||s?.available_probes||[]).filter(p=>method==='dc'||p.unit==='V');
  const picked=cfg.probe_ids||[];
  root.innerHTML=`<form id="sweep-settings" class="analysis-toolbar"><label>Method<select id="sweep-method">${sweepMethods.map(([id,label])=>`<option value="${id}" ${id===method?'selected':''} ${targets.some(t=>sweepMethod(t.parameter)===id)?'':'disabled'}>${label}</option>`).join('')}</select></label><label>Target<select id="sweep-target">${filtered.map(t=>`<option value="${esc(key(t))}" ${key(t)===targetKey?'selected':''}>${esc(t.label)} · ${esc(t.unit)}</option>`).join('')}</select></label><label>Start<input name="start" value="${esc(cfg.start)}" required></label><label>Stop<input name="stop" value="${esc(cfg.stop)}" required></label><label>Points<input name="points" type="number" min="2" max="201" value="${cfg.points||41}" required></label><label>Spacing<select name="spacing"><option value="linear" ${cfg.spacing!=='logarithmic'?'selected':''}>Linear</option><option value="logarithmic" ${cfg.spacing==='logarithmic'?'selected':''}>Logarithmic</option></select></label><button class="primary">Run sweep</button><button type="button" data-sweep="reset">Use configured settings</button><details class="instrument-settings"><summary>Measurements & range</summary><p class="field-note">${target?`${esc(target.minimum)} – ${esc(target.maximum)} ${esc(target.unit)}. `:''}${method==='dc'?'Select up to six measurements.':method==='loop'?'The native engine measures this dynamic amplifier’s differential return ratio.':'Select up to three voltage outputs.'} Leave all unchecked for the native default. ${['wiper','temperature','light'].includes(target?.parameter)?'Enter numeric endpoints in the displayed unit.':'Endpoints accept engineering units.'}</p><div class="scope-probe-list">${probes.map(p=>`<label class="checkbox-label"><input type="checkbox" name="probe" value="${esc(p.id)}" ${picked.includes(p.id)?'checked':''}>${esc(p.label)} · ${esc(p.unit)}</label>`).join('')}</div></details><p id="sweep-feedback" class="settings-feedback error" role="status"></p></form><div class="analysis-source"><label>Retained sweep<select id="sweep-result"></select></label><button type="button" data-sweep="keep"></button><button type="button" data-sweep="cancel">Cancel active sweep</button></div><p id="sweep-progress" class="field-note" role="status"></p><p id="sweep-summary" class="analysis-summary" role="status"></p><label>Result trace<select id="sweep-trace"></select></label><canvas id="instrument-chart" class="instrument-chart" tabindex="0" aria-label="Sweep results; click for nearest stored point, or use arrow keys, Home and End"></canvas><form id="sweep-point" class="analysis-toolbar"><label>Exact point index<input name="index" type="number" min="0" value="0" required></label><button>Set point cursor</button><button type="button" data-sweep="clear">Clear cursor</button></form><div id="sweep-readings"></div><div class="analysis-export"><button data-sweep="json">Download sweep JSON</button><button data-sweep="csv">Download sweep CSV</button></div><details><summary>Native measurements & assumptions</summary><div id="sweep-details"></div><button data-sweep="inspect">Show captured source & assumptions</button><div id="sweep-source"></div></details>`;
  formGuard=api.guards();draft=false;domain=null;
 }
 function update() {
  if(!active())return;
  if(owner!==identity()){owner=identity();build();}
  const s=current(),result=s?.result,operations=s?.operations||[],selected=operations.find(o=>o.id===s.selected_operation),work=operations.find(o=>['queued','running','canceling'].includes(o.status));
  document.getElementById('scope-origin').textContent='SWEEP';
  for(const el of root.querySelectorAll('button,input,select'))el.disabled=pending;
  root.querySelector('#sweep-settings button').disabled=pending||!!work||!chosen();
  const select=root.querySelector('#sweep-result'),options=operations.map(o=>`<option value="${esc(o.id)}">${esc(o.id)} · ${esc(o.kind)} · ${esc(o.status)}${o.kept_in_project?' · kept':''}</option>`).join('');
  if(select.innerHTML!==options)select.innerHTML=options;select.value=s?.selected_operation||'';select.disabled=pending||!operations.length;
  const cancel=root.querySelector('[data-sweep="cancel"]');cancel.disabled=pending||!work||work.status==='canceling';cancel.dataset.operation=work?.id||'';
  const keep=root.querySelector('[data-sweep="keep"]');keep.textContent=selected?.kept_in_project?'Kept in project':'Keep in project';keep.setAttribute('aria-pressed',String(!!selected?.kept_in_project));keep.disabled=pending||!result;
  root.querySelector('#sweep-progress').textContent=work?`${work.id} · ${work.status} · ${work.completed_points} / ${work.total_points} points`:operations.at(-1)?.error?.message||'';
  root.querySelector('#sweep-summary').textContent=result?`${s.selected_operation} · ${result.target.label} · ${result.points.length} exact points · ${api.fmt(result.start_si,result.target.unit)} to ${api.fmt(result.stop_si,result.target.unit)} · ${result.spacing||'linear'}${selected?.source_unchanged?' · live source unchanged':' · captured source'}`:selected?.error?.message||'Run a sweep to capture native measurements while preserving your live circuit.';
  const trace=root.querySelector('#sweep-trace'),traces=(result?.probes||[]).map(p=>`<option value="${esc(p.id)}">${esc(p.label)} · ${esc(p.unit)}</option>`).join('');
  if(trace.innerHTML!==traces)trace.innerHTML=traces;trace.value=(result?.probes.find(p=>p.id===s?.trace_id)||result?.probes[0])?.id||'';trace.disabled=pending||!result;
  const index=result?.probes.findIndex(p=>p.id===trace.value)??-1,probe=result?.probes[index],point=result?.points[s.cursor_index];
  domain=plotInstrument(api,root.querySelector('#instrument-chart'),(result?.points||[]).map(p=>({x:p.input,y:p.values[index]})),{log:result?.spacing==='logarithmic',xUnit:result?.target.unit||'',yUnit:probe?.unit||'',markers:point?[{x:point.input,label:String(s.cursor_index)}]:[]});
  const input=root.querySelector('#sweep-point input');input.max=Math.max(0,(result?.points.length||0)-1);if(document.activeElement!==input)input.value=s?.cursor_index??0;
  for(const el of root.querySelectorAll('#sweep-point button,#sweep-point input,[data-sweep="json"],[data-sweep="csv"],[data-sweep="inspect"]'))el.disabled=pending||!result;
  root.querySelector('#sweep-readings').textContent=point?`Point ${s.cursor_index} · ${api.fmt(point.input,result.target.unit,7)} · ${result.probes.map((p,i)=>`${p.label} ${api.fmt(point.values[i],p.unit,7)}`).join(' · ')}`:'Click the plot or enter an index to select an exact captured point. No interpolation.';
  const details=result?resultTable({statistics:result.statistics,noise:result.noise,loop_margins:result.loop_margins,ac_bands:result.ac_bands}):'',output=root.querySelector('#sweep-details');if(output.innerHTML!==details)output.innerHTML=details;
  const source=root.querySelector('#sweep-source'),metadata=inspected&&inspected.owner===identity()&&inspected.id===s?.selected_operation?resultTable(inspected.metadata):'';if(source.innerHTML!==metadata)source.innerHTML=metadata;
 }
 async function run(name,params,guard=api.guards(),expected=owner) {
  if(pending)throw new Error('Wait for the current sweep action to finish.');
  if(expected!==identity())throw new Error('Sweep source changed. Review the current controls.');
  pending=true;update();
  try{return await api.command(name,params,{guard});}finally{pending=false;update();}
 }
 const fail=error=>{const output=root.querySelector('#sweep-feedback');if(active()&&output)output.textContent=error.message;else api.toast(error.message,true);};
 root.addEventListener('input',e=>{if(!active())return;if(e.target.closest('#sweep-settings')&&!draft){draft=true;formGuard=api.guards();}const point=e.target.closest('#sweep-point');if(point&&!point._context)point._context={id:current()?.selected_operation,guard:api.guards(),owner};});
 root.addEventListener('change',async e=>{
  if(!active()||pending)return;
  try {
   if(e.target.id==='sweep-method'){method=e.target.value;targetKey='';build(false);update();}
   else if(e.target.id==='sweep-target'){targetKey=e.target.value;build(false);update();}
   else if(e.target.id==='sweep-result')await run('sweep.select',{id:e.target.value});
   else if(e.target.id==='sweep-trace')await run('sweep.trace.select',{id:e.target.value});
   else if(e.target.closest('#sweep-settings')&&!draft){draft=true;formGuard=api.guards();}
  }catch(error){fail(error);}
 });
 root.addEventListener('submit',async e=>{
  if(!active())return;e.preventDefault();const form=e.target;if(!form.reportValidity())return;
  try {
   const values=Object.fromEntries(new FormData(form));
   if(form.id==='sweep-settings'){
    const probes=[...form.querySelectorAll('[name="probe"]:checked')].map(p=>p.value);
    await run('sweep.start',sweepRequest(values,chosen(),probes),draft?formGuard:api.guards());draft=false;formGuard=api.guards();root.querySelector('#sweep-feedback').textContent='';
   }else if(form.id==='sweep-point'){const context=form._context||{id:current().selected_operation,guard:api.guards(),owner};await run('sweep.cursor.set',{operation_id:context.id,index:Number(values.index)},context.guard,context.owner);form._context=null;}
  }catch(error){fail(error);}
 });
 root.addEventListener('click',async e=>{
  if(!active()||pending)return;const button=e.target.closest('[data-sweep]');
  try {
   if(button&&!button.disabled){const action=button.dataset.sweep,s=current();
    if(action==='reset'){build();update();}
    else if(action==='cancel')await run('sweep.cancel',{id:button.dataset.operation});
    else if(action==='keep')await run('sweep.keep.set',{id:s.selected_operation,keep:button.getAttribute('aria-pressed')!=='true'});
    else if(action==='clear')await run('sweep.cursor.set',{operation_id:s.selected_operation,index:null});
    else if(action==='json'||action==='csv')await api.download('sweep-'+action,{id:s.selected_operation,guard:api.guards()});
    else if(action==='inspect'){const id=s.selected_operation,capturedOwner=identity(),inspection=await run('sweep.inspect',{id});inspected={id,owner:capturedOwner,metadata:inspection.metadata};update();}
   }else if(e.target.id==='instrument-chart'&&domain&&current()?.result){const s=current(),index=nearestSweepPoint(s.result.points,domain.at(e.clientX),s.result.spacing==='logarithmic');if(index!==null)await run('sweep.cursor.set',{operation_id:s.selected_operation,index});}
  }catch(error){fail(error);}
 });
 root.addEventListener('keydown',async e=>{
  if(!active()||pending||e.target.id!=='instrument-chart'||!current()?.result)return;
  const s=current(),last=s.result.points.length-1,index=({'ArrowLeft':Math.max(0,(s.cursor_index??1)-1),'ArrowRight':Math.min(last,(s.cursor_index??-1)+1),Home:0,End:last})[e.key];
  if(index==null)return;e.preventDefault();try{await run('sweep.cursor.set',{operation_id:s.selected_operation,index});}catch(error){fail(error);}
 });
 return {show(){root.dataset.instrument='sweep';owner='';update();},update};
}
