"""macOS runtime rehearsal with development-file and IP-network denial.

The controller remains unrestricted. All copied external ELab processes and
children inherit the generated policy. This is not a security boundary offered
by the application, a clean OS installation, or a distribution/signing check.
"""
import argparse
from contextlib import ExitStack
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import plistlib
import re
import shlex
import shutil
import signal
import socket
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[1]

PROBE = r'''
import errno,json,pathlib,socket,sys
configuration=json.loads(pathlib.Path(sys.argv[1]).read_text())
results={"allowed_read":pathlib.Path(configuration["allowed"]).read_text()=="ELab allowed control\n","files":{},"network":{}}
for name,path in configuration["canaries"].items():
    try:
        with open(path,"rb") as source:source.read(1)
        results["files"][name]={"denied":False}
    except OSError as error:
        results["files"][name]={"denied":isinstance(error,PermissionError) and error.errno in (errno.EPERM,errno.EACCES),"errno":error.errno}
for name,entry in configuration["network"].items():
    family=socket.AF_INET6 if entry["ipv6"] else socket.AF_INET
    kind=socket.SOCK_DGRAM if entry["udp"] else socket.SOCK_STREAM
    with socket.socket(family,kind) as client:
        client.settimeout(2)
        try:
            address=tuple(entry["address"])
            if entry["udp"]:client.sendto(b"ELab network control",address)
            else:client.connect(address)
            results["network"][name]={"denied":False}
        except OSError as error:
            results["network"][name]={"denied":isinstance(error,PermissionError) and error.errno in (errno.EPERM,errno.EACCES),"errno":error.errno}
print(json.dumps(results))
'''


def digest(path):
    value = hashlib.sha256()
    with path.open('rb') as source:
        while block := source.read(1024 * 1024):
            value.update(block)
    return value.hexdigest()


def regular(path):
    if path.is_symlink() or not path.is_file():
        raise ValueError('expected a regular file: '+str(path))
    return path


def bundle_tree(folder):
    if folder.is_symlink() or not folder.is_dir():
        raise ValueError('expected an app directory')
    result = {}
    for path in [folder, *sorted(folder.rglob('*'))]:
        if path.is_symlink() or not (path.is_file() or path.is_dir()):
            raise ValueError('app contains a link or special entry')
        result[path.relative_to(folder).as_posix()] = dict(
            mode=path.stat().st_mode & 0o777, sha256=digest(path) if path.is_file() else None)
    return result


def bundle_executable(folder, sealed_sha256=None):
    bundle_tree(folder)
    info = plistlib.loads(regular(folder/'Contents/Info.plist').read_bytes())
    if not isinstance(info, dict):
        raise ValueError('invalid bundle property list')
    name = info.get('CFBundleExecutable')
    if not isinstance(name, str) or not name or Path(name).name != name or name in ('.', '..') or '\\' in name:
        raise ValueError('invalid bundle executable')
    executable = regular(folder/'Contents/MacOS'/name)
    manifest = json.loads(regular(folder/'Contents/Resources/APP_MANIFEST.json').read_text())
    if not isinstance(manifest, dict) or manifest.get('private_build') is not True:
        raise ValueError('private bundle manifest/executable mismatch')
    source_sha256 = manifest.get('source_binary_sha256')
    if not isinstance(source_sha256, str) or not re.fullmatch('[0-9a-f]{64}', source_sha256):
        raise ValueError('private bundle manifest/executable mismatch')
    # Signing changes Mach-O bytes after the packager records its source hash.
    # Accept those bytes only against an explicitly supplied independent pin;
    # never silently trust an arbitrary signed binary or a self-reported hash.
    expected = source_sha256 if sealed_sha256 is None else sealed_sha256
    if not isinstance(expected, str) or not re.fullmatch('[0-9a-f]{64}', expected) or expected != digest(executable):
        raise ValueError('private bundle manifest/executable mismatch')
    return executable


def native_dependencies(output):
    dependencies = []
    for line in output.splitlines():
        if not line.startswith((' ', '\t')):
            continue
        path = line.strip().split(' (', 1)[0]
        if not path:
            continue
        normalized = os.path.normpath(path)
        if not normalized.startswith(('/System/Library/', '/usr/lib/')):
            raise ValueError('unbundled non-system native dependency: '+path)
        dependencies.append(path)
    if not dependencies:
        raise ValueError('native dependency inspection returned no libraries')
    return sorted(set(dependencies))


