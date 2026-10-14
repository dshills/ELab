#!/usr/bin/env python3
"""Exercise shared clean/dirty close admission in an owned session."""
import argparse
import importlib.util
import json
from pathlib import Path

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='owned validation session; closes it on success')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    client = helper.Client(args.connect, command_preconditions=True)
    try:
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        dirty = client.ask('simulation.wait')
        client.ask('app.close', {}, True, 'unsaved-changes')
        assert client.ask('project.inspect') == dirty
        generation = dirty['document']['generation']
        client.ask('project.clear', {'discard_changes': True}, True)
        before = client.ask('project.inspect')
        client.ask('app.close', {'discard_changes': True, 'expected_document_generation': generation}, True, 'conflict')
        assert client.ask('project.inspect') == before
        client.ask('example.reset', {}, True)
        client.ask('simulation.wait')
        closed = client.ask('app.close', {'discard_changes': True}, True)
        assert closed['closed'] and closed['status'] == 'closed'
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'checks': ['dirty close refuses without circuit/history/physics change',
                             'stale close generation cannot close replacement', 'explicit discard admits semantic close'],
                  'limits': ['public semantic command workflow; physical OS close-button/Cmd+Q delivery remains a separate check']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps(report))
    finally:
        client.close()


if __name__ == '__main__':
    main()
