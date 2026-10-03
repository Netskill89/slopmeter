<h1 align="center">SLOPMETER</h1>
<p align="center">An AION 2 DPS meter for Linux and Wayland</p>
<p align="center"><strong>Alpha</strong> · <a href="https://github.com/Netskill89/slopmeter/releases">Download</a> · <a href="https://github.com/Netskill89/slopmeter/issues">Report an issue</a> · <a href="CONTRIBUTING.md">Contribute</a></p>

---

SlopMeter brings a native Linux overlay to AION 2 running through Steam/Proton.
Built to fill the gap in Linux support among major DPS meters, it tracks your
character and confirmed group members.

## Screenshots

<table>
  <tr><th>DPS overlay</th><th>Skill breakdown</th></tr>
  <tr>
    <td align="center"><a href="docs/images/boss-health-overlay.png"><img src="docs/images/boss-health-overlay.png" width="340" alt="SlopMeter showing boss health and five class-coloured player bars"></a></td>
    <td align="center"><a href="docs/images/eight-skill-details.png"><img src="docs/images/eight-skill-details.png" width="620" alt="SlopMeter damage breakdown with eight skills, icons, damage, critical rate, and contribution"></a></td>
  </tr>
</table>

Click a screenshot to view it at full size.

## Features

- **Party DPS:** class icons, coloured player bars, and damage contribution.
- **Boss encounters:** captured boss health and separate combat sessions.
- **Fight history:** the last 10 sessions, with per-player skill breakdowns.
- **Customisation:** bar styles, dimensions, spacing, opacity, and refresh rate.
- **Test mode:** a five-player encounter with boss health and detailed skill data.

## Installation

Download an **x86_64** build from [Releases](https://github.com/Netskill89/slopmeter/releases).
Both formats bundle the application libraries and capture helper.

| Format | Launch |
| --- | --- |
| **AppImage** | Make executable and run the downloaded file. |
| **tar.gz** | Extract the archive and run `AppRun` inside its folder. |

### AppImage

```sh
chmod +x SlopMeter-0.1.0-alpha.2-x86_64.AppImage
./SlopMeter-0.1.0-alpha.2-x86_64.AppImage --appimage-extract-and-run
```

The extraction option works on systems without FUSE. With FUSE available,
you can also double-click the executable AppImage.

### tar.gz

```sh
tar -xzf SlopMeter-0.1.0-alpha.2-x86_64.tar.gz
cd SlopMeter-0.1.0-alpha.2-x86_64
./AppRun
```

Run `./install.sh` to add an application-menu launcher. Keep the extracted
folder in place after installing the launcher.

### First launch

1. Launch SlopMeter as your normal desktop user.
2. Approve the desktop permission dialog if prompted to start packet capture.
3. Log into your AION 2 character. If the meter shows **Relog**, log out of the
   character and back in while SlopMeter is running.

Use the tray icon to show or hide the overlay and select a display. Open Settings
with the gear button; **Test mode** previews the layout with a complete encounter.

Under **Settings → Network capture**, choose Automatic, a specific network adapter,
or All interfaces. Click **Apply and restart capture** to activate the selection,
then relog your character. The selection is saved for future launches.

## Compatibility

### Verified environment

| Component | Tested configuration |
| --- | --- |
| Distribution | CachyOS |
| Desktop | KDE Plasma 6.7 |
| Display session | Wayland |
| CPU | AMD Ryzen 9 9950X |
| GPU | NVIDIA GeForce RTX 5070 Ti |
| Game runtime | Steam / Proton |

### Requirements

- An **x86_64 Linux** desktop with Wayland layer-shell support.
- Working graphics drivers and **Polkit (`pkexec`)** for capture authorization.
- **glibc 2.41+** for CI release builds; builds from newer systems may require a
  newer glibc.

Other distributions are **untested**. Automatic game-display detection uses KDE's
KWin integration; other supported desktops can select a display from the tray.
GNOME's default compositor does not provide the required layer-shell protocol.

### Alpha status

| Feature | Current support |
| --- | --- |
| Boss health | Current HP is read from network packets. Remaining HP percentage is shown when maximum HP is known. |

Boss health needs captured health updates and an identified boss. If those packets
have not been received, the meter cannot show its health yet. Game updates may
require decoder fixes. Test mode uses simulated data throughout.

## Updates

Open **Settings → Check updates** to check for a newer release, then choose
**Download update**. Close SlopMeter before replacing the AppImage or installing
the new tarball. Settings and fight history are preserved. Alpha builds include
prerelease updates.

## Development

```sh
git clone https://github.com/Netskill89/slopmeter.git
cd slopmeter
bash scripts/build.sh
bash scripts/test.sh
./build/ui/slopmeter
```

See [Contributing](CONTRIBUTING.md) for build dependencies and pull-request
instructions, [Implementation notes](docs/technical.md) for architecture, and
[Release instructions](docs/releases.md) for CI and packaging.

## License

[GPL-3.0](LICENSE). See [Third-party notices](THIRD_PARTY.md) for dependencies and
game artwork. SlopMeter is not affiliated with NCSOFT.
