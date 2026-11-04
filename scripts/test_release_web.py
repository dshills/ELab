import copy
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import release_web as release

COMMIT = 'a'*40


def browser_report():
    return dict(stats=dict(expected=3, skipped=2, unexpected=0, flaky=0), errors=[],
                config=dict(projects=[dict(name=n) for n in ['chromium','firefox','webkit']]),
                suites=[dict(specs=[dict(tests=[dict(projectName=n, status='expected',
                         results=[dict(status='passed')]) for n in ['chromium','firefox','webkit']])])])


class ReleaseTests(unittest.TestCase):
    def test_browser_receipt_requires_each_engine_and_no_flaky_or_failed_tests(self):
        report = browser_report()
        self.assertEqual(release.browser_summary(report)['expected'], 3)
        for change in ['flaky', 'unexpected', 'missing-engine', 'no-results', 'error']:
            bad = copy.deepcopy(report)
            if change in ['flaky', 'unexpected']:
                bad['stats'][change] = 1
            elif change == 'missing-engine':
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
            steps = [dict(name=n, status='passed', exit_code=0) for n,_ in release.gate_plan('macos')]
            report = dict(passed=True, gates_passed=True, inputs_unchanged=True, profile='macos',
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
        for failure in ['download','runtime','main-change','existing-tag','candidate','none']:
            with self.subTest(failure=failure), tempfile.TemporaryDirectory() as temporary:
                directory=Path(temporary)
                archive=directory/'ELab.zip';archive.write_bytes(b'qualified archive')
                for name in ['release.json','SHA256SUMS','RELEASE-NOTES.md']:
                    (directory/name).write_text(name)
                manifest=dict(version='1.0.0-alpha.1',build='1',source_commit=COMMIT,
                              archive=archive.name,publication_ready=failure!='candidate')
                calls=[];main_reads=0
                def run(args,**kwargs):
                    nonlocal main_reads
                    calls.append([str(a) for a in args])
                    if args[:2] == ['gh','api']:
                        if 'matching-refs' in args[2]:
                            return json.dumps([{'ref':'refs/tags/v1.0.0-alpha.1'}] if failure=='existing-tag' else [])
                        main_reads+=1
                        return json.dumps({'object':{'sha':('b'*40 if failure=='main-change' and main_reads>1 else COMMIT)}})
                    if args[:3] == ['gh','release','download']:
                        destination=Path(args[args.index('--dir')+1])
                        for name in [archive.name,'release.json','SHA256SUMS']:
                            (destination/name).write_bytes((directory/name).read_bytes())
                        if failure=='download': (destination/archive.name).write_bytes(b'changed by server')
                    return ''
                args=type('Args',(),dict(directory=directory,make_public=True))()
                with patch.object(release,'verify_stage',return_value=(manifest,{})), patch.object(release,'source',return_value=(COMMIT,'main')), patch.object(release,'run',side_effect=run), patch.object(release,'extract_and_qualify') as runtime:
                    if failure=='runtime': runtime.side_effect=ValueError('shipping backend failed')
                    if failure=='none': self.assertTrue(release.publish(args)['published'])
                    else:
                        with self.assertRaises(ValueError): release.publish(args)
                public=[c for c in calls if c[:3]==['gh','release','edit']]
                self.assertEqual(bool(public),failure=='none')
                if failure=='download': runtime.assert_not_called()
                if failure in ['candidate','existing-tag']:
                    self.assertFalse(any(c[:3]==['gh','release','create'] for c in calls))

    def test_staged_bytes_and_package_identity_are_rechecked_before_any_publication(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory=Path(temporary)
            qualification=directory/'qualification';qualification.mkdir()
            steps=[dict(name=n,status='passed',exit_code=0) for n,_ in release.gate_plan('macos')]
            quality=dict(passed=True,gates_passed=True,inputs_unchanged=True,profile='macos',source_commit=COMMIT,steps=steps)
            (qualification/'report.json').write_text(json.dumps(quality))
            (qualification/'inputs.json').write_text('{}')
            (directory/'browsers.json').write_text(json.dumps(browser_report()))
            (directory/'ELab.zip').write_bytes(b'archive')
            receipt=dict(source_commit=COMMIT,source_clean=True,test_fixture=False,version='1.0.0-alpha.1',build='1')
            (directory/'ELab Web.package.json').write_text(json.dumps(receipt))
            manifest=dict(version=receipt['version'],build='1',source_commit=COMMIT,archive='ELab.zip',
                          archive_sha256=release.checksum(directory/'ELab.zip'),
                          package_receipt_sha256=release.checksum(directory/'ELab Web.package.json'),
                          browser_report_sha256=release.checksum(directory/'browsers.json'),
                          quality_report_sha256=release.checksum(qualification/'report.json'))
            (directory/'release.json').write_text(json.dumps(manifest))
            with patch.object(release,'source_inputs',return_value={}):
                release.verify_stage(directory)
                (directory/'ELab.zip').write_bytes(b'changed')
                with self.assertRaises(ValueError): release.verify_stage(directory)
                (directory/'ELab.zip').write_bytes(b'archive')
                receipt['source_commit']='b'*40
                (directory/'ELab Web.package.json').write_text(json.dumps(receipt))
                manifest['package_receipt_sha256']=release.checksum(directory/'ELab Web.package.json')
                (directory/'release.json').write_text(json.dumps(manifest))
                with self.assertRaises(ValueError): release.verify_stage(directory)
