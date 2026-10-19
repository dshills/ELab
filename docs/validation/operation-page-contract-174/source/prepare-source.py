from pathlib import Path
import ast
import hashlib
import json
import shutil

root = Path('/Users/dshills/Development/projects/ELab')
stage = root/'build/operation-pages-174-work'
out = Path('/private/tmp/elab-operation-pages-source')
out.mkdir(mode=0o700, exist_ok=False)
shutil.copyfile('/private/tmp/elab-operation-pages-174/recorded-rc.elab', out/'recorded-rc.elab')
shutil.copyfile(stage/'elab-agent-source', out/'elab-agent')
(out/'elab-agent').chmod(0o700)
plan = json.loads((stage/'plan.json').read_text())
plan.update({'source_commit': 'f6528ec3 plus documented metadata repair',
             'release': 'unpackaged development source; no clean-release claim',
             'agent_sha256': hashlib.sha256((out/'elab-agent').read_bytes()).hexdigest(),
             'sources': {n: hashlib.sha256((root/n).read_bytes()).hexdigest() for n in plan['sources']}})
(stage/'plan-source.json').write_text(json.dumps(plan, indent=2)+'\n')
source = (stage/'collect.py').read_text()
source = source.replace('One frozen /174 recorded-page contract probe', 'One repaired development recorded-page contract probe')
source = source.replace("OUT = Path('/private/tmp/elab-operation-pages-174')", "OUT = Path('/private/tmp/elab-operation-pages-source')")
source = source.replace("plan = json.loads((STAGE/'plan.json').read_text())", "BINARY = OUT/'elab-agent'\nplan = json.loads((STAGE/'plan-source.json').read_text())")
source = source.replace("(RUNTIME/'elab-agent').read_bytes()", "BINARY.read_bytes()")
source = source.replace("process = subprocess.Popen([str(RUNTIME/'agent')],", "process = subprocess.Popen(['/usr/bin/sandbox-exec', '-f', str(RUNTIME/'runtime.sb'), str(BINARY)],")
source = source.replace("    assert response['id'] == request['id'] and response['version'] == 1", "    assert response['version'] == 1\n    if error == 'invalid-json':\n        assert raw_params is not None and response['id'] == '' and response['revision'] == 0\n    else:\n        assert response['id'] == request['id']")
source = source.replace("read_case('duplicate-limit', 'operation.samples', {}, 'invalid-params', raw_params=raw)", "read_case('duplicate-limit', 'operation.samples', {}, 'invalid-json', raw_params=raw)")
ast.parse(source)
(stage/'collect-source.py').write_text(source)
