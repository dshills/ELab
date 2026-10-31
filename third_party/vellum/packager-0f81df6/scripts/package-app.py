#!/usr/bin/env python3
"""Package a prebuilt app; public Mac candidates require audit approval and notices."""
import argparse
import importlib.util
import hashlib
import gzip
import json
import os
from pathlib import Path
import platform
import plistlib
import posixpath
import re
import shutil
import stat
import tarfile
import tempfile
import zipfile

import sys

if __name__ == "__main__":
    sys.dont_write_bytecode = True

import app_notices
import app_icons
import macos_deployment

from native_archive import regular_stream, unchanged

from artifact_io import publish_exclusive
from child_process import run_child, check_child_output
from sdk_environment import child_environment

NATIVE_TOOL_TIMEOUT_SECONDS = 60

ROOT = Path(__file__).resolve().parents[1]


def load_release_policy():
    path = Path(__file__).with_name("release-sdk.py")
    if path.is_symlink() or not path.is_file():
        raise ValueError("public app packaging requires its release policy script")
    spec = importlib.util.spec_from_file_location("vellum_app_release_policy", path)
    if spec is None or spec.loader is None:
        raise ValueError("public app release policy loader is unavailable")
    policy = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(policy)
    return policy


def valid_name(value):
    if not isinstance(value, str) or value in {"", ".", ".."} or value.upper() == "APP_MANIFEST.JSON" or any(c in value for c in '/\\:*?"<>|') or any(ord(c) < 32 or ord(c) == 127 for c in value) or value.endswith((" ", ".")) or value.startswith("-"):
        return False
    stem = value.split(".", 1)[0].upper()
    return stem not in {"CON", "PRN", "AUX", "NUL", *("COM" + str(i) for i in range(1, 10)), *("LPT" + str(i) for i in range(1, 10))}


def validate(binary, name, identifier, version, target, identity, build_version):
    if binary.is_symlink() or not binary.is_file():
        raise ValueError("app binary must be a regular file")
    if not valid_name(name) or not re.fullmatch(r"[A-Za-z0-9]+(?:[.-][A-Za-z0-9]+)+",identifier):
        raise ValueError("invalid app name or bundle identifier")
    if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+",version):
        raise ValueError("app version must be numeric major.minor.patch")
    if not re.fullmatch(r"[0-9]+(?:\.[0-9]+){0,2}", build_version):
        raise ValueError("build version must contain one to three numeric components")
    with binary.open("rb") as source:
        magic = source.read(4)
    expected = {"darwin":{b"\xcf\xfa\xed\xfe",b"\xfe\xed\xfa\xcf",b"\xca\xfe\xba\xbe"},"linux":{b"\x7fELF"}}
    if target == "windows":
        with binary.open("rb") as source:
            header = source.read(64)
            offset = int.from_bytes(header[60:64],"little") if len(header)==64 else 0
            source.seek(offset)
            signature = source.read(4)
        if not magic.startswith(b"MZ") or offset < 64 or signature != b"PE\x00\x00":
            raise ValueError("Windows app requires a PE executable")
    elif magic not in expected.get(target,set()):
        raise ValueError("binary format does not match app target")
    if identity is not None and target != "darwin":
        raise ValueError("signing is implemented only for explicitly selected macOS identities")
    if target == "darwin" and platform.system() != "Darwin":
        raise ValueError("macOS dependency inspection/signing needs a macOS host")


def darwin_dependencies(binary):
    output = check_child_output(["otool","-L",str(binary)],text=True, env=child_environment(), timeout=NATIVE_TOOL_TIMEOUT_SECONDS)
    dependencies = set()
    for line in output.splitlines():
        if not line.startswith(("\t", " ")):
            continue  # Each architecture of a universal binary has its own header.
        value = line.strip().split(" (",1)[0]
        if value and not posixpath.normpath(value).startswith(("/System/Library/","/usr/lib/")):
            dependencies.add(value)
    if dependencies:
        raise ValueError("non-system native dependencies require an audited bundle closure: " + ", ".join(sorted(dependencies)))


def normalize_layout(stage):
    stage.chmod(0o755)
    for path in stage.rglob("*"):
        path.chmod(0o755 if path.is_dir() or path.stat().st_mode & 0o111 else 0o644)


def normalized_tar_member(member, executable_name):
    member.uid = member.gid = member.mtime = 0
    member.uname = member.gname = ""
    member.mode = 0o755 if member.isdir() or member.name == executable_name else 0o644
    return member


