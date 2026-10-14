#!/bin/sh
set -eu
ELAB_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ELAB_ROOT"
export GOWORK=off
export GOCACHE="${ELAB_GOCACHE:-$ELAB_ROOT/build/go-cache}"
export CGO_LDFLAGS_ALLOW='-Wl,-rpath,@executable_path/native'
mkdir -p build
go build -o build/elab-agent ./cmd/elab-agent
go build -o build/elab-conformance ./cmd/elab-conformance
cd desktop
go build -o ../build/elab .
