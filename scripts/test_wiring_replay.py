"""Reject mixed source/release/producer identities before replaying an agent."""
import copy
import hashlib
import json
from pathlib import Path
import stat
import tempfile
import unittest
from unittest.mock import patch

import wiring_replay as replay


class WiringReleaseIdentityTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix='elab-wiring-identity-')
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.source = 'a'*40
        self.original = {'desktop/main.go': b'package main\n', 'workbench/model.go': b'package workbench\n',
                         'go.mod': b'module test\n', 'go.sum': b'checksums\n'}
        ledger = dict(source_sha256={'desktop/main.go': hashlib.sha256(self.original['desktop/main.go']).hexdigest()})
        self.original[replay.LEDGER] = (json.dumps(ledger)+'\n').encode()
        for name, data in self.original.items():
            path = self.root/name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
        self.pin = dict(source_commit='pinned-sdk')
        path = self.root/'third_party/vellum/sdk.lock.json'
        path.parent.mkdir(parents=True)
        path.write_text(json.dumps(self.pin))
        self.review = self.root/'build/review'
        self.files = {'elab-agent': b'agent', 'ELab.app/Contents/MacOS/ELab': b'native',
                      'Evidence/Agent-build-info.txt': ('vcs.revision='+self.source+'\nvcs.modified=false\n').encode()}
        for name, data in self.files.items():
            path = self.review/name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
        self.manifest = dict(source_record_commit=self.source, source_build_clean=True,
                             files={name: dict(sha256=replay.digest(self.review/name),
                                              mode=oct(stat.S_IMODE((self.review/name).stat().st_mode))) for name in self.files})
        (self.review/'RELEASE_MANIFEST.json').write_text(json.dumps(self.manifest))
        self.receipt = dict(source_commit=self.source, clean_build_vcs=True, sdk=self.pin,
                            review_path=str(self.review), manifest_file_entries=len(self.files),
                            version='0.1.0', build='1', agent_sha256=replay.digest(self.review/'elab-agent'),
                            native_sha256=replay.digest(self.review/'ELab.app/Contents/MacOS/ELab'))
        self.path = self.root/'receipt.json'
        self.save_receipt()
        blobs = patch.object(replay, 'git_blob', side_effect=lambda root, source, name: self.original[name])
        trees = patch.object(replay, 'git_files', return_value={name: replay.git_identity(data) for name, data in self.original.items()})
        blobs.start(); trees.start()
        self.addCleanup(blobs.stop); self.addCleanup(trees.stop)

    def save_receipt(self):
        self.path.write_text(json.dumps(self.receipt))

    def validate(self):
        return replay.validate_release(self.root, self.path)

    def test_matching_sources_review_and_build_identity_are_reported(self):
        proof = self.validate()
        self.assertEqual(proof['source_commit'], self.source)
        self.assertEqual(proof['production_source_files_verified'], 1)
        self.assertEqual(proof['runtime_inputs_verified'], 4)
        self.assertEqual(proof['agent'], str(self.review/'elab-agent'))

    def test_current_ledger_cannot_be_reported_as_older_review_source(self):
        (self.root/replay.LEDGER).write_text(json.dumps(dict(source_sha256={})))
        with patch('runtime_isolation.pinned_child_runner') as runner:
            with self.assertRaisesRegex(ValueError, 'source ledger differs'):
                replay.run(self.root, self.path, self.root/'new-output')
            runner.assert_not_called()
        self.assertFalse((self.root/'new-output').exists())

    def test_runtime_changes_or_new_inputs_refuse_even_with_fresh_ui_ledger(self):
        (self.root/'workbench/model.go').write_bytes(b'changed solver\n')
        with self.assertRaisesRegex(ValueError, 'runtime source differs'):
            self.validate()
        (self.root/'workbench/model.go').write_bytes(self.original['workbench/model.go'])
        (self.root/'desktop/new.go').write_bytes(b'package main\n')
        with self.assertRaisesRegex(ValueError, 'input set differs'):
            self.validate()

    def test_sdk_agent_native_build_and_manifest_identity_must_match(self):
        for changed in ('sdk', 'agent', 'native', 'build'):
            with self.subTest(changed=changed):
                if changed == 'sdk':
                    (self.root/'third_party/vellum/sdk.lock.json').write_text('{}')
                elif changed == 'agent':
                    self.receipt['agent_sha256'] = '0'*64
                elif changed == 'native':
                    self.receipt['native_sha256'] = '0'*64
                else:
                    self.receipt['clean_build_vcs'] = False
                self.save_receipt()
                with self.assertRaises(ValueError):
                    self.validate()
                (self.root/'third_party/vellum/sdk.lock.json').write_text(json.dumps(self.pin))
                self.receipt.update(agent_sha256=replay.digest(self.review/'elab-agent'),
                                    native_sha256=replay.digest(self.review/'ELab.app/Contents/MacOS/ELab'),
                                    clean_build_vcs=True)
        self.save_receipt()
        (self.review/'Evidence/Agent-build-info.txt').write_text('vcs.modified=true\n')
        with self.assertRaisesRegex(ValueError, 'manifest mismatch'):
            self.validate()

    def test_links_and_existing_artifacts_preserve_files(self):
        output = self.root/'valuable'
        output.mkdir(); marker = output/'saved.elab'; marker.write_text('retain')
        with patch('runtime_isolation.pinned_child_runner') as runner:
            with self.assertRaises(FileExistsError):
                replay.run(self.root, self.path, output)
            runner.assert_not_called()
        self.assertEqual(marker.read_text(), 'retain')
        agent = self.review/'elab-agent'
        agent.rename(self.root/'kept-agent')
        agent.symlink_to(self.root/'kept-agent')
        with self.assertRaises(ValueError):
            self.validate()

    def test_worker_rejects_producer_or_log_drift_before_client_execution(self):
        output = self.root/'observations'; output.mkdir()
        log = output/'focused.txt'; log.write_text('PASS\n')
        sources = dict(test='digest')
        producer = dict(release=self.validate(), sources=sources, test_command=replay.TEST_ARGS,
                        test_exit_code=0, observations_sha256=replay.digest(log))
        with patch.object(replay, 'producer_sources', return_value=sources), patch.object(replay, 'replay') as client:
            for field in ('sources', 'observations_sha256'):
                damaged = copy.deepcopy(producer)
                damaged[field] = {} if field == 'sources' else 'wrong'
                (output/'producer.json').write_text(json.dumps(damaged))
                with self.assertRaises(ValueError):
                    replay.worker(self.root, self.path, output)
            client.assert_not_called()

    def test_only_fresh_complete_passing_observations_are_admitted(self):
        path = self.root/'run.txt'
        valid = '\n'.join('named-wiring-observation '+json.dumps(dict(case=name)) for name in sorted(replay.CASES))+'\nPASS\n'
        path.write_text(valid)
        self.assertEqual(set(replay.observations(path)), replay.CASES)
        for text in (valid+'FAIL\n', valid+'ok package (cached)\n', 'PASS\n', valid+valid):
            with self.subTest(text=text):
                path.write_text(text)
                with self.assertRaises(ValueError):
                    replay.observations(path)


if __name__ == '__main__':
    unittest.main()