def sb_string(value):
    if any(ord(c) < 32 or ord(c) == 127 for c in value):
        raise ValueError('sandbox paths cannot contain control characters')
    return '"'+value.replace('\\', '\\\\').replace('"', '\\"')+'"'


def sandbox_profile(roots):
    if not roots or any(not p.is_absolute() or p == Path('/') for p in roots):
        raise ValueError('expected explicit development directories')
    filters = ' '.join('(subpath '+sb_string(str(p))+')' for p in roots)
    return ('(version 1)\n(allow default)\n'
            '(deny file-read* file-write* '+filters+')\n'
            '(deny network-outbound (remote ip "*:*"))\n')


def rehearsal_profile(roots, protect_geometry=False):
    profile = sandbox_profile(roots)
    if protect_geometry:
        profile += '(deny file-write* (subpath '+sb_string(str(Path.home()/'Library/Cogent Core'))+'))\n'
    return profile


def create_output(path, roots):
    if any(path == root or path.is_relative_to(root) for root in roots):
        raise ValueError('runtime artifacts must be outside blocked development directories')
    path.mkdir(mode=0o700)  # Never reuse, delete or replace an existing destination.


def launcher(path, profile, binary, working):
    content = '#!/bin/sh\nset -eu\ncd -- '+shlex.quote(str(working))+'\nexec /usr/bin/sandbox-exec -f '+shlex.quote(str(profile))+' '+shlex.quote(str(binary))+' "$@"\n'
    with path.open('x') as output:
        output.write(content)
    path.chmod(0o700)


def pinned_child_runner():
    # The qualification controller may use development tools. Its children may
    # not. Reuse the pinned SDK's tested process-group cleanup implementation.
    previous = sys.dont_write_bytecode
    try:
        sys.dont_write_bytecode = True
        spec = importlib.util.spec_from_file_location('runtime_prepare_sdk', ROOT/'scripts/prepare-vellum.py')
        prepare = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(prepare)
        sdk = prepare.prepare_sdk()
        spec = importlib.util.spec_from_file_location('runtime_child_process', sdk/'scripts/child_process.py')
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        return sdk, module.run_child
    finally:
        sys.dont_write_bytecode = previous


def development_canaries(sdk):
    values = subprocess.check_output(['go', 'env', '-json', 'GOMODCACHE', 'GOCACHE', 'GOROOT'], text=True, timeout=30, env=dict(os.environ, GOTOOLCHAIN='local', GOWORK='off', GOPROXY='off'))
    go = json.loads(values)
    developer = Path(subprocess.check_output(['xcode-select', '--print-path'], text=True, timeout=10).strip()).resolve()
    candidates = {
        'repository': (ROOT, ROOT/'README.md'),
        'sdk': (sdk, sdk/'go.mod'),
        'sibling_vellum': (ROOT.parent/'Vellum', ROOT.parent/'Vellum/go.mod'),
        'go_toolchain': (Path(go['GOROOT']), Path(go['GOROOT'])/'VERSION'),
        'go_build_cache': (Path(go['GOCACHE']), Path(go['GOCACHE'])/'README'),
        'project_go_cache': (ROOT/'build/go-cache', ROOT/'build/go-cache/README'),
    }
    modules = Path(go['GOMODCACHE'])
    modfiles = sorted((modules/'cache/download/golang.org/x/text/@v').glob('*.mod'))
    if modules.exists():
        if not modfiles:
            raise ValueError('existing module cache lacks the expected text-module read control')
        candidates['go_module_cache'] = (modules, modfiles[-1])
    compiler = next((p for p in [developer/'usr/bin/xcodebuild', developer/'usr/bin/clang'] if p.is_file()), None)
    if compiler is None:
        raise ValueError('developer toolchain read control unavailable')
    candidates['developer_sdk'] = (developer, compiler)
    result = {}
    for name, (root, canary) in candidates.items():
        if not root.exists():
            continue
        root, canary = root.resolve(), canary.resolve()
        if not canary.is_relative_to(root):
            raise ValueError('read control escaped its development directory')
        regular(canary)
        with canary.open('rb') as source:
            source.read(1)  # Missing/unreadable controls cannot establish a denial.
        result[name] = dict(root=str(root), canary=str(canary))
    return result


