export const escapeHTML=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
export function labelFor(key){const units={ambient_c:'Ambient temperature (°C)',temperature_c:'Temperature (°C)',illuminance_lux:'Illuminance (lux)',id:'Component / result',version:'Model version'};if(units[key])return units[key];const suffixes=[['_n_m_s_per_rad','N·m·s/rad'],['_v_s_per_rad','V·s/rad'],['_kg_m2','kg·m²'],['_k_per_w','K/W'],['_j_per_k','J/K'],['_v_per_s','V/s'],['_per_k','1/K'],['_ohm','Ω'],['_hz','Hz'],['_ah','Ah'],['_uw','µW'],['_v','V'],['_a','A'],['_f','F'],['_h','H'],['_s','s'],['_w','W']];let unit='';for(const[suffix,title]of suffixes)if(key.endsWith(suffix)){key=key.slice(0,-suffix.length);unit=' ('+title+')';break;}return key.replaceAll('_',' ').replace(/\b(esr|pwm|rpm|gbw|ctr|soc|rms)\b/gi,v=>v.toUpperCase()).replace(/^./,v=>v.toUpperCase())+unit;}

export function resolveSchema(schema,registry={},depth=0){
 if(depth>12)return {};let result={...(schema||{})};
 if(result.$ref){result={...resolveSchema(registry[result.$ref],registry,depth+1),...result};delete result.$ref;}
 if(result.anyOf){const choices=result.anyOf.map(s=>resolveSchema(s,registry,depth+1)).filter(s=>s.type!=='null');const text=choices.find(s=>s.type==='string'),numeric=choices.find(s=>s.type==='number');result={...result,...(numeric&&text?{type:'string'}:choices[0]||{})};delete result.anyOf;}
 return result;
}
export function schemaFields(schema,initial,registry={},path='',required=true){
 const s=resolveSchema(schema,registry),value=initial??s.default;
 if(s.type==='object'||s.properties){const fields=Object.entries(s.properties||{}).map(([key,child])=>schemaFields(child,value?.[key],registry,path?path+'.'+key:key,(s.required||[]).includes(key))).join('');return path?`<fieldset class="schema-group"><legend>${escapeHTML(labelFor(path.split('.').at(-1)))}</legend>${fields}</fieldset>`:fields;}
 const name=escapeHTML(path),title=escapeHTML(s.title||labelFor(path.split('.').at(-1))),hint=s.description?`<small class="field-note">${escapeHTML(s.description)}</small>`:'';
 if(s.type==='array'){
  const values=Array.isArray(value)?value:[];
  return `<fieldset class="schema-array" data-array="${name}"><legend>${title}</legend><div data-array-rows>${values.map((v,i)=>`<div class="schema-array-row">${schemaFields(s.items,v,registry,path+'.'+i,true)}<button type="button" data-remove-array>Remove</button></div>`).join('')}</div><button type="button" data-add-array="${name}">Add ${title.toLowerCase()}</button>${hint}</fieldset>`;
 }
 if(s.type==='boolean')return `<label class="checkbox-label"><input type="checkbox" name="${name}" ${value?'checked':''}>${title}${hint}</label>`;
 const current=value??s.const??'';
 if(s.const!=null||path.split('.').at(-1)==='version'&&s.enum?.length===1)return `<input type="hidden" name="${name}" value="${escapeHTML(s.const??s.enum[0])}">`;
 if(s.enum)return `<label>${title}<select name="${name}" ${required?'required':''}>${!required?'<option value="">Default</option>':''}${s.enum.map(v=>`<option value="${escapeHTML(v)}" ${v===current?'selected':''}>${escapeHTML(s['x-option-labels']?.[v]||v)}</option>`).join('')}</select>${hint}</label>`;
 return `<label>${title}<input name="${name}" ${s.type==='number'||s.type==='integer'?`type="number" step="${s.type==='integer'?'1':'any'}"`:'type="text"'} value="${escapeHTML(current)}" ${required?'required':''} ${s.const!=null?'readonly':''} ${s.minimum!=null?`min="${s.minimum}"`:''} ${s.maximum!=null?`max="${s.maximum}"`:''}>${hint}</label>`;
}
function fieldSchema(schema,path,registry){let s=resolveSchema(schema,registry);for(const key of path.split('.'))s=resolveSchema(s.type==='array'?s.items:s.properties?.[key],registry);return s;}
export function readSchemaForm(form,schema,registry={}){
 function read(s,path,required){s=resolveSchema(s,registry);if(s.type==='object'||s.properties){const out={};for(const [key,child]of Object.entries(s.properties||{})){const v=read(child,path?path+'.'+key:key,(s.required||[]).includes(key));if(v!==undefined)out[key]=v;}return out;}
  if(s.type==='array'){const group=[...form.querySelectorAll('[data-array]')].find(el=>el.dataset.array===path),rows=group?.querySelector('[data-array-rows]')?.children||[];return [...rows].map((_,i)=>read(s.items,path+'.'+i,true));}
  const input=form.elements.namedItem(path);if(!input)return s.const??s.default;
  if(s.type==='boolean')return input.checked;
  if(input.value===''){if(!required)return undefined;throw new Error(labelFor(path.split('.').at(-1))+' is required.');}
  if(s.type==='number'||s.type==='integer'){const n=Number(input.value);if(!Number.isFinite(n))throw new Error(labelFor(path)+' must be finite.');return n;}
  return input.value;
 }
 return read(schema,'',true);
}
export function bindSchemaArrays(container,schemaFor,registry){
 container.addEventListener('click',event=>{const add=event.target.closest('[data-add-array]'),remove=event.target.closest('[data-remove-array]');if(!add&&!remove)return;const form=event.target.closest('form'),schema=schemaFor(form);if(!schema)return;
  if(remove){const row=remove.closest('.schema-array-row'),rows=row.parentElement;row.remove();const prefix=rows.parentElement.dataset.array;[...rows.children].forEach((el,i)=>{const first=el.querySelector('[name],[data-array]'),old=first?.getAttribute('name')||first?.dataset.array;if(!old)return;const oldPrefix=prefix+'.'+old.slice(prefix.length+1).split('.')[0],nextPrefix=prefix+'.'+i;for(const field of el.querySelectorAll('[name],[data-array],[data-add-array]'))for(const attr of ['name','data-array','data-add-array']){const value=field.getAttribute(attr);if(value===oldPrefix||value?.startsWith(oldPrefix+'.'))field.setAttribute(attr,nextPrefix+value.slice(oldPrefix.length));}});return;}
  const path=add.dataset.addArray,rows=add.parentElement.querySelector('[data-array-rows]'),s=fieldSchema(schema,path,registry),row=document.createElement('div');row.className='schema-array-row';row.innerHTML=schemaFields(s.items,undefined,registry,path+'.'+rows.children.length,true)+'<button type="button" data-remove-array>Remove</button>';rows.append(row);
 });
}
export function modelForms(part,catalog){
 const device=part.device_parameters||{},power=part.power||{},defaults=catalog.model_defaults||{},recipes=[];
 const add=(title,name,initial)=>recipes.push({title,name,initial:{id:part.id,...initial}});
 const models={thyristor:['Thyristor model','thyristor.configure'],photo:['Photodiode model','photodiode.configure'],jfet:['JFET model','jfet.configure'],current_source:['Current source','current-source.configure'],electronic_load:['Electronic load','load.configure'],speaker:['Loudspeaker','speaker.configure'],pwm:['PWM controller','pwm.configure'],sound:['Sounder','sound.configure']};
 for(const[key,[title,name]]of Object.entries(models))if(device[key])add(title,name,{parameters:device[key]});
 for(const[key,name]of Object.entries({motor:'motor.configure',transformer:'transformer.configure'}))if(power[key])add(labelFor(key)+' parameters',name,{parameters:power[key]});
 if(power.battery)add('Battery parameters','battery.configure',power.battery);
 if(power.optocoupler)add('Optocoupler','optocoupler.configure',power.optocoupler);
 if(['resistor','diode','npn','pnp'].includes(part.kind))add('Temperature & self-heating','thermal.configure',{enabled:Boolean(part.thermal),parameters:part.thermal||defaults.thermal?.[part.kind]});
 if(['npn','pnp'].includes(part.kind))add('Transistor profile','transistor.profile.set',{enabled:Boolean(device.bipolar),parameters:device.bipolar||defaults.bipolar});
 if(part.kind==='opamp')add('Bandwidth & slew limits','opamp.dynamics.set',{enabled:Boolean(device.opamp_dynamics),parameters:device.opamp_dynamics||{gain_bandwidth_hz:1000,slew_v_per_s:1000}});
 if(part.kind==='potentiometer')add('Live wiper position','potentiometer.wiper.set',{position:part.wiper_position??.5});
 if(part.kind==='thermistor')add('Live temperature','sensor.temperature.set',{temperature_c:part.temperature_c??27});
 if(part.kind==='ldr')add('Live light level','sensor.light.set',{illuminance_lux:part.illuminance_lux??100});
 if(power.motor)add('Live motor load','motor.load.set',{load_n_m_s_per_rad:power.motor.load_n_m_s_per_rad});
 if(device.photo)add('Live incident light','photodiode.light.set',{incident_light_uw:device.photo.incident_light_uw});
 return recipes.filter(r=>catalog.commands.some(c=>c.id===r.name)).map(r=>{const c=catalog.commands.find(c=>c.id===r.name);return `<details class="section model-settings"><summary>${escapeHTML(r.title)}</summary><form class="settings-form schema-model-form" data-command="${r.name}" data-component-id="${escapeHTML(part.id)}">${schemaFields(c.input_schema,r.initial,catalog.schemas).replace('name="id"','name="id" readonly')}<p class="field-note">${escapeHTML(c.description)}</p><button class="primary" type="submit">Apply ${escapeHTML(r.title.toLowerCase())}</button><p class="settings-feedback" role="status"></p></form></details>`;}).join('');
}

