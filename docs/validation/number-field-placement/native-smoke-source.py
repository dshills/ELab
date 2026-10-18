import json,os,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/placement-number-repair';sys.path.insert(0,str(root/'scripts'))
from runtime_isolation import pinned_child_runner
_,run=pinned_child_runner();started=time.monotonic()
with (base/'native-smoke-final.txt').open('x') as log:p=run([str(base/'ELab-placement-final'),'--smoke','--smoke-timeout','2m'],cwd=root,env=dict(os.environ,PYTHONDONTWRITEBYTECODE='1'),stdout=log,stderr=-2,timeout=150)
text=(base/'native-smoke-final.txt').read_text();result=dict(passed=p.returncode==0 and '"event":"elab.native.smoke","passed":true' in text,exit_code=p.returncode,seconds=time.monotonic()-started,scope='Development candidate against isolated repaired SDK; no clean copied-runtime/release claim')
(base/'native-smoke-final.json').write_text(json.dumps(result,indent=2)+'\n');print(json.dumps(result))
