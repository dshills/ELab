#!/usr/bin/env python3
"""Verify and materialize ELab's pinned Vellum SDK from public source."""

import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[1]
REPOSITORY = "https://github.com/dshills/vellum"
BOOTSTRAP = {
    "scripts/check-sdk.py", "scripts/child_process.py", "scripts/sdk_environment.py",
    "scripts/native_gpu.py", "scripts/native_archive.py", "scripts/native_linkers.py",
}
MODULES = {
    "github.com/dshills/vellum": "",
    "github.com/dshills/vellum/drivers/cogent": "drivers/cogent",
    "cogentcore.org/core": "drivers/cogent/build/upstream",
    "github.com/go-text/typesetting": "drivers/cogent/build/typesetting",
}
WORKER = r'''
import importlib.util, json, pathlib, sys
bootstrap, archive, digest, destination = sys.argv[1:]
sys.path.insert(0, bootstrap)
spec = importlib.util.spec_from_file_location("pinned_vellum_integrity", pathlib.Path(bootstrap) / "check-sdk.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
verified = module.verify_archive(pathlib.Path(archive), digest)
root = module.extract_verified(pathlib.Path(archive), pathlib.Path(destination), verified)
print(json.dumps({"root": str(root)}))
'''


def digest(path):
    result = hashlib.sha256()
    with path.open("rb") as source:
        while block := source.read(1024 * 1024):
            result.update(block)
    return result.hexdigest()


def relative(name):
    if not isinstance(name, str) or not name or "\\" in name:
        raise ValueError("invalid SDK relative path")
    value = PurePosixPath(name)
    if value.is_absolute() or ".." in value.parts or value.as_posix() != name:
        raise ValueError("invalid SDK relative path")
    return Path(*value.parts)


def regular(path):
    if path.is_symlink() or not path.is_file():
        raise ValueError(f"SDK requires a regular file: {path}")


def load_pin(root):
    pin = json.loads((root / "third_party/vellum/sdk.lock.json").read_text())
    if pin.get("schema") != 2 or pin.get("private_build") is not True:
        raise ValueError("unsupported SDK pin")
    for name in ["sha256", "manifest_sha256"]:
        if not re.fullmatch(r"[0-9a-f]{64}", pin.get(name, "")):
            raise ValueError("invalid SDK digest")
    if not re.fullmatch(r"[0-9a-f]{40}", pin.get("source_commit", "")):
        raise ValueError("invalid SDK source commit")
    if (pin.get("repository") != REPOSITORY
            or not re.fullmatch(r"[0-9a-f]{40}", pin.get("upstream_commit", ""))):
        raise ValueError("invalid public SDK repository/commit")
    patch = pin.get("compatibility_patch")
    if patch is not None:
        if (not isinstance(patch, dict) or set(patch) != {"path", "sha256"}
                or not re.fullmatch(r"[0-9a-f]{64}", patch.get("sha256", ""))
                or relative(patch.get("path")).parent != Path("third_party/vellum/patches")):
            raise ValueError("invalid SDK compatibility patch pin")
        path = root / relative(patch["path"])
        regular(path)
        if digest(path) != patch["sha256"]:
            raise ValueError("SDK compatibility patch differs from its trusted pin")
    elif pin["source_commit"] != pin["upstream_commit"]:
        raise ValueError("derived SDK source requires a pinned compatibility patch")
    if pin.get("module") != "github.com/dshills/vellum" or set(pin.get("bootstrap", {})) != BOOTSTRAP:
        raise ValueError("invalid SDK module/bootstrap pin")
    if any(not re.fullmatch(r"[0-9a-f]{64}", value) for value in pin["bootstrap"].values()):
        raise ValueError("invalid SDK bootstrap digest")
    if not isinstance(pin.get("files"), int) or not 1 <= pin["files"] <= 20_000:
        raise ValueError("invalid SDK file count")
    if not isinstance(pin.get("archive_bytes"), int) or not 1 <= pin["archive_bytes"] <= 512 * 1024 * 1024:
        raise ValueError("invalid SDK archive size")
    if relative(pin["archive"]).parent != Path("third_party/vellum"):
        raise ValueError("SDK archive must be in third_party/vellum")
    cache = relative(pin["cache"])
    if cache.parent != Path("build") or not cache.name.startswith(".vellum-sdk-"):
        raise ValueError("SDK cache must be an owned build directory")
    if relative(pin["prefix"]).parts != (pin["prefix"],) or pin["prefix"] != "vellum-" + pin["version"]:
        raise ValueError("invalid SDK archive prefix")
    return pin


def verify_archive_identity(archive, pin):
    regular(archive)
    if archive.stat().st_size != pin["archive_bytes"] or digest(archive) != pin["sha256"]:
        raise ValueError("SDK archive differs from its trusted pin")


