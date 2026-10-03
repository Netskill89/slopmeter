# Contributing

Open an issue for bugs or send a focused pull request (GitHub) / merge request
(GitLab). Include what changed, why, and how you tested it. Do not commit private
captures, character/account details, generated binaries, or build directories.

## Local build

Install Go 1.26+ separately if your distribution provides an older version.

CachyOS / Arch:

```sh
sudo pacman -S --needed go libpcap base-devel cmake pkgconf qt6-base qt6-declarative qt6-wayland qt6-svg qt6-imageformats layer-shell-qt wayland nodejs python
```

Debian 13:

```sh
sudo apt install libpcap-dev build-essential cmake pkg-config qt6-base-dev qt6-declarative-dev qt6-wayland qt6-svg-dev qt6-image-formats-plugins liblayershellqtinterface-dev libwayland-dev qml6-module-qtquick qml6-module-qtquick-controls qml6-module-qtquick-layouts qml6-module-qtquick-window qml6-module-qtqml-workerscript qml6-module-qtquick-templates nodejs python3
```

```sh
bash scripts/build.sh
bash scripts/test.sh
./build/ui/slopmeter -read tests/fixtures/boss.jsonl
```

Race-enabled Go tests compile the decoder’s CGO packet-capture dependency and
require the libpcap development headers listed above. Release builds use
`CGO_ENABLED=0`.

Tests use synthetic traffic and replay fixtures; they do not need capture
privileges or a running game. The QML self-test runs offscreen. For overlay/display
changes, also test on a real Wayland desktop and include a screenshot.

## Pull / merge requests

1. Fork the project and create a branch for your change.
2. Keep changes focused, run the build/tests above, and update relevant docs.
3. Open a request against `main` and complete the included template.

Contributions are under the project's GPL-3.0 license. Preserve third-party
attribution when changing parsers or assets. For protocol reports, share a
minimal synthetic reproducer rather than your full network traffic.
