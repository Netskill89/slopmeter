# Changelog

## Unreleased

## 0.1.0-alpha.4

- Detect classes from class-specific casts as well as hits; refresh missing class metadata in retained finished fights when identity arrives late.
- Register a desktop identity automatically for source and portable launches, fixing the missing-app portal warning.
- Use HTTPS over HTTP/1.1 for release checks to avoid Qt 6.11's closed-socket read in its HTTP/2 handler.
- Added a GitHub release check and an "Update available" link beside the version, including newer alpha releases.
- Added the SlopMeter application icon to the tray and application windows.
- Launching the app again restores the existing meter instead of starting a second capture.
- Added a taskbar restore window when hiding the meter on a desktop without a tray.
- Fixed boss health display when NPC identity arrives after damage.
- Read validated extended boss HP fields even when extra packet fields change the total length.
- Added boss-health capture diagnostics in Settings to distinguish missing NPC identity, catalogue entries and HP packets.
- Kept ordinary enemies hidden; elite support still depends on NPC catalogue classification.
- Include non-ignored new files in local source archives so packaged source remains complete.

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
