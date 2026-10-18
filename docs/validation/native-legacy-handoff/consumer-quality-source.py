import os,sys,time,json
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-handoff-repair';sys.path.insert(0,str(root/'scripts'))
from runtime_isolation import pinned_child_runner
_,run=pinned_child_runner();env=dict(os.environ,PYTHONDONTWRITEBYTECODE='1')
started=time.monotonic()
name=sys.argv[1] if len(sys.argv)>1 else 'consumer-full-check'
with (base/(name+'.txt')).open('x') as log:p=run(['./scripts/check.sh'],cwd=root,env=env,stdout=log,stderr=-2,timeout=600)
result=dict(passed=p.returncode==0,exit_code=p.returncode,elapsed_seconds=time.monotonic()-started,scope='Complete consumer quality gate against new pinned SDK')
(base/(name+'.json')).write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result))
