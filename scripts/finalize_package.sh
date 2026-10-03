#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
root=$PWD
version=$(tr -d '\n' < VERSION)
arch=$(uname -m)
appdir="$root/build/AppDir"
export APPIMAGE_EXTRACT_AND_RUN=1
export PATH="$root/.tools:$PATH"
# Both formats contain the same bundled Qt libraries/plugins and launcher.
QT_QPA_PLATFORM=offscreen QT_QUICK_BACKEND=software "$appdir/AppRun" --ui-self-test -read "$root/tests/fixtures/boss.jsonl"
ARCH="$arch" appimagetool --runtime-file "$root/.tools/runtime-x86_64" "$appdir" "$root/dist/SlopMeter-$version-$arch.AppImage"
APPIMAGE_EXTRACT_AND_RUN=1 QT_QPA_PLATFORM=offscreen QT_QUICK_BACKEND=software "$root/dist/SlopMeter-$version-$arch.AppImage" --ui-self-test -read "$root/tests/fixtures/boss.jsonl"
bundle="SlopMeter-$version-$arch"
if [[ -d "build/$bundle" ]]; then
    python3 -c 'import shutil,sys; shutil.rmtree(sys.argv[1])' "build/$bundle"
fi
mkdir -p "build/$bundle"
cp -a "$appdir/." "build/$bundle/"
cp scripts/install.sh "build/$bundle/install.sh"
chmod +x "build/$bundle/install.sh"
tar -czf "dist/$bundle.tar.gz" -C build "$bundle"
# GPL releases include the corresponding application source (dependencies in go.mod).
python3 scripts/source_archive.py "dist/SlopMeter-$version-source.tar.gz"
(cd dist && sha256sum "SlopMeter-$version-$arch.AppImage" "SlopMeter-$version-$arch.tar.gz" "SlopMeter-$version-source.tar.gz" > SHA256SUMS)
