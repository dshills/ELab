"""Verify the retained source proof without launching any application."""
import gzip
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import stat

ROOT = Path(__file__).resolve().parents[3]
BASE = Path(__file__).parent


def read(name):
    path = BASE/name
    return json.loads(gzip.decompress(path.read_bytes()) if path.suffix == '.gz' else path.read_text())


def main():
    checkpoint = read('checkpoint.json')
    for name, expected in checkpoint['evidence_sha256'].items():
        assert hashlib.sha256((BASE/name).read_bytes()).hexdigest() == expected, name
    identity = read('development-identity.json')
    for name, expected in identity['source_sha256'].items():
        assert hashlib.sha256((ROOT/name).read_bytes()).hexdigest() == expected, name
    quality = read('quality.json')
    assert quality['passed'] and quality['exit_code'] == 0
    assert 'FAIL' not in (BASE/'quality.txt').read_text()
    assert 'FAIL' in (BASE/'before-regression.txt').read_text()
    spec = importlib.util.spec_from_file_location('producer', BASE/'reproduce.py')
    producer = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(producer)
    for folder, expected_pass in [('old-public',False),('fixed-public',True)]:
        report = read(folder+'/report.json.gz')
        assert report['passed'] == expected_pass and 'error' not in report
        assert set(report['cases']) == {'nmos','pmos'}
        for kind, case in report['cases'].items():
            traces = {trace['probe']['id']:trace['points'] for trace in case['live']['scope']['traces']}
            count = len(traces[producer.PROBES[0]])
            assert all(len(traces[p]) == count for p in producer.PROBES)
            rows = []
            for index in range(count):
                at = traces[producer.PROBES[0]][index]['time_s']
                assert all(traces[p][index]['time_s'] == at for p in producer.PROBES)
                rows.append(dict(time_s=at, values=[traces[p][index]['value'] for p in producer.PROBES]))
            assert producer.check_rows(rows,kind) == case['live_check']
            captured = [row for page in case['pages'] for row in page['rows']]
            assert producer.check_rows(captured,kind) == case['captured_check']
            assert case['passed'] == (expected_pass or kind == 'nmos')
            path = BASE/folder/(kind+'.elab')
            assert hashlib.sha256(path.read_bytes()).hexdigest() == case['saved_sha256']
            document = json.loads(path.read_text())
            assert document['graph']['components'] == case['live']['components']
            assert document['graph']['wires'] == case['live']['wires']
            entry = document['recordings']['entries'][0]['data']
            assert entry['metadata'] == case['capture']['metadata'] and entry['rows'] == captured
            assert case['reopened']['paused'] and not case['reopened']['document']['dirty']
            assert case['reopened']['components'] == case['live']['components']
            assert case['reopened']['wires'] == case['live']['wires']
            for index, page in enumerate(case['pages']):
                assert page['offset'] == index*200 and page['total'] == len(captured)
                assert [probe['id'] for probe in page['probes']] == producer.PROBES
        if expected_pass:
            assert report['agent_sha256'] == identity['agent_sha256']
    preservation = read('preservation.json')
    for name, expected in preservation['review_manifests'].items():
        folder = ROOT/'build'/name
        manifest = folder/'RELEASE_MANIFEST.json'
        assert hashlib.sha256(manifest.read_bytes()).hexdigest() == expected['manifest_sha256']
        files = json.loads(manifest.read_text())['files']
        assert len(files) == expected['entries']
        assert {str(p.relative_to(folder)) for p in folder.rglob('*') if p.is_file()} == set(files)|{'RELEASE_MANIFEST.json'}
        for member, identity in files.items():
            path = folder/member
            assert not path.is_symlink() and hashlib.sha256(path.read_bytes()).hexdigest() == identity['sha256']
            assert oct(stat.S_IMODE(path.stat().st_mode)) == identity['mode']
    assert hashlib.sha256((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes()).hexdigest() == preservation['canonical_native_sha256']
    for name, key in [('third_party/vellum/sdk.lock.json','sdk_lock_sha256'),('research/13-native-command-inventory.json','native_ledger_sha256')]:
        assert hashlib.sha256((ROOT/name).read_bytes()).hexdigest() == preservation[key]
    for name, expected in json.loads((ROOT/'research/13-native-command-inventory.json').read_text())['source_sha256'].items():
        assert hashlib.sha256((ROOT/name).read_bytes()).hexdigest() == expected
    for target in re.findall(r'!?\[[^\]]*\]\(([^)]+)\)', (BASE/'README.md').read_text()):
        assert (BASE/target.strip('<>')).exists(),target
    print(json.dumps(dict(passed=True, old_pmos_precision_passed=False, repaired_live_and_capture_passed=True,
                          preserved_reviews=len(preservation['review_manifests']), quality_s=quality['duration_s'],
                          clean_replacement_qualified=False, evidence_files=len(checkpoint['evidence_sha256']))))


if __name__ == '__main__':
    main()
