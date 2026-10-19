async function runDeclaredLedTrials(appHandle, first, last, expectedPID) {
 const fs=await import('node:fs/promises'); const root='/private/tmp/elab-latency-cohort-led-173'; const results=[];
 async function exists(p){try{await fs.access(p);return true;}catch(e){if(e.code==='ENOENT')return false;throw e;}}
 async function read(p){try{return JSON.parse(await fs.readFile(p,'utf8'));}catch(e){if(e.code==='ENOENT'||e instanceof SyntaxError)return null;throw e;}}
 async function wait(p,seconds=25,alternative){const end=Date.now()+seconds*1000;while(Date.now()<end){const value=await read(p);if(value)return value;if(alternative){const failed=await read(alternative);if(failed)throw new Error('Worker completed before ready: '+JSON.stringify(failed));}await new Promise(r=>setTimeout(r,3));}throw new Error('Timed out: '+p);}
 async function write(p,value){await fs.writeFile(p,JSON.stringify(value,null,2),{flag:'wx',mode:0o600});}
 for(let n=first;n<=last;n++){
  const id='led-'+String(n).padStart(2,'0');const out=root+'/trials/'+id;
  const action={id,input_call_invoked:false,input_call_completed:false,eligible_latency_observations:0,latency_gate_passed:null,events:[],controller_clock_scope:'Ordered wall time only; no OS input-event timestamp'};
  let fileCreated=false;
  const persist=async()=>{await fs.writeFile(out+'/capture-action.next',JSON.stringify(action,null,2),{mode:0o600});await fs.rename(out+'/capture-action.next',out+'/capture-action.json');};
  const event=async(name,detail={})=>{action.events.push({sequence:action.events.length+1,name,wall_time_utc:new Date().toISOString(),...detail});await persist();};
  try{
   const ready=await wait(out+'/ready.json',25,out+'/done.json');if(ready.owned_pid!==expectedPID||ready.part!=='Limit'||ready.target!=='2k')throw new Error('Wrong owned trial identity');
   await write(out+'/capture-action.json',action);fileCreated=true;
   const before=await appHandle.getAXState({emit:false,disableDiffing:true});await fs.writeFile(out+'/before-draft.ax.txt',before,{flag:'wx',mode:0o600});
   if(!before.includes('Limit · Resistor')||!before.includes('2.917 mA'))throw new Error('Expected fresh Limit inspector/baseline absent');
   let field=before.match(/^\s*(\d+) text field \(settable\) Component value,/m);if(!field)throw new Error('Value field absent');
   await appHandle.click(Number(field[1]));await appHandle.pressKey('super+a');await appHandle.typeText('2k');
   const draft=await appHandle.getAXState({emit:false,disableDiffing:true});await fs.writeFile(out+'/before-apply.ax.txt',draft,{flag:'wx',mode:0o600});
   const valueLine=draft.split('\n').find(l=>l.includes('text field (settable) Component value,'));if(!valueLine||!valueLine.includes('Value: 2k'))throw new Error('Draft not observed');
   const apply=draft.match(/^\s*(\d+) button Apply value,/m);if(!apply)throw new Error('Apply button absent');
   await write(out+'/draft-ready.json',{id,typed:'2k',apply_index:Number(apply[1])});await wait(out+'/armed.json',15,out+'/done.json');
   await fs.writeFile(out+'/capture-start.request','CUA controller ready.',{flag:'wx',mode:0o600});await event('capture_start_requested');
   const until=Date.now()+5000;let baseline;
   while(!baseline){if(Date.now()>until)throw new Error('Baseline frame deadline');try{const lines=(await fs.readFile(out+'/capture/frames.jsonl','utf8')).split('\n');for(const line of lines){try{const r=JSON.parse(line);if(r.event==='frame'&&r.png&&r.display_mach_ticks){baseline=r;break;}}catch{}}}catch(e){if(e.code!=='ENOENT')throw e;}if(!baseline)await new Promise(r=>setTimeout(r,3));}
   await event('baseline_image_received',{frame_index:baseline.index,png:baseline.png});const marker=id.replace('-','_');
   await fs.writeFile(out+'/capture/marker.next',marker,{flag:'wx',mode:0o600});await fs.rename(out+'/capture/marker.next',out+'/capture/marker.request');await event('marker_requested',{id:marker});
   const ack=await wait(out+'/capture/marker-'+marker+'.json',2);if(ack.id!==marker||typeof ack.mach_ticks!=='string'||BigInt(ack.mach_ticks)<=0n||ack.frame_count_before<1)throw new Error('Invalid acknowledgment');
   action.marker_acknowledgment=ack;await event('complete_acknowledgment_validated',{id:marker});action.input_call_invoked=true;await event('input_call_will_invoke',{api:'CUA App.click',apply_index:Number(apply[1])});
   await appHandle.click(Number(apply[1]));action.input_call_completed=true;await event('input_call_completed');
   const after=await appHandle.getAXState({emit:false,disableDiffing:true});await fs.writeFile(out+'/after-apply.ax.txt',after,{flag:'wx',mode:0o600});action.after_ax_has_expected_readout=after.includes('1.477 mA');await event('after_input_ax_observed');
  }catch(e){action.error=String(e);await fs.writeFile(out+'/abort.request',String(e),{flag:'wx',mode:0o600});}
  finally{if(fileCreated)await persist();await write(out+'/controller.finished',{id,input_call_completed:action.input_call_completed,error:action.error??null});}
  const done=await wait(out+'/done.json',25);results.push({id,status:done.status,input_completed:action.input_call_completed,error:action.error??null});
  if(n!==20)await fs.writeFile(out+'/next.request','Previous result observed by CUA controller.',{flag:'wx',mode:0o600});
 }
 return results;
}