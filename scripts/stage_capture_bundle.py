#!/usr/bin/env python3
"""Create a self-contained capture helper payload for desktop authorization."""
from pathlib import Path
import shutil
import subprocess
import sys
appdir=Path(sys.argv[1]).resolve()
output=appdir/'usr/lib/slopmeter-capture-helper'
(output/'bin').mkdir(parents=True,exist_ok=True)
(output/'lib').mkdir(exist_ok=True)
for name in ['dumpcap','setpriv']:
    helper=appdir/'usr/bin'/name
    destination=output/'bin'/name
    shutil.copyfile(helper,destination)
    destination.chmod(0o755)
    subprocess.run(['patchelf','--set-rpath','$ORIGIN/../lib',str(destination)],check=True)
    for line in subprocess.check_output(['ldd',str(helper)],text=True).splitlines():
        if '=>' not in line:continue
        path=Path(line.split('=>',1)[1].strip().split(' ',1)[0])
        if path.is_absolute() and path.name not in ['libc.so.6','libm.so.6','libdl.so.2','libpthread.so.0','librt.so.1','libgcc_s.so.1','libstdc++.so.6']:
            target=output/'lib'/path.name
            shutil.copyfile(path,target)
            target.chmod(0o644)
            subprocess.run(['patchelf','--set-rpath','$ORIGIN',str(target)],check=True)
# Avoid copying the entire Qt bundle into the helper's cache.
subprocess.run(['tar','-czf',str(appdir/'usr/share/slopmeter-capture-helper.tar.gz'),'-C',str(output),'.'],check=True)
shutil.rmtree(output)
for name in ['dumpcap','setpriv']:(appdir/'usr/bin'/name).unlink()
