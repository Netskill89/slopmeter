#!/usr/bin/env python3
"""Publish tag artifacts to permanent GitLab packages, then create a release."""
import json
import os
from pathlib import Path
import urllib.request
base = os.environ['CI_API_V4_URL']+'/projects/'+os.environ['CI_PROJECT_ID']
tag = os.environ['CI_COMMIT_TAG']
version = Path('VERSION').read_text().strip()
assert tag == 'v'+version, 'Tag does not match VERSION'
headers = {'JOB-TOKEN': os.environ['CI_JOB_TOKEN']}
links = []
for file in sorted(Path('dist').iterdir()):
    if not file.is_file(): continue
    address = f'{base}/packages/generic/slopmeter/{version}/{file.name}'
    request = urllib.request.Request(address, data=file.read_bytes(), headers=headers, method='PUT')
    with urllib.request.urlopen(request, timeout=120) as response: response.read()
    links.append({'name':file.name, 'url':address, 'link_type':'package'})
payload = {'name':f'SlopMeter {version}', 'tag_name':tag, 'description':f'Linux x86_64 release. Installation: {os.environ["CI_PROJECT_URL"]}/-/blob/{tag}/README.md', 'assets':{'links':links}}
request = urllib.request.Request(base+'/releases', data=json.dumps(payload).encode(), headers=dict(headers, **{'Content-Type':'application/json'}), method='POST')
with urllib.request.urlopen(request, timeout=30) as response: response.read()
print('Published', tag)
