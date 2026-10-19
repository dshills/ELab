"""Replay preserved authorized PNGs; writes data-only results to a new directory.

Requires the exact ignored local frame archives declared by the existing proof.
Does not alter that proof, create recordings, launch ELab, or send UI input.
"""
from pathlib import Path
import argparse
import importlib.util
import io
import json
import subprocess
import tempfile

from PIL import Image
from prepare import admit_existing_frame, first_complete_frame, sha, wait_and_admit


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--tool', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=False)
    source = Path(__file__).resolve().parent
    root = source.parents[1]
    proof = root / 'docs/validation/latency-followup-173'
    spec = importlib.util.spec_from_file_location('recorded_frame_store', proof / 'frame_store.py')
    store = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(store)
    tool = args.tool.resolve()
    tool_sha = sha(tool.read_bytes())
    cases = []
    with tempfile.TemporaryDirectory(prefix='elab-baseline-offline-') as temp:
        capture = Path(temp)
        for kind, count in [('led', 20), ('rc', 4)]:
            profile_path = source / 'profiles' / (kind + '.json')
            profile_raw = profile_path.read_bytes()
            profile = json.loads(profile_raw)
            for n in range(1, count + 1):
                trial_id = f'{kind}2-{n:02}'
                trial = proof / kind / 'session/trials' / trial_id
                journal = store.contents(trial / 'capture/frames.jsonl')
                row = first_complete_frame(journal)
                images = store.load_images(trial / 'capture')
                candidates = [('first', row, kind == 'led' or n in (1, 3))]
                # Accepted new values must not be mistaken for an old baseline.
                if kind == 'led' or n == 1:
                    rows = [json.loads(line) for line in journal.splitlines()]
                    last = [r for r in rows if r.get('png')][-1]
                    candidates.append(('fresh', last, False))
                for role, frame, should_admit in candidates:
                    raw = images[frame['png']]
                    image = Image.open(io.BytesIO(raw)).convert('RGBA')
                    independent_regions = []
                    for r in profile['regions']:
                        crop = image.crop(r['rect'])
                        if crop.getchannel('A').getextrema() != (255, 255):
                            raise ValueError('Unexpected nonopaque recorded display')
                        independent_regions.append(dict(name=r['name'], rgb_sha256=sha(crop.convert('RGB').tobytes())))
                    independent_admission = all(r['rgb_sha256'] == ref['rgb_sha256'] for r, ref in zip(independent_regions, profile['regions']))
                    if independent_admission != should_admit:
                        raise ValueError('Recorded case no longer matches its declared expectation: ' + trial_id + '/' + role)
                    image_path = capture / frame['png']
                    image_path.write_bytes(raw)
                    attempt = 'offline-' + trial_id + '-' + role + '-20261019'
                    bindings = dict(trial_id=trial_id, attempt_id=attempt, tool=tool, tool_sha256=tool_sha,
                                    profile=profile_path, profile_sha256=sha(profile_raw))
                    if role == 'first':
                        (capture / 'frames.jsonl').write_bytes(journal)
                        receipt = wait_and_admit(capture, capture / (trial_id + '-receipt.json'), **bindings)
                    else:
                        receipt = admit_existing_frame(capture, frame, timeout=1.5, **bindings)
                    if receipt['admitted'] != should_admit:
                        raise ValueError('Go/Pillow admission differs: ' + trial_id + '/' + role)
                    if receipt['result']['regions'] != [dict(r, matches=r['rgb_sha256'] == ref['rgb_sha256'])
                                                      for r, ref in zip(independent_regions, profile['regions'])]:
                        raise ValueError('Go/Pillow normalized RGB hashes differ')
                    expected = dict(trial_id=trial_id, attempt_id=attempt, frame_index=frame['index'],
                        png=frame['png'], png_sha256=frame['png_sha256'], tool_sha256=tool_sha,
                        profile_sha256=sha(profile_raw), width=profile['width'], height=profile['height'],
                        regions=[r['name'] for r in profile['regions']])
                    cases.append(dict(name=trial_id + '/' + role, should_admit=should_admit,
                        source_journal_sha256=sha(journal), source_frame_store_sha256=sha((trial / 'capture/frame-store.json').read_bytes()),
                        receipt=receipt, expected=expected))
                    image_path.unlink()
    cases_path = args.output / 'cases.json'
    cases_path.write_text(json.dumps(cases, indent=2) + '\n')
    process = subprocess.run(['/opt/homebrew/bin/node', str(source / 'replay_controller.mjs'), str(cases_path)],
                             capture_output=True, text=True, check=True, timeout=10)
    (args.output / 'controller-output.json').write_text(process.stdout)
    result = dict(passed=True, cases=len(cases), independent_png_decoder='Pillow', go_png_decoder='image/png',
        tool_sha256=tool_sha, first_admitted=sum(c['should_admit'] for c in cases if c['name'].endswith('/first')),
        first_refused=sum(not c['should_admit'] for c in cases if c['name'].endswith('/first')),
        fresh_state_refused=sum(c['name'].endswith('/fresh') for c in cases),
        late_occlusion_not_detected_by_initial_guard=['rc2-03'], actual_capture=False, actual_input=False,
        overall_prd_performance_pass=None,
        source_sha256={p.relative_to(root).as_posix(): sha(p.read_bytes()) for p in source.rglob('*')
                       if p.is_file() and '__pycache__' not in p.parts})
    (args.output / 'result.json').write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps(result))


if __name__ == '__main__':
    main()
