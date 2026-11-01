#!/usr/bin/env python3
"""Exercise a copied web backend without launching UI or opening a browser."""
import argparse
import hashlib
import http.cookiejar
import json
import os
from pathlib import Path
import shutil
import subprocess
import time
import urllib.request
import uuid

from package_web import verify_bundle


def qualify(app, artifacts):
    app, artifacts = Path(app).resolve(), Path(artifacts).absolute()
    if artifacts.exists() or artifacts.is_symlink():
        raise ValueError('qualification output must be new')
    verify_bundle(app)
    artifacts.mkdir(parents=True)
    copied = artifacts / 'copied' / app.name
    copied.parent.mkdir()
    shutil.copytree(app, copied, symlinks=False)
    verify_bundle(copied)
    executable = copied / 'Contents/Resources/elab-web'
    metadata = json.loads((copied / 'Contents/Resources/PACKAGE.json').read_text())
    for relative, expected in metadata['executables'].items():
        if hashlib.sha256((copied / relative).read_bytes()).hexdigest() != expected:
            raise ValueError('copied executable changed: ' + relative)
    env = dict(os.environ, PATH='/usr/bin:/bin:/usr/sbin:/sbin')
    if shutil.which('go', path=env['PATH']) or shutil.which('node', path=env['PATH']):
        raise ValueError('qualification PATH must exclude Go and Node')
    data = artifacts / 'data'
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}),
                                         urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
    process = None
    checks = []
    with (artifacts / 'backend.log').open('w') as log:
        def start(name):
            nonlocal process
            reply = artifacts / (name + '.json')
            process = subprocess.Popen([str(executable), '--addr', '127.0.0.1:0',
                                        '--data-dir', str(data), '--startup-file', str(reply)],
                                       cwd=copied.parent, env=env, stdout=log, stderr=subprocess.STDOUT)
            deadline = time.monotonic() + 15
            while time.monotonic() < deadline:
                if reply.exists():
                    try:
                        info = json.loads(reply.read_text())
                        return reply, info
                    except json.JSONDecodeError:
                        pass
                if process.poll() is not None:
                    raise ValueError('copied backend exited during startup; see backend.log')
                time.sleep(.02)
            raise TimeoutError('copied backend did not start')

        def request(info, path, value=None):
            req = urllib.request.Request(info['url'] + path,
                                         data=json.dumps(value).encode() if value is not None else None,
                                         headers={'Content-Type': 'application/json'})
            with opener.open(req, timeout=10) as response:
                return json.loads(response.read())

        def stop(reply):
            subprocess.run([str(executable), '--stop-host', str(reply)], cwd=copied.parent,
                           env=env, stdout=log, stderr=subprocess.STDOUT, check=True, timeout=50)
            if process.wait(timeout=10) != 0:
                raise ValueError('copied backend did not exit normally')
            if (data / '.host.json').exists():
                raise ValueError('active-host metadata remained after normal exit')

        try:
            reply, info = start('first')
            state = request(info, 'api/state')['state']
            for name in ['app.js', 'scope-controls.mjs', 'recording-analysis.mjs', 'spectrum.mjs', 'sweep.mjs']:
                with opener.open(info['url'] + name, timeout=10) as response:
                    if response.status != 200 or 'javascript' not in response.headers.get('Content-Type', ''):
                        raise ValueError('embedded module unavailable: ' + name)
            checks.append('copied embedded assets run with no Go/Node in PATH and outside the source directory')
            result = request(info, 'api/command', {
                'version': 1, 'id': uuid.uuid4().hex, 'key': uuid.uuid4().hex,
                'command': 'component.set', 'params': {'id': 'R1', 'value': '22k'},
                'expected_command_revision': state['command_revision'],
                'expected_document_generation': state['document']['generation']})
            if result.get('error'):
                raise ValueError('native edit refused')
            stop(reply)
            checks.append('normal control Quit drains and exits the copied backend')
            reply, info = start('second')
            restored = request(info, 'api/state')['state']
            if not any(p['id'] == 'R1' and p['value'] == 22000 for p in restored['components']):
                raise ValueError('acknowledged edit did not survive restart')
            checks.append('same browser workspace restores acknowledged 22 kΩ edit after restart')
            stop(reply)
        finally:
            if process is not None and process.poll() is None:
                process.terminate()
                process.wait(timeout=40)
    report = {'passed': True, 'source_commit': metadata['source_commit'],
              'source_clean': metadata['source_clean'], 'checks': checks,
              'scope': 'Copied backend and embedded assets; no GUI, browser, clean-account, or notarization claim.'}
    (artifacts / 'qualification.json').write_text(json.dumps(report, indent=2) + '\n')
    # Launch replies contain control credentials; keep them private local fixtures.
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--app', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    print(json.dumps(qualify(args.app, args.artifacts), indent=2))


if __name__ == '__main__':
    main()
