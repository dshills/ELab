import {test} from 'node:test';
import assert from 'node:assert/strict';
import {freeSelectionOffset} from './static/selection.mjs';
test('group placement avoids occupied destinations and respects boundary positions',()=>{
 const parts=[{x:0,y:0},{x:10,y:0}],used=[...parts,{x:100,y:100},{x:-90,y:100}];
 const offset=freeSelectionOffset(parts,used);assert.ok(offset);
 for(const p of parts)assert.ok(!used.some(q=>q.x===p.x+offset.x&&q.y===p.y+offset.y));
 const corner=freeSelectionOffset([{x:10000,y:10000}],[]);assert.ok(corner.x<0&&corner.y<0);
 assert.equal(freeSelectionOffset([{x:-10000,y:-10000},{x:10000,y:10000}],[]),null);
});