def application_archive(stage, output, name, target):
    if target == "windows":
        with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED) as archive:
            for source in [stage, *sorted(stage.rglob("*"))]:
                relative = source.relative_to(stage).as_posix()
                filename = name + ("/" + relative if relative != "." else "")
                if source.is_dir():
                    filename += "/"
                record = zipfile.ZipInfo(filename, (1980, 1, 1, 0, 0, 0))
                record.create_system = 3
                record.compress_type = zipfile.ZIP_DEFLATED
                mode = 0o040755 if source.is_dir() else (0o100755 if source.name == name + ".exe" else 0o100644)
                record.external_attr = mode << 16
                if source.is_dir():
                    record.external_attr |= 0x10
                archive.writestr(record, b"" if source.is_dir() else source.read_bytes())
    else:
        with output.open("wb") as destination:
            with gzip.GzipFile(fileobj=destination, mode="wb", filename="", mtime=0) as zipped:
                with tarfile.open(fileobj=zipped, mode="w|", format=tarfile.PAX_FORMAT) as archive:
                    archive.add(stage, arcname=name, filter=lambda member: normalized_tar_member(member, name + "/" + name))


def binary_digest(path):
    # Stream the admitted regular file, refusing growth or observed mutation.
    stream, before = regular_stream(path, (1 << 63) - 1)
    digest, count = hashlib.sha256(), 0
    with stream:
        while chunk := stream.read(64 * 1024):
            count += len(chunk)
            if count > before.st_size:
                raise ValueError("app executable changed while hashing")
            digest.update(chunk)
        unchanged(stream, before)
    if count != before.st_size:
        raise ValueError("app executable changed while hashing")
    return digest.hexdigest()


def document_type_info(records):
    if records is None:
        return {}
    if not isinstance(records, list) or not 1 <= len(records) <= 8:
        raise ValueError("document types require one to eight records")
    extensions, identifiers, types, exports = set(), set(), [], []
    for record in records:
        if not isinstance(record, dict) or set(record) != {"extension", "uti", "name"}:
            raise ValueError("invalid document type fields")
        extension, uti, name = record["extension"], record["uti"], record["name"]
        if (not isinstance(extension, str) or not re.fullmatch(r"[a-z0-9]{1,16}", extension)
                or not isinstance(uti, str) or len(uti) > 160 or not re.fullmatch(r"[A-Za-z0-9]+(?:[.-][A-Za-z0-9]+)+", uti)
                or not isinstance(name, str) or not 1 <= len(name) <= 80 or any(ord(c) < 32 for c in name)
                or extension in extensions or uti in identifiers):
            raise ValueError("invalid or duplicate document type")
        extensions.add(extension); identifiers.add(uti)
        types.append({"CFBundleTypeName": name, "CFBundleTypeRole": "Editor", "LSHandlerRank": "Alternate",
                      "CFBundleTypeExtensions": [extension], "LSItemContentTypes": [uti]})
        exports.append({"UTTypeIdentifier": uti, "UTTypeDescription": name, "UTTypeConformsTo": ["public.data"],
                        "UTTypeTagSpecification": {"public.filename-extension": [extension]}})
    return {"VellumHandlesOpenDocuments": True, "CFBundleDocumentTypes": types, "UTExportedTypeDeclarations": exports}


def verify_app(stage, name, binary_sha256, manifest_bytes, notice_record, icon_record=None):
    if binary_digest(stage/"Contents/MacOS"/name) != binary_sha256:
        raise ValueError("packaged app executable changed")
    resources = stage/"Contents/Resources"
    if app_notices.read_regular(resources/"APP_MANIFEST.json", len(manifest_bytes)) != manifest_bytes:
        raise ValueError("packaged app manifest changed")
    app_icons.verify(resources, icon_record)
    data=json.loads(manifest_bytes)
    declared = document_type_info(data.get("document_types"))
    info = plistlib.loads((stage/"Contents/Info.plist").read_bytes())
    deployment=data.get("macos_deployment")
    macos_deployment.verify(info,deployment,macos_deployment.inspect(stage/"Contents/MacOS"/name) if deployment is not None else None)
    if any(info.get(key) != value for key, value in declared.items()):
        raise ValueError("packaged document type metadata changed")
    if icon_record is not None and plistlib.loads((stage/"Contents/Info.plist").read_bytes()).get("CFBundleIconFile") != app_icons.ICON_NAME:
        raise ValueError("packaged app icon metadata changed")
    if notice_record is not None:
        app_notices.verify(resources/app_notices.DIRECTORY, notice_record["artifact_sha256"], notice_record)


