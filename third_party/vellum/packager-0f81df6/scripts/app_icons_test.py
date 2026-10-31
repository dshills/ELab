"""Icon admission, native conversion, bundle signing and mutation regressions."""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import plistlib
import struct
import subprocess
import tempfile
import unittest
from unittest import mock
import zlib

import app_icons

spec = importlib.util.spec_from_file_location("icon_package_app", Path(__file__).with_name("package-app.py"))
app = importlib.util.module_from_spec(spec)
spec.loader.exec_module(app)


def png(width=1024, height=1024):
    def chunk(kind, data):
        return struct.pack(">I", len(data)) + kind + data + struct.pack(">I", zlib.crc32(kind + data))
    pixels = (b"\x00" + bytes((20, 110, 240, 255)) * width) * height
    return (app_icons.PNG_SIGNATURE + chunk(b"IHDR", struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0)) +
            chunk(b"IDAT", zlib.compress(pixels)) + chunk(b"IEND", b""))


class IconChecks(unittest.TestCase):
    def test_invalid_sources_fail_before_tools_or_output(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for name, data in (("bad.svg", b"svg"), ("bad.png", b"wrong"),
                               ("small.png", png(16, 16)), ("wide.png", png(1024, 16)),
                               ("empty.icns", b"icns" + (8).to_bytes(4, "big")),
                               ("bad.icns", b"icns" + (16).to_bytes(4, "big") + b"ic07" + (99).to_bytes(4, "big"))):
                with self.subTest(name=name), mock.patch.object(app_icons, "tool") as tool:
                    source = root / name
                    source.write_bytes(data)
                    with self.assertRaises(ValueError):
                        app_icons.install(source, root)
                    tool.assert_not_called()
                    self.assertFalse((root / app_icons.ICON_NAME).exists())
            actual = root / "real.png"
            actual.write_bytes(png())
            link = root / "link.png"
            link.symlink_to(actual)
            with self.assertRaises(ValueError):
                app_icons.read_icon(link)
            fifo = root / "fifo.png"
            if hasattr(os, "mkfifo"):
                os.mkfifo(fifo)
                with self.assertRaises(ValueError):
                    app_icons.read_icon(fifo)
            with actual.open("wb") as stream:
                stream.truncate(app_icons.MAX_ICON_BYTES + 1)
            with self.assertRaises(ValueError):
                app_icons.read_icon(actual)

    def test_non_mac_icon_request_is_rejected(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            binary = root / "binary"
            binary.write_bytes(b"\x7fELFfixture")
            with self.assertRaisesRegex(ValueError, "only for macOS"):
                app.package(binary, root / "out", "Test", "org.example.test", "1.0.0", "linux", icon=root / "missing.png")
            self.assertFalse((root / "out").exists())

    @unittest.skipUnless(platform.system() == "Darwin", "Apple icon tools required")
    def test_native_png_and_icns_round_trip(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "Source with spaces.png"
            data = png()
            source.write_bytes(data)
            resources = root / "App.app/Contents/Resources"
            resources.mkdir(parents=True)
            record = app_icons.install(source, resources)
            self.assertEqual(record["source_sha256"], hashlib.sha256(data).hexdigest())
            self.assertEqual(record["source_format"], "png")
            app_icons.verify(resources, record)
            self.assertEqual([p.name for p in resources.parent.iterdir()], ["Resources"])
            with self.assertRaises(FileExistsError):
                app_icons.install(source, resources)
            copied = root / "Copied"
            copied.mkdir()
            second = app_icons.install(resources / app_icons.ICON_NAME, copied)
            self.assertEqual(second["sha256"], record["sha256"])
            self.assertEqual(second["source_format"], "icns")
            decoded = root / "check.iconset"
            app_icons.tool(["/usr/bin/iconutil", "--convert", "iconset", "--output", str(decoded), str(copied / app_icons.ICON_NAME)])
            for size in (16, 32, 128, 256, 512):
                for scale in (1, 2):
                    suffix = "@2x" if scale == 2 else ""
                    pixels = (decoded / f"icon_{size}x{size}{suffix}.png").read_bytes()
                    self.assertEqual(struct.unpack(">II", pixels[16:24]), (size * scale, size * scale))
            (copied / app_icons.ICON_NAME).write_bytes(b"changed")
            with self.assertRaisesRegex(ValueError, "icon changed"):
                app_icons.verify(copied, second)

    @unittest.skipUnless(platform.system() == "Darwin", "Apple icon tools required")
    def test_native_conversion_uses_snapshot_after_source_changes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "source.png"
            original = png()
            source.write_bytes(original)
            resources = root / "Resources"
            resources.mkdir()
            actual_tool = app_icons.tool
            calls = 0
            def mutate_then_convert(argv):
                nonlocal calls
                calls += 1
                if calls == 1:
                    source.write_bytes(b"replaced during conversion")
                return actual_tool(argv)
            with mock.patch.object(app_icons, "tool", side_effect=mutate_then_convert):
                record = app_icons.install(source, resources)
            self.assertEqual(record["source_sha256"], hashlib.sha256(original).hexdigest())
            app_icons.verify(resources, record)

    @unittest.skipUnless(platform.system() == "Darwin", "Apple icon tools required")
    def test_native_corrupt_png_refuses_and_leaves_no_bundle(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "corrupt.png"
            source.write_bytes(png()[:33])  # plausible IHDR; no pixels
            with self.assertRaises(subprocess.CalledProcessError):
                app.package(Path("/usr/bin/true"), root / "Bad.app", "Bad", "org.vellum.icon.bad", "1.0.0", "darwin", icon=source)
            self.assertFalse((root / "Bad.app").exists())

    @unittest.skipUnless(platform.system() == "Darwin", "Apple icon tools required")
    def test_structurally_valid_icns_without_images_refuses(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "no-images.icns"
            body = b"icns" + (24).to_bytes(4, "big") + b"TOC " + (16).to_bytes(4, "big") + b"ic07" + (8).to_bytes(4, "big")
            app_icons.validate_icns(body)  # framing is deliberately valid
            source.write_bytes(body)
            with self.assertRaises((subprocess.CalledProcessError, ValueError)):
                app.package(Path("/usr/bin/true"), root / "Bad.app", "Bad", "org.vellum.icon.bad", "1.0.0", "darwin", icon=source)
            self.assertFalse((root / "Bad.app").exists())

    def test_icon_is_present_before_signing_and_verified_after_publication(self):
        for tamper in (None, "icon", "plist"):
            with self.subTest(tamper=tamper), tempfile.TemporaryDirectory() as temporary, mock.patch.object(app.platform, "system", return_value="Darwin"), mock.patch.object(app, "darwin_dependencies"), mock.patch.object(app.macos_deployment,"inspect",return_value=dict(minimum="27.0",architectures={"arm64":dict(minimum="27.0",sdk="27.0")})):
                root = Path(temporary)
                binary = root / "binary"
                binary.write_bytes(b"\xcf\xfa\xed\xfefixture")
                output = root / "Test.app"
                payload = b"fixture icon bytes"
                record = {"file": app_icons.ICON_NAME, "sha256": hashlib.sha256(payload).hexdigest()}
                def install(source, resources):
                    (resources / app_icons.ICON_NAME).write_bytes(payload)
                    return record
                def signing(argv, **kwargs):
                    bundle = Path(argv[-1])
                    self.assertEqual((bundle / "Contents/Resources" / app_icons.ICON_NAME).read_bytes(), payload)
                    self.assertEqual(plistlib.loads((bundle / "Contents/Info.plist").read_bytes())["CFBundleIconFile"], app_icons.ICON_NAME)
                    if bundle == output and tamper:
                        if tamper == "icon":
                            (bundle / "Contents/Resources" / app_icons.ICON_NAME).write_bytes(b"tampered")
                        else:
                            info=plistlib.loads((bundle / "Contents/Info.plist").read_bytes())
                            info["CFBundleIconFile"]="Wrong.icns"
                            (bundle / "Contents/Info.plist").write_bytes(plistlib.dumps(info))
                with mock.patch.object(app_icons, "install", side_effect=install), mock.patch.object(app, "run_child", side_effect=signing):
                    if tamper:
                        with self.assertRaisesRegex(ValueError, "icon.*changed"):
                            app.package(binary, output, "Test", "org.example.test", "1.0.0", "darwin", "-", icon=root / "icon.png")
                        self.assertFalse(output.exists())
                    else:
                        app.package(binary, output, "Test", "org.example.test", "1.0.0", "darwin", "-", icon=root / "icon.png")
                        manifest = json.loads((output / "Contents/Resources/APP_MANIFEST.json").read_text())
                        self.assertEqual(manifest["app_icon"], record)


if __name__ == "__main__":
    unittest.main()
