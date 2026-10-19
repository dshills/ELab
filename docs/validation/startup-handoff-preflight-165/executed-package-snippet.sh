ELAB_SDK_CACHE=$(python3 -B scripts/prepare-vellum.py | python3 -c 'import json, sys; print(json.load(sys.stdin)["sdk"])')
python3 -B "$ELAB_SDK_CACHE/scripts/package-app.py" --binary /Users/dshills/Development/projects/ELab/build/first-version-165-tools/build/elab \
  --output /private/tmp/elab-readme-packaging-165/ELab-local.app --name ELab --identifier com.dshills.elab \
  --version 0.0.0 --build-version 1 --platform darwin