def package(binary, output, name, identifier, version, target, identity=None, build_version="1", notice_bundle=None, *, public=False, icon=None, document_types=None):
    declared = document_type_info(document_types)
    if declared and target != "darwin":
        raise ValueError("document type registration is implemented only for macOS")
    if icon is not None and target != "darwin":
        raise ValueError("app icons are implemented only for macOS")
    if type(public) is not bool:
        raise ValueError("public mode must be boolean")
    if public and target != "darwin":
        raise ValueError("public app packaging is implemented only for macOS")
    if public and notice_bundle is None:
        raise ValueError("public app packaging requires explicit binary-bound notices")
    if notice_bundle is not None and target != "darwin":
        raise ValueError("app notice bundles are implemented only for macOS")
    validate(binary,name,identifier,version,target,identity,build_version)
    metadata_bytes = (ROOT / "release/version.json").read_bytes()
    metadata = json.loads(metadata_bytes)
    release_policy = load_release_policy() if public else None
    if public:
        checkout = Path(release_policy.git_output(ROOT, ["rev-parse", "--show-toplevel"], text=True).strip()).resolve()
        if checkout != ROOT.resolve():
            raise ValueError("public app packaging requires its source checkout root")
        release_source = release_policy.source_state(ROOT)
        release_policy.release_gates(ROOT, metadata, True)
        if release_policy.source_state(ROOT) != release_source:
            raise ValueError("public app release source changed during authorization")
    else:
        release_source = None
    def recheck_authorization():
        if not public:
            return
        if (ROOT / "release/version.json").read_bytes() != metadata_bytes:
            raise ValueError("public app release metadata changed during packaging")
        release_policy.release_gates(ROOT, metadata, True)
        if release_policy.source_state(ROOT) != release_source:
            raise ValueError("public app release source changed during packaging")
    sdk_release = metadata["version"]
    audit = "approved" if public else "pending"
    if output.exists() or output.is_symlink():
        raise ValueError("app output already exists")
    if target == "darwin":
        if output.suffix.lower() != ".app":
            raise ValueError("macOS app output must have a .app suffix")
    output.parent.mkdir(parents=True,exist_ok=True)
    with tempfile.TemporaryDirectory(prefix=".vellum-app-",dir=output.parent) as directory:
        stage = Path(directory)/("application.app" if target=="darwin" else name)
        stage.mkdir()
        payload = stage/"Contents/MacOS" if target=="darwin" else stage
        payload.mkdir(parents=True,exist_ok=True)
        executable = payload/(name+".exe" if target=="windows" else name)
        shutil.copyfile(binary,executable);executable.chmod(0o755)
        validate(executable,name,identifier,version,target,identity,build_version)
        if target == "darwin":
            darwin_dependencies(executable)
            deployment=macos_deployment.inspect(executable)
        manifest = {"schema":1,"name":name,"identifier":identifier,"version":version,"build_version":build_version,"sdk_release":sdk_release,"target":target,"source_binary_sha256":binary_digest(executable),"private_build":not public,"signed_identity":identity,"notarized":False,"native_distribution_audit":audit,"target_runtime":"unverified"}
        if document_types is not None:
            manifest["document_types"] = document_types
        if public:
            manifest["release_authorization"] = {
                "metadata_sha256": hashlib.sha256(metadata_bytes).hexdigest(),
                "source": release_source,
                "scope": "producer release-audit declaration; not independent binary-source attestation",
            }
        if target=="darwin":
            manifest["macos_deployment"]=deployment
            resources = stage/"Contents/Resources"
            resources.mkdir()
            if notice_bundle is not None:
                manifest["component_notices"] = app_notices.copy(notice_bundle, resources/app_notices.DIRECTORY, manifest["source_binary_sha256"])
            info={"CFBundlePackageType":"APPL","CFBundleName":name,"CFBundleDisplayName":name,"CFBundleExecutable":name,"CFBundleIdentifier":identifier,"CFBundleVersion":build_version,"CFBundleShortVersionString":version,"NSHighResolutionCapable":True}
            info["LSMinimumSystemVersion"]=deployment["minimum"]
            info.update(declared)
            if icon is not None:
                manifest["app_icon"] = app_icons.install(icon, resources)
                info["CFBundleIconFile"] = app_icons.ICON_NAME
            (stage/"Contents/Info.plist").write_bytes(plistlib.dumps(info,sort_keys=True))
            manifest_bytes = (json.dumps(manifest,indent=2)+"\n").encode()
            (resources/"APP_MANIFEST.json").write_bytes(manifest_bytes)
            normalize_layout(stage)
            notice_record = manifest.get("component_notices")
            verify_app(stage, name, manifest["source_binary_sha256"], manifest_bytes, notice_record, manifest.get("app_icon"))
            if identity is not None:
                run_child(["codesign","--force","--sign",identity,str(stage)],check=True, env=child_environment(), timeout=NATIVE_TOOL_TIMEOUT_SECONDS)
                run_child(["codesign","--verify","--strict",str(stage)],check=True, env=child_environment(), timeout=NATIVE_TOOL_TIMEOUT_SECONDS)
            packaged_sha256 = binary_digest(executable) if identity is not None else manifest["source_binary_sha256"]
            verify_app(stage, name, packaged_sha256, manifest_bytes, notice_record, manifest.get("app_icon"))
            # Recheck public authorization before output reservation. This remains
            # sampled local provenance, not protection against transient reversal.
            recheck_authorization()
            # Exclusive reservation; sampled identity guards preserve replacements.
            output.mkdir()
            owned = output.lstat()
            def still_owned():
                try:
                    current = output.lstat()
                    return stat.S_ISDIR(current.st_mode) and (current.st_dev, current.st_ino) == (owned.st_dev, owned.st_ino)
                except FileNotFoundError:
                    return False
            def require_owned():
                if not still_owned():
                    raise ValueError("app output directory was replaced")
            try:
                require_owned()
                output.chmod(0o755)
                for child in stage.iterdir():
                    require_owned()
                    shutil.move(str(child),output/child.name)
                require_owned()
                if identity is not None:
                    run_child(["codesign","--verify","--strict",str(output)],check=True, env=child_environment(), timeout=NATIVE_TOOL_TIMEOUT_SECONDS)
                verify_app(output, name, packaged_sha256, manifest_bytes, notice_record, manifest.get("app_icon"))
                recheck_authorization()
                require_owned()
            except BaseException:
                if still_owned():
                    shutil.rmtree(output)
                raise
        else:
            (stage/"APP_MANIFEST.json").write_text(json.dumps(manifest,indent=2)+"\n")
            artifact = Path(directory)/"application.archive"
            normalize_layout(stage)
            application_archive(stage, artifact, name, target)
            artifact.chmod(0o644)
            publish_exclusive(artifact,output)
    result = {"output":str(output),"private_build":not public,"target":target,"signed":identity is not None,"notarized":False,"native_distribution_audit":audit}
    if target == "darwin":
        result["packaged_binary_sha256"] = packaged_sha256
    return result


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary",required=True,type=Path)
    parser.add_argument("--output",required=True,type=Path)
    parser.add_argument("--name",required=True)
    parser.add_argument("--identifier",required=True)
    parser.add_argument("--version",required=True,help="numeric app release version: major.minor.patch")
    parser.add_argument("--build-version",default="1",help="numeric bundle build version (one to three components); increment for distribution")
    parser.add_argument("--platform",dest="target",choices=["darwin","windows","linux"],default=platform.system().lower())
    parser.add_argument("--notice-bundle",type=Path,help="explicit binary-bound original notice bundle for a macOS app; rights review remains pending")
    parser.add_argument("--icon",type=Path,help="macOS app icon: .icns or 1024x1024 PNG; embedded before signing")
    parser.add_argument("--document-types",type=Path,help="bounded JSON array of extension/uti/name registrations; existing default handlers are preserved")
    parser.add_argument("--public",action="store_true",help="require existing source release-audit gates and binary-bound notices; does not publish or notarize")
    parser.add_argument("--sign-identity",help="explicit macOS code-signing identity; '-' requests ad-hoc signing")
    args=parser.parse_args()
    records=None
    if args.document_types is not None:
        records=json.loads(app_notices.read_regular(args.document_types,64<<10))
    print(json.dumps(package(args.binary.absolute(),args.output.absolute(),args.name,args.identifier,args.version,args.target,args.sign_identity,args.build_version,args.notice_bundle,public=args.public,icon=args.icon,document_types=records),indent=2))


if __name__=="__main__":main()
