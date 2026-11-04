// Own read-only polling for one page lifetime, including back/forward restores.
// Mutating commands use their separate queue and are never aborted here.
export function pagePolling({read,success,failure,delay,events=window}){
 let ready=false,present=true,active=false,timer,controller;
 async function tick(){
  if(!active||controller)return;
  const pending=new AbortController();controller=pending;
  try{await read(pending.signal);if(active&&!pending.signal.aborted)success();}
  catch(error){if(active&&!pending.signal.aborted)failure(error);}
  finally{
   controller=null;
   if(active)timer=setTimeout(tick,delay());
  }
 }
 function resume(){if(!ready||!present||active)return;active=true;if(!controller)void tick();}
 function suspend(){active=false;clearTimeout(timer);timer=undefined;controller?.abort();}
 // WebKit can cancel network reads before pagehide. Abort while the old
 // document is still active; a cancelled navigation keeps its normal retry.
 events.addEventListener('beforeunload',()=>controller?.abort());
 events.addEventListener('pagehide',()=>{present=false;suspend();});
 events.addEventListener('pageshow',event=>{present=true;if(event.persisted)resume();});
 return {start(){ready=true;resume();},stop(){ready=false;suspend();}};
}
