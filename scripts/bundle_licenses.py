#!/usr/bin/env python3
"""Retain distribution copyright/license notices for bundled shared libraries."""
from pathlib import Path
import shutil
import subprocess
import sys
appdir=Path(sys.argv[1])
out=appdir/'usr/share/doc/slopmeter/bundled-licenses'
out.mkdir(parents=True,exist_ok=True)
packages=set()
for library in list(appdir.rglob('*.so*'))+[appdir/'usr/bin/dumpcap',appdir/'usr/bin/setpriv']:
    name=library.name
    try:
        lines=subprocess.check_output(['dpkg-query','-S',f'*/{name}'],stderr=subprocess.DEVNULL,text=True).splitlines()
        for line in lines:
            package=line.split(': /',1)[0].split(':',1)[0]
            source=Path('/usr/share/doc')/package/'copyright'
            if source.is_file():
                shutil.copyfile(source,out/(package+'.copyright'))
                packages.add(package)
    except (FileNotFoundError,subprocess.CalledProcessError):
        # Native Arch development builds: official CI uses Debian copyright files.
        try:
            path=subprocess.check_output(['bash','-c','ldconfig -p'],stderr=subprocess.DEVNULL,text=True)
            matches=[line.split('=>')[-1].strip() for line in path.splitlines() if line.strip().startswith(name+' ')]
            executable=shutil.which(name)
            if executable:matches.append(executable)
            for item in matches:
                package=subprocess.check_output(['pacman','-Qoq',item],stderr=subprocess.DEVNULL,text=True).strip()
                source=Path('/usr/share/licenses')/package
                if source.is_dir():shutil.copytree(source,out/package,dirs_exist_ok=True)
        except (FileNotFoundError,subprocess.CalledProcessError):pass
(out/'README.txt').write_text('Distribution copyright files for bundled libraries.\nDebian sources: https://sources.debian.org/ and https://deb.debian.org/debian/pool/\nQt sources/licenses: https://code.qt.io/ and https://www.qt.io/licensing/\nLayerShellQt sources: https://invent.kde.org/plasma/layer-shell-qt\nShared libraries remain dynamically linked and may be replaced in the extracted AppDir.\n')
