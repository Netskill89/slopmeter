#!/usr/bin/env bash
set -euo pipefail
# Run inside the disposable Debian 13 CI container, as root.
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends wireshark-common libpcap-dev build-essential cmake pkg-config python3 nodejs curl ca-certificates file patchelf desktop-file-utils libwayland-dev liblayershellqtinterface-dev layer-shell-qt qt6-base-dev qt6-declarative-dev qt6-wayland qt6-svg-dev qt6-svg-plugins qt6-image-formats-plugins qml6-module-qtquick qml6-module-qtquick-controls qml6-module-qtquick-layouts qml6-module-qtquick-window qml6-module-qtqml-workerscript qml6-module-qtquick-templates
