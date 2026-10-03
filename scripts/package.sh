#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version=$(tr -d '\n' < VERSION)
arch=$(uname -m)
[[ "$arch" == x86_64 ]] || { echo 'Release packaging currently supports x86_64 only' >&2; exit 1; }
root=$PWD
appdir="$root/build/AppDir"
if [[ -e "$appdir" ]]; then echo 'Move/remove build/AppDir before packaging again' >&2; exit 1; fi
mkdir -p "$appdir/usr/bin" "$appdir/usr/share/doc/slopmeter" dist
install -m755 build/ui/slopmeter build/slopmeter-capture "$appdir/usr/bin/"
cp LICENSE THIRD_PARTY.md README.md "$appdir/usr/share/doc/slopmeter/"
cp -r LICENSES "$appdir/usr/share/doc/slopmeter/"
cp VERSION "$appdir/usr/share/doc/slopmeter/"
python3 scripts/fetch_release_tools.py
export PATH="$root/.tools:$PATH"
export APPIMAGE_EXTRACT_AND_RUN=1
# Bundled binutils can be too old for modern RELR sections. Build is already stripped.
export NO_STRIP=1
export QMAKE=$(python3 scripts/stage_qt_plugins.py)
export QML_SOURCES_PATHS="$root/ui"
export EXTRA_QT_MODULES='svg;waylandclient'
export EXTRA_PLATFORM_PLUGINS=$(python3 -c 'import pathlib,subprocess,os; p=pathlib.Path(subprocess.check_output([os.environ["QMAKE"],"-query","QT_INSTALL_PLUGINS"],text=True).strip())/"platforms"; print(";".join(sorted(x.name for x in p.glob("*.so") if "wayland" in x.name or "offscreen" in x.name)))')
linuxdeploy --appdir "$appdir" --executable "$appdir/usr/bin/slopmeter" --desktop-file packaging/slopmeter.desktop --icon-file packaging/slopmeter.svg --plugin qt
dumpcap_path=$(command -v dumpcap)
linuxdeploy --appdir "$appdir" --executable "$dumpcap_path" --executable "$(command -v setpriv)"
python3 scripts/bundle_licenses.py "$appdir"
python3 scripts/stage_capture_bundle.py "$appdir"
# LayerShellQt's integration is loaded dynamically; ldd cannot discover it.
plugins=$("$QMAKE" -query QT_INSTALL_PLUGINS)
mkdir -p "$appdir/usr/plugins/wayland-shell-integration" "$appdir/usr/plugins/wayland-graphics-integration-client"
layer_plugin="$plugins/wayland-shell-integration/liblayer-shell.so"
[[ -f "$layer_plugin" ]] || { echo 'LayerShellQt Wayland plugin is missing' >&2; exit 1; }
for plugin in "$plugins"/wayland-shell-integration/*.so "$plugins"/wayland-graphics-integration-client/*.so; do
    [[ -f "$plugin" ]] || continue
    relative=${plugin#"$plugins/"}
    install -m755 "$plugin" "$appdir/usr/plugins/$relative"
    linuxdeploy --appdir "$appdir" --library "$appdir/usr/plugins/$relative"
done
install -m755 packaging/AppRun "$appdir/AppRun"
python3 scripts/bundle_licenses.py "$appdir"
bash scripts/finalize_package.sh
