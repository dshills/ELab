// Disposable host only. Never attach to a developer's running library.
import {mkdtempSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {fileURLToPath} from 'node:url';
import {resolve,join} from 'node:path';
import {spawn} from 'node:child_process';

if(!process.env.CI&&process.env.ELAB_BROWSER_TESTS!=='1') {
 console.error('Browser journeys are disabled locally. Set ELAB_BROWSER_TESTS=1 explicitly when desktop testing is available.');
 process.exit(1);
}
const root=fileURLToPath(new URL('../../',import.meta.url));
const data=mkdtempSync(join(tmpdir(),'elab-browser-'));
const child=spawn(resolve(root,'build/elab-web-browser'),['--addr','127.0.0.1:18788','--data-dir',data],{cwd:tmpdir(),stdio:'inherit'});
let stopped=false;
for(const signal of ['SIGINT','SIGTERM'])process.on(signal,()=>{stopped=true;child.kill('SIGTERM');});
child.once('error',error=>{console.error(error);rmSync(data,{recursive:true,force:true});process.exitCode=1;});
child.once('exit',(code,signal)=>{
 rmSync(data,{recursive:true,force:true});
 process.exitCode=stopped?0:code|| (signal?1:0);
});
