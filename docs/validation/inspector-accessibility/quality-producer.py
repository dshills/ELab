import json,os,subprocess,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/ax-recovery-diagnostic';started=time.monotonic()
with (base/'full-check-final.txt').open('x') as log:p=subprocess.run(['./scripts/check.sh'],cwd=root,env=dict(os.environ,PYTHONDONTWRITEBYTECODE='1'),stdout=log,stderr=-2,timeout=600)
r=dict(passed=p.returncode==0,exit_code=p.returncode,seconds=time.monotonic()-started,scope='Complete consumer gate after inspector grouping and native invalid-value accessibility assertions');(base/'full-check-final.json').write_text(json.dumps(r,indent=2)+'\n');print(json.dumps(r));print((base/'full-check-final.txt').read_text()[-1600:]);raise SystemExit(p.returncode)
