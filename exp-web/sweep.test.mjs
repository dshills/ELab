import test from 'node:test';
import assert from 'node:assert/strict';
import {sweepRequest,sweepDefaults,nearestSweepPoint} from './static/sweep.mjs';

test('sweep requests preserve engineering quantities, multiple outputs and native default semantics',()=>{
 const target={component_id:'VIN',parameter:'frequency'};
 const values={start:'1 kHz',stop:'10 Hz',spacing:'logarithmic',points:'9'};
 const automatic=sweepRequest(values,target);
 assert.equal('probe_ids' in automatic.settings,false,'automatic output must omit the list, not submit an invalid empty list');
 assert.equal(automatic.settings.start,'1 kHz');
 assert.equal(automatic.settings.stop,'10 Hz');
 assert.equal(automatic.settings.points,9);
 assert.equal(automatic.settings.spacing,'logarithmic');
 assert.equal(automatic.settings.component_id,'VIN');
 assert.equal(automatic.settings.parameter,'frequency');
 assert.equal('probe_ids' in sweepRequest(values,target,[]).settings,false,'explicitly empty selection also requests native defaults');
 assert.deepEqual(sweepRequest(values,target,['v:C1','v:R1']).settings.probe_ids,['v:C1','v:R1']);
 assert.deepEqual(sweepRequest(values,{component_id:'U2',parameter:'loop-frequency'},['v:C1']).settings.probe_ids,['loop:U2'],'the native loop method requires the selected amplifier’s loop probe');
 assert.equal(sweepDefaults({component_id:'PD1',parameter:'light'}).stop,'1000');
 assert.equal(sweepDefaults(target).spacing,'logarithmic');
});
test('plot selection returns an exact captured index for descending, logarithmic and invalid data',()=>{
 const points=[{input:1000},{input:100},{input:10},{input:1}];
 assert.equal(nearestSweepPoint(points,40,true),1,'log distance chooses 100 rather than the linear-nearest 10');
 assert.equal(nearestSweepPoint(points,40),2);
 assert.equal(nearestSweepPoint(points,0,true),null);
 assert.equal(nearestSweepPoint(points,NaN),null);
 assert.equal(nearestSweepPoint([],20),null);
 assert.equal(nearestSweepPoint([{input:NaN},{input:0},{input:10}],10,true),2);
 assert.equal(nearestSweepPoint([{input:1},{input:3}],2),0,'ties choose a stable stored point');
});
