"""macOS bundle icons: snapshot explicit artwork before native conversion/signing."""
import hashlib
from pathlib import Path
import struct
import tempfile

from child_process import run_child
from native_archive import regular_stream, unchanged
from sdk_environment import child_environment

MAX_ICON_BYTES = 16 << 20
ICON_NAME = "AppIcon.icns"
TOOL_TIMEOUT_SECONDS = 60
PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"


def read_icon(path):
    stream, before = regular_stream(path, MAX_ICON_BYTES)
    with stream:
        data = stream.read(MAX_ICON_BYTES + 1)
        unchanged(stream, before)
    if len(data) != before.st_size or not data or len(data) > MAX_ICON_BYTES:
        raise ValueError("app icon is empty, changed, or exceeds 16 MiB")
    return data


def validate_icns(data):
    if len(data) < 16 or data[:4] != b"icns" or int.from_bytes(data[4:8], "big") != len(data):
        raise ValueError("app icon must be a valid ICNS container")
    offset = 8
    while offset < len(data):
        if offset + 8 > len(data):
            raise ValueError("truncated ICNS element")
        size = int.from_bytes(data[offset + 4:offset + 8], "big")
        if size <= 8 or offset + size > len(data):
            raise ValueError("invalid ICNS element length")
        offset += size


def tool(argv):
    run_child(argv, check=True, env=child_environment(), timeout=TOOL_TIMEOUT_SECONDS)


def install(source, resources):
    """Install a fixed-name full-color icon from a bounded, regular source file.

    Native tools read only a detached snapshot. No source paths enter the bundle.
    Output is subsequently sealed by the packager's existing signature pipeline.
    """
    source, resources = Path(source), Path(resources)
    kind = source.suffix.lower()
    if kind not in {".png", ".icns"}:
        raise ValueError("app icon must be .png or .icns")
    data = read_icon(source)
    if kind == ".png":
        if (len(data) < 33 or data[:8] != PNG_SIGNATURE or
                data[8:16] != b"\x00\x00\x00\rIHDR" or
                struct.unpack(">II", data[16:24]) != (1024, 1024)):
            raise ValueError("PNG app icon must be 1024 by 1024 pixels")
    else:
        validate_icns(data)
    destination = resources / ICON_NAME
    with tempfile.TemporaryDirectory(prefix=".vellum-icon-") as temporary:
        work = Path(temporary)
        snapshot = work / ("source" + kind)
        snapshot.write_bytes(data)
        if kind == ".png":
            iconset = work / "AppIcon.iconset"
            iconset.mkdir()
            for size in (16, 32, 128, 256, 512):
                for scale in (1, 2):
                    suffix = "@2x" if scale == 2 else ""
                    pixels = str(size * scale)
                    tool(["/usr/bin/sips", "--resampleHeightWidth", pixels, pixels,
                          str(snapshot), "--out", str(iconset / f"icon_{size}x{size}{suffix}.png")])
            converted = work / ICON_NAME
            tool(["/usr/bin/iconutil", "--convert", "icns", "--output", str(converted), str(iconset)])
            icon = read_icon(converted)
            validate_icns(icon)
        else:
            icon = data
        # Decode a staged copy even for supplied ICNS; container framing alone
        # does not establish that macOS can read an image representation.
        candidate = work / "validated.icns"
        candidate.write_bytes(icon)
        decoded = work / "decoded.iconset"
        tool(["/usr/bin/iconutil", "--convert", "iconset", "--output", str(decoded), str(candidate)])
        if not any(decoded.glob("*.png")):
            raise ValueError("app icon contains no decodable image representations")
        with destination.open("xb") as output:
            output.write(icon)
    return {"file": ICON_NAME, "sha256": hashlib.sha256(icon).hexdigest(),
            "source_format": kind[1:], "source_sha256": hashlib.sha256(data).hexdigest()}


def verify(resources, record):
    if record is None:
        return
    if hashlib.sha256(read_icon(resources / ICON_NAME)).hexdigest() != record["sha256"]:
        raise ValueError("packaged app icon changed")
