"""Vellum-authored, Mac-only SDK compatibility modules for WebGPU blank imports."""
from contextlib import contextmanager
import hashlib
from pathlib import Path
import tempfile

PREFIX = "github.com/oliverbestmann/webgpu/libs-"
DIRECTORIES = {PREFIX + target: "drivers/cogent/build/libs-" + target
               for target in ("android", "darwin", "ios", "linux", "windows")}
ROLE = "Vellum-authored SDK linker compatibility source; original upstream linker-module distributions are not selected"
POLICY = "darwin-arm64-cgo-only; other native targets fail compilation"
# This license applies only to the newly authored compatibility source below.
LICENSE = b'MIT License\n\nCopyright (c) 2026 Davin Hills\n\nPermission is hereby granted, free of charge, to any\nperson obtaining a copy of this software and associated\ndocumentation files (the "Software"), to deal in the\nSoftware without restriction, including without\nlimitation the rights to use, copy, modify, merge,\npublish, distribute, sublicense, and/or sell copies of\nthe Software, and to permit persons to whom the Software\nis furnished to do so, subject to the following\nconditions:\n\nThe above copyright notice and this permission notice\nshall be included in all copies or substantial portions\nof the Software.\n\nTHE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF\nANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED\nTO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A\nPARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT\nSHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY\nCLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION\nOF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR\nIN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER\nDEALINGS IN THE SOFTWARE.\n'


def sources():
    result = {}
    for module in DIRECTORIES:
        target = module.removeprefix(PREFIX)
        guard = {"linux": "linux && !android", "darwin": "darwin && !ios && (!arm64 || !cgo)"}.get(target, target)
        result[module] = {
            "go.mod": ("module " + module + "\n").encode(),
            "LICENSE": LICENSE,
            "package.go": b"// Copyright (c) 2026 Davin Hills\n// SPDX-License-Identifier: MIT\n\n// Package lib satisfies a WebGPU blank import in Vellum's Mac SDK.\npackage lib\n",
            "unsupported.go": ("//go:build " + guard + "\n\n// Copyright (c) 2026 Davin Hills\n// SPDX-License-Identifier: MIT\n\npackage lib\n\n// Refuse unsupported native targets before creating an executable.\nvar _ = VellumSDKRequiresDarwinARM64WithCGO\n").encode(),
        }
        if target == "darwin":
            result[module]["link.go"] = b'''//go:build darwin && !ios && arm64 && cgo

// Copyright (c) 2026 Davin Hills
// SPDX-License-Identifier: MIT

package lib

/*
#cgo LDFLAGS: ${SRCDIR}/arm64/libwgpu_native.a -framework Metal -framework QuartzCore
*/
import "C"
'''
    return result


def checksum(data):
    return hashlib.sha256(data).hexdigest()


def declaration():
    return {"schema": 1, "role": ROLE, "target_policy": POLICY,
            "modules": {module: {"directory": DIRECTORIES[module],
                                 "files": {name: checksum(data) for name, data in sorted(files.items())}}
                        for module, files in sources().items()}}


def replacements(gpu):
    if gpu is None:
        return {}
    if "owned_linkers" not in gpu:
        return {PREFIX + "darwin": DIRECTORIES[PREFIX + "darwin"]}
    if gpu["owned_linkers"] != declaration():
        raise ValueError("invalid owned linker source declaration")
    return dict(DIRECTORIES)


def validate(gpu, files):
    """Couple authored bytes and exact module paths to the SDK file manifest."""
    if gpu.get("owned_linkers") != declaration():
        raise ValueError("invalid owned linker source declaration")
    for module, bodies in sources().items():
        directory = DIRECTORIES[module]
        for name, data in bodies.items():
            if files.get(directory + "/" + name) != checksum(data):
                raise ValueError("owned linker source or license checksum mismatch")


@contextmanager
def inventory_tree(gpu, generated):
    """Inventory the exact authored graph that consumers receive, before archiving."""
    if gpu is None or "owned_linkers" not in gpu:
        yield None
        return
    with tempfile.TemporaryDirectory(prefix="vellum-owned-sdk-linkers-") as temporary:
        base = Path(temporary)
        directories = {}
        for module, files in sources().items():
            relative = DIRECTORIES[module]
            directory = base / relative
            directory.mkdir(parents=True)
            for name, data in files.items():
                if generated.get(relative + "/" + name) != data:
                    raise ValueError("inventory linker differs from SDK source")
                (directory / name).write_bytes(data)
            directories[module] = directory
        yield {"directories": directories, "declaration": gpu["owned_linkers"]}


def sdk_inventory(root, gpu):
    if gpu is None or "owned_linkers" not in gpu:
        return None
    return {"directories": {module: root / relative for module, relative in replacements(gpu).items()},
            "declaration": gpu["owned_linkers"]}


def inventory_item(root, module, context, *, copy_notices, max_notice_bytes):
    """Separate resolution metadata from the locally authored effective source."""
    name = module["Path"]
    if context is None or name not in context["directories"]:
        return None
    if context["declaration"] != declaration():
        raise ValueError("invalid inventory owned linker declaration")
    directory = context["directories"][name]
    effective = module.get("Replace", {})
    if (not effective.get("Dir") or Path(effective["Dir"]).resolve() != directory.resolve()
            or effective.get("Version") or effective.get("Sum")):
        raise ValueError("inventory graph does not select owned linker")
    from native_gpu import read_regular
    for filename, body in sources()[name].items():
        if read_regular(directory / filename, 65536) != body:
            raise ValueError("inventory owned linker bytes changed")
    if max_notice_bytes is not None and len(LICENSE) > max_notice_bytes:
        raise ValueError("notice exceeds inventory file bound: LICENSE")
    selected_sum = None
    sums = root / "drivers/cogent/go.sum"
    if sums.is_file():
        for line in sums.read_text().splitlines():
            fields = line.split()
            if len(fields) == 3 and fields[:2] == [name, module.get("Version")]:
                if selected_sum is not None and selected_sum != fields[2]:
                    raise ValueError("conflicting original linker resolution checksums")
                selected_sum = fields[2]
    notice = {"file": "LICENSE", "sdk_file": DIRECTORIES[name] + "/LICENSE",
              "sha256": checksum(LICENSE), "bytes": len(LICENSE)}
    return {"module": name, "version": None, "sum": None,
            "selected_requirement": {"version": module.get("Version"), "sum": selected_sum,
                                     "basis": "source module requirement/checksum; original distribution not selected"},
            "effective_source": {"kind": "vellum-owned-sdk-linker", "role": ROLE,
                                 "target_policy": POLICY, **declaration()["modules"][name]},
            "license_files": [notice], "module_license_files": ["LICENSE"],
            "status": "owned-source-declaration"}
