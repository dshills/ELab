"""Read declared local frame archives without capture, extraction or regeneration."""
from pathlib import Path
import gzip
import hashlib
import json
import tarfile

ROOT = Path('/Users/dshills/Development/projects/ELab')
ARCHIVE_ROOT = ROOT / 'build/validation-evidence/latency-led3-185'


def contents(path):
    return path.read_bytes() if path.exists() else gzip.decompress(path.with_name(path.name + '.gz').read_bytes())


def read(path):
    return json.loads(contents(path))


def load_images(capture):
    report = read(capture / 'report.json')
    expected_count, expected_bytes = report['unique_images'], report['image_bytes']
    assert 0 < expected_count <= 300 and 0 < expected_bytes <= 64 * 1024 * 1024
    pointer = capture / 'frame-store.json'
    if not pointer.exists():
        # Initial analysis uses originals in the terminal private collection.
        paths = list(capture.glob('*.png'))
        assert len(paths) == expected_count and sum(p.stat().st_size for p in paths) == expected_bytes
        images = {p.name: p.read_bytes() for p in paths}
    else:
        store = read(pointer)
        relative = Path(store['archive_path'])
        if relative.is_absolute() or '..' in relative.parts:
            raise ValueError('Frame archive must use its declared repository-relative path')
        archive_path = (ROOT / relative).resolve()
        if not archive_path.is_relative_to(ARCHIVE_ROOT.resolve()):
            raise ValueError('Frame archive is outside the declared local evidence directory')
        if not archive_path.is_file():
            raise FileNotFoundError(f'Restore the declared local archive: {archive_path}; never regenerate a trial')
        assert 0 < store['archive_bytes'] <= expected_bytes + 1024 * expected_count + 65536
        assert archive_path.stat().st_size == store['archive_bytes']
        raw = archive_path.read_bytes()
        assert len(raw) == store['archive_bytes']
        assert hashlib.sha256(raw).hexdigest() == store['archive_sha256']
        manifest = read(capture / 'image-manifest.json')
        assert len(manifest) == store['member_count'] == expected_count
        assert sum(info['bytes'] for info in manifest.values()) == store['image_bytes'] == expected_bytes
        images = {}
        with tarfile.open(archive_path, 'r:xz') as archive:
            for member in archive:
                assert member.isfile() and member.mode == 0o600 and member.name not in images
                assert Path(member.name).name == member.name and member.name in manifest
                info = manifest[member.name]
                assert member.size == info['bytes']
                image = archive.extractfile(member).read()
                assert len(image) == info['bytes'] and hashlib.sha256(image).hexdigest() == info['sha256']
                assert member.name == 'frame-' + info['sha256'] + '.png'
                images[member.name] = image
        assert set(images) == set(manifest)
    assert len(images) == expected_count and sum(map(len, images.values())) == expected_bytes
    return images
