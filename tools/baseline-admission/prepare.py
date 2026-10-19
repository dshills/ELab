"""Analyze the first existing complete recorder PNG; never capture or send input.

A worker may call wait_and_admit immediately after starting its bounded recorder,
before waiting for recorder exit. Failure must not shorten/restart the recording.
The controller must independently validate the returned receipt before its marker.
"""
from pathlib import Path
import hashlib
import json
import re
import stat
import subprocess
import time


SHA = re.compile(r"[0-9a-f]{64}")


def require(condition, message):
    if not condition:
        raise ValueError(message)


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def bounded_file(path, limit):
    path = Path(path)
    info = path.lstat()
    require(stat.S_ISREG(info.st_mode), "Expected a regular nonsymlink file")
    require(info.st_size <= limit, "File byte budget exceeded")
    with path.open("rb") as stream:
        raw = stream.read(limit + 1)
    require(len(raw) <= limit, "File byte budget exceeded")
    return raw


def first_complete_frame(raw):
    """Ignore only the unterminated final line of a live journal."""
    require(len(raw) <= 2 << 20, "Journal byte budget exceeded")
    for line in raw.split(b"\n")[:-1]:
        row = json.loads(line)
        require(isinstance(row, dict), "Journal row must be an object")
        require(row.get("event") in ("frame", "marker"), "Unknown recorder event")
        require(row["event"] != "marker", "Marker already requested before admission")
        require(type(row.get("status")) is int, "Missing recorder frame status")
        if row["status"] == 0:
            require(bool(row.get("png")), "Complete frame is missing its PNG")
        if row.get("png"):
            return row
    return None


def validate_frame(row, width, height):
    require(row.get("event") == "frame" and type(row.get("status")) is int and row["status"] == 0,
            "Expected a complete frame")
    require(type(row.get("index")) is int and 1 <= row["index"] <= 300, "Invalid frame index")
    require(type(row.get("width_pixels")) is int and row["width_pixels"] == width and
            type(row.get("height_pixels")) is int and row["height_pixels"] == height,
            "Unexpected frame dimensions")
    for key in ("frame_geometry_validated", "target_geometry_matches_at_receipt",
                "presentation_matches_converted_display_time"):
        require(row.get(key) is True, "Missing recorder geometry/clock validation: " + key)
    for key in ("display_mach_ticks", "received_mach_ticks"):
        value = row.get(key)
        require(isinstance(value, str) and value.isascii() and value.isdigit() and int(value) > 0,
                "Invalid recorder timestamp")
    # Composition presentation timestamps may be ahead of callback receipt. The
    # recorder reports this explicitly; baseline content admission must not
    # invent a clock-order restriction. Timing qualification stays independent.
    future = int(row["display_mach_ticks"]) > int(row["received_mach_ticks"])
    require(row.get("display_time_after_receipt") is future, "Inconsistent display/receipt flag")
    digest = row.get("png_sha256")
    require(isinstance(digest, str) and SHA.fullmatch(digest), "Invalid PNG identity")
    require(row.get("png") == "frame-" + digest + ".png", "Invalid PNG basename")


def admit_existing_frame(capture, row, *, trial_id, attempt_id, tool, tool_sha256,
                         profile, profile_sha256, timeout=1.0):
    """Pinned, bounded file analysis. Exceptions are negative admission decisions."""
    require(isinstance(trial_id, str) and re.fullmatch(r"[a-z0-9-]{1,64}", trial_id), "Invalid trial ID")
    require(isinstance(attempt_id, str) and re.fullmatch(r"[a-zA-Z0-9-]{16,80}", attempt_id), "Invalid attempt ID")
    for digest in (tool_sha256, profile_sha256):
        require(isinstance(digest, str) and SHA.fullmatch(digest), "Invalid pinned identity")
    require(0 < timeout <= 1.5, "Invalid analysis timeout")
    require(sha(bounded_file(tool, 64 << 20)) == tool_sha256, "Analysis tool identity changed")
    profile_raw = bounded_file(profile, 64 << 10)
    require(sha(profile_raw) == profile_sha256, "Baseline profile identity changed")
    declared = json.loads(profile_raw)
    validate_frame(row, declared["width"], declared["height"])
    image = Path(capture) / row["png"]
    require(sha(bounded_file(image, 64 << 20)) == row["png_sha256"], "Recorder PNG identity changed")
    process = subprocess.run([str(tool), "--profile", str(profile), "--image", str(image),
                              "--sha256", row["png_sha256"]],
                             stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout)
    require(len(process.stdout) <= 64 << 10 and len(process.stderr) <= 64 << 10,
            "Analysis output budget exceeded")
    result = json.loads(process.stdout)
    admitted = (process.returncode == 0 and isinstance(result, dict) and
                result.get("version") == 1 and result.get("admitted") is True and
                result.get("code") == "admitted" and
                result.get("profile_sha256") == profile_sha256 and
                result.get("image_sha256") == row["png_sha256"])
    return dict(version=1, admitted=admitted, trial_id=trial_id, attempt_id=attempt_id,
                frame_index=row["index"], png=row["png"], png_sha256=row["png_sha256"],
                tool_sha256=tool_sha256, profile_sha256=profile_sha256,
                tool_exit_code=process.returncode, result=result)


def wait_and_admit(capture, receipt_path, *, deadline_seconds=1.5, **bindings):
    """Write one exclusive receipt; missing/invalid data never admits a trial.

    Use a new attempt_id from the controller's capture-start handshake. A receipt
    from a previous attempt cannot authorize a marker in the current attempt.
    This function does not own the recorder: the worker still waits for its fixed
    duration and records an unavailable outcome when admission is negative.
    """
    require(0 < deadline_seconds <= 1.5, "Invalid baseline deadline")
    deadline = time.monotonic() + deadline_seconds
    try:
        row = None
        while row is None:
            remaining = deadline - time.monotonic()
            require(remaining > 0, "First complete frame deadline exceeded")
            journal = Path(capture) / "frames.jsonl"
            try:
                row = first_complete_frame(bounded_file(journal, 2 << 20))
            except FileNotFoundError:
                row = None
            if row is None:
                time.sleep(min(.003, remaining))
        receipt = admit_existing_frame(capture, row, timeout=max(.001, deadline - time.monotonic()), **bindings)
        require(time.monotonic() <= deadline, "Baseline analysis deadline exceeded")
    except (ValueError, KeyError, TypeError, OSError, subprocess.SubprocessError) as error:
        receipt = dict(version=1, admitted=False, trial_id=bindings.get("trial_id"),
                       attempt_id=bindings.get("attempt_id"), error=type(error).__name__ + ": " + str(error))
    with Path(receipt_path).open("x", encoding="utf-8") as stream:
        json.dump(receipt, stream, indent=2)
        stream.write("\n")
    return receipt
