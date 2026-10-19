import gzip,hashlib,io,json,shutil,sys,tarfile
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');base=Path(__file__).parent;out=base/'observations'
destination=root/'research/ui-audit/first-version/native-pause-161';destination.mkdir()
sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
sha=lambda path:hashlib.sha256(path.read_bytes()).hexdigest()

def compressed(source,target):
    with target.open('wb') as f:
        with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:z.write(source.read_bytes())
    assert gzip.decompress(target.read_bytes())==source.read_bytes()

report=json.loads((out/'report.json').read_text());assert report['passed'] and report['normal_closure'] and len(report['cycles'])==6
assert 'completion_failure' not in report and 'error' not in report
for name in ['setup.py','observe.py','finish.py','references.py','independent-references.json','identity.json',
             'observer-output.txt','focused-source-tests.txt','focused-source-tests-sandbox.txt']:
    shutil.copyfile(base/name,destination/name)
for path in out.iterdir():
    if path.suffix=='.elab' or path.name in ['cleanup.json','fresh-agent-reopen.json.gz','native-host.txt']:
        shutil.copyfile(path,destination/path.name)
    elif path.name.endswith('-completion.json'):
        compressed(path,destination/(path.name+'.gz'))
compressed(out/'report.json',destination/'native-session.json.gz')
captures=[]
with (destination/'ui-captures.tar.gz').open('wb') as f:
    with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:
        with tarfile.open(fileobj=z,mode='w') as tar:
            for path in sorted(out.iterdir()):
                if not path.is_file() or not(path.name.endswith('.ax.txt') or path.suffix in ['.jpg','.png']):continue
                payload=path.read_bytes();actual=path.stem+'.jpg' if payload[:2]==b'\xff\xd8' else path.name
                info=tarfile.TarInfo(actual);info.size=len(payload);info.mode=0o644;tar.addfile(info,io.BytesIO(payload))
                captures.append(dict(original_name=path.name,archive_name=actual,bytes=len(payload),sha256=sha(path)))
with tarfile.open(destination/'ui-captures.tar.gz') as tar:
    for row in captures:assert hashlib.sha256(tar.extractfile(row['archive_name']).read()).hexdigest()==row['sha256']
(destination/'ui-capture-index.json').write_text(json.dumps(dict(scope='Unmodified CUA images/accessibility observations; JPEG naming follows bytes. Endpoint credentials and executable backups excluded.',captures=captures),indent=2)+'\n')
for suffix in ['.jpg','.ax.txt']:shutil.copyfile(out/('cycle-6-paused'+suffix),destination/('cycle-6-paused'+suffix))
(destination/'cycles.json').write_text(json.dumps(dict(passed=True,cycles=report['cycles'],single_clicks_per_cycle=2,
    stable_paused_observation_seconds=.12,historical_cause_resolved=False,
    scope='Six bounded native cycles with explicit completion and full reference parity; original earlier incomplete observation remains unexplained. These observations do not measure visible p95 latency or prove absence of every intermittent input issue.'),indent=2)+'\n')
tests=(base/'focused-source-tests.txt').read_text();assert 'github.com/dshills/elab/desktop\t2.157s' in tests and '\nok ' in tests
(destination/'source-tests.json').write_text(json.dumps(dict(passed=True,command=['go','test','-race','-tags=offscreen','-count=1','-run','^(TestRenderedPauseAcceptsLiveSamplesWithoutInterveningCommands|TestMountedButtonOwnerPauseAcceptsLiveRepaintDuringHeldPress|TestRenderedSimulationButtonRefusesAgentDocumentReplacement)$','./...'],working_directory=str(root/'desktop'),exit_code=0,package_test_seconds=2.157,
    sandbox_attempt='Failed before tests because Go build-cache reads were denied; preserved separately.',
    warnings='Existing AppKit accessibility deprecations and duplicate libobjc link warning.',
    source_sha256={name:sha(root/name) for name in ['desktop/main.go','desktop/simulation_button_context_test.go','desktop/button_context_offscreen_test.go']}),indent=2)+'\n')
canonical=root/'build/ELab.app/Contents/MacOS/ELab';assert sha(canonical)=='914c1ba7547d9d1608cc1c8a0ff1bdeca42918e9343108b712e933c43e5d5b7c'
(destination/'preservation.json').write_text(json.dumps(dict(passed=True,current_review=release,canonical_native_sha256=sha(canonical),canonical_promoted=False),indent=2)+'\n')
shutil.copyfile(base/'package.py',destination/'package.py')
print('Packaged six-cycle reproduction, observed completions, source race checks, save/reopens, normal closure and preservation')
