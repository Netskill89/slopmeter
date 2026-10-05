#!/usr/bin/env python3
"""Create corresponding source without build products or local capture data."""
from pathlib import Path
import subprocess
import sys
import tarfile
root = Path(__file__).resolve().parents[1]
# Include tracked and non-ignored new files so local packages have complete source.
# A fresh pre-init workspace uses explicit roots.
try:
    paths = subprocess.check_output(['git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z'], cwd=root, stderr=subprocess.DEVNULL).decode().split('\0')
except subprocess.CalledProcessError:
    allowed = ['cmd', 'ui', 'scripts', 'packaging', 'tests', 'docs', 'LICENSES', '.github', '.gitlab']
    paths = [str(p.relative_to(root)) for name in allowed for p in (root/name).rglob('*') if p.is_file()]
    paths += [p.name for p in root.iterdir() if p.is_file() and (p.suffix in ['.go', '.md'] or p.name in ['go.mod', 'go.sum', 'VERSION', 'LICENSE', '.gitignore', '.gitattributes', '.gitlab-ci.yml'])]
version = (root/'VERSION').read_text().strip()
with tarfile.open(sys.argv[1], 'w:gz') as out:
    for name in sorted(set(paths)):
        p = root/name
        if name and p.is_file() and '__pycache__' not in p.parts:
            out.add(p, arcname=f'SlopMeter-{version}-source/{name}', recursive=False)
