#!/usr/bin/env node
import {readdirSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import {resolve} from 'node:path';
import {spawnSync} from 'node:child_process';
const root=fileURLToPath(new URL('../',import.meta.url));
const run=args=>{const result=spawnSync(process.execPath,args,{cwd:root,stdio:'inherit'});if(result.error)throw result.error;if(result.status!==0)process.exit(result.status||1);};
for(const file of readdirSync(resolve(root,'static')).filter(name=>/\.(mjs|js)$/.test(name)))run(['--check','static/'+file]);
run(['--test',...readdirSync(resolve(root)).filter(name=>name.endsWith('.test.mjs')).map(name=>name)]);
