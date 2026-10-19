#!/bin/sh
# Compiles and checks clocks only. No capture/permission/window operation.
set -eu
ELAB_CLOCK_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$ELAB_CLOCK_ROOT"
mkdir -p build/frame-probe-clock
xcrun swiftc -swift-version 6 -parse-as-library -warnings-as-errors \
  -target arm64-apple-macosx14.0 \
  -module-cache-path build/frame-probe-tools/module-cache \
  tools/frame-probe-clock/main.swift -o build/frame-probe-clock/frame-probe-clock
build/frame-probe-clock/frame-probe-clock --check
