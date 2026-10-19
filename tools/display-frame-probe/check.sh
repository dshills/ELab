#!/bin/sh
# Offline compilation and synthetic checks only: no UI/access preflight/capture.
set -eu
ELAB_DISPLAY_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$ELAB_DISPLAY_ROOT"
mkdir -p build/display-frame-probe-tools
xcrun swiftc -swift-version 6 -parse-as-library -warnings-as-errors \
  -target arm64-apple-macosx14.0 -module-cache-path build/display-frame-probe-tools/module-cache \
  tools/display-frame-probe/main.swift tools/display-frame-probe/support.swift \
  tools/display-frame-probe/geometry.swift tools/display-frame-probe/collector.swift \
  tools/display-frame-probe/checks.swift tools/display-frame-probe/journal_checks.swift -o build/display-frame-probe-tools/display-frame-probe
build/display-frame-probe-tools/display-frame-probe --self-test
