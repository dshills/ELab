import {test} from 'node:test';
import assert from 'node:assert/strict';
import {browserEnvironment} from './static/support.mjs';
test('support browser details identify engines without exporting raw UA, URLs or unbounded dimensions',()=>{
 const ua='Mozilla/5.0 (Macintosh) Chrome/147.0.1 Safari/537.36 owner-secret https://private.test/?token=secret /Users/private';
 const report=browserEnvironment(ua,1440,1000,2);
 assert.deepEqual(report,{family:'Chrome',version:'147.0.1',platform:'macOS',viewport:{width:1440,height:1000},device_pixel_ratio:2});
 assert.equal(JSON.stringify(report).includes('secret'),false);
 assert.equal(browserEnvironment('Version/26.0 Safari/605.1').family,'Safari');
 assert.equal(browserEnvironment('Firefox/144.0').family,'Firefox');
 assert.equal(browserEnvironment('Chrome/147.0 Edg/147.0').family,'Edge');
 assert.equal(browserEnvironment('custom-owner-secret',Infinity,-5,NaN).family,'Other');
 assert.deepEqual(browserEnvironment('',Infinity,-5,NaN).viewport,{width:0,height:0});
});