// Use native inventories to make identifiers discoverable without weakening
// native command validation. The original schema stays immutable.
export function commandSchema(command,state,registry={}){
 const name=command.id,recorded=name.startsWith('recording.')||name.startsWith('spectrum.analyze');
 const probes=recorded?(state.recording?.scope?.available||state.spectrum?.available||[]):(state.scope?.available||[]);
 const captures=(state.operations||[]).filter(o=>o.status==='completed').map(o=>({id:o.id,label:o.id+(o.kept_in_project?' · kept':'')}));
 const parts=(state.components||[]).map(p=>({id:p.id,label:p.id+' · '+p.name}));
 const pick=(s,choices)=>choices.length?{...s,enum:choices.map(c=>c.id),'x-option-labels':Object.fromEntries(choices.map(c=>[c.id,c.label||c.id]))}:s;
 function visit(schema,key=''){
  let s={...resolveSchema(schema,registry)};
  if(s.properties)s.properties=Object.fromEntries(Object.entries(s.properties).map(([k,v])=>[k,visit(v,k)]));
  if(s.items)s.items=visit(s.items,key==='probe_ids'?'probe_id':key==='ids'&&name.includes('probes')?'probe_id':'');
  if(key==='probe_id')s=pick(s,probes);
  if(['source_id','component_id'].includes(key))s=pick(s,parts.filter(p=>key!=='source_id'||state.components.find(c=>c.id===p.id)?.kind.endsWith('source')));
  if(key==='reference_id'||key==='operation_id')s=pick(s,captures);
  if(key==='id'){
   if(name==='audio.render')s=pick(s,parts.filter(p=>['speaker','piezo','active-buzzer'].includes(state.components.find(c=>c.id===p.id)?.kind)));
   else if(name.startsWith('audio.'))s=pick(s,state.audio_preview?[{id:state.audio_preview.id}]:[]);
   else if(name.startsWith('sweep.'))s=pick(s,(state.dc_sweep?.operations||[]).map(o=>({id:o.id,label:o.id+' · '+o.status})));
   else if(name.startsWith('recording.')||name.startsWith('operation.')||name==='spectrum.analyze')s=pick(s,captures);
   else if(name==='measurement.pin.add')s=pick(s,probes);
   else if(name.startsWith('measurement.pin.'))s=pick(s,(state.pinned_readings||[]).map(p=>({id:p.definition.id,label:p.probe.label})));
  }
  if(key==='window'&&name.startsWith('spectrum.'))s={...s,enum:['hann','rectangular']};
  if(key==='mode'&&name==='tolerance.run')s={...s,enum:['dc','ac-gain']};
  if(key==='direction'&&name==='display.preferences.set')s={...s,enum:['conventional','electron']};
  if(key==='direction'&&name==='recording.step')s={...s,enum:['next','previous']};
  if(key==='cursor')s={...s,enum:['a','b']};
  if(key==='source'&&name==='spectrum.cursor.set')s={...s,enum:['live','recorded']};
  if(key==='paper'&&name==='export.page.set')s={...s,enum:['letter','a4']};
  if(key==='orientation'&&name==='export.page.set')s={...s,enum:['landscape','portrait']};
  if(key==='spacing')s={...s,enum:['linear','logarithmic']};
  return s;
 }
 return visit(command.input_schema);
}
