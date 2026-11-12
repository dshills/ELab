"""Build and inspect standalone ELab web-host release ZIPs."""
import hashlib
import platform
import re
from pathlib import Path
import shutil
import stat
import subprocess
import zipfile

from package_web import copy_licenses

ROOT = Path(__file__).resolve().parents[1]
TARGETS = ('linux-amd64', 'darwin-arm64', 'darwin-amd64', 'windows-amd64')
MAX_FILE_BYTES = 128 << 20


def native_target():
    system = {'Linux': 'linux', 'Darwin': 'darwin', 'Windows': 'windows'}.get(platform.system())
    arch = {'x86_64': 'amd64', 'amd64': 'amd64', 'arm64': 'arm64', 'aarch64': 'arm64'}.get(
        platform.machine().lower())
    return f'{system}-{arch}'


def executable_name(target):
    return 'elab.exe' if target.startswith('windows-') else 'elab'


def archive_name(version, target):
    return f"ELab-Web-{version}-{target.replace('darwin-', 'macos-')}.zip"


def sha256(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def build_archive(target, version, build, output, temporary, env):
    if target not in TARGETS:
        raise ValueError('unsupported release target')
    work = temporary/target
    work.mkdir()
    binary = work/executable_name(target)
    os_name, arch = target.split('-')
    target_env = dict(env, GOOS=os_name, GOARCH=arch, CGO_ENABLED='0', GOAMD64='v1', GOARM64='v8.0')
    subprocess.run(['go', 'build', '-trimpath', '-buildvcs=true', '-ldflags',
                    f'-X main.releaseVersion={version} -X main.releaseBuild={build}',
                    '-o', str(binary), '.'], env=target_env, cwd=ROOT, check=True, timeout=900)
    build_info = subprocess.check_output(['go', 'version', '-m', binary.name],
                                         cwd=work, env=env, text=True, timeout=30)
    copy_licenses(work, build_info)
    (work/'BUILD.txt').write_text(build_info, encoding='utf-8')
    launch = r'.\elab.exe' if os_name == 'windows' else './elab'
    (work/'START-HERE.txt').write_text(
        f'ELab Web {version} (build {build})\n\n'
        'Extract the whole ZIP into a folder and open a terminal in that folder.\n'
        f'Run {launch}, then open http://localhost:8788 in your browser.\n'
        'Leave the terminal open while working. Press Ctrl+C there to save and stop ELab.\n'
        'Go, Node and a network connection are not required.\n'
        'Projects are saved on this computer. Use Projects > Back up library before upgrading.\n',
        encoding='utf-8')
    archive = output/archive_name(version, target)
    files = {}
    with zipfile.ZipFile(archive, 'x', compression=zipfile.ZIP_DEFLATED) as zipped:
        for path in sorted(p for p in work.rglob('*') if p.is_file()):
            name = path.relative_to(work).as_posix()
            files[name] = sha256(path)
            entry = zipfile.ZipInfo.from_file(path, name)
            entry.create_system = 3
            entry.compress_type = zipfile.ZIP_DEFLATED
            entry.external_attr = (stat.S_IFREG | (0o755 if path == binary else 0o644)) << 16
            with path.open('rb') as source, zipped.open(entry, 'w') as destination:
                shutil.copyfileobj(source, destination)
    return {'name': archive.name, 'sha256': sha256(archive), 'files': files}


def inspect_archive(archive, entry, target, destination=None):
    """Verify the exact inventory before extracting ordinary files only."""
    if archive.is_symlink() or sha256(archive) != entry['sha256']:
        raise ValueError('archive bytes differ: '+archive.name)
    files = entry['files']
    if (not {executable_name(target), 'BUILD.txt', 'START-HERE.txt', 'Licenses/Go.txt',
             'Licenses/README.txt'} <= set(files) or len(files) > 100):
        raise ValueError('archive is missing its host or documentation')
    with zipfile.ZipFile(archive) as zipped:
        entries = zipped.infolist()
        if len(entries) != len(files) or {item.filename for item in entries} != set(files):
            raise ValueError('archive inventory differs: '+archive.name)
        if len({item.filename.casefold() for item in entries}) != len(entries):
            raise ValueError('archive paths collide on a case-insensitive filesystem')
        for item in entries:
            name = item.filename
            parts = name.split('/')
            # Only portable file names are extracted, including on Windows.
            mode = item.external_attr >> 16
            top_level = name in (executable_name(target), 'BUILD.txt', 'START-HERE.txt')
            license_file = parts[0] == 'Licenses' and len(parts) > 1 and all(
                re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._@-]*', p)
                and not p.endswith('.') and not re.fullmatch(
                    r'(?:CON|PRN|AUX|NUL|COM[1-9]|LPT[1-9])(?:\..*)?', p, re.I)
                for p in parts)
            if (not (top_level or license_file) or not stat.S_ISREG(mode)
                    or not 0 < item.file_size <= MAX_FILE_BYTES
                    or (name == executable_name(target) and not mode & stat.S_IXUSR)):
                raise ValueError('invalid release archive entry: '+name)
        for item in entries:
            name = item.filename
            with zipped.open(item) as stream:
                payload = stream.read(MAX_FILE_BYTES+1)
            if len(payload) != item.file_size or hashlib.sha256(payload).hexdigest() != files[name]:
                raise ValueError('archive payload differs: '+name)
            if destination is not None:
                path = destination.joinpath(*name.split('/'))
                path.parent.mkdir(parents=True, exist_ok=True)
                with path.open('xb') as stream:
                    stream.write(payload)
                path.chmod(0o755 if name == executable_name(target) else 0o644)
    if destination is not None:
        return destination/executable_name(target)
