import gzip, hashlib, io, json, shutil, sys, tarfile
from pathlib import Path

root = Path('/Users/dshills/Development/projects/ELab')
base = root / 'build/native-flasher-161-retry'
destination = root / 'research/ui-audit/first-version/native-flasher-161'
destination.mkdir()
sys.path.insert(0, str(root / 'scripts'))
from wiring_replay import validate_release

release = validate_release(root, root / 'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
report = json.loads((base / 'observations/report.json').read_text())
assert report['passed'] and report['normal_closure']
cleanup = json.loads((base / 'observations/cleanup.json').read_text())
assert cleanup['process_absent'] and cleanup['fresh_agent_reopen_passed']
canonical = root / 'build/ELab.app/Contents/MacOS/ELab'
assert sha(canonical) == '914c1ba7547d9d1608cc1c8a0ff1bdeca42918e9343108b712e933c43e5d5b7c'

def compressed(source, target):
    with target.open('wb') as f:
        with gzip.GzipFile(filename='', mode='wb', fileobj=f, mtime=0) as z:
            z.write(source.read_bytes())
    assert gzip.decompress(target.read_bytes()) == source.read_bytes()

for label, source in [('retry', base), ('first-attempt', root / 'build/native-flasher-161')]:
    target = destination / label
    target.mkdir()
    for name in ['identity.json', 'observe.py', 'setup.py', 'observer-output.txt', 'preflight.py', 'preflight.json', 'preflight-output.txt',
                 'finish.py', 'finish-first-attempt.py', 'finish-first-attempt.txt', 'finish-second-attempt.py', 'finish-second-attempt.txt']:
        if (source / name).is_file():
            shutil.copyfile(source / name, target / name)
    observations = source / 'observations'
    compressed(observations / 'report.json', target / 'native-session.json.gz')
    for name in ['cleanup.json', 'native-host.txt', 'cursor-10.json', 'cursor-20.json',
                 'Native timed flasher.elab', 'Agent timed flasher.elab', 'fresh-agent-reopen.json.gz']:
        if (observations / name).is_file():
            shutil.copyfile(observations / name, target / name)
    captures = []
    with (target / 'ui-captures.tar.gz').open('wb') as f:
        with gzip.GzipFile(filename='', mode='wb', fileobj=f, mtime=0) as gz:
            with tarfile.open(fileobj=gz, mode='w') as tar:
                for path in sorted(observations.iterdir()):
                    if not path.is_file() or not (path.name.endswith('.ax.txt') or path.suffix in ['.jpg', '.png']):
                        continue
                    payload = path.read_bytes()
                    actual = path.stem + '.jpg' if payload[:2] == b'\xff\xd8' else path.name
                    info = tarfile.TarInfo(actual)
                    info.size = len(payload)
                    info.mode = 0o644
                    tar.addfile(info, io.BytesIO(payload))
                    captures.append(dict(original_name=path.name, archive_name=actual, bytes=len(payload), sha256=sha(path)))
    with tarfile.open(target / 'ui-captures.tar.gz') as tar:
        for item in captures:
            assert hashlib.sha256(tar.extractfile(item['archive_name']).read()).hexdigest() == item['sha256']
    (target / 'ui-capture-index.json').write_text(json.dumps(dict(
        scope='Unmodified CUA image and accessibility observations; JPEG bytes use .jpg in the archive, including first-attempt files originally misnamed .png. No resizing or image editing. Endpoint/transport credentials and executable backups are excluded.',
        captures=captures), indent=2) + '\n')

for name in ['timing-10', 'edited-C2', 'timing-20']:
    for suffix in ['.jpg', '.ax.txt']:
        shutil.copyfile(base / 'observations' / (name + suffix), destination / (name + suffix))

periods = report['periods']
metrics = dict(passed=True, thresholds=dict(led_rising_current_a=.001, led_low_max_a=.00001,
    led_high_min_a=.002, kcl_residual_max_a=1e-9, absolute_power_residual_max_w=1e-8),
    periods=periods, capacitance_period_ratio=report['capacitance_period_ratio'],
    scope='Independent time-scaling and signal/conservation invariants supplement same-solver agent parity. Edges are sampled crossings bracketed within 5 ms; this is not hardware or closed-form absolute-period validation.')
for label, stage in [('10', 'paused-10'), ('20', 'paused-20')]:
    s = next(e['state'] for e in report['events'] if e['name'] == stage)
    metrics[label + '_sample_time_s'] = s['transient']['sample']['time_s']
    metrics[label + '_kcl_residual_a'] = s['mixed']['kcl_residual_a']
    metrics[label + '_power_residual_w'] = s['mixed']['power_residual_w']
    timing = next(e['state'] for e in report['events'] if e['name'] == 'timing-' + label)
    metrics[label + '_cursor_measurement'] = {k: timing['scope'][k] for k in ['a', 'b', 'delta_s', 'interval_hz', 'readings']}
(destination / 'measurements.json').write_text(json.dumps(metrics, indent=2) + '\n')
(destination / 'stages.json').write_text(json.dumps(dict(passed=True,
    stage_names=[e['name'] for e in report['events']],
    reference_stages=list(report['reference']), clock_10_s=report['running-10-clock-times'],
    clock_20_s=report['running-20-clock-times']), indent=2) + '\n')
(destination / 'preservation.json').write_text(json.dumps(dict(passed=True,
    canonical_native_sha256=sha(canonical), current_review=release,
    custom_stopped_bundle_restored=cleanup['stopped_bundle_restored'], canonical_promoted=False), indent=2) + '\n')
shutil.copyfile(base / 'package.py', destination / 'package.py')
print('Packaged raw journals, unmodified native observations, failures, reopening and release preservation. Add narrative/checkpoint before sealing evidence hashes.')
