#!/bin/sh
set -eu
ELAB_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ELAB_ROOT"
export GOWORK=off
export GOCACHE="${ELAB_GOCACHE:-$ELAB_ROOT/build/go-cache}"
export CGO_LDFLAGS_ALLOW='-Wl,-rpath,@executable_path/native'
./scripts/lint.sh
python3 -m unittest discover -s scripts -p 'test_*.py'
go test -race ./...
go vet ./...
cd desktop
go test -race ./...
go vet ./...
