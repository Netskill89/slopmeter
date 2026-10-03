#!/usr/bin/env python3
"""Expose only required image plugins to linuxdeploy, avoiding host KDE add-ons."""
from pathlib import Path
import os
import subprocess
import sys
real_qmake=os.environ.get('QMAKE','qmake6')
plugins=Path(subprocess.check_output([real_qmake,'-query','QT_INSTALL_PLUGINS'],text=True).strip())
# Debian separates SVG runtime plugins from qt6-svg-dev. Check before deploy.
for relative in ['iconengines/libqsvgicon.so', 'imageformats/libqsvg.so', 'imageformats/libqwebp.so']:
    if not (plugins/relative).is_file():
        sys.exit(f'Missing required Qt plugin: {plugins/relative}. On Debian install qt6-svg-plugins and qt6-image-formats-plugins; on Arch install qt6-svg and qt6-imageformats.')
root=Path(__file__).resolve().parents[1]
stage=root/'build/qt-plugins';stage.mkdir(exist_ok=True)
for directory in plugins.iterdir():
    if not directory.is_dir():continue
    destination=stage/directory.name
    if directory.name in ['imageformats','platformthemes','styles']:
        destination.mkdir(exist_ok=True)
        if directory.name=='imageformats':
            for name in ['libqwebp.so','libqsvg.so','libqjpeg.so','libqgif.so','libqico.so']:
                original=directory/name
                if original.exists() and not (destination/name).exists():(destination/name).symlink_to(original)
    elif not destination.exists():destination.symlink_to(directory,target_is_directory=True)
wrapper=root/'build/qmake-package'
# Forward all query results, overriding only the plugin directory.
import shlex
wrapper.write_text('#!/usr/bin/env bash\nset -euo pipefail\n'+
    'if [[ "$*" == "-query QT_INSTALL_PLUGINS" ]]; then printf "%s\\n" '+shlex.quote(str(stage))+';\n'+
    'elif [[ "$*" == "-query" ]]; then '+shlex.quote(real_qmake)+' -query | sed '+shlex.quote('s|^QT_INSTALL_PLUGINS:.*|QT_INSTALL_PLUGINS:'+str(stage)+'|')+';\n'+
    'else exec '+shlex.quote(real_qmake)+' "$@"; fi\n')
wrapper.chmod(0o755)
print(wrapper)
