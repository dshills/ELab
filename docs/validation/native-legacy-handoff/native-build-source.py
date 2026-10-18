import os,sys,json,hashlib,subprocess
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-handoff-repair';sys.path.insert(0,str(root/'scripts'))
from runtime_isolation import pinned_child_runner
_,run=pinned_child_runner()
s=(root/'desktop/go.mod').read_text().replace('../build/.vellum-sdk-677548d',str(base/'sdk')).replace('replace github.com/dshills/elab => ..','replace github.com/dshills/elab => '+str(root))
(base/'desktop.mod').write_text(s);(base/'desktop.sum').write_bytes((root/'desktop/go.sum').read_bytes())
env=dict(os.environ,GOWORK='off',GOCACHE=str(root/'build/go-cache'),GOPROXY='off',GOTOOLCHAIN='local',CGO_LDFLAGS_ALLOW='-Wl,-rpath,@executable_path/native')
with (base/'native-build.txt').open('x') as log:run(['go','build','-modfile',str(base/'desktop.mod'),'-buildvcs=true','-o',str(base/'ELab-handoff'),'.'],cwd=root/'desktop',env=env,stdout=log,stderr=-2,timeout=150,check=True)
info=subprocess.check_output(['go','version','-m',str(base/'ELab-handoff')],text=True);(base/'native-build-info.txt').write_text(info)
print(json.dumps(dict(passed=True,binary_sha256=hashlib.sha256((base/'ELab-handoff').read_bytes()).hexdigest(),scope='Development repair candidate, alternate modfile, isolated SDK copy; no clean release claim')))
