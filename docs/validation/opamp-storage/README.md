# Powered storage and recording consistency evidence

The [parent receipt](../opamp-storage-validation.json) binds the frozen sources, binaries, pinned SDK, results and limits. `quality.json` and the full logs retain ordinary build, formatting/imports/Staticcheck, race/vet, Python, native smoke and four-adapter runs. The canonical app bundles retain their previous byte/mode trees.

`mosfet-before.log.txt` retains 20 real failed refusal tests. `storage-before.log.txt` retains three more accepted-inconsistent-state cases. `public-before.log.txt` shows the buffered RLC numerical workflow reaching the same decoder gap. Later focused/full logs pass after repair. `mosfet-fixture-position-failure.log.txt` is a separate fixture error: overlapping test positions correctly failed schematic validation; only those fixture positions were corrected.

`conformance-report.json.gz` is the exact original JSON compressed losslessly. It includes four adapters' graphs, observations, 668 coarse/refined storage rows, 7,996 reference checks, 232 file-refusal outcomes and normalized public call traces. `sample-audit.json` records a fresh typed review that recomputes references/coverage and compares prior op-amp/NMOS numerical/command evidence. `review.go.txt` is its executed source. To reproduce from the repository root, choose a new owned directory:

```sh
mkdir build/.storage-report-review
cp docs/validation/opamp-storage/review.go.txt build/.storage-report-review/main.go
GOWORK=off go run ./build/.storage-report-review \
  docs/validation/opamp-storage/conformance-report.json.gz \
  docs/validation/opamp-performance/conformance-report.json.gz
```

`review-mutations.go.txt` and `review-negative-checks.py.txt` retain the executed corruption driver. Its typed Go mutations preserve raw call-parameter number lexemes; rewriting them through Python had triggered the prior exact-call comparison before the intended mutation. The final eight altered reports fail for their expected reasons, retained in `reviewer-negative-checks.json`.

`file-fixtures.tar.gz` contains only 248 explicitly generated positive/corrupt circuit files and 13 retained compatibility inputs. `file-fixtures-index.json` pins every regular member's bytes and SHA-256. Process endpoint/recovery files are excluded. `compatibility-final.json` records loading all 13 retained inputs, preserving their bytes, and exact re-export of five recorded JSON files. An initial reader wrongly required every saved project to have a capture; the original inverter intentionally has none. The corrected reader compares the exact asset count present in each document.

To extract the retained inputs into a new owned directory and verify their contents:

```python
from pathlib import Path
import hashlib, json, tarfile
stage = Path("build/.storage-fixture-inputs")
stage.mkdir(mode=0o700)
index = json.loads(Path("docs/validation/opamp-storage/file-fixtures-index.json").read_text())["members"]
seen = set()
with tarfile.open("docs/validation/opamp-storage/file-fixtures.tar.gz", "r:gz") as archive:
    for member in archive:
        target = stage / member.name
        assert member.isfile() and target.resolve().is_relative_to(stage.resolve()) and member.name not in seen
        data = archive.extractfile(member).read()
        assert hashlib.sha256(data).hexdigest() == index[member.name]["sha256"]
        target.parent.mkdir(parents=True, exist_ok=True)
        with target.open("xb") as output:
            output.write(data)
        seen.add(member.name)
assert seen == set(index)
```

The executed `compatibility.go.txt` reader takes the input root followed by a new output directory. Copy it to `main.go` in a new owned build subdirectory and run it with `build/.storage-fixture-inputs/compat` as the input root. Those retained files were hashed now; their directory labels do not prove that they were immutable since earlier qualification runs.

These checks establish internal consistency and the documented fixture coverage. They do not replay arbitrary history, prove hardware agreement, qualify maximum capacity or complete physical input/accessibility, or promote the application.
