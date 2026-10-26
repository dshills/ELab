"""Bounded Mac ar/link-map observations; never certify source or license provenance."""
import hashlib
import os
from pathlib import Path
import re
import stat

MAX_ARCHIVE_BYTES = 1 << 30
MAX_MEMBERS = 10000
MAX_NAME_BYTES = 4096
MAX_MAP_BYTES = 128 << 20
MAX_OBJECTS = 20000


def regular_stream(path, bound):
    path = Path(path)
    if path.is_symlink():
        raise ValueError("native input cannot be a symlink")
    stream = os.fdopen(os.open(path, os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_NONBLOCK", 0)), "rb")
    info = os.fstat(stream.fileno())
    if not stat.S_ISREG(info.st_mode) or info.st_size > bound:
        stream.close()
        raise ValueError("native input is nonregular or exceeds byte bound")
    return stream, info


def unchanged(stream, before):
    after = os.fstat(stream.fileno())
    if (before.st_dev, before.st_ino, before.st_size, before.st_mtime_ns, before.st_ctime_ns) != (after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns):
        raise ValueError("native input changed during observation")


def archive_members(path):
    """Read BSD extended or short ar names, retaining exact payload hashes only.

    GNU long-name tables, thin archives and malformed/ambiguous inputs refuse.
    No archive member is extracted or executed. Names are observations, not crate
    version/source/license identities. Header/name padding is outside payload hash.
    """
    stream, before = regular_stream(path, MAX_ARCHIVE_BYTES)
    digest = hashlib.sha256()
    members = []
    with stream:
        def read(size):
            data = stream.read(size)
            if len(data) != size:
                raise ValueError("truncated native archive")
            digest.update(data)
            return data
        if read(8) != b"!<arch>\n":
            raise ValueError("unsupported native archive format")
        while stream.tell() < before.st_size:
            if len(members) == MAX_MEMBERS:
                raise ValueError("native archive exceeds member bound")
            header = read(60)
            if header[58:] != b"`\n" or not re.fullmatch(rb"[0-9]+ *", header[48:58]):
                raise ValueError("invalid native archive header")
            size = int(header[48:58])
            if size > before.st_size - stream.tell():
                raise ValueError("native archive payload exceeds remaining bytes")
            raw_name = header[:16].rstrip(b" ")
            if raw_name == b"//":
                raise ValueError("unsupported GNU archive name table")
            name_size = 0
            if raw_name.startswith(b"#1/"):
                if not re.fullmatch(rb"#1/[0-9]+", raw_name):
                    raise ValueError("invalid BSD archive name")
                name_size = int(raw_name[3:])
                if name_size > MAX_NAME_BYTES or name_size > size:
                    raise ValueError("native archive name exceeds bound")
                raw_name = read(name_size).rstrip(b"\x00")
            elif raw_name not in {b"/"}:
                raw_name = raw_name.removesuffix(b"/")
            if not raw_name or len(raw_name) > MAX_NAME_BYTES or b"\x00" in raw_name or raw_name == b"//" or raw_name.startswith(b"/") and raw_name != b"/":
                raise ValueError("unsupported native archive name")
            name = raw_name.decode("utf-8", errors="strict")
            if name != "/" and ("/" in name or "\\" in name or any(ord(c) < 32 or ord(c) == 127 for c in name)):
                raise ValueError("invalid native archive member name")
            payload_bytes = size - name_size
            remaining, payload_hash = payload_bytes, hashlib.sha256()
            while remaining:
                data = read(min(remaining, 65536)); payload_hash.update(data); remaining -= len(data)
            members.append({"ordinal": len(members), "name": name, "bytes": payload_bytes, "sha256": payload_hash.hexdigest()})
            if size % 2 and read(1) != b"\n":
                raise ValueError("invalid native archive alignment")
        unchanged(stream, before)
    return {"bytes": before.st_size, "sha256": digest.hexdigest(), "members": members}


def linker_objects(path):
    """Parse only the ld object table; symbols/sections are hashed, not retained."""
    stream, before = regular_stream(path, MAX_MAP_BYTES)
    digest, objects = hashlib.sha256(), []
    architecture, active, completed, total = None, False, False, 0
    with stream:
        while line := stream.readline(65537):
            if len(line) > 65536:
                raise ValueError("linker map line exceeds bound")
            total += len(line)
            if total > MAX_MAP_BYTES:
                raise ValueError("linker map exceeds byte bound")
            digest.update(line)
            if completed:
                continue
            text = line.decode("utf-8", errors="strict").rstrip("\n")
            if text.startswith("# Arch: "):
                if architecture is not None:
                    raise ValueError("duplicate linker architecture")
                architecture = text[8:]
            elif text == "# Object files:":
                if active:
                    raise ValueError("duplicate linker object table")
                active = True
            elif text == "# Sections:" and active:
                active, completed = False, True
            elif active:
                match = re.fullmatch(r"\[\s*([0-9]+)\]\s+(.+)", text)
                if not match or int(match[1]) != len(objects):
                    raise ValueError("invalid/nonsequential linker object table")
                if len(objects) == MAX_OBJECTS:
                    raise ValueError("linker map exceeds object bound")
                if len(match[2].encode("utf-8")) > MAX_NAME_BYTES:
                    raise ValueError("linker object name exceeds bound")
                objects.append(match[2])
        unchanged(stream, before)
    if not completed or architecture not in {"arm64", "x86_64"} or not objects or objects[0] != "linker synthesized":
        raise ValueError("incomplete/unsupported Mac linker map")
    return {"architecture": architecture, "bytes": before.st_size, "sha256": digest.hexdigest(), "objects": objects}


def selected_archive_members(map_record, archive_path, archive_record):
    """Map names to exact member bytes; caller must bind map to its build/binary."""
    archive_path = Path(archive_path).resolve()
    by_name = {}
    for member in archive_record["members"]:
        by_name.setdefault(member["name"], []).append(member)
    selected = []
    for index, name in enumerate(map_record["objects"]):
        match = re.fullmatch(r"(.+\.a)\(([^()]+)\)", name)
        if match and Path(match[1]).is_absolute() and Path(match[1]).resolve() == archive_path:
            matches = by_name.get(match[2], [])
            if len(matches) != 1:
                raise ValueError("selected native member is absent or ambiguous: " + match[2])
            selected.append({"linker_object_index": index, **matches[0]})
    return selected
