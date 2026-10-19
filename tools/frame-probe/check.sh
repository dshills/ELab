#!/bin/sh
# Compilation and pure checks only. This never invokes --capture.
set -eu
ELAB_FRAME_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$ELAB_FRAME_ROOT"
mkdir -p build/frame-probe-tools
xcrun swiftc -swift-version 6 -parse-as-library -warnings-as-errors \
  -target arm64-apple-macosx14.0 \
  -module-cache-path build/frame-probe-tools/module-cache \
  tools/frame-probe/main.swift tools/frame-probe/journal_checks.swift \
  -o build/frame-probe-tools/frame-probe
build/frame-probe-tools/frame-probe --self-test
