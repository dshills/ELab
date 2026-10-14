#!/bin/sh
set -eu
ELAB_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
export GOWORK=off
export GOCACHE="${ELAB_GOCACHE:-$ELAB_ROOT/build/go-cache}"
export STATICCHECK_CACHE="$ELAB_ROOT/build/staticcheck-cache"
export CGO_LDFLAGS_ALLOW='-Wl,-rpath,@executable_path/native'
mkdir -p "$ELAB_ROOT/build/tools"
cd "$ELAB_ROOT/tools"
go build -mod=readonly -o "$ELAB_ROOT/build/tools/staticcheck" honnef.co/go/tools/cmd/staticcheck
go build -mod=readonly -o "$ELAB_ROOT/build/tools/goimports" golang.org/x/tools/cmd/goimports
cd "$ELAB_ROOT"
ELAB_UNFORMATTED=$("$ELAB_ROOT/build/tools/goimports" -l cmd desktop internal transport workbench)
if [ -n "$ELAB_UNFORMATTED" ]; then
    printf '%s\n' "$ELAB_UNFORMATTED"
    exit 1
fi
"$ELAB_ROOT/build/tools/staticcheck" ./...
cd "$ELAB_ROOT/desktop"
"$ELAB_ROOT/build/tools/staticcheck" ./...
