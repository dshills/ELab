#!/bin/sh
# Compilation and pure layout checks only; --show is never invoked here.
set -eu
ELAB_PATCH_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$ELAB_PATCH_ROOT"
mkdir -p build/display-calibration-fixture-tools
xcrun swiftc -swift-version 6 -parse-as-library -warnings-as-errors \
  -target arm64-apple-macosx14.0 -module-cache-path build/display-calibration-fixture-tools/module-cache \
  tools/display-calibration-fixture/main.swift tools/display-calibration-fixture/layout.swift \
  tools/display-frame-probe/support.swift tools/display-frame-probe/geometry.swift \
  -o build/display-calibration-fixture-tools/calibration-fixture
build/display-calibration-fixture-tools/calibration-fixture --self-test
