"""Check pinned notices for the packaged agent's added runtime dependencies."""
import hashlib
import json
from pathlib import Path, PurePosixPath
import re
import subprocess

ROOT=Path(__file__).resolve().parents[1]



def bundle_path(name):
    """Map module @version paths into the unchanged pinned packager alphabet.

    Licenses/AgentMCP retains original paths and bytes. The app's bound notice
    copy uses this documented spelling; callers must reject mapped collisions.
    """
    path=PurePosixPath(name)
    if path.is_absolute() or not path.parts or str(path)!=name:
        raise ValueError('invalid agent notice path')
    parts=[part.replace('@','-at-') for part in path.parts]
    if any(not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._-]*',part) or part.endswith('.') for part in parts):
        raise ValueError('invalid agent notice path')
    return 'NOTICES/AgentMCP/'+'/'.join(parts)


def verify(root=ROOT):
    folder=root/'third_party/agent-mcp'
    data=json.loads((folder/'notices.json').read_text())
    if data.get('schema')!=1:
        raise ValueError('invalid agent notice schema')
    paths=list(folder.rglob('*'))
    if folder.is_symlink() or any(p.is_symlink() or not (p.is_file() or p.is_dir()) for p in paths):
        raise ValueError('agent notices contain a link/special entry')
    actual={p.relative_to(folder).as_posix() for p in paths if p.is_file()}
    if actual!=set(data['files'])|{'notices.json'}:
        raise ValueError('agent notice inventory differs')
    for name,expected in data['files'].items():
        path=Path(name)
        if path.is_absolute() or '..' in path.parts or hashlib.sha256((folder/path).read_bytes()).hexdigest()!=expected:
            raise ValueError('agent notice bytes differ: '+name)
    # Gonum is covered by the existing separate supplied notice bundle.
    declared=json.loads(subprocess.check_output(['go','mod','edit','-json'],cwd=root,text=True))['Require']
    wanted={r['Path']:r['Version'] for r in declared if r['Path']!='gonum.org/v1/gonum'}
    if wanted!={name:entry['version'] for name,entry in data['modules'].items()}:
        raise ValueError('agent notice versions differ from go.mod')
    sums={tuple(line.split()[:2]):line.split()[2] for line in (root/'go.sum').read_text().splitlines()}
    owned=[]
    for name,entry in data['modules'].items():
        if sums.get((name,entry['version']))!=entry['sum'] or not entry['notice_files']:
            raise ValueError('agent notice module checksum differs: '+name)
        for notice in entry['notice_files']:
            if notice not in data['files'] or not notice.startswith(name+'@'+entry['version']+'/'):
                raise ValueError('invalid module notice ownership')
            owned.append(notice)
    if len(owned)!=len(set(owned)) or set(owned)!=set(data['files'])-{'README.md'}:
        raise ValueError('agent notice ownership differs')
    return data


if __name__=='__main__':
    checked=verify()
    print(json.dumps(dict(passed=True,modules=len(checked['modules']),files=len(checked['files']))))
