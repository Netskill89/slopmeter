# Changelog

## 0.1.0-alpha.3

- Fixed overlay focus and display-switch loops with the Debian-packaged LayerShellQt.
- Kept Settings and damage details independent of the non-focusable overlay surface.

- Added a version label to the meter footer.
- Removed in-app update checking and downloads; updates are planned for a future release.

## 0.1.0-alpha.2

- Added saved network-interface selection in Settings, including Automatic,
  individual adapters, and All interfaces.
- Added interface refresh and an explicit capture restart when applying a change.
- Fixed blank interface selection after the adapter list loads.
- Prevented unnecessary capture restarts when the selected interface is already active.
- Clarified that capture needs a character relog after switching interfaces.
- Fixed missing libpcap development headers in CI test builds.
- Renamed the Go module to `github.com/Netskill89/slopmeter`.

## 0.1.0-alpha.1

First alpha release: Wayland overlay, party DPS, boss HP, fight history,
skill breakdown, configurable player bars, and release update checks.
