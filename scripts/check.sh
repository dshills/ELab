#!/bin/sh
set -eu
ELAB_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
ELAB_QUALITY_PROFILE=${ELAB_QUALITY_PROFILE:-native}
ELAB_QUALITY_ARTIFACTS=${ELAB_QUALITY_ARTIFACTS:-"$ELAB_ROOT/build/quality-$(date -u +%Y%m%dT%H%M%SZ)-$$"}
exec python3 -B "$ELAB_ROOT/scripts/quality.py" --profile "$ELAB_QUALITY_PROFILE" --artifacts "$ELAB_QUALITY_ARTIFACTS"
