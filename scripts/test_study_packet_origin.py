import copy
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import unittest
from unittest.mock import patch

import study_packet_origin as origin

ROOT = Path(__file__).resolve().parents[1]


class StudyPacketOriginTests(unittest.TestCase):
    def test_current_packet_retains_zero_participants_and_no_current_runtime_claim(self):
        result = subprocess.run([sys.executable, '-B', str(ROOT/'research/first-use-study/verify-packet.py')],
                                cwd=ROOT, capture_output=True, text=True, timeout=30, check=True)
        report = json.loads(result.stdout)
        self.assertTrue(report['packet_integrity_verified'])
        self.assertEqual(report['measured_participants'], 0)
        self.assertIsNone(report['study_gate_passed'])
        self.assertFalse(report['current_runtime_qualified'])
        self.assertFalse(report['study_run'])

    def test_altered_receipt_release_and_ledger_are_refused_before_acceptance(self):
        folder = ROOT/'research/first-use-study'
        checkpoint = json.loads((folder/'checkpoint.json').read_text())
        identity = json.loads((folder/'fixtures/preflight.json').read_text())['release']
        receipt = (ROOT/'research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json').read_bytes()
        for field in ['receipt_sha256', 'source_commit', 'agent_sha256', 'ledger_sha256']:
            with self.subTest(field=field):
                damaged = copy.deepcopy(identity)
                damaged[field] = '0'*len(damaged[field])
                with self.assertRaises(ValueError):
                    origin.validate_origin(ROOT, checkpoint, damaged, receipt)
        with self.assertRaises(ValueError):
            origin.validate_origin(ROOT, checkpoint, identity, receipt+b' ')
        with patch.object(origin, 'git_blob', return_value=b'{"source_sha256":{}}'):
            with self.assertRaisesRegex(ValueError, 'ledger identity'):
                origin.validate_origin(ROOT, checkpoint, identity, receipt)

    def test_archived_source_drift_is_not_hidden_by_a_matching_ledger_hash(self):
        source = 'a'*40
        original = b'package main\n'
        ledger = json.dumps({'source_sha256': {'desktop/main.go': hashlib.sha256(original).hexdigest()}}).encode()
        receipt = dict(source_commit=source, clean_build_vcs=True, version='1.0.0', build='1',
                       agent_sha256='b'*64, native_sha256='c'*64)
        raw = json.dumps(receipt).encode()
        checkpoint = dict(review_receipt_sha256=hashlib.sha256(raw).hexdigest(), build='1.0.0 /1',
                          agent_sha256='b'*64, native_sha256='c'*64)
        identity = dict(source_commit=source, release='1.0.0 /1', receipt_sha256=checkpoint['review_receipt_sha256'],
                        agent_sha256='b'*64, ledger_sha256=hashlib.sha256(ledger).hexdigest(),
                        production_source_files_verified=1)
        with patch.object(origin, 'git_blob', side_effect=lambda r, s, n: ledger if n==origin.LEDGER else b'drift'):
            with self.assertRaisesRegex(ValueError, 'source hash differs'):
                origin.validate_origin(ROOT, checkpoint, identity, raw)
