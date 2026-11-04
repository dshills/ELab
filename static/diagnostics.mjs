export function circuitDiagnostics(api){
 const panel=document.getElementById('circuit-guide');let request=null,key='',items=[];
 function current(){return api.state();}
 function mark(){
  const objects=new Set(items.flatMap(v=>v.object_ids||[])),wires=new Set(items.flatMap(v=>v.wire_ids||[])),pins=new Set(items.flatMap(v=>v.terminal_refs||[]));
  for(const el of document.querySelectorAll('#schematic [data-object],#schematic .pin-hit')){
   const id=el.dataset.object,ref=el.closest('[data-pin]')?.dataset.pin,related=id?objects.has(id)||wires.has(id):pins.has(ref);
   el.classList.toggle('diagnostic-target',related);
   if(related){el.setAttribute('aria-describedby','diagnostic-summary');el.dataset.diagnostic='true';}else if(el.dataset.diagnostic){el.removeAttribute('aria-describedby');delete el.dataset.diagnostic;}
  }
 }
 function button(text,kind,target){const b=document.createElement('button');b.textContent=text;b.dataset.diagnosticAction=kind;if(target)b.dataset.target=target;return b;}
 function update(){
  const state=current();if(!state)return;
  if(request&&(request.revision!==state.command_revision||request.generation!==state.document.generation))request=null;
  items=[...(state.diagnostic_presentation?.items||[]),...(request?[request.help]:[])];
  const next=JSON.stringify([items,state.undo_entry?.command_revision]);
  panel.hidden=!items.length;panel.classList.toggle('hidden',!items.length);
  if(next!==key){key=next;const open=new Set([...panel.querySelectorAll('details[open]')].map(d=>d.dataset.code));panel.replaceChildren();
   const heading=document.createElement('h2');heading.id='diagnostics-heading';heading.textContent='Circuit checks';panel.append(heading);
   const summary=document.createElement('p');summary.id='diagnostic-summary';summary.setAttribute('role','status');summary.textContent=`${items.length} circuit ${items.length===1?'check':'checks'} to review. Related parts and terminals have dashed outlines.`;panel.append(summary);
   for(const help of items){const article=document.createElement('article'),title=document.createElement('h3'),action=document.createElement('p'),controls=document.createElement('div');
    title.textContent=help.summary;action.textContent=help.action;controls.className='selection-actions';article.append(title,action,controls);
    const known=new Set([...state.components.map(p=>p.id),...state.wires.map(w=>w.id)]);
    for(const id of [...(help.object_ids||[]),...(help.wire_ids||[])])if(known.has(id))controls.append(button(`Show ${id}`,'show',id));
    for(const ref of help.terminal_refs||[])if(api.scene().parts.some(p=>p.pins.some(pin=>pin.ref===ref)))controls.append(button(`Connect ${ref}`,'connect',ref));
    if(['open-terminal','unconnected-primitive','missing-ground','reference-disconnected'].includes(help.code)&&state.undo_entry?.name?.startsWith('Delete'))controls.append(button('Undo deletion','undo'));
    if(['source-sampling','lc-sampling'].includes(help.code))controls.append(button('Fit sample interval','sampling'));
    if(['integration-time-limit','integration-attempt-limit','mixed-integration-failed','solve-failed','mixed-initialize-failed'].includes(help.code))controls.append(button('Review simulation settings','settings'),button('Restart simulation','restart'));
    const details=document.createElement('details'),label=document.createElement('summary'),technical=document.createElement('p');details.dataset.code=help.source+':'+help.code;details.open=open.has(details.dataset.code);label.textContent='Technical details';technical.textContent=`${help.code} (${help.source}): ${help.details}`;details.append(label,technical);article.append(details);panel.append(article);
   }
  }mark();
 }
 panel.onclick=e=>{const b=e.target.closest('[data-diagnostic-action]');if(!b)return;const kind=b.dataset.diagnosticAction,target=b.dataset.target;
  const run=async()=>{if(kind==='show')await api.show(target);else if(kind==='connect')await api.connect(target);else if(kind==='undo')await api.undo();else if(kind==='sampling')await api.command('simulation.sampling.fit');else if(kind==='restart')await api.command('simulation.restart');else if(kind==='settings')api.settings();};
  run().catch(()=>{});
 };
 return {update,mark,failure(help,revision,generation){if(help&&current()?.command_revision===revision&&current()?.document.generation===generation){request={help,revision,generation};key='';update();}},reset(){request=null;key='';}};
}
