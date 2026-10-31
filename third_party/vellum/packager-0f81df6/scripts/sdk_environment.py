"""Controlled Go/native child environment for SDK verification and packaging.

This reduces ambient overrides and unrelated tokens; it is not a process sandbox.
"""
import os

CHILD_ENVIRONMENT = {
    "PATH", "HOME", "USERPROFILE", "SYSTEMROOT", "WINDIR", "SYSTEMDRIVE",
    "APPDATA", "LOCALAPPDATA", "TEMP", "TMP", "TMPDIR", "LANG", "LC_ALL", "LC_CTYPE",
    "GOPATH", "GOCACHE", "GOMODCACHE", "GOMAXPROCS", "GOMEMLIMIT", "GORACE",
    "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy", "SSL_CERT_FILE", "SSL_CERT_DIR",
}
RUNTIME_ENVIRONMENT = {
    "DISPLAY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR", "XAUTHORITY", "DBUS_SESSION_BUS_ADDRESS",
    "GPU_DEVICE", "GPU_COMPUTE_DEVICE", "VK_ICD_FILENAMES", "VK_DRIVER_FILES", "LIBGL_ALWAYS_SOFTWARE",
}

GO_CONFIGURATION = {
    "GOWORK": "off", "GOENV": "off", "GOFLAGS": "-mod=readonly", "GOTOOLCHAIN": "local",
    "GOPROXY": "https://proxy.golang.org,direct", "GOSUMDB": "sum.golang.org",
    "GONOSUMDB": "", "GONOPROXY": "", "GOINSECURE": "", "GOPRIVATE": "",
    "CGO_ENABLED": "1", "CGO_LDFLAGS_ALLOW": r"^-Wl,-rpath,@executable_path/native$",
}


def child_environment(runtime=False):
    allowed = CHILD_ENVIRONMENT | (RUNTIME_ENVIRONMENT if runtime else set())
    return {**{name: value for name, value in os.environ.items() if name in allowed}, **GO_CONFIGURATION}
