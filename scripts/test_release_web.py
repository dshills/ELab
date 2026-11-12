import copy
import json
import stat
import zipfile
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import release_web as release
from portable_web import executable_name

COMMIT = 'a'*40


def browser_report():
    projects = ['chromium', 'firefox', 'webkit', 'chromium-large-circuits',
                'firefox-large-circuits', 'webkit-large-circuits']
    return dict(stats=dict(expected=6, skipped=2, unexpected=0, flaky=0), errors=[],
                config=dict(projects=[dict(name=n) for n in projects]),
                suites=[dict(specs=[dict(tests=[dict(projectName=n, status='expected',
                         results=[dict(status='passed')]) for n in projects])])])


class ReleaseTests(unittest.TestCase):
    def test_browser_receipt_requires_each_project_and_no_flaky_or_failed_tests(self):
        report = browser_report()
        self.assertEqual(release.browser_summary(report)['expected'], 6)
        for change in ['flaky', 'unexpected', 'missing-project', 'no-results', 'error']:
            bad = copy.deepcopy(report)
            if change in ['flaky', 'unexpected']:
                bad['stats'][change] = 1
            elif change == 'missing-project':
                bad['config']['projects'].pop()
            elif change == 'no-results':
                bad['suites'][0]['specs'][0]['tests'].pop()
            else:
                bad['errors'].append({'message':'failed server'})
            with self.subTest(change=change), self.assertRaises(ValueError):
                release.browser_summary(bad)

    def test_quality_receipt_cannot_reuse_an_old_commit_changed_source_or_partial_plan(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            steps = [dict(name=n, status='passed', exit_code=0) for n,_ in release.gate_plan('portable')]
            report = dict(passed=True, gates_passed=True, inputs_unchanged=True, profile='portable',
                          source_commit=COMMIT, steps=steps)
            (directory/'inputs.json').write_text('{"main.go":"hash"}')
            for change in ['none','commit','source','failed','partial','unfrozen']:
                bad = copy.deepcopy(report)
                inputs = {'main.go':'hash'}
                if change == 'commit': bad['source_commit']='b'*40
                elif change == 'source': inputs['main.go']='new'
                elif change == 'failed': bad['steps'][1]['exit_code']=1
                elif change == 'partial': bad['steps'].pop()
                elif change == 'unfrozen': bad['inputs_unchanged']=False
                (directory/'report.json').write_text(json.dumps(bad))
                with self.subTest(change=change):
                    if change == 'none': release.check_quality(directory, COMMIT, inputs)
                    else:
                        with self.assertRaises(ValueError): release.check_quality(directory, COMMIT, inputs)

    def test_draft_download_failure_or_main_change_cannot_publish(self):
        for failure in ['download','runtime','main-change','existing-tag','candidate','receipts','none']:
            with self.subTest(failure=failure), tempfile.TemporaryDirectory() as temporary:
                directory = Path(temporary)
                archives = {target: dict(name=release.archive_name('1.0.0-alpha.1', target))
                            for target in release.TARGETS}
                assets = [entry['name'] for entry in archives.values()] + ['release.json', 'SHA256SUMS']
                for name in [*assets, 'RELEASE-NOTES.md']:
                    (directory/name).write_text(name)
                manifest = dict(version='1.0.0-alpha.1', build='1', source_commit=COMMIT,
                                archives=archives, publication_ready=failure!='candidate')
                calls = []
                main_reads = 0

                def run(args, **kwargs):
                    nonlocal main_reads
                    calls.append([str(a) for a in args])
                    if args[:2] == ['gh','api']:
                        if 'matching-refs' in args[2]:
                            return json.dumps([{'ref':'refs/tags/v1.0.0-alpha.1'}] if failure=='existing-tag' else [])
                        main_reads += 1
                        return json.dumps({'object':{'sha':('b'*40 if failure=='main-change' and main_reads>1 else COMMIT)}})
                    if args[:3] == ['gh','release','download']:
                        destination = Path(args[args.index('--dir')+1])
                        for name in assets:
                            (destination/name).write_bytes((directory/name).read_bytes())
                        if failure == 'download':
                            (destination/assets[-3]).write_bytes(b'changed by server')
                    return ''

                args = type('Args', (), dict(directory=directory, make_public=True))()
                with (patch.object(release,'verify_stage',return_value=manifest),
                      patch.object(release,'verify_runtime_receipts') as receipts,
                      patch.object(release,'source',return_value=(COMMIT,'main')),
                      patch.object(release,'run',side_effect=run),
                      patch.object(release,'inspect_archive'),
                      patch.object(release,'qualify_archive') as runtime):
                    if failure == 'runtime':
                        runtime.side_effect = ValueError('shipping backend failed')
                    if failure == 'receipts':
                        receipts.side_effect = ValueError('missing platform verification')
                    if failure == 'none':
                        self.assertTrue(release.publish(args)['published'])
                    else:
                        with self.assertRaises(ValueError):
                            release.publish(args)
                self.assertEqual(any(c[:3]==['gh','release','edit'] for c in calls), failure=='none')
                if failure == 'download':
                    runtime.assert_not_called()
                if failure in ['candidate','existing-tag','receipts']:
                    self.assertFalse(any(c[:3]==['gh','release','create'] for c in calls))

    def make_stage(self, directory):
        qualification = directory/'qualification'
        qualification.mkdir()
        steps = [dict(name=n, status='passed', exit_code=0) for n,_ in release.gate_plan('portable')]
        quality = dict(passed=True, gates_passed=True, inputs_unchanged=True, profile='portable',
                       source_commit=COMMIT, steps=steps)
        (qualification/'report.json').write_text(json.dumps(quality))
        (qualification/'inputs.json').write_text('{}')
        (directory/'browsers.json').write_text(json.dumps(browser_report()))
        version = '1.0.0-alpha.1'
        archives = {}
        for target in release.TARGETS:
            name = release.archive_name(version, target)
            files = {}
            with zipfile.ZipFile(directory/name, 'w') as zipped:
                for entry in [executable_name(target), 'START-HERE.txt', 'BUILD.txt',
                              'Licenses/Go.txt', 'Licenses/README.txt']:
                    content = qualification/'fixture'
                    content.write_text(f'{target}: {entry}')
                    item = zipfile.ZipInfo(entry)
                    item.external_attr = (stat.S_IFREG | 0o755) << 16
                    zipped.writestr(item, content.read_bytes())
                    files[entry] = release.checksum(content)
            digest = release.checksum(directory/name)
            archives[target] = dict(name=name, sha256=digest, files=files)
            (qualification/f'runtime-{target}.json').write_text(json.dumps(dict(
                source_commit=COMMIT, source_clean=True, archive_sha256=digest, target=target,
                passed=True, checks=['example'])))
        manifest = dict(schema=2, version=version, build='1', source_commit=COMMIT, archives=archives,
                        browser_report_sha256=release.checksum(directory/'browsers.json'),
                        quality_report_sha256=release.checksum(qualification/'report.json'),
                        browsers=release.browser_summary(browser_report()))
        (directory/'SHA256SUMS').write_text(release.checksum_text(archives))
        (directory/'RELEASE-NOTES.md').write_text('example release notes')
        manifest['release_notes_sha256'] = release.checksum(directory/'RELEASE-NOTES.md')
        (directory/'release.json').write_text(json.dumps(manifest))
        return manifest

    def test_publish_requires_all_exact_archives_and_platform_receipts(self):
        with tempfile.TemporaryDirectory() as temporary, patch.object(release,'source_inputs',return_value={}):
            directory = Path(temporary)
            manifest = self.make_stage(directory)
            self.assertEqual(release.verify_stage(directory), manifest)
            release.verify_runtime_receipts(directory, manifest)
            target = 'windows-amd64'
            receipt = directory/'qualification'/f'runtime-{target}.json'
            wrong = json.loads(receipt.read_text())
            wrong['archive_sha256'] = '0'*64
            receipt.write_text(json.dumps(wrong))
            with self.assertRaisesRegex(ValueError, 'runtime receipt'):
                release.verify_runtime_receipts(directory, manifest)
            archive = directory/manifest['archives'][target]['name']
            archive.write_bytes(b'changed')
            with self.assertRaisesRegex(ValueError, 'archive bytes differ'):
                release.verify_stage(directory)

    def test_zip_verification_refuses_unsafe_entries_before_extracting_them(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            target = 'linux-amd64'
            manifest = self.make_stage(directory)
            entry = manifest['archives'][target]
            archive = directory/entry['name']
            name = 'Licenses/../escape'
            with zipfile.ZipFile(archive, 'a') as zipped:
                item = zipfile.ZipInfo(name)
                item.external_attr = (stat.S_IFREG | 0o644) << 16
                zipped.writestr(item, 'no')
            entry['sha256'] = release.checksum(archive)
            entry['files'][name] = '0'*64
            with self.assertRaisesRegex(ValueError, 'invalid release archive entry'):
                release.inspect_archive(archive, entry, target, directory/'out')
            self.assertFalse((directory/'out'/'escape').exists())
