from pathlib import Path
import hashlib
import json
import os
import subprocess
import time

root = Path('/Users/dshills/Development/projects/ELab')
out = Path(__file__).resolve().parent
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
old = json.loads((root / 'docs/validation/sweep-smoke-admission/quality-default.json').read_text())
before = {name: sha(root / name) for name in old['source_sha256']}
changed = [name for name in before if before[name] != old['source_sha256'][name]]
assert changed == ['desktop/project_work.go'], changed
assert sha(root / 'docs/validation/export-progress-labels/original-project_work.go.txt') == old['source_sha256']['desktop/project_work.go']
environment = os.environ.copy()
environment['GOPROXY'] = 'off'
environment['GOWORK'] = 'off'
start = time.monotonic()
with (out / 'stdout.txt').open('x') as stdout, (out / 'stderr.txt').open('x') as stderr:
    result = subprocess.run(['./scripts/check.sh'], cwd=root, env=environment,
                            stdout=stdout, stderr=stderr, timeout=1800)
after = {name: sha(root / name) for name in before}
report = dict(passed=result.returncode == 0 and after == before, exit_code=result.returncode,
              elapsed_s=time.monotonic()-start, command='./scripts/check.sh',
              changed_go_sources=changed, source_sha256=before, sources_unchanged_during_checks=after == before,
              quality_runner_sha256=sha(root / 'scripts/check.sh'), environment=dict(GOPROXY='off', GOWORK='off'))
(out / 'report.json').write_text(json.dumps(report, indent=2) + '\n')
print(json.dumps({key: value for key, value in report.items() if key != 'source_sha256'}))
raise SystemExit(0 if report['passed'] else 1)