def validate_controls(result, names):
    if result.get('allowed_read') is not True or set(result.get('files', {})) != set(names):
        raise ValueError('missing sandbox read controls')
    expected = {'ipv4_tcp', 'ipv4_udp', 'ipv6_tcp', 'ipv6_udp'}
    if set(result.get('network', {})) != expected:
        raise ValueError('missing IP network controls')
    for entry in [*result['files'].values(), *result['network'].values()]:
        if not isinstance(entry, dict) or entry.get('denied') is not True or type(entry.get('errno')) is not int or entry['errno'] not in (1, 13):
            raise ValueError('sandbox did not prove a permission denial')


def controls(directory, profile, canaries, run_child):
    allowed = directory/'allowed-control.txt'
    allowed.write_text('ELab allowed control\n')
    configuration = dict(allowed=str(allowed), canaries={k: v['canary'] for k, v in canaries.items()}, network={})
    with ExitStack() as stack:
        for ipv6 in (False, True):
            for udp in (False, True):
                family = socket.AF_INET6 if ipv6 else socket.AF_INET
                kind = socket.SOCK_DGRAM if udp else socket.SOCK_STREAM
                server = stack.enter_context(socket.socket(family, kind))
                server.settimeout(2)
                server.bind(('::1' if ipv6 else '127.0.0.1', 0))
                if not udp:
                    server.listen(1)
                address = server.getsockname()
                # Prove the local target works before asking the sandbox to deny it.
                with socket.socket(family, kind) as client:
                    client.settimeout(2)
                    if udp:
                        client.sendto(b'control', address)
                        if server.recvfrom(32)[0] != b'control':
                            raise ValueError('unrestricted UDP control failed')
                    else:
                        client.connect(address)
                        connection, _ = server.accept()
                        connection.close()
                name = ('ipv6' if ipv6 else 'ipv4')+('_udp' if udp else '_tcp')
                configuration['network'][name] = dict(ipv6=ipv6, udp=udp, address=address)
        config_path = directory/'controls-input.json'
        config_path.write_text(json.dumps(configuration, indent=2)+'\n')
        probe = directory/'probe.py'
        probe.write_text(PROBE)
        command = ['/usr/bin/sandbox-exec', '-f', str(profile), sys.executable, '-I', str(probe), str(config_path)]
        completed = run_child(command, cwd=directory, capture_output=True, text=True, timeout=20)
        (directory/'controls-stderr.log.txt').write_text(completed.stderr)
        completed.check_returncode()
        result = json.loads(completed.stdout)
        (directory/'controls.json').write_text(json.dumps(result, indent=2)+'\n')
        validate_controls(result, canaries)
        return result


def execute_stage(result, directory, working, name, command, timeout, run_child, save):
    stage = dict(name=name, command=command, status='running')
    result['stages'].append(stage)
    save()
    print('Starting '+name, flush=True)
    started = time.monotonic()
    try:
        with (directory/(name+'.log.txt')).open('x') as log:
            completed = run_child(command, cwd=working, stdout=log, stderr=subprocess.STDOUT, timeout=timeout, check=True)
    except BaseException as error:
        stage.update(status='interrupted' if isinstance(error, (KeyboardInterrupt, SystemExit)) else 'failed',
                     error_type=type(error).__name__, exit_code=getattr(error, 'returncode', None), elapsed_seconds=time.monotonic()-started)
        save()
        raise
    stage.update(exit_code=completed.returncode, status='completed', elapsed_seconds=time.monotonic()-started)
    save()


