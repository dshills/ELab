import {test} from 'node:test';
import assert from 'node:assert/strict';
import {pagePolling} from './static/polling.mjs';
const flush=()=>new Promise(resolve=>setImmediate(resolve));
const shown=()=>{const e=new Event('pageshow');Object.defineProperty(e,'persisted',{value:true});return e;};

test('leaving aborts an in-flight read without reporting a disconnect or polling again',async()=>{
 const events=new EventTarget(),outcomes=[];let signal,reads=0;
 const polling=pagePolling({events,read:s=>{signal=s;reads++;return new Promise((resolve,reject)=>s.addEventListener('abort',()=>reject(s.reason),{once:true}));},success:()=>outcomes.push('connected'),failure:()=>outcomes.push('disconnected'),delay:()=>0});
 polling.start();assert.equal(reads,1);events.dispatchEvent(new Event('pagehide'));await flush();
 assert.equal(signal.aborted,true);assert.deepEqual(outcomes,[]);assert.equal(reads,1);polling.stop();
});

test('back/forward restoration resumes one chain and genuine failures remain visible',async()=>{
 const events=new EventTarget(),outcomes=[];let reads=0,finish;
 const polling=pagePolling({events,read:()=>{reads++;return new Promise((resolve,reject)=>{finish={resolve,reject};});},success:()=>outcomes.push('connected'),failure:()=>outcomes.push('disconnected'),delay:()=>0});
 polling.start();events.dispatchEvent(new Event('pagehide'));events.dispatchEvent(shown());events.dispatchEvent(shown());
 // A response from the previous page lifetime must not announce a connection.
 finish.resolve();await flush();assert.deepEqual(outcomes,[]);
 await new Promise(resolve=>setTimeout(resolve,5));assert.equal(reads,2);
 finish.reject(new Error('real network outage'));await flush();assert.deepEqual(outcomes,['disconnected']);polling.stop();
});

test('a hidden page finishing boot stays stopped until it is restored',async()=>{
 const events=new EventTarget();let reads=0;
 const polling=pagePolling({events,read:async()=>{reads++;},success:()=>{},failure:()=>{},delay:()=>1000});
 events.dispatchEvent(new Event('pagehide'));polling.start();await flush();assert.equal(reads,0);
 events.dispatchEvent(shown());await flush();assert.equal(reads,1);polling.stop();
});


test('a cancelled navigation aborts the old read and continues updating the current page',async()=>{
 const events=new EventTarget(),outcomes=[];let reads=0,finish;
 const polling=pagePolling({events,read:signal=>{reads++;return new Promise((resolve,reject)=>{finish=resolve;signal.addEventListener('abort',()=>reject(signal.reason),{once:true});});},success:()=>outcomes.push('connected'),failure:()=>outcomes.push('disconnected'),delay:()=>0});
 polling.start();events.dispatchEvent(new Event('beforeunload'));await flush();
 await new Promise(resolve=>setTimeout(resolve,5));assert.equal(reads,2);assert.deepEqual(outcomes,[]);
 finish();await flush();assert.deepEqual(outcomes,['connected']);polling.stop();
});
