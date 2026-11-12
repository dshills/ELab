// WebKit can report a fetch started during reload as an uncaught page error.
// Use request events for background reads so navigation cancellation stays local.
export function readPollJSON(url, signal) {
 return new Promise((resolve, reject) => {
  signal.throwIfAborted();
  const request = new XMLHttpRequest();
  const abort = () => request.abort();
  const detach = () => signal.removeEventListener('abort', abort);
  request.open('GET', url);
  request.addEventListener('load', () => {
   if (request.status === 304) {
    resolve(null);
    return;
   }
   try {
    const data = JSON.parse(request.responseText);
    const message = typeof data?.error === 'string' ? data.error : '';
    if (request.status < 200 || request.status >= 300 || message) {
     throw new Error(message || 'The server refused this request.');
    }
    resolve(data);
   } catch (error) {
    reject(error);
   }
  }, {once: true});
  request.addEventListener('error', () => reject(new Error('The state read failed.')), {once: true});
  request.addEventListener('abort', () => reject(signal.reason || new DOMException('The state read was aborted.', 'AbortError')), {once: true});
  request.addEventListener('loadend', detach, {once: true});
  signal.addEventListener('abort', abort, {once: true});
  try {
   request.send();
  } catch (error) {
   detach();
   reject(error);
  }
 });
}

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
 events.addEventListener('beforeunload',()=>{
  if(controller)controller.abort();
  else if(active){
   clearTimeout(timer);
   timer=setTimeout(tick,delay());
  }
 });
 events.addEventListener('pagehide',()=>{present=false;suspend();});
 events.addEventListener('pageshow',event=>{present=true;if(event.persisted)resume();});
 return {start(){ready=true;resume();},stop(){ready=false;suspend();}};
}
