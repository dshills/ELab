import hashlib,json,os,subprocess,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/ax-recovery-diagnostic';binary=base/'elab';started=time.monotonic()
with (base/'native-smoke-before.txt').open('x') as log:
 p=subprocess.run([str(binary),'--smoke','--smoke-timeout','2m','--no-recovery','--no-appearance'],cwd=root,env=dict(os.environ,PYTHONDONTWRITEBYTECODE='1'),stdout=log,stderr=-2,timeout=150)
text=(base/'native-smoke-before.txt').read_text();r=dict(passed=p.returncode==0 and '"event":"elab.native.smoke","passed":true' in text,exit_code=p.returncode,seconds=time.monotonic()-started,binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),scope='Development native diagnostic; same pinned SDK, added smoke assertions only; not a clean release or external CUA acceptance')
(base/'native-smoke-before.json').write_text(json.dumps(r,indent=2)+'\n');print(json.dumps(r));print('\n'.join(line for line in text.splitlines() if 'led-inspector.accessibility' in line or 'not ready' in line or 'smoke failed' in line)[-12000:])
