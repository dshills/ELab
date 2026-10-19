"""Read-only local-store refusal checks; only metadata copies change in memory."""
from pathlib import Path
import copy
import json
from unittest.mock import patch

import frame_store

folder = Path(__file__).resolve().parent
capture = folder / 'led/session/trials/led2-01/capture'
original_read = frame_store.read
images = frame_store.load_images(capture)
assert images
rejected = []


def refuses(name, filename, change, error):
    def altered(path):
        value = original_read(path)
        if path.name == filename:
            value = copy.deepcopy(value)
            change(value)
        return value

    with patch.object(frame_store, 'read', altered):
        try:
            frame_store.load_images(capture)
        except error:
            rejected.append(name)
            return
    raise AssertionError('Corruption admitted: ' + name)


refuses('absolute_archive_path', 'frame-store.json', lambda d: d.update(archive_path='/private/tmp/not-an-evidence-archive'), ValueError)
refuses('escaping_archive_path', 'frame-store.json', lambda d: d.update(archive_path='../not-an-evidence-archive'), ValueError)
refuses('missing_archive', 'frame-store.json', lambda d: d.update(archive_path='build/validation-evidence/latency-followup-173/missing-counterexample.tar.xz'), FileNotFoundError)
refuses('changed_archive_sha', 'frame-store.json', lambda d: d.update(archive_sha256='0' * 64), AssertionError)
refuses('changed_archive_size', 'frame-store.json', lambda d: d.update(archive_bytes=d['archive_bytes'] + 1), AssertionError)
refuses('changed_member_count', 'frame-store.json', lambda d: d.update(member_count=d['member_count'] + 1), AssertionError)
refuses('changed_member_hash', 'image-manifest.json', lambda d: next(iter(d.values())).update(sha256='0' * 64), AssertionError)
refuses('over_image_budget', 'report.json', lambda d: d.update(image_bytes=64 * 1024 * 1024 + 1), AssertionError)
assert len(rejected) == 8
print(json.dumps(dict(passed=True, read_only=True, images_verified=len(images), rejected=rejected)))
