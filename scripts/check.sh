#!/bin/sh
set -eu
ELAB_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
case "$(uname -s)" in
    Darwin) ELAB_DEFAULT_PROFILE=macos ;;
    *) ELAB_DEFAULT_PROFILE=portable ;;
esac
ELAB_QUALITY_PROFILE=${ELAB_QUALITY_PROFILE:-$ELAB_DEFAULT_PROFILE}
ELAB_QUALITY_ARTIFACTS=${ELAB_QUALITY_ARTIFACTS:-"$ELAB_ROOT/build/quality-$(date -u +%Y%m%dT%H%M%SZ)-$$"}
exec python3 -B "$ELAB_ROOT/scripts/quality.py" --profile "$ELAB_QUALITY_PROFILE" --artifacts "$ELAB_QUALITY_ARTIFACTS"
