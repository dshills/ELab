#!/usr/bin/env python3
"""Assemble, qualify and archive a new local unsigned ELab candidate on macOS."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys

from acceptance_client import AcceptanceClient
import quality
import agent_notices
import runtime_isolation

ROOT = Path(__file__).resolve().parents[1]
GUIDES = ['agent-api', 'workbench-workflows', 'application-icon', 'discrete-class-ab',
          'ac-frequency-sweeps', 'spectrum-analysis', 'transistor-profiles',
          'tolerance-analysis', 'loudspeaker', 'class-ab-speaker', 'passive-losses',
          'current-devices', 'jfets', 'feedback-and-step-response', 'circuit-blocks',
          'temperature', 'noise-analysis', 'photodiodes', 'thyristors', 'analysis-pages', 'agent-setup', 'workspace-layout']
MANIFEST = 'ELAB_PACKAGE.json'


def check(condition, message):
    if not condition:
        raise ValueError(message)


def sha(path):
    return runtime_isolation.digest(path)


def tree(folder):
    """Only regular payload files/directories; links and special entries refuse."""
    check(folder.is_dir() and not folder.is_symlink(), 'expected a regular package directory')
    result = {}
    for path in sorted(folder.rglob('*')):
        check(not path.is_symlink() and (path.is_file() or path.is_dir()), 'package contains a link/special entry')
        if path.is_file():
            name = path.relative_to(folder).as_posix()
            if name != MANIFEST:
                result[name] = dict(sha256=sha(path), mode=stat.S_IMODE(path.stat().st_mode))
    return result


def exclusive(path):
    check(not os.path.lexists(path), 'output already exists: ' + str(path))
    path.mkdir(parents=True, exist_ok=False)


def separate(*paths):
    resolved = [p.resolve() for p in paths]
    for i, path in enumerate(resolved):
        for other in resolved[i+1:]:
            check(path != other and path not in other.parents and other not in path.parents,
                  'input/output directories must not overlap')


def read_package(folder):
    data = json.loads((folder / MANIFEST).read_text())
    check(data.get('schema') == 1 and data.get('files') == tree(folder), 'package file/mode identities changed')
    for name in ['app', 'agent']:
        value = data.get(name)
        check(isinstance(value, str) and not Path(value).is_absolute() and '..' not in Path(value).parts,
              'invalid packaged path')
    return data


def run(args, artifacts, name, cwd=ROOT, env=None, timeout=900):
    with (artifacts / (name + '.log')).open('x') as stream:
        code, _ = quality.execute_owned(args, cwd, env or os.environ.copy(), stream, timeout)
    check(code == 0, name + ' failed; inspect its log')


def assemble(args):
    separate(args.output, args.work, args.examples, args.licenses)
    check(re.fullmatch(r'\d+\.\d+\.\d+', args.version) is not None and args.build > 0, 'invalid version/build')
    check(not subprocess.check_output(['git', 'status', '--porcelain'], cwd=ROOT, text=True).strip(),
          'commit source before assembly')
    check(not os.path.lexists(args.output) and not os.path.lexists(args.work), 'assembly outputs must be new')
    exclusive(args.work)
    args._owned = args.work
    exclusive(args.output)
    source = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    env = quality.go_environment(ROOT)
    run([sys.executable, '-B', 'scripts/prepare-vellum.py'], args.work, 'sdk', env=env)
    for name, command in [('agent', ['go', 'build', '-o', str(args.output/'elab-agent'), './cmd/elab-agent']),
                          ('controller', ['go', 'build', '-o', str(args.work/'elab-conformance'), './cmd/elab-conformance']),
                          ('native', ['go', '-C', 'desktop', 'build', '-o', str(args.work/'elab'), '.'])]:
        run(command, args.work, 'build-'+name, env=env)
    # Copy explicit curated inputs after rejecting links, never replacing sources.
    for source_dir, name in [(args.examples, 'Examples'), (args.licenses, 'Licenses')]:
        tree(source_dir)
        shutil.copytree(source_dir, args.output/name)
    # Refresh only the newly owned output copy from the authoritative pinned
    # source, never the caller's supplied license corpus or an older package.
    agent_notices.verify(ROOT)
    added = args.output/'Licenses/AgentMCP'
    if added.exists():
        shutil.rmtree(added)
    shutil.copytree(ROOT/'third_party/agent-mcp', added)
    notices = args.work/'notices'
    notices.mkdir()
    files, components = {}, []
    for folder in ['Vellum', 'Gonum', 'AgentMCP']:
        source_dir = args.output/'Licenses'/folder
        check(source_dir.is_dir(), 'missing explicit '+folder+' notices')
        names = []
        for path in sorted(source_dir.rglob('*')):
            if path.is_file():
                relative = path.relative_to(source_dir).as_posix()
                name = agent_notices.bundle_path(relative) if folder == 'AgentMCP' else 'NOTICES/'+folder+'/'+relative
                check(name not in files, 'duplicate app notice path')
                dest = notices/name
                dest.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(path, dest)
                files[name] = dict(bytes=path.stat().st_size, sha256=sha(path))
                names.append(name)
        components.append(dict(id=folder+' supplied notices', notice_files=names))
    path = args.output/'Licenses/Vellum-packager-LICENSE'
    check(path.is_file(), 'missing packager notice')
    name = 'NOTICES/Vellum-packager-LICENSE'
    shutil.copyfile(path, notices/name)
    files[name] = dict(bytes=path.stat().st_size, sha256=sha(path))
    components.append(dict(id='Pinned Vellum packager', notice_files=[name]))
    inventory = dict(schema=1, target='darwin', artifact_sha256=sha(args.work/'elab'),
                     distribution_audit='pending', applicability='review-required',
                     scope='Explicit supplied notices bound to this artifact; no new rights clearance',
                     files=files, components=components)
    (notices/'INVENTORY.json').write_text(json.dumps(inventory, indent=2)+'\n')
    run([sys.executable, '-B', 'scripts/package_app.py', '--binary', str(args.work/'elab'),
         '--output', str(args.output/'ELab.app'), '--version', args.version, '--build-version', str(args.build),
         '--notice-bundle', str(notices)], args.work, 'package', env=env)
    for name in GUIDES:
        shutil.copyfile(ROOT/'docs'/(name+'.md'), args.output/(name+'.md'))
    for label, binary in [('native', args.output/'ELab.app/Contents/MacOS/ELab'),
                          ('agent', args.output/'elab-agent'), ('controller', args.work/'elab-conformance')]:
        info = subprocess.check_output(['go', 'version', '-m', str(binary)], text=True)
        check('vcs.revision='+source in info and 'vcs.modified=false' in info, 'build must match clean source')
        (args.work/(label+'-build-info.txt')).write_text(info)
    (args.output/'README.md').write_text('ELab '+args.version+' /'+str(args.build)+
        '\n\nUnsigned/ad-hoc candidate. Matching app/agent and curated examples; see included guides.\n'
        'Qualification is separate. Native OS-input, participant, clean-account and public signing gates remain distinct.\n')
    data = dict(schema=1, version=args.version, build=args.build, source_commit=source,
                app='ELab.app', agent='elab-agent', files=tree(args.output))
    (args.output/MANIFEST).write_text(json.dumps(data, indent=2)+'\n')
    (args.work/'assembly.json').write_text(json.dumps(dict(passed=True, package=str(args.output),
        controller=str(args.work/'elab-conformance'), source_commit=source), indent=2)+'\n')


def qualify(args):
    separate(args.package, args.artifacts)
    package = read_package(args.package)
    exclusive(args.artifacts)
    args._owned = args.artifacts
    agent = args.package/package['agent']
    client = AcceptanceClient(agent)
    examples = []
    try:
        parts = client.ask('parts.list')
        commands = client.ask('capabilities.list')['capabilities']
        catalog = client.ask('examples.list')
        example_dir = args.package/'Examples'
        paths = sorted(example_dir.glob('*.elab'))
        check({p.stem for p in paths} == {e['id'] for e in catalog}, 'curated examples do not match source IDs')
        for path in paths:
            expected = json.loads(path.read_text())['graph']
            client.ask('project.load', dict(path=str(path), discard_changes=True), mutate=True)
            state = client.ask('simulation.wait')
            check(state['paused'] and not state['document']['dirty'], 'example must reopen paused/clean')
            check(dict(components=state['components'], wires=state['wires']) == expected, 'saved graph differs')
            client.ask('example.load', dict(id=path.stem, discard_changes=True), mutate=True)
            client.ask('simulation.pause', mutate=True)
            state = client.ask('simulation.wait')
            check(dict(components=state['components'], wires=state['wires']) == expected,
                  'curated example differs from current built-in graph: '+path.stem)
            examples.append(path.stem)
    finally:
        client.close()
    runtime = None
    if args.runtime:
        check(args.controller is not None, '--runtime requires matching --controller')
        controller = args.controller.resolve(strict=True)
        info = subprocess.check_output(['go', 'version', '-m', str(controller)], text=True)
        check('vcs.revision='+package['source_commit'] in info and 'vcs.modified=false' in info,
              'runtime controller does not match clean package source')
        runtime = runtime_isolation.run(argparse.Namespace(app=args.package/package['app'],
            app_executable_sha256=package['files'][package['app']+'/Contents/MacOS/ELab']['sha256'],
            agent_binary=agent, conformance_binary=controller, artifacts=args.artifacts/'runtime',
            protect_geometry=True))
        check(runtime['passed'], 'runtime qualification failed')
    check(read_package(args.package) == package, 'qualification modified package')
    result = dict(passed=True, manifest_sha256=sha(args.package/MANIFEST), source_commit=package['source_commit'],
                  examples=examples, component_kinds=len(parts), commands=len(commands),
                  agent_exit_code=client.process.returncode, runtime_checked=runtime is not None)
    (args.artifacts/'qualification.json').write_text(json.dumps(result, indent=2)+'\n')


def archive(args):
    separate(args.package, args.artifacts)
    package = read_package(args.package)
    qualified = json.loads(args.qualification.read_text())
    check(qualified.get('passed') is True and qualified.get('manifest_sha256') == sha(args.package/MANIFEST)
          and qualified.get('source_commit') == package['source_commit'], 'qualification does not match exact package')
    check(not os.path.lexists(args.archive), 'archive already exists')
    check(args.package.resolve() not in args.archive.resolve().parents, 'archive must be outside package')
    exclusive(args.artifacts)
    args._owned = args.artifacts
    args.archive.parent.mkdir(parents=True, exist_ok=True)
    run(['ditto', '-c', '-k', '--sequesterRsrc', '--keepParent', str(args.package), str(args.archive)], args.artifacts, 'archive')
    import zipfile
    with zipfile.ZipFile(args.archive) as zipped:
        check(zipped.testzip() is None, 'archive CRC failed')
        expected = {args.package.name+'/'+name for name in package['files']} | {args.package.name+'/'+MANIFEST}
        actual = [n for n in zipped.namelist() if not n.endswith('/') and not n.startswith('__MACOSX/')]
        check(len(actual) == len(set(actual)) and set(actual) == expected, 'archive payload membership differs')
        for name, info in package['files'].items():
            member = zipped.getinfo(args.package.name+'/'+name)
            check(hashlib.sha256(zipped.read(member)).hexdigest() == info['sha256'] and stat.S_IMODE(member.external_attr >> 16) == info['mode'], 'archive bytes/mode differ')
        check(zipped.read(args.package.name+'/'+MANIFEST) == (args.package/MANIFEST).read_bytes(), 'archive manifest differs')
    extracted = args.artifacts/'extracted'
    extracted.mkdir()
    run(['ditto', '-x', '-k', str(args.archive), str(extracted)], args.artifacts, 'extract')
    check(read_package(extracted/args.package.name) == package, 'OS extraction differs')
    for label, app in [('source', args.package/'ELab.app'), ('extracted', extracted/args.package.name/'ELab.app')]:
        run(['codesign', '--verify', '--deep', '--strict', str(app)], args.artifacts, 'seal-'+label)
    result = dict(passed=True, archive=str(args.archive), bytes=args.archive.stat().st_size,
                  sha256=sha(args.archive), payload_files=len(expected), manifest_sha256=sha(args.package/MANIFEST))
    (args.artifacts/'archive.json').write_text(json.dumps(result, indent=2)+'\n')


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='action', required=True)
    p = sub.add_parser('assemble')
    for name in ['output', 'work', 'examples', 'licenses']:
        p.add_argument('--'+name, type=lambda s: Path(s).absolute(), required=True)
    p.add_argument('--version', required=True)
    p.add_argument('--build', type=int, required=True)
    p = sub.add_parser('check')
    p.add_argument('--package', type=lambda s: Path(s).absolute(), required=True)
    p.add_argument('--artifacts', type=lambda s: Path(s).absolute(), required=True)
    p.add_argument('--runtime', action='store_true')
    p.add_argument('--controller', type=Path)
    p = sub.add_parser('archive')
    for name in ['package', 'archive', 'artifacts', 'qualification']:
        p.add_argument('--'+name, type=lambda s: Path(s).absolute(), required=True)
    args = parser.parse_args(argv)
    if sys.platform != 'darwin':
        parser.error('unsigned native release qualification requires macOS')
    try:
        with quality.termination_signals():
            {'assemble': assemble, 'check': qualify, 'archive': archive}[args.action](args)
    except (Exception, KeyboardInterrupt) as error:
        owned = getattr(args, '_owned', None)
        if owned is not None:
            try:
                with (owned/'failure.json').open('x') as stream:
                    json.dump(dict(passed=False, action=args.action, error_type=type(error).__name__, error=str(error)), stream, indent=2)
            except OSError as write_error:
                print('Could not record release failure: '+str(write_error), file=sys.stderr)
        if isinstance(error, KeyboardInterrupt):
            return 128+getattr(error, 'signum', 2)
        if not isinstance(error, (ValueError, OSError, subprocess.SubprocessError)):
            raise
        parser.error(str(error))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
