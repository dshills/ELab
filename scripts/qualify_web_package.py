#!/usr/bin/env python3
"""Exercise a copied web backend without launching UI or opening a browser."""
import argparse
import hashlib
import http.cookiejar
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import time
import urllib.request
import urllib.parse
import zipfile
import uuid

from package_web import verify_bundle
from distribution import verify_distribution


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
    if metadata.get('notarization_requested'):
        verify_distribution(copied)
    for relative, expected in metadata['executables'].items():
        if hashlib.sha256((copied / relative).read_bytes()).hexdigest() != expected:
            raise ValueError('copied executable changed: ' + relative)
    return exercise_host(executable, metadata, artifacts)


def exercise_host(executable, metadata, artifacts):
    """Exercise an exact extracted host with an isolated library and no toolchain."""
    artifacts.mkdir(parents=True, exist_ok=True)
    empty_path = artifacts/'empty-path'
    empty_path.mkdir()
    env = dict(os.environ, PATH=str(empty_path))
    data = artifacts / 'data'
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}),
                                         urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
    process = None
    checks = []
    with (artifacts / 'backend.log').open('w') as log:
        def launch_args(reply=None):
            args = [str(executable), '--addr', '127.0.0.1:0', '--data-dir', str(data)]
            if reply is not None:
                args.extend(['--startup-file', str(reply)])
            return args

        def start(name):
            nonlocal process
            reply = artifacts / (name + '.json')
            process = subprocess.Popen(launch_args(reply),
                                       cwd=artifacts, env=env, stdout=log, stderr=subprocess.STDOUT)
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

        def transfer(info, path, body):
            req = urllib.request.Request(info['url'] + path, data=body,
                                         headers={'Content-Type': 'application/zip'})
            with opener.open(req, timeout=15) as response:
                return response.headers, response.read()

        def stop(reply):
            subprocess.run([str(executable), '--stop-host', str(reply)], cwd=artifacts,
                           env=env, stdout=log, stderr=subprocess.STDOUT, check=True, timeout=50)
            if process.wait(timeout=10) != 0:
                raise ValueError('copied backend did not exit normally')
            if (data / '.host.json').exists():
                raise ValueError('active-host metadata remained after normal exit')

        try:
            reply, info = start('first')
            about = request(info, 'api/about')
            if (about.get('version') != metadata['version'] or about.get('build') != metadata['build']
                    or about.get('source_commit') != metadata['source_commit']
                    or about.get('source_modified') != (not metadata['source_clean'])):
                raise ValueError('running backend build identity differs from sealed package')
            checks.append('running backend exposes matching version, build and VCS identity')
            if metadata.get('target') and f"{about.get('os')}-{about.get('architecture')}" != metadata['target']:
                raise ValueError('running backend platform differs from the release target')
            # With no startup reply requested, a second CLI invocation
            # must refuse rather than opening another writer.
            duplicate = subprocess.run(launch_args(), cwd=artifacts, env=env,
                                       stdout=log, stderr=subprocess.STDOUT, timeout=10)
            if duplicate.returncode == 0:
                raise ValueError('a second host acquired the same library')
            checks.append('a second host cannot acquire the active library')
            state = request(info, 'api/state')['state']
            for name in ['app.js', 'scope-controls.mjs', 'recording-analysis.mjs', 'spectrum.mjs', 'sweep.mjs', 'agent.mjs', 'library-backup.mjs', 'support.mjs']:
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
                raise ValueError('browser workbench edit refused')
            for kind, mime, signature in [
                ('pdf', 'application/pdf', b'%PDF-'),
                ('png', 'image/png', bytes.fromhex('89504e470d0a1a0a')),
            ]:
                # Export completion can change the revision; re-read the guards.
                state = request(info, 'api/state')['state']
                download = urllib.parse.urlencode({'revision': state['command_revision'],
                    'generation': state['document']['generation'], 'kind': kind})
                req = urllib.request.Request(info['url']+'api/download?'+download, data=b'')
                with opener.open(req, timeout=30) as response:
                    payload = response.read(8*1024*1024+1)
                    if (response.headers.get_content_type() != mime or not payload.startswith(signature)
                            or not 100 < len(payload) <= 8*1024*1024):
                        raise ValueError('copied backend returned an invalid '+kind+' download')
            checks.append('embedded workbench downloads schematic PDF and PNG')
            current = request(info, 'api/state')['state']
            query = urllib.parse.urlencode({'revision': current['command_revision'],
                                            'generation': current['document']['generation']})
            headers, backup = transfer(info, 'api/library/backup?' + query, b'')
            if headers.get('X-ELab-Backup-SHA256') != hashlib.sha256(backup).hexdigest():
                raise ValueError('copied backend backup archive checksum differs')
            with zipfile.ZipFile(io.BytesIO(backup)) as archive:
                manifest = json.loads(archive.read('manifest.json'))
                if (manifest['format'] != 'elab-library' or len(manifest['records']) != 1
                        or manifest['records'][0]['kind'] != 'workspaces'):
                    raise ValueError('copied backend backup omitted the unsaved workspace')
                record = manifest['records'][0]
                if hashlib.sha256(archive.read(record['file'])).hexdigest() != record['sha256']:
                    raise ValueError('copied backend backup document checksum differs')
            _, preview = transfer(info, 'api/library/restore?' + query + '&preview=true', backup)
            if json.loads(preview)['contents']['workspaces'] != 1:
                raise ValueError('copied backend backup preview differs')
            _, imported = transfer(info, 'api/library/restore?' + query, backup)
            if json.loads(imported)['restored']['workspaces'] != 1:
                raise ValueError('copied backend failed to restore a workspace copy')
            unchanged = request(info, 'api/state')['state']
            if unchanged['command_revision'] != current['command_revision']:
                raise ValueError('library restore replaced the open workspace')
            checks.append('copied backend backs up unsaved work, validates a preview and restores a separate recovery copy')
            stop(reply)
            checks.append('normal control Quit drains and exits the copied backend')
            reply, info = start('second')
            restored = request(info, 'api/state')['state']
            if not any(p['id'] == 'R1' and p['value'] == 22000 for p in restored['components']):
                raise ValueError('acknowledged edit did not survive restart')
            checks.append('same browser workspace restores acknowledged 22 kΩ edit after restart')
            if not any(entry.get('checkpoint') for entry in request(info, 'api/projects')['history']):
                raise ValueError('restored backup workspace disappeared after restart')
            checks.append('restored backup recovery copy remains discoverable after restart')
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
