import json
from pathlib import Path
import shutil
import tempfile
import unittest

import agent_notices

ROOT=Path(__file__).resolve().parents[1]


class AgentNoticeTests(unittest.TestCase):
    def fixture(self):
        directory=tempfile.TemporaryDirectory(prefix='elab-notice-test-');self.addCleanup(directory.cleanup)
        root=Path(directory.name)
        shutil.copytree(ROOT/'third_party/agent-mcp',root/'third_party/agent-mcp')
        for name in ['go.mod','go.sum']:shutil.copyfile(ROOT/name,root/name)
        return root

    def test_pinned_runtime_notices_verify(self):
        data=agent_notices.verify(ROOT)
        self.assertEqual(len(data['modules']),9)
        self.assertIn('github.com/modelcontextprotocol/go-sdk',data['modules'])

    def test_changed_missing_extra_and_linked_notices_refuse(self):
        for change in ['bytes','missing','extra','link']:
            with self.subTest(change=change):
                root=self.fixture();folder=root/'third_party/agent-mcp'
                data=json.loads((folder/'notices.json').read_text())
                file=folder/next(n for n in data['files'] if n.endswith('/LICENSE'))
                if change=='bytes':file.write_text('changed')
                elif change=='missing':file.unlink()
                elif change=='extra':(folder/'unexpected').write_text('extra')
                else:
                    file.unlink();file.symlink_to(root/'go.mod')
                with self.assertRaises(ValueError):agent_notices.verify(root)

    def test_module_version_and_checksum_drift_refuse(self):
        for change in ['version','sum']:
            root=self.fixture()
            if change=='version':
                p=root/'go.mod';p.write_text(p.read_text().replace('go-sdk v1.8.0','go-sdk v1.8.1'))
            else:
                p=root/'go.sum';lines=p.read_text().splitlines();lines=[line.replace(line.split()[2],'h1:changed') if line.startswith('github.com/modelcontextprotocol/go-sdk v1.8.0 ') else line for line in lines];p.write_text('\n'.join(lines)+'\n')
            with self.assertRaises(ValueError):agent_notices.verify(root)