def verify_cache(cache, pin):
    if cache.is_symlink() or not cache.is_dir():
        raise ValueError("SDK cache is not a directory")
    manifest_path = cache / "SDK_MANIFEST.json"
    regular(manifest_path)
    if manifest_path.stat().st_size > 8 * 1024 * 1024 or digest(manifest_path) != pin["manifest_sha256"]:
        raise ValueError("SDK cache manifest differs from its trusted pin")
    manifest = json.loads(manifest_path.read_bytes())
    source, release = manifest.get("source", {}), manifest.get("release", {})
    files = manifest.get("files", {})
    if manifest.get("schema") != 1 or manifest.get("private_build") is not True or source.get("clean") is not True or source.get("commit") != pin["source_commit"]:
        raise ValueError("SDK cache has the wrong source identity")
    if release.get("module") != pin["module"] or release.get("version") != pin["version"] or len(files) != pin["files"]:
        raise ValueError("SDK cache has the wrong module/file inventory")
    required = {(Path(path) / "go.mod").as_posix() for path in MODULES.values()}
    if not required.issubset(files) or "PRIVATE_BUILD.txt" not in files:
        raise ValueError("SDK cache lacks required modules/private-build metadata")
    actual = set()
    for directory, directories, names in os.walk(cache, followlinks=False):
        parent = Path(directory)
        if any((parent / name).is_symlink() for name in directories):
            raise ValueError("SDK cache contains a directory symlink")
        for name in names:
            path = parent / name
            regular(path)
            actual.add(path.relative_to(cache).as_posix())
    if actual != set(files) | {"SDK_MANIFEST.json"}:
        raise ValueError("SDK cache file inventory changed; preserve it before preparing a fresh copy")
    for name, expected in files.items():
        if not re.fullmatch(r"[0-9a-f]{64}", expected) or digest(cache / relative(name)) != expected:
            raise ValueError(f"SDK cache file changed: {name}; preserve edits before preparing a fresh copy")


def verify_replacements(root, cache):
    env = dict(os.environ, GOWORK="off", GOFLAGS="", GO111MODULE="on")
    result = subprocess.run(["go", "mod", "edit", "-json"], cwd=root / "desktop", env=env,
                            capture_output=True, text=True, check=True, timeout=30)
    replacements = json.loads(result.stdout).get("Replace", [])
    for name, subdir in MODULES.items():
        matches = [entry for entry in replacements if entry["Old"]["Path"] == name]
        if len(matches) != 1 or matches[0]["Old"].get("Version") or matches[0]["New"].get("Version"):
            raise ValueError(f"SDK module needs one unversioned local replacement: {name}")
        path = (root / "desktop" / matches[0]["New"]["Path"]).resolve()
        if path != (cache / subdir).resolve():
            raise ValueError(f"SDK module replacement differs from the pin: {name}")


def prepare_sdk(root=ROOT):
    root = root.resolve()
    pin = load_pin(root)
    archive, cache = root / relative(pin["archive"]), root / relative(pin["cache"])
    if not cache.resolve().is_relative_to(root):
        raise ValueError("SDK cache must stay inside the project")
    verify_archive_identity(archive, pin)  # Before reading or executing bootstrap code.
    verify_replacements(root, cache)
    if cache.exists() or cache.is_symlink():
        verify_cache(cache, pin)
        return cache
    cache.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".vellum-sdk-stage-", dir=cache.parent) as temporary:
        stage = Path(temporary)
        bootstrap = stage / "bootstrap"
        bootstrap.mkdir()
        # The complete archive is already trusted by digest. Extract only the
        # explicitly pinned verifier sources; never execute a path from
        # archive metadata or use tarfile.extractall for the SDK payload.
        with tarfile.open(archive, "r:gz") as source:
            for name in sorted(BOOTSTRAP):
                member = source.getmember(pin["prefix"] + "/" + name)
                if not member.isfile() or member.size > 1024 * 1024:
                    raise ValueError("invalid SDK bootstrap member")
                data = source.extractfile(member).read()
                if hashlib.sha256(data).hexdigest() != pin["bootstrap"][name]:
                    raise ValueError("SDK bootstrap checksum mismatch")
                (bootstrap / Path(name).name).write_bytes(data)
        payload = stage / "payload"
        result = subprocess.run([sys.executable, "-I", "-c", WORKER, str(bootstrap), str(archive),
                                 pin["sha256"], str(payload)], capture_output=True, text=True,
                                check=True, timeout=120)
        extracted = Path(json.loads(result.stdout)["root"])
        if extracted != payload / pin["prefix"]:
            raise ValueError("SDK verifier returned an unexpected root")
        verify_cache(extracted, pin)
        try:
            extracted.rename(cache)
        except OSError:
            # A concurrent preparer may have published the same complete SDK.
            # Never replace or delete an existing cache, including local edits.
            if not cache.exists():
                raise
            verify_cache(cache, pin)
    return cache


def main():
    try:
        path = prepare_sdk()
        print(json.dumps({"passed": True, "sdk": str(path), "source": load_pin(ROOT)["source_commit"]}))
    except (OSError, ValueError, KeyError, subprocess.SubprocessError, tarfile.TarError) as error:
        print(f"Vellum SDK preparation failed: {error}", file=sys.stderr)
        if isinstance(error, subprocess.CalledProcessError) and error.stderr:
            print(error.stderr[-4000:], file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
