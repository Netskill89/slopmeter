# Implementation notes

SlopMeter has a Go packet-decoding backend and a C++/Qt Quick frontend. The GUI
starts `slopmeter-capture` and reads JSON snapshots over stdout. The backend and
its Go tests live in `cmd/slopmeter-capture/`; embedded game catalogues live in
its `data/` subdirectory. `ui/` contains the frontend, and `scripts/` and
`packaging/` contain build and release tooling. Live capture uses
a bundled `dumpcap` helper, reusing a permitted system helper when available.
Polkit authorizes only capture when necessary; the decoder and GUI run unprivileged.

## Capture and scope

- a2kit decodes a2log/pcap streams and live TCP traffic.
- Automatic uses dumpcap’s default adapter. Settings can select a Linux network
  adapter or `any`, with an explicit capture restart. Enumeration is unprivileged;
  selection is saved with QSettings and ignored during replay. Restarting ends
  the current session and requires a relog for character/group discovery.
- The Self event must identify the character before damage is accepted.
- Only self and members confirmed by group packets are counted. Nearby players
  are excluded; membership leases avoid retaining stale group members.
- Packets are decoded continuously. The 50–1000 ms setting controls snapshot
  publication, not capture. Default: 200 ms. Updating it does not restart a fight.
- A bounded snapshot publisher prevents a slow GUI from blocking accounting.

## Encounters

Open-world fights end after five seconds without damage. A recognised boss
starts a boss-only session; add damage is excluded. Boss death, full-health reset,
zone/login changes, or a conservative 90-second idle fallback end the encounter.
Elapsed time includes boss mechanics. Final results remain visible until the
next combat begins. The health bar appears only for recognised bosses and uses
captured HP/max-HP; damage is never subtracted to invent health.

The last 10 sessions are persisted with atomic file replacement at
`$XDG_STATE_HOME/aiondps/history.json`, or `~/.local/state/aiondps/history.json`.
The legacy path is retained for upgrades. Replay defaults to in-memory history;
`-history PATH` explicitly persists it. Invalid history files are not overwritten.

Skill variants share a base skill. Damage, contribution, crit evidence, min/max,
hits, hits/s, and observed casts are tracked separately. Multi-hit damage does
not imply multiple casts.

## Overlay

The native layer-shell surface stays stationary over the display. Dragging moves
the QML panel inside it; an input mask makes the rest of the display click-through.
KDE display detection uses an ephemeral KWin script to follow the AION 2 window.
Other layer-shell desktops can use the tray's display selector.

Settings and damage details use separate ordinary windows. Settings are saved
with QSettings; the old AionDPS settings are migrated once. Test mode shares the
real bar delegates/models and displays five simulated players with eight skills
each and boss health. It never writes fake history
or stops real capture.

## Tests

Go tests cover party filtering, HP/encounter transitions, history, skill stats,
publication, runtime refresh, and helper lifecycle.
The Node test covers game-window matching. The Python integration test streams
400 synthetic hits through the actual capture path while changing refresh rates.
The QML self-test checks drag geometry, input masks, settings persistence, bar
styles, compact percentages, history retention, and secondary-window lifecycle.
