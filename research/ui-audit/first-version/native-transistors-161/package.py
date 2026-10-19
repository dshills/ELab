import gzip, hashlib, io, json, shutil, sys, tarfile
from pathlib import Path

root = Path('/Users/dshills/Development/projects/ELab')
base = Path(__file__).parent
destination = root/'research/ui-audit/first-version/native-transistors-161'
destination.mkdir()
sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
release = validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
reports = {}

def compressed(source,target):
    with target.open('wb') as f:
        with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:
            z.write(source.read_bytes())
    assert gzip.decompress(target.read_bytes()) == source.read_bytes()

for label,source in [('first-attempt',root/'build/native-transistors-161'),('mosfet-retry',base)]:
    target = destination/label
    target.mkdir()
    for name in ['references.py','independent-references.json','observe.py','setup.py','identity.json',
                 'preflight.py','preflight.json.gz','preflight-correction.txt','observer-output.txt','finish.py','retry-protocol.txt']:
        if (source/name).is_file():
            shutil.copyfile(source/name,target/name)
    observations = source/'observations'
    reports[label] = json.loads((observations/'report.json').read_text())
    compressed(observations/'report.json',target/'native-session.json.gz')
    for path in observations.iterdir():
        if path.suffix == '.elab' or path.name in ['cleanup.json','native-host.txt','fresh-agent-reopen.json.gz']:
            shutil.copyfile(path,target/path.name)
        elif path.name.endswith('-completion.json'):
            compressed(path,target/(path.name+'.gz'))
    captures = []
    with (target/'ui-captures.tar.gz').open('wb') as f:
        with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:
            with tarfile.open(fileobj=z,mode='w') as tar:
                for path in sorted(observations.iterdir()):
                    if not path.is_file() or not (path.name.endswith('.ax.txt') or path.suffix in ['.jpg','.png']):
                        continue
                    payload = path.read_bytes()
                    actual = path.stem+'.jpg' if payload[:2] == b'\xff\xd8' else path.name
                    info = tarfile.TarInfo(actual)
                    info.size,info.mode = len(payload),0o644
                    tar.addfile(info,io.BytesIO(payload))
                    captures.append(dict(original_name=path.name,archive_name=actual,bytes=len(payload),sha256=sha(path)))
    with tarfile.open(target/'ui-captures.tar.gz') as tar:
        for item in captures:
            assert hashlib.sha256(tar.extractfile(item['archive_name']).read()).hexdigest() == item['sha256']
    (target/'ui-capture-index.json').write_text(json.dumps(dict(
        scope='Unmodified CUA accessibility/image observations. JPEG bytes retain .jpg; no resizing/re-encoding. Endpoint credentials and binary backups excluded.',
        captures=captures),indent=2)+'\n')

first,retry = reports['first-attempt'],reports['mosfet-retry']
assert not first['passed'] and first['checks']['npn-save']['passed']
assert retry['passed'] and retry['normal_closure']
for source,name in [(root/'build/native-transistors-161','npn-step-high'),(base,'nmos-paused'),(base,'pmos-step-low')]:
    for suffix in ['.jpg','.ax.txt']:
        shutil.copyfile(source/'observations'/(name+suffix),destination/(name+suffix))
checks = dict(passed=True,scope='Completed NPN sequence from first attempt plus completed MOSFET retry; original first overall failure remains false.',
    npn={name:value for name,value in first['checks'].items() if name.startswith('npn-')},
    mosfet=retry['checks'],first_overall_passed=first['passed'],first_error=first['error'],
    strict_1mv_pmos_drain_voltage_qualified=False,
    precision_limit='PMOS cutoff drain-node point reference differs by 3.218 mV in the retained native trace. Signed-current smoke uses the fixed existing 2 nA current bound and monotone LED/load-line voltage envelope; no strict 1 mV accuracy claim.')
(destination/'checks.json').write_text(json.dumps(checks,indent=2)+'\n')
canonical = root/'build/ELab.app/Contents/MacOS/ELab'
assert sha(canonical) == '914c1ba7547d9d1608cc1c8a0ff1bdeca42918e9343108b712e933c43e5d5b7c'
(destination/'preservation.json').write_text(json.dumps(dict(passed=True,current_review=release,
    canonical_native_sha256=sha(canonical),canonical_promoted=False,
    reference_sources_sha256={name:sha(root/name) for name in ['scripts/freeze-electrical-reference.py',
        'internal/conformance/testdata/electrical-reference.json','internal/conformance/mosfet_references.go']}),indent=2)+'\n')
shutil.copyfile(base/'package.py',destination/'package.py')
print('Packaged both attempts, bounded checks, raw observations, six fresh reopens, known precision gap and restoration evidence')
