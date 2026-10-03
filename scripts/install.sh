#!/usr/bin/env bash
set -euo pipefail
bundle=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
[[ -x "$bundle/AppRun" ]] || { echo 'Run install.sh from the extracted release bundle' >&2; exit 1; }
data_root=${XDG_DATA_HOME:-$HOME/.local/share}
mkdir -p "$HOME/.local/bin" "$data_root/applications" "$data_root/icons/hicolor/scalable/apps"
# Keep the extracted directory in place; launchers point to it.
ln -sfn "$bundle/AppRun" "$HOME/.local/bin/slopmeter"
python3 - "$bundle" "$data_root" <<'PY'
import pathlib, sys
bundle, data = map(pathlib.Path, sys.argv[1:])
# Desktop Entry Exec escaping is different from shell escaping.
exe=str(bundle/'AppRun').replace('\\','\\\\').replace('"','\\"').replace('`','\\`').replace('$','\\$').replace('%','%%')
source=(bundle/'slopmeter.desktop').read_text().replace('Exec=slopmeter',f'Exec="{exe}"')
(data/'applications/slopmeter.desktop').write_text(source)
(data/'icons/hicolor/scalable/apps/slopmeter.svg').write_bytes((bundle/'slopmeter.svg').read_bytes())
PY
printf 'Installed SlopMeter launcher. Keep this directory: %s\n' "$bundle"
printf 'Launch from your application menu or ~/.local/bin/slopmeter.\n'
