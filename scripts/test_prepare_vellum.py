import hashlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location("prepare_vellum", ROOT / "scripts/prepare-vellum.py")
prepare = importlib.util.module_from_spec(spec)
spec.loader.exec_module(prepare)


class PrepareVellumTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        pin = prepare.load_pin(ROOT)
        prepare.verify_archive_identity(ROOT / pin["archive"], pin)
        with tarfile.open(ROOT / pin["archive"], "r:gz") as archive:
            cls.bootstrap = {name: archive.extractfile(pin["prefix"] + "/" + name).read()
                             for name in prepare.BOOTSTRAP}

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="elab-sdk-test-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name).resolve()
        (self.root / "third_party/vellum").mkdir(parents=True)
        (self.root / "desktop").mkdir()
        self.cache = self.root / "build/.vellum-sdk-fixture"
        self.files = dict(self.bootstrap)
        for module, relative in prepare.MODULES.items():
            self.files[(Path(relative) / "go.mod").as_posix()] = f"module {module}\n\ngo 1.25.6\n".encode()
        self.files["PRIVATE_BUILD.txt"] = b"Private SDK fixture.\n"
        self.write_archive()
        replacements = "\n".join(f" {name} => ../build/.vellum-sdk-fixture/{path}" for name, path in prepare.MODULES.items())
        self.mod = "module example.com/elab-sdk-fixture\n\ngo 1.25.6\n\nreplace (\n" + replacements + "\n)\n"
        (self.root / "desktop/go.mod").write_text(self.mod)

    def write_archive(self, extra=None):
        manifest = {"schema": 1, "release": {"module": "github.com/dshills/vellum", "version": "v0.0.0-fixture"},
                    "source": {"commit": "a" * 40, "clean": True}, "private_build": True,
                    "files": {name: hashlib.sha256(data).hexdigest() for name, data in self.files.items()}}
        raw = json.dumps(manifest).encode()
        self.archive = self.root / "third_party/vellum/fixture.tar.gz"
        with tarfile.open(self.archive, "w:gz") as archive:
            for name, data in [*self.files.items(), ("SDK_MANIFEST.json", raw), *(extra or [])]:
                member = tarfile.TarInfo("vellum-v0.0.0-fixture/" + name)
                member.size = len(data)
                member.mode = 0o644
                archive.addfile(member, io.BytesIO(data))
        self.pin = {"schema": 1, "archive": "third_party/vellum/fixture.tar.gz",
                    "archive_bytes": self.archive.stat().st_size,
                    "sha256": prepare.digest(self.archive), "manifest_sha256": hashlib.sha256(raw).hexdigest(),
                    "cache": "build/.vellum-sdk-fixture", "prefix": "vellum-v0.0.0-fixture", "version": "v0.0.0-fixture",
                    "module": "github.com/dshills/vellum", "private_build": True,
                    "source_commit": "a" * 40, "files": len(self.files),
                    "bootstrap": {name: hashlib.sha256(self.files[name]).hexdigest() for name in prepare.BOOTSTRAP}}
        self.write_pin()

    def write_pin(self):
        (self.root / "third_party/vellum/sdk.lock.json").write_text(json.dumps(self.pin))

    def test_real_verifier_extracts_and_cache_hit_does_not_run_bootstrap(self):
        self.assertEqual(prepare.prepare_sdk(self.root), self.cache)
        self.assertEqual((self.cache / "go.mod").read_bytes(), self.files["go.mod"])
        original_run = subprocess.run
        with mock.patch.object(prepare.subprocess, "run", wraps=original_run) as run:
            self.assertEqual(prepare.prepare_sdk(self.root), self.cache)
            self.assertEqual(len(run.call_args_list), 1)
            self.assertEqual(run.call_args.args[0][:3], ["go", "mod", "edit"])

    def test_changed_archive_refuses_before_any_bootstrap_or_go_command(self):
        with self.archive.open("ab") as output:
            output.write(b"changed")
        with mock.patch.object(prepare.subprocess, "run") as run:
            with self.assertRaisesRegex(ValueError, "archive differs"):
                prepare.prepare_sdk(self.root)
            run.assert_not_called()
        self.assertFalse(self.cache.exists())

    def test_pin_requires_complete_explicit_verifier_imports_before_execution(self):
        original = dict(self.pin["bootstrap"])
        for changed in [{k: v for k, v in original.items() if k != "scripts/native_gpu.py"},
                        dict(original, **{"scripts/unexpected.py": "0" * 64})]:
            with self.subTest(imports=sorted(changed)):
                self.pin["bootstrap"] = changed
                self.write_pin()
                with mock.patch.object(prepare.subprocess, "run") as run:
                    with self.assertRaisesRegex(ValueError, "bootstrap"):
                        prepare.prepare_sdk(self.root)
                    run.assert_not_called()
                self.assertFalse(self.cache.exists())

    def test_modified_or_extra_cache_files_are_preserved(self):
        prepare.prepare_sdk(self.root)
        changed = self.cache / "go.mod"
        changed.write_text("local developer edit\n")
        with self.assertRaisesRegex(ValueError, "cache file changed"):
            prepare.prepare_sdk(self.root)
        self.assertEqual(changed.read_text(), "local developer edit\n")
        changed.write_bytes(self.files["go.mod"])
        extra = self.cache / "local.go"
        extra.write_text("package local\n")
        with self.assertRaisesRegex(ValueError, "inventory changed"):
            prepare.prepare_sdk(self.root)
        self.assertEqual(extra.read_text(), "package local\n")

    def test_existing_unknown_cache_is_not_replaced(self):
        self.cache.mkdir(parents=True)
        notes = self.cache / "notes.txt"
        notes.write_text("preserve me")
        with self.assertRaises(ValueError):
            prepare.prepare_sdk(self.root)
        self.assertEqual(notes.read_text(), "preserve me")

    def test_sdk_replacement_drift_and_version_shadow_are_rejected(self):
        mod = self.root / "desktop/go.mod"
        for text in [self.mod.replace("../build/.vellum-sdk-fixture/drivers/cogent/build/upstream", "../../Vellum/drivers/cogent/build/upstream"),
                     self.mod + "\nreplace github.com/dshills/vellum v0.0.0 => ../../Vellum\n"]:
            with self.subTest(text=text):
                mod.write_text(text)
                with self.assertRaisesRegex(ValueError, "replacement"):
                    prepare.prepare_sdk(self.root)
                self.assertFalse(self.cache.exists())

    def test_cache_symlink_is_refused_without_touching_target(self):
        prepare.prepare_sdk(self.root)
        target = self.root / "developer-notes"
        target.write_text("do not read or replace")
        path = self.cache / "go.mod"
        path.unlink()
        try:
            path.symlink_to(target)
        except OSError as error:
            self.skipTest(str(error))
        with self.assertRaisesRegex(ValueError, "regular file"):
            prepare.prepare_sdk(self.root)
        self.assertEqual(target.read_text(), "do not read or replace")

    def test_unsafe_lock_and_archive_paths_do_not_escape_owned_staging(self):
        self.pin["cache"] = "../outside"
        self.write_pin()
        with self.assertRaisesRegex(ValueError, "relative path"):
            prepare.prepare_sdk(self.root)
        self.write_archive(extra=[("../escaped.txt", b"must not escape")])
        with self.assertRaises(subprocess.CalledProcessError) as refusal:
            prepare.prepare_sdk(self.root)
        self.assertIn("unsafe SDK archive member", refusal.exception.stderr)
        self.assertFalse((self.root / "escaped.txt").exists())
        self.assertFalse(self.cache.exists())


if __name__ == "__main__":
    unittest.main()
