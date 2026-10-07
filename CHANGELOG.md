# Changelog

## Unreleased

## 0.1.0-alpha.6

- Add saved Up/Down bar growth, keeping the bottom or top edge fixed as player bars are added or removed.
- Add a zoomable per-player skill usage timeline with small skill icons and cast-time hover details.
- Save observed cast timestamps with new combat history entries; include a sample timeline in Test mode.
- Remember the last detected character name and allow editing it in Settings without restarting capture.
- Match remembered names against fresh player metadata, recheck name-based identity after zone arrivals, and prefer confirmed self identity on relog.

## 0.1.0-alpha.5

- Decode captured area names and instance event IDs; display area names in the meter, fight history and damage details.
- Handle modern NPC spawn layouts and validated current-HP records; attach late boss metadata to the existing fight.
- Keep confirmed boss sessions active through damage pauses and temporary loss of visibility; finish on confirmed map transitions.
- Add a manual damage reset button between Settings and Minimise, preserving fight history and capture.
- Count validated periodic damage ticks while excluding heals, buffs, and damage to self or confirmed group members.
- Verify DPS arithmetic and shared fight timing against A2Tools/Aion2Flow reference formulas.
- Decode party roster gear score and combat power; show below player names, or on name hover for bars shorter than 38 pixels.
- Add saved settings for score visibility, total damage, damage contribution, and abbreviated DPS.
- Use muted class colors based on the A2Tools palette.
- Update the missing-character tooltip to suggest relogging or changing zone.
- Limit CI packaging to release tags and manual validation runs; keep build and test checks on branches and PRs.

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
