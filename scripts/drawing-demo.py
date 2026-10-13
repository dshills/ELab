#!/usr/bin/env python3
"""Build an LED circuit from blank through the shared schematic command API."""
import argparse
import json
from pathlib import Path
import subprocess
import time

ROOT = Path(__file__).resolve().parents[1]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--connect", help="private endpoint descriptor for a running app")
    parser.add_argument("--output", type=Path, help="save the result report")
    args = parser.parse_args()
    argv = [str(ROOT / "build/elab-agent")]
    if args.connect:
        argv += ["--connect", args.connect]
    client = subprocess.Popen(argv, stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
    sequence = 0
    revision = 0
    observations = []

    def request(command, params=None, mutation=False, key=None, expect_error=None):
        nonlocal sequence, revision
        sequence += 1
        envelope = {"version": 1, "id": f"demo-{sequence}", "command": command, "params": params or {}}
        if mutation:
            envelope.update(expected_revision=revision, key=key or f"demo-{time.time_ns()}-{sequence}")
        client.stdin.write(json.dumps(envelope) + "\n")
        client.stdin.flush()
        line = client.stdout.readline()
        if not line:
            raise RuntimeError("agent client ended without a response")
        response = json.loads(line)
        revision = response["revision"]
        if expect_error:
            assert response.get("error", {}).get("code") == expect_error, response
        else:
            assert "error" not in response, response
        return response

    try:
        catalog = request("capabilities.list")["result"]["capabilities"]
        request("project.inspect")
        request("project.clear", {"discard_changes": True}, mutation=True)
        ids = {}
        for kind, position in [("dc-source", {"x": -200, "y": 0}), ("resistor", {"x": 0, "y": -110}), ("led", {"x": 200, "y": 0}), ("ground", {"x": 0, "y": 110})]:
            placed = request("component.place", {"kind": kind, "position": position}, True)["result"]
            ids[kind] = placed["selected"]
            assert placed["status"] == "incomplete"
        pairs = [(ids["dc-source"] + ".positive", ids["resistor"] + ".1"), (ids["resistor"] + ".2", ids["led"] + ".anode"), (ids["led"] + ".cathode", ids["ground"] + ".reference"), (ids["ground"] + ".reference", ids["dc-source"] + ".negative")]
        for index, (source, destination) in enumerate(pairs):
            wired = request("wire.connect", {"from": source, "to": destination}, True)["result"]
            if index < 3:
                assert wired["status"] == "incomplete"
        baseline = request("simulation.wait")["result"]
        assert abs(baseline["result"]["current_a"] - 0.00291679) < 2e-8
        retry_key = f"demo-resistance-{time.time_ns()}"
        request("component.set", {"id": ids["resistor"], "value": "2k"}, True, retry_key)
        edited = request("simulation.wait")["result"]
        assert abs(edited["result"]["current_a"] - 0.00147672) < 2e-8
        assert request("component.set", {"id": ids["resistor"], "value": "2k"}, True, retry_key)["duplicate"]
        # A cached response identifies its original revision; inspect current
        # state before sending the next mutation.
        request("project.inspect")
        broken = request("object.delete", {"id": baseline["wires"][1]["id"]}, True)["result"]
        assert broken["status"] == "incomplete" and broken["result"]["stale"]
        request("edit.undo", mutation=True)
        recovered = request("simulation.wait")["result"]
        assert recovered["circuit"]["connected"] and not recovered["result"]["stale"]
        request("simulation.pause", mutation=True)
        paused = request("component.set", {"id": ids["resistor"], "value": "1k"}, True)["result"]
        assert paused["status"] == "paused" and paused["result"]["stale"]
        request("simulation.resume", mutation=True)
        final = request("simulation.wait")["result"]
        assert not final["result"]["stale"]
        request("component.set", {"id": ids["resistor"], "value": "0"}, True, expect_error="invalid-params")
        unchanged = request("project.inspect")["result"]
        assert unchanged["circuit"]["resistance_ohm"] == 1000
        run = unchanged["run_id"]
        request("component.select", {"id": ids["led"]}, True)
        request("component.move", {"id": ids["resistor"], "position": {"x": 30, "y": -100}}, True)
        request("component.rotate", {"id": ids["resistor"]}, True)
        request("edit.undo", mutation=True)
        request("edit.undo", mutation=True)
        request("view.pan", {"dx": 20, "dy": 10}, True)
        request("view.zoom", {"factor": 1.25}, True)
        result = request("view.fit", mutation=True)["result"]
        assert result["run_id"] == run
        observations = {
            "passed": True,
            "transport": "desktop-or-headless-socket" if args.connect else "headless-stdio",
            "capability_count": len(catalog),
            "current_1k_a": baseline["result"]["current_a"],
            "current_2k_a": edited["result"]["current_a"],
            "checks": ["build from blank", "terminal-derived nets", "move/rotate/undo without solver restart", "live edit", "retry deduplication", "incomplete/stale feedback", "undo recovery", "paused edit", "resume", "invalid input refusal", "selection/view without solver restart"],
            "component_ids": ids,
            "final": result,
        }
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(observations, indent=2) + "\n")
        print(json.dumps({k: v for k, v in observations.items() if k != "final"}, indent=2))
    finally:
        client.stdin.close()
        try:
            client.wait(timeout=5)
        except subprocess.TimeoutExpired:
            client.terminate()
            client.wait(timeout=5)


if __name__ == "__main__":
    main()
