Party payload layouts in cmd/slopmeter-capture/party.go were derived from the GPL-3.0 Aion2Flow source:
https://github.com/cloris-chan/Aion2Flow/blob/main/src/Aion2Flow.Protocol/Packets/PacketPlayerGroupParser.cs
Copyright the Aion2Flow contributors. This application is distributed under
GPL-3.0; see LICENSE. a2kit remains MIT licensed under its own upstream license.

NPC health layouts in cmd/slopmeter-capture/encounter.go and the NPC definitions in cmd/slopmeter-capture/data/bosses.json
also derive from Aion2Flow commit 8c7c3f3ce7770382afc646e74a5d97bca79b24aa:
- Packet008DRemainHpParser.cs, Packet4136Parser.cs, Packet4036CreateParser.cs,
  and PacketNpcStateFields.cs in src/Aion2Flow.Protocol/Packets.
- src/Aion2Flow.Resources/Packs/shared.bin and en-US.bin, decoded according to
  ResourcePackReader.cs (format 14), with pack checksums verified before extraction.
Boss definitions, English NPC names, HP display divisors, and PC skill/class
mappings were retained.

Class metadata parsing and class codes in cmd/slopmeter-capture/classes.go derive from
Packet4536PcMetadataParser.cs, NicknameParserUtil.cs and
PacketCharacterClassMapper.cs at the same Aion2Flow commit. The embedded
skill/class mapping in cmd/slopmeter-capture/data/skill-classes.json was extracted from shared.bin
SkillDefinitions and SkillBaseProjections sections (format 14), with its checksum
verified. Only class-bearing PC skills and their projections are retained;
CombatantClassEvidence.cs provides the reference for direct damage evidence.
Unknown.svg is the fallback for an unrecognised class.

Skill names/base projections in cmd/slopmeter-capture/data/skills.json and the skill-code/icon mapping
were extracted from Aion2Flow's en-US.bin (SkillNames section), shared.bin
(SkillBaseProjections section), and Generated/SkillIconCatalog.g.cs at commit
8c7c3f3ce7770382afc646e74a5d97bca79b24aa. Pack checksums were verified.
425 WebP images in ui/skill-icons and the Brawler class image come from
src/Aion2Flow/Assets/Images in that repository. AION 2 game artwork belongs to
NCSOFT; source of the bundled skill images:
https://github.com/cloris-chan/Aion2Flow/tree/8c7c3f3ce7770382afc646e74a5d97bca79b24aa/src/Aion2Flow/Assets/Images/Skills
The other class images were supplied by the project author;
ui/class-icons contains PNG format conversions, with SpiritMaster mapped to
Elementalist. Combat crit flag type 3 was cross-checked with
Packet0438DamageParser.cs. Uses are observed a2kit Cast events; damage hit counts
are not treated as uses or successful-attempt denominators.

Release bundles also contain dynamically linked Qt libraries (LGPL/GPL options)
and LayerShellQt (LGPL), alongside their dependency libraries. Distribution
copyright/license notices are retained under
usr/share/doc/slopmeter/bundled-licenses in the release AppDir. Qt source is
available from https://code.qt.io/ and distribution sources from
https://sources.debian.org/; LayerShellQt source is at
https://invent.kde.org/plasma/layer-shell-qt. The MIT license of a2kit is
reproduced in LICENSES/a2kit-MIT.txt. Original application source and build
scripts are included with tagged releases.

Release capture payloads include Wireshark dumpcap (GPL-2.0-or-later) and
util-linux setpriv (GPL-2.0-or-later), with their dependency libraries.
Sources: https://www.wireshark.org/download/src/ and
https://www.kernel.org/pub/linux/utils/util-linux/ . Distribution license notices
are included with the bundled-library notices in the release.
