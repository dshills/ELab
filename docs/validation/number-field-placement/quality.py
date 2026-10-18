import os,sys,time,json
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/placement-number-repair';sys.path.insert(0,str(root/'scripts'))
from runtime_isolation import pinned_child_runner
_,run=pinned_child_runner();env=dict(os.environ,GOWORK='off',GOCACHE=str(root/'build/go-cache'),GOPROXY='off',GOTOOLCHAIN='local',CGO_LDFLAGS_ALLOW='-Wl,-rpath,@executable_path/native')
commands=[['go','test','-race','-count=1','./...'],['go','vet','./...'],['go','test','-race','-tags=offscreen','-count=1','-run','^Test(NumberSettlement|ManagedLegacyBoundary|ReviewFieldSettlement|ManagedTextFieldFocusLoss)','./...'],['go','vet','-tags=offscreen','./...']]
results=[]
name=sys.argv[1] if len(sys.argv)>1 else 'sdk-quality'
with (base/(name+'.txt')).open('x') as log:
 for cmd in commands:
  started=time.monotonic();p=run(cmd,cwd=base/'sdk/drivers/cogent',env=env,stdout=log,stderr=-2,timeout=180);result=dict(command=cmd,exit_code=p.returncode,seconds=time.monotonic()-started);results.append(result);print(json.dumps(result),flush=True)
  if p.returncode:break
(base/(name+'.json')).write_text(json.dumps(dict(passed=len(results)==len(commands) and all(r['exit_code']==0 for r in results),stages=results),indent=2)+'\n')
