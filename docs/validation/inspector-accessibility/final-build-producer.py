import hashlib,json,os,subprocess,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/ax-recovery-diagnostic';identity=json.loads((base/'development-identity.json').read_text());inputs=identity['production_source_sha256']
def verify():
 for name,sha in inputs.items():assert hashlib.sha256((root/name).read_bytes()).hexdigest()==sha,name
verify();started=time.monotonic()
with (base/'final-build.txt').open('x') as log:p=subprocess.run(['go','build','-o',str(base/'elab-final'),'.'],cwd=root/'desktop',env=dict(os.environ,GOWORK='off',GOCACHE=str(root/'build/go-cache'),CGO_LDFLAGS_ALLOW='-Wl,-rpath,@executable_path/native'),stdout=log,stderr=-2,timeout=120)
assert p.returncode==0;verify();sha=hashlib.sha256((base/'elab-final').read_bytes()).hexdigest();assert sha==identity['native_sha256'],(sha,identity['native_sha256']);info=subprocess.check_output(['go','version','-m',str(base/'elab-final')],cwd=root,text=True);(base/'native-build-info.txt').write_text(info);assert 'vcs.revision=40cfd37d2560' in info and 'vcs.modified=true' in info
r=dict(passed=True,exit_code=p.returncode,seconds=time.monotonic()-started,native_sha256=sha,exact_binary_matches_native_replay_and_smoke=True,source_inputs_unchanged=True,scope='Working-tree development build, not clean packaged release');(base/'final-build.json').write_text(json.dumps(r,indent=2)+'\n');print(json.dumps(r))
