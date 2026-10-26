"""Bound explicit app notice payloads to pre-sign bytes, without approving rights."""
import hashlib
import os
from pathlib import Path, PurePosixPath
import re

from native_gpu import read_regular, sha_value, strict_json

INVENTORY = "INVENTORY.json"
DIRECTORY = "THIRD_PARTY_NOTICES"
MAX_MANIFEST_BYTES = 1 << 20
MAX_FILE_BYTES = 4 << 20
MAX_TOTAL_BYTES = 16 << 20
MAX_FILES = 1024
MAX_COMPONENTS = 512


def notice_name(name):
    if (type(name) is not str or len(name) > 1024 or "\\" in name
            or not name.startswith("NOTICES/") or len(name.split("/")) > 16
            or any(not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]*", part)
                   or part.endswith(".") for part in name.split("/"))):
        raise ValueError("invalid app notice path")
    return name


def regular_path(root, name):
    current = root
    for part in PurePosixPath(name).parts:
        current /= part
        if current.is_symlink():
            raise ValueError("app notices cannot contain symlinks")
    return current


def exact_members(root, names):
    """Only declared files/directories are visited; refuse extras immediately."""
    directories = set()
    for name in names:
        directories.update(str(parent) for parent in PurePosixPath(name).parents
                           if str(parent) != ".")
    pending, observed = [(root, "")], set()
    while pending:
        path, prefix = pending.pop()
        with os.scandir(path) as entries:
            for entry in entries:
                name = prefix + entry.name
                if entry.is_symlink():
                    raise ValueError("app notices cannot contain symlinks")
                if name in directories and entry.is_dir(follow_symlinks=False):
                    pending.append((Path(entry.path), name + "/"))
                elif name in names and entry.is_file(follow_symlinks=False):
                    observed.add(name)
                else:
                    raise ValueError("app notice member coverage mismatch: " + name)
    if observed != names:
        raise ValueError("app notice member coverage mismatch")


def load(root, binary_sha256):
    """Return bounded admitted bytes; source mutations cannot alter that snapshot.

    Paths and bytes are sampled, not isolated against adversarial concurrent
    directory replacement. No file is interpreted as an instruction or executed.
    """
    root = Path(root)
    if root.is_symlink() or not root.is_dir():
        raise ValueError("app notice bundle must be a directory without symlinks")
    root = root.resolve()
    raw = read_regular(regular_path(root, INVENTORY), MAX_MANIFEST_BYTES)
    inventory = strict_json(raw)
    keys = {"schema", "target", "artifact_sha256", "distribution_audit",
            "applicability", "scope", "files", "components"}
    if (type(inventory) is not dict or set(inventory) != keys
            or type(inventory["schema"]) is not int or inventory["schema"] != 1
            or inventory["target"] != "darwin"
            or not sha_value(inventory["artifact_sha256"])
            or inventory["artifact_sha256"] != binary_sha256
            or inventory["distribution_audit"] != "pending"
            or inventory["applicability"] != "review-required"
            or type(inventory["scope"]) is not str
            or not 0 < len(inventory["scope"]) <= 4096):
        raise ValueError("invalid app notice inventory or executable binding")
    files, components = inventory["files"], inventory["components"]
    if (type(files) is not dict or not 0 < len(files) <= MAX_FILES
            or type(components) is not list or not 0 < len(components) <= MAX_COMPONENTS):
        raise ValueError("invalid app notice coverage or count bound")
    folded = {INVENTORY.casefold(): INVENTORY}
    declared_total = len(raw)
    for name, identity in files.items():
        notice_name(name)
        for prefix in [PurePosixPath(name), *PurePosixPath(name).parents]:
            spelling = str(prefix)
            if spelling == ".":
                continue
            key = spelling.casefold()
            if key in folded and folded[key] != spelling:
                raise ValueError("case-colliding app notice paths")
            folded[key] = spelling
        if (type(identity) is not dict or set(identity) != {"bytes", "sha256"}
                or type(identity["bytes"]) is not int
                or not 0 <= identity["bytes"] <= MAX_FILE_BYTES
                or not sha_value(identity["sha256"])):
            raise ValueError("invalid app notice file identity or byte bound")
        declared_total += identity["bytes"]
    if declared_total > MAX_TOTAL_BYTES:
        raise ValueError("app notice aggregate byte bound exceeded")
    referenced, identities = set(), set()
    for component in components:
        if (type(component) is not dict or set(component) != {"id", "notice_files"}
                or type(component["id"]) is not str or not 0 < len(component["id"]) <= 512
                or any(ord(c) < 32 or ord(c) == 127 for c in component["id"])
                or component["id"] in identities
                or type(component["notice_files"]) is not list
                or not 0 < len(component["notice_files"]) <= MAX_FILES):
            raise ValueError("invalid app notice component")
        identities.add(component["id"])
        local = set()
        for name in component["notice_files"]:
            if type(name) is not str or name not in files or name in local:
                raise ValueError("invalid app notice component reference")
            local.add(name)
        referenced.update(local)
    if referenced != set(files):
        raise ValueError("unreferenced app notice file")
    names = {INVENTORY, *files}
    exact_members(root, names)
    payloads = {INVENTORY: raw}
    for name, identity in sorted(files.items()):
        content = read_regular(regular_path(root, name), identity["bytes"])
        if len(content) != identity["bytes"] or hashlib.sha256(content).hexdigest() != identity["sha256"]:
            raise ValueError("app notice checksum or byte count mismatch")
        payloads[name] = content
    exact_members(root, names)
    record = {"directory": DIRECTORY, "inventory_sha256": hashlib.sha256(raw).hexdigest(),
              "artifact_sha256": binary_sha256, "files": len(files), "components": len(components),
              "bytes": declared_total, "applicability": "review-required",
              "distribution_audit": "pending", "trust": "caller-selected-not-attested"}
    return record, payloads


def copy(source, destination, binary_sha256):
    record, payloads = load(source, binary_sha256)
    destination.mkdir()
    for name, content in payloads.items():
        path = destination / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(content)
    verify(destination, binary_sha256, record)
    return record


def verify(root, binary_sha256, expected):
    record, _ = load(root, binary_sha256)
    if record != expected:
        raise ValueError("packaged app notices changed")
