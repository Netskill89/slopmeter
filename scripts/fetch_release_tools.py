#!/usr/bin/env python3
"""Download checksum-pinned official packaging tools; never execute an unverified file."""
import hashlib
import json
from pathlib import Path
import urllib.request
root = Path(__file__).resolve().parents[1]
folder = root/'.tools'
folder.mkdir(exist_ok=True)
for tool in json.loads((root/'packaging/tools.lock.json').read_text()):
    destination = folder/tool['name']
    if destination.exists() and hashlib.sha256(destination.read_bytes()).hexdigest() == tool['sha256']:
        continue
    print('Downloading', tool['name'], flush=True)
    request = urllib.request.Request(tool['url'], headers={'User-Agent': 'SlopMeter-release-build'})
    with urllib.request.urlopen(request, timeout=120) as response:
        payload = response.read(64*1024*1024+1)
    if len(payload)>64*1024*1024 or hashlib.sha256(payload).hexdigest()!=tool['sha256']:
        raise SystemExit(f"Checksum mismatch for {tool['name']}; review/update tools.lock.json against the official release")
    temporary = destination.with_suffix('.tmp')
    temporary.write_bytes(payload)
    temporary.chmod(0o755)
    temporary.replace(destination)
