"""Data-only producer refusal tests; no application/recorder is launched."""
from pathlib import Path
import json
import tempfile
import unittest
from unittest.mock import patch

import prepare


class PrepareTests(unittest.TestCase):
    def test_partial_line_is_not_a_frame(self):
        self.assertIsNone(prepare.first_complete_frame(b'{"event":"frame","status":0,"png":"x"}'))
        self.assertEqual(prepare.first_complete_frame(b'{"event":"frame","status":0,"png":"x"}\n{')["png"], "x")

    def test_malformed_complete_lines_and_premature_marker_refuse(self):
        for raw in [b'bad\n', b'null\n', b'{"event":"other"}\n',
                    b'{"event":"marker"}\n', b'{"event":"frame","status":0}\n',
                    b'{"event":"frame","png":"x"}\n', b'x' * ((2 << 20) + 1)]:
            with self.subTest(raw=raw[:30]), self.assertRaises(ValueError):
                prepare.first_complete_frame(raw)

    def test_missing_frame_deadline_writes_only_negative_receipt(self):
        with tempfile.TemporaryDirectory() as temp:
            path = Path(temp) / 'receipt.json'
            with patch.object(prepare, 'admit_existing_frame') as checker:
                result = prepare.wait_and_admit(temp, path, deadline_seconds=.01,
                                               trial_id='rc-test', attempt_id='unique-offline-attempt')
            self.assertIs(result['admitted'], False)
            self.assertEqual(json.loads(path.read_text()), result)
            checker.assert_not_called()
            with self.assertRaises(FileExistsError):
                prepare.wait_and_admit(temp, path, deadline_seconds=.001)

    def test_bad_complete_journal_does_not_reach_checker(self):
        with tempfile.TemporaryDirectory() as temp:
            folder = Path(temp)
            (folder / 'frames.jsonl').write_text('bad\n')
            with patch.object(prepare, 'admit_existing_frame') as checker:
                result = prepare.wait_and_admit(temp, folder / 'receipt.json', deadline_seconds=.01)
            self.assertIs(result['admitted'], False)
            checker.assert_not_called()

    def test_nonsymlink_and_byte_budgets(self):
        with tempfile.TemporaryDirectory() as temp:
            folder = Path(temp)
            (folder / 'raw').write_bytes(b'abcd')
            (folder / 'link').symlink_to(folder / 'raw')
            self.assertEqual(prepare.bounded_file(folder / 'raw', 4), b'abcd')
            for path, limit in [(folder / 'raw', 3), (folder / 'link', 4), (folder, 4)]:
                with self.assertRaises(ValueError):
                    prepare.bounded_file(path, limit)

    def test_geometry_clock_filename_and_index_refusals(self):
        row = dict(event='frame', status=0, index=1, width_pixels=1053, height_pixels=802,
                   frame_geometry_validated=True, target_geometry_matches_at_receipt=True,
                   presentation_matches_converted_display_time=True, display_time_after_receipt=False,
                   display_mach_ticks='10', received_mach_ticks='11',
                   png='frame-' + 'a' * 64 + '.png', png_sha256='a' * 64)
        prepare.validate_frame(row, 1053, 802)
        prepare.validate_frame(dict(row, display_mach_ticks='12', display_time_after_receipt=True), 1053, 802)
        for key, value in [('event', 'marker'), ('status', 1), ('status', False), ('index', 0),
                           ('index', True), ('width_pixels', 1), ('height_pixels', 1),
                           ('frame_geometry_validated', False), ('target_geometry_matches_at_receipt', False),
                           ('presentation_matches_converted_display_time', False),
                           ('display_time_after_receipt', True), ('display_mach_ticks', '12'),
                           ('received_mach_ticks', None), ('png', '../x.png'), ('png_sha256', 'bad')]:
            with self.subTest(key=key, value=value), self.assertRaises(ValueError):
                prepare.validate_frame(dict(row, **{key: value}), 1053, 802)


if __name__ == '__main__':
    unittest.main()
