#!/usr/bin/env python3
"""Exercise the real backend/control pipe with a deterministic fake capture helper.

No network capture or privileges. All 400 hits must survive rate changes.
"""
import json
import os
from pathlib import Path
import selectors
import statistics
import subprocess
import tempfile
import time

BACKEND = Path(os.environ.get("SLOPMETER_BACKEND", str(Path(__file__).resolve().parents[1] / "build/slopmeter-capture")))
HELPER = r'''#!/usr/bin/python3
import base64,datetime,json,struct,sys,time

def varint(n):
    b=bytearray()
    while n>=128:b.append((n&127)|128);n>>=7
    b.append(n);return bytes(b)

def message(opcode,payload,at):
    print(json.dumps({"t":at,"opcode":opcode,"flags":["server"],"src":"10.0.0.2:13328","dst":"10.0.0.1:10000","payload":base64.b64encode(payload).decode()}),flush=True)

print(json.dumps({"schema":"a2log/v0.1","decoder":"fake-live-test","source":{"kind":"feed"},"t0":datetime.datetime.now(datetime.timezone.utc).isoformat()}),flush=True)
message("33 36",bytes([1,1,32,0,0,7,4])+b"Self",0)
p=varint(7)+varint(4)+varint(0)+varint(1)+struct.pack("<I",11020000)+bytes([1,2,0,0,0,0])+struct.pack("<I",1)+varint(10000)+varint(10)+bytes([1,0])
start=time.monotonic()
for n in range(400):
    delay=start+n*.01-time.monotonic()
    if delay>0:time.sleep(delay)
    message("04 38",p,n*10)
'''


def main():
    with tempfile.TemporaryDirectory(prefix="aiondps-refresh-") as directory:
        helper = Path(directory) / "dumpcap"
        helper.write_text(HELPER)
        helper.chmod(0o755)
        env = dict(os.environ, PATH=directory + os.pathsep + os.environ.get("PATH", ""))
        process = subprocess.Popen(
            [str(BACKEND), "-json", "-interface", "test0", "-interval", "200ms", "-history", str(Path(directory) / "history.json")],
            env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=1,
        )
        selector = selectors.DefaultSelector()
        selector.register(process.stdout, selectors.EVENT_READ)
        start = time.monotonic()
        frames = []
        phase = 0
        try:
            while time.monotonic() - start < 8:
                elapsed = time.monotonic() - start
                if phase == 0 and elapsed >= .9:
                    process.stdin.write('{"intervalMs":50}\n'); process.stdin.flush(); phase = 1
                if phase == 1 and elapsed >= 1.9:
                    process.stdin.write('{"intervalMs":1000}\n'); process.stdin.flush(); phase = 2
                ready = selector.select(.01)
                if not ready:
                    if process.poll() is not None:
                        break
                    continue
                line = process.stdout.readline()
                if not line:
                    break
                frames.append((time.monotonic() - start, json.loads(line)))
            process.wait(timeout=2)
            assert process.returncode == 0, process.stderr.read()
            final = frames[-1][1]
            assert final["actors"][0]["damage"] == 4000, final
            assert final["actors"][0]["skills"][0]["hits"] == 400, final
            assert final["session"] == 1 and len(final["history"]) == 1, final
            for lo, hi, expected in [(.25, .88, .2), (1.1, 1.88, .05), (2.5, 4, 1.)]:
                times = [at for at, state in frames if lo < at < hi and state["active"]]
                gaps = [right - left for left, right in zip(times, times[1:])]
                assert gaps, (lo, hi, times)
                median = statistics.median(gaps)
                assert abs(median - expected) < max(.025, expected * .2), (expected, median)
                print(f"Runtime interval {expected * 1000:.0f} ms: observed median {median * 1000:.1f} ms")
            print("All 400 hits retained; interval changes preserved the fight and history.")
        finally:
            selector.close()
            if process.poll() is None:
                process.kill()
                process.wait()


if __name__ == "__main__":
    main()
