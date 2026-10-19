async function applyCompositionCalibration173(appHandle, applyIndex) {
 const fs=await import('node:fs/promises');const out='/private/tmp/elab-display-causal-input-173';
 const action={input_call_invoked:false,input_call_completed:false,eligible_latency_observations:0,latency_gate_passed:null,controller_clock_scope:'Ordered wall-time observations only; no OS input-event timestamp',events:[]};
 await fs.writeFile(out+'/capture-action.json',JSON.stringify(action,null,2),{flag:'wx',mode:0o600});
 const persist=async()=>{await fs.writeFile(out+'/capture-action.next',JSON.stringify(action,null,2),{mode:0o600});await fs.rename(out+'/capture-action.next',out+'/capture-action.json');};
 const event=async(name,detail={})=>{action.events.push({sequence:action.events.length+1,name,wall_time_utc:new Date().toISOString(),...detail});await persist();};
 try{
  await fs.writeFile(out+'/capture-start.request','CUA controller ready.',{flag:'wx',mode:0o600});await event('capture_start_requested');
  const deadline=Date.now()+5000;let baseline;
  while(!baseline){if(Date.now()>deadline)throw new Error('First-image deadline');try{for(const line of (await fs.readFile(out+'/capture/frames.jsonl','utf8')).split('\n').filter(Boolean)){try{const row=JSON.parse(line);if(row.event==='frame'&&row.png&&row.display_mach_ticks){baseline=row;break;}}catch{}}}catch(error){if(error.code!=='ENOENT')throw error;}if(!baseline)await new Promise(resolve=>setTimeout(resolve,5));}
  await event('baseline_image_received',{frame_index:baseline.index,png:baseline.png});
  const id='composition_led01';await fs.writeFile(out+'/capture/marker.next',id,{flag:'wx',mode:0o600});await fs.rename(out+'/capture/marker.next',out+'/capture/marker.request');await event('marker_requested',{id});
  let ack;while(!ack){if(Date.now()>deadline)throw new Error('Complete acknowledgment deadline');try{const value=JSON.parse(await fs.readFile(out+'/capture/marker-'+id+'.json','utf8'));if(value.id!==id||typeof value.mach_ticks!=='string'||BigInt(value.mach_ticks)<=0n||BigInt(value.mach_ticks)>18446744073709551615n||value.frame_count_before<1)throw new Error('Invalid complete acknowledgment');ack=value;}catch(error){if(error.code!=='ENOENT'&&!(error instanceof SyntaxError))throw error;}if(!ack)await new Promise(resolve=>setTimeout(resolve,2));}
  action.marker_acknowledgment=ack;await event('complete_acknowledgment_validated',{id});action.input_call_invoked=true;await event('input_call_will_invoke',{api:'CUA App.click',apply_index:applyIndex});
  await appHandle.click(applyIndex);action.input_call_completed=true;await event('input_call_completed');
  const after=await appHandle.getAXState({disableDiffing:true,emit:false});await fs.writeFile(out+'/after-apply.ax.txt',after,{flag:'wx',mode:0o600});action.after_ax_has_expected_readout=after.includes('1.477 mA');await event('after_input_ax_observed');
 }catch(error){action.error=String(error);}finally{await persist();}return action;
}