import json,os,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/placement-number-repair';sys.path.insert(0,str(root/'scripts'))
from runtime_isolation import pinned_child_runner
_,run=pinned_child_runner();env=dict(os.environ,PYTHONDONTWRITEBYTECODE='1',GOPROXY='off',GOTOOLCHAIN='local',GOCACHE=str(root/'build/go-cache'))
started=time.monotonic()
with (base/'export.txt').open('x') as log:p=run([sys.executable,'-B',str(base/'source/scripts/release-sdk.py'),'--output',str(root/'third_party/vellum/vellum-b22478c.tar.gz')],cwd=base/'source',env=env,stdout=log,stderr=-2,timeout=600)
r=dict(passed=p.returncode==0,exit_code=p.returncode,seconds=time.monotonic()-started);(base/'export-result.json').write_text(json.dumps(r,indent=2)+'\n');print(json.dumps(r))
