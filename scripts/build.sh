#!/bin/sh
set -eu
ELAB_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ELAB_ROOT"
export GOWORK=off
export GOCACHE="${ELAB_GOCACHE:-$ELAB_ROOT/build/go-cache}"
mkdir -p build
go build -o build/elab .
go build -o build/elab-agent ./cmd/elab-agent
go build -o build/elab-conformance ./cmd/elab-conformance
