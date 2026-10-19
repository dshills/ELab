"""Compare MOSFET cutoff voltage to an independent scalar reference via public APIs."""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import sys
import time

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT/'scripts'))
from demo_session import connect_demo_client

spec = importlib.util.spec_from_file_location('helper', ROOT/'scripts/persistence-demo.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
PROBES = ['v:M1', 'v:M1.gate-source', 'i:M1.drain', 'i:D1', 'v:VIN.positive', 'i:M1.gate']


def reference():
    vt = 1.380649e-23*300.15/1.602176634e-19
    current = 1e-9
    voltage = 5-1000*current-(2*vt*math.log1p(current/1e-20)+current)
    omitted = current*math.exp((-voltage+current)/(1.7*vt))
    assert omitted < 1e-40
    return dict(drain_magnitude_v=voltage, current_a=current,
                body_reverse_exponential_bound_a=omitted, voltage_tolerance_v=.001)


def check_rows(rows, kind):
    sign, off = (1, 0) if kind == 'nmos' else (-1, 5)
    target = reference()
    maximum, selected, inputs = 0., [], set()
    assert len(rows) >= 503
    for row in rows:
        assert len(row['values']) == 6 and all(math.isfinite(v) for v in row['values'])
        drain, gate, current, led, source, gate_current = row['values']
        nominal = 0 if abs(source) < 1e-9 else 5
        assert abs(source-nominal) < 1e-9
        inputs.add(nominal)
        assert gate_current == 0 and abs(current-sign*led) < 1e-9
        if nominal != off:
            continue
        selected.append(row)
        assert abs(current-sign*1e-9) < 1e-12 and abs(led-1e-9) < 1e-10
        maximum = max(maximum, abs(drain-sign*target['drain_magnitude_v']))
    assert inputs == {0, 5} and len(selected) >= 2
    for at in [.2, .5]:
        corners = [row for row in rows if abs(row['time_s']-at) < 1e-14]
        assert {round(row['values'][4]) for row in corners} == {0, 5}
    return dict(passed=maximum <= .001, total_rows=len(rows), cutoff_rows=len(selected),
                maximum_drain_error_v=maximum, reference=target)


def run_case(client, kind, assets):
    client.ask('example.load', dict(id=kind, discard_changes=True), True)
    client.ask('simulation.pause', {}, True)
    client.ask('simulation.restart', {}, True)
    client.ask('simulation.wait')
    client.ask('scope.probes.set', dict(ids=PROBES), True)
    client.ask('simulation.step', dict(duration_s=.2), True)
    client.ask('simulation.step', dict(duration_s=.3), True)
    live = client.ask('simulation.wait')
    traces = {trace['probe']['id']: trace['points'] for trace in live['scope']['traces']}
    count = len(traces[PROBES[0]])
    assert all(len(traces[p]) == count for p in PROBES)
    rows = []
    for index in range(count):
        at = traces[PROBES[0]][index]['time_s']
        assert all(traces[p][index]['time_s'] == at for p in PROBES)
        rows.append(dict(time_s=at, values=[traces[p][index]['value'] for p in PROBES]))
    live_check = check_rows(rows, kind)
    client.ask('simulation.verify', dict(duration_s=.5, probe_ids=PROBES), True)
    state = client.ask('project.inspect')
    identity = state['last_operation']
    deadline = time.monotonic()+20
    while True:
        capture = client.ask('operation.inspect', dict(id=identity))
        if capture['operation']['status'] == 'completed':
            break
        assert capture['operation']['status'] not in ['failed','canceled'] and time.monotonic() < deadline
        time.sleep(.01)
    after = client.ask('simulation.wait')
    for key in ['components','wires','paused','run_id','transient','mixed','scope','npn','result']:
        assert after.get(key) == live.get(key), key
    pages, captured = [], []
    offset = 0
    while True:
        page = client.ask('operation.samples', dict(id=identity, offset=offset, limit=200))
        pages.append(page)
        assert [p['id'] for p in page['probes']] == PROBES
        captured.extend(page['rows'])
        offset = page['next_offset']
        if offset == page['total']:
            break
    captured_check = check_rows(captured, kind)
    assert captured == rows
    path = assets/(kind+'.elab')
    client.ask('project.save', dict(path=str(path)), True)
    fresh = helper.Client(command_preconditions=True, agent_binary=AGENT)
    try:
        fresh.ask('project.load', dict(path=str(path)), True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['paused'] and not reopened['document']['dirty']
        metadata = fresh.ask('operation.inspect', dict(id=reopened['last_operation']))['metadata']
        assert metadata == capture['metadata']
        document = json.loads(path.read_text())
        assert document['recordings']['entries'][0]['data']['rows'] == captured
    finally:
        fresh.close()
    return dict(passed=live_check['passed'] and captured_check['passed'], live=live, live_check=live_check,
                capture=capture, pages=pages, captured_check=captured_check, reopened=reopened,
                saved_sha256=hashlib.sha256(path.read_bytes()).hexdigest())


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent-binary', type=Path, required=True)
    parser.add_argument('--connect', help='Only an independently owned, fresh blank native endpoint')
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    AGENT = args.agent_binary.resolve()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    report = dict(passed=False, agent_sha256=hashlib.sha256(AGENT.read_bytes()).hexdigest(), cases={})
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True, agent_binary=AGENT)
    try:
        for kind in ['nmos','pmos']:
            report['cases'][kind] = run_case(client, kind, assets)
        report['passed'] = all(case['passed'] for case in report['cases'].values())
    except BaseException as error:
        report['error'] = type(error).__name__+': '+str(error)
        raise
    finally:
        (assets/'report.json').write_text(json.dumps(report,indent=2)+'\n')
        if args.connect:
            client.ask('project.inspect')
            client.ask('app.close', dict(discard_changes=True), True)
        client.close()
    print(json.dumps(dict(passed=report['passed'], cases={kind:dict(live=case['live_check'],captured=case['captured_check']) for kind,case in report['cases'].items()})))
    raise SystemExit(0 if report['passed'] else 1)