def run(args):
    if platform.system() != 'Darwin' or not Path('/usr/bin/sandbox-exec').is_file():
        raise ValueError('this runtime rehearsal requires macOS sandbox-exec')
    app = args.app.absolute()
    executable = bundle_executable(app, args.app_executable_sha256)
    agent, controller = regular(args.agent_binary.absolute()), regular(args.conformance_binary.absolute())
    dependencies = native_dependencies(subprocess.check_output(['otool', '-L', str(executable)], text=True, timeout=30))
    sdk, run_child = pinned_child_runner()
    canaries = development_canaries(sdk)
    roots = [Path(v['root']) for v in canaries.values()]
    directory = args.artifacts.absolute().resolve()
    create_output(directory, roots)
    result = dict(passed=False, scope='macOS copied runtime with development-file and outbound IP denial; controller unrestricted',
                  geometry_write_protected=bool(getattr(args, 'protect_geometry', False)),
                  app_source=str(app), source_bundle=bundle_tree(app), source_agent_sha256=digest(agent),
                  source_controller_sha256=digest(controller), system_dependencies=dependencies, blocked_development=canaries, stages=[],
                  executable_identity_source='explicit sealed-byte pin' if args.app_executable_sha256 else 'manifest source hash',
                  executable_sha256=digest(executable), manifest_source_sha256=json.loads((app/'Contents/Resources/APP_MANIFEST.json').read_text())['source_binary_sha256'])
    report = directory/'runtime-report.json'
    def save():
        report.write_text(json.dumps(result, indent=2)+'\n')
    save()
    try:
        copied = directory/'ELab.app'
        shutil.copytree(app, copied)
        shutil.copyfile(agent, directory/'elab-agent')
        (directory/'elab-agent').chmod(0o700)
        if bundle_tree(copied) != result['source_bundle'] or digest(directory/'elab-agent') != result['source_agent_sha256']:
            raise ValueError('runtime copy changed source bytes/modes')
        working = directory/'empty-working-directory'
        working.mkdir()
        profile = directory/'runtime.sb'
        profile.write_text(rehearsal_profile(roots, getattr(args, 'protect_geometry', False)))
        result['controls'] = controls(directory, profile, canaries, run_child)
        launcher(directory/'native', profile, copied/'Contents/MacOS'/executable.name, working)
        launcher(directory/'agent', profile, directory/'elab-agent', working)
        commands = [
            ('native-smoke', [str(directory/'native'), '--smoke', '--smoke-timeout', '2m'], 180),
            ('public-suites', [str(controller), '--agent-binary', str(directory/'agent'), '--native-binary', str(directory/'native'),
                '--project-contract-suite', '--electrical-suite', '--inductor-suite', '--opamp-suite', '--opamp-storage-suite', '--capacity-suite',
                '--artifacts', str(directory/'public')], 900),
        ]
        for name, command, timeout in commands:
            execute_stage(result, directory, working, name, command, timeout, run_child, save)
        public = json.loads((directory/'public/report.json').read_text())
        if not public.get('passed') or [a['name'] for a in public['adapters']] != ['in-process', 'headless-stdio', 'headless-socket', 'native-socket']:
            raise ValueError('public conformance report incomplete')
        result['public_report_sha256'] = digest(directory/'public/report.json')
        result['copied_bundle_unchanged'] = bundle_tree(copied) == result['source_bundle'] == bundle_tree(app)
        result['agent_unchanged'] = digest(agent) == digest(directory/'elab-agent') == result['source_agent_sha256']
        result['controller_unchanged'] = digest(controller) == result['source_controller_sha256']
        if not all(result[k] for k in ('copied_bundle_unchanged', 'agent_unchanged', 'controller_unchanged')):
            raise ValueError('runtime inputs changed during rehearsal')
        result['passed'] = True
        save()
    except BaseException as error:
        result['error'] = type(error).__name__+': '+str(error)
        save()
        raise
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--app', type=Path, required=True)
    parser.add_argument('--protect-geometry', action='store_true', help='add a write denial for the owner shared window-geometry directory')
    parser.add_argument('--app-executable-sha256', help='Independent post-signing executable SHA-256; required when signing changed the packager source bytes')
    parser.add_argument('--agent-binary', type=Path, default=ROOT/'build/elab-agent')
    parser.add_argument('--conformance-binary', type=Path, default=ROOT/'build/elab-conformance')
    parser.add_argument('--artifacts', type=Path, required=True, help='new directory outside the repository and development caches')
    args = parser.parse_args()
    previous = signal.signal(signal.SIGTERM, lambda signum, frame: sys.exit(128+signum))
    try:
        result = run(args)
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        parser.exit(1, str(error)+'\n')
    except KeyboardInterrupt:
        parser.exit(130, 'runtime rehearsal interrupted; owned stage cleanup attempted\n')
    finally:
        signal.signal(signal.SIGTERM, previous)
    print(json.dumps(dict(passed=result['passed'], report=str(args.artifacts/'runtime-report.json'))))


if __name__ == '__main__':
    main()
