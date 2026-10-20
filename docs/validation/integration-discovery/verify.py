"""Replay numerical-discovery evidence without starting an app or changing files."""
from pathlib import Path
import copy
import gzip
import hashlib
import importlib.util
import json
import sys

if sys.flags.optimize:
    raise SystemExit('Verification requires assertions; optimized Python is refused')

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]


def sha(data):
    return hashlib.sha256(data).hexdigest()


def data(relative):
    return gzip.decompress((HERE/'raw'/(relative+'.gz')).read_bytes())


def read(relative):
    return json.loads(data(relative))


def main():
    checkpoint = json.loads((HERE/'checkpoint.json').read_text())
    actual = {str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name != 'checkpoint.json'}
    assert actual == set(checkpoint['files'])
    for name, digest in checkpoint['files'].items():
        p = HERE/name
        assert not p.is_symlink() and p.resolve().is_relative_to(HERE) and sha(p.read_bytes()) == digest, name
    for name, entry in json.loads((HERE/'manifest.json').read_text()).items():
        stored = (HERE/name).read_bytes(); raw = gzip.decompress(stored)
        assert sha(stored) == entry['stored_sha256'] and sha(raw) == entry['original_sha256']
        assert len(raw) == entry['original_bytes']
    report = json.loads((HERE/'report.json').read_text())
    for relative, digest in report['dependencies'].items():
        assert sha((ROOT/relative).read_bytes()) == digest, relative
    quality = read('stage/full-quality.json')
    assert quality['exit_code'] == 0 and quality['sources_unchanged'] and quality['source_files'] == 658
    assert quality['command'] == './scripts/check.sh' and quality['environment_overrides'] == dict(GOPROXY='off')
    assert quality['source_sha256'] == read('stage/source-before-quality.json')
    for name, digest in quality['source_sha256'].items():
        assert sha((ROOT/name).read_bytes()) == digest, name
    receipt_path = ROOT/'research/ui-audit/first-version/geometry-action-review-checkpoint.json'
    assert sha(receipt_path.read_bytes()) == report['preserved_receipt_sha256']
    receipt = json.loads(receipt_path.read_text())
    old_sources = receipt['quality']['source_sha256']
    # The gate runner also froze three existing files in a separate tools module;
    # they are not application inputs or new production changes.
    extras = {'tools/baseline-admission/'+name for name in ('admission.go','admission_test.go','main.go')}
    assert set(quality['source_sha256'])-set(old_sources) == extras | {'workbench/integration_discovery_test.go'}
    new_sources = {p:digest for p,digest in quality['source_sha256'].items() if p not in extras}
    assert len(new_sources) == 655
    assert len(old_sources) == 654
    assert set(new_sources)-set(old_sources) == {'workbench/integration_discovery_test.go'}
    assert not set(old_sources)-set(new_sources)
    assert {p for p in old_sources if old_sources[p] != new_sources[p]} == {'workbench/value_capabilities.go'}
    assert sha(data('stage/value_capabilities-original.go.txt')) == old_sources['workbench/value_capabilities.go']
    assert sha(data('stage/value_capabilities-final.go.txt')) == new_sources['workbench/value_capabilities.go']
    assert sha(data('stage/integration_discovery_test.go.txt')) == new_sources['workbench/integration_discovery_test.go']
    old_log = data('stage/original-focused.log').decode()
    assert 'FAIL' in old_log
    for name in ('mixed-capacitors','network-capacitors','waveform-capacitors','single-rc','npn-phases','inductors'):
        assert '--- FAIL: TestIntegrationDiscoveryMatchesLiveEngineFamilies/'+name in old_log
    assert data('stage/fixed-focused.log').decode().startswith('ok')
    assert 'Ran 65 tests' in data('stage/full-quality.log').decode()
    for prefix, expected_good, expected_count in (('original-initial',False,3), ('original-final',True,6), ('fixed-final',True,6)):
        plan = read(prefix+'/plan.json')
        assert plan['client_sha256'] == report['dependencies'][plan['client_source']]
        binary = Path(plan['agent_path']).with_name('elab-agent')
        assert sha(binary.read_bytes()) == (report['development_agent_sha256'] if prefix=='fixed-final' else receipt['agent_sha256'])
        observation = read(prefix+'/report.json')
        assert observation['passed'] is expected_good and len(observation['observations']) == expected_count
        assert (observation['failure'] is None) is expected_good
        if not expected_good:
            assert 'not-found' in observation['failure'] and 'npn-pulse' not in [o['name'] for o in observation['observations']]
        assert len(observation['process_exits']) == 1
        outcome = observation['process_exits'][0]
        rows = [json.loads(line) for line in data(prefix+'/'+outcome['name']+'-protocol.jsonl').splitlines()]
        assert len(rows) == outcome['requests']*2
        pairs = [(rows[i]['request'], rows[i+1]['response']) for i in range(0,len(rows),2)]
        failures = []
        for i, (request,response) in enumerate(pairs):
            assert request['id'] == response['id'] == f'junction-{i+1}'
            if response.get('error'):
                failures.append((request,response))
        if expected_good:
            assert not failures and outcome['requests'] == 44
        else:
            assert len(failures) == 1 and failures[0][0]['params']['id'] == 'npn-pulse'
            assert failures[0][1]['error']['code'] == 'not-found' and outcome['requests'] == 27
        assert outcome['exit_code'] == 0 and pairs[-1][0]['command'] == 'app.close'
        final = pairs[-1][1]['result']
        assert final['closed'] and final['shutdown'] == outcome['shutdown']
        assert outcome['shutdown']['status'] == 'completed'
        assert all(outcome['shutdown'][key] for key in ('workers_joined','recovery_finalized','exit_ready'))
        assert not data(prefix+'/'+outcome['name']+'-stderr.txt')
        # Both filtered queries are bounded by two identical full snapshots.
        indices = [i for i,(request,_) in enumerate(pairs) if request['command']=='capabilities.list']
        assert len(indices) == 3 and indices == list(range(indices[0],indices[0]+3))
        assert pairs[indices[0]-1][0]['command'] == pairs[indices[-1]+1][0]['command'] == 'project.inspect'
        assert pairs[indices[0]-1][1]['result'] == pairs[indices[-1]+1][1]['result']
    assert 'FileExistsError' in data('stage/original-refused-collision.log').decode()
    assert not read('stage/comparison.json')['refused_output_collision_started_agent']
    assert read('original-final/ordinary.json') == read('fixed-final/ordinary.json')
    old, new = read('original-final/details.json'), read('fixed-final/details.json')
    engines = ['elab.mixed-mna-sdirk2.v1','elab.network-mna-sdirk2.v1','elab.ideal-rc-exact.v1',
               'elab.npn-switch-quasistatic.v1','elab.rlc-mna-radau5.v1','elab.network-waveforms-mna-sdirk2.v1']
    for command in ('component.place','component.set'):
        x,y = copy.deepcopy(old[command]),copy.deepcopy(new[command])
        old_text = x['capabilities'][0]['input_schema']['x-network-numerics'].pop('time_integration')
        new_text = y['capabilities'][0]['input_schema']['x-network-numerics'].pop('time_integration')
        assert 'retain backward Euler' in old_text and 'retain backward Euler' not in new_text
        assert 'historical' in new_text and 'SDIRK2' in new_text and all(engine in new_text for engine in engines)
        assert x == y, command
    sys.path.insert(0,str(ROOT/'scripts'))
    spec = importlib.util.spec_from_file_location('discovery_state_parity',ROOT/'scripts/first-version-parity.py')
    parity = importlib.util.module_from_spec(spec); spec.loader.exec_module(parity)
    for name, engine in zip(('flasher','rc-high-pass','rc','npn-pulse','rlc','waveform'), engines):
        for suffix in ('baseline','live'):
            left,right = (read(prefix+'/'+name+'-'+suffix+'.json') for prefix in ('original-final','fixed-final'))
            assert left['transient']['engine'] == right['transient']['engine'] == engine
            assert left['transient']['sample']['time_s'] == right['transient']['sample']['time_s'] == (0 if suffix=='baseline' else .001)
            for key in ('components','wires','transient_settings','display_preferences','source_sampling','source_warnings','scope_probes'):
                assert left.get(key) == right.get(key),(name,suffix,key)
            for key in ('result','mixed','transient','scope'):
                parity.compare(left.get(key),right.get(key),name+'.'+suffix+'.'+key)
    runtime = read('stage/development-runtime.json')
    assert not runtime['release_candidate'] and sha(Path(runtime['copied_agent']).read_bytes()) == runtime['agent_sha256']
    assert sha(Path(runtime['development_agent']).read_bytes()) == runtime['agent_sha256'] == report['development_agent_sha256']
    from package_review import members
    folder = Path(receipt['review_path']); actual = members(folder); actual.pop('RELEASE_MANIFEST.json')
    assert actual == json.loads((folder/'RELEASE_MANIFEST.json').read_text())['files'] and len(actual) == 418
    assert sha((ROOT/receipt['sdk']['archive']).read_bytes()) == receipt['sdk']['sha256']
    assert sha((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes()) == receipt['canonical']['native_sha256']
    assert not any(report[key] for key in ('solver_changed','native_input','new_release','owner_acceptance','full_mvp_complete'))
    print(json.dumps(dict(corrected_filtered_fields=2,live_engine_families=6,matched_physics_states=12,
                          retained_requests=115,normal_process_exits=3,source_files=655,
                          full_quality_exit=0,solver_changed=False,new_release=False)))


if __name__ == '__main__':
    main()
