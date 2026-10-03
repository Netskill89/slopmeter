package main

import (
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"github.com/nuriland/a2kit/game"
)

//go:embed data/skill-classes.json
var skillClassData []byte

var skillClasses = func() map[game.Skill]string {
	var classes map[game.Skill]string
	if err := json.Unmarshal(skillClassData, &classes); err != nil {
		panic(err)
	}
	return classes
}()

// Player identity tail and class codes: Aion2Flow Packet4536PcMetadataParser
// and PacketCharacterClassMapper (GPL-3.0; see THIRD_PARTY.md).
func className(code uint32) string {
	if code >= 5 && code <= 36 {
		return []string{"Gladiator", "Templar", "Ranger", "Assassin", "Elementalist", "Sorcerer", "Cleric", "Chanter"}[(code-5)/4]
	}
	if code >= 45 && code <= 48 {
		return "Brawler"
	}
	return ""
}

func (s *scope) observeClass(m aMessage) {
	// Only confirmed participants' direct damage provides fallback evidence.
	if hit, ok := m.event.(game.Hit); ok && s.accepts(hit.Actor, m.t) && s.classes[hit.Actor] == "" {
		if name := skillClasses[hit.Skill]; name != "" {
			if s.classes == nil {
				s.classes = make(map[game.Entity]string)
			}
			s.classes[hit.Actor] = name
		}
	}
	if m.opcode != 0x3645 {
		return
	}
	id, rest, ok := readVar(m.payload)
	if !ok || id == 0 {
		return
	}
	// Walk prefix varints until the documented name marker, then require the
	// already-decoded identity's exact name and a valid class tail.
	for len(rest) > 0 {
		if (rest[0] == 0x07 || rest[0] == 0x17) && len(rest) >= 2 {
			n := int(rest[1])
			end := 2 + n
			if n > 0 && n <= 72 && end+4 <= len(rest) && s.names[game.Entity(id)] == string(rest[2:end]) {
				name := className(binary.LittleEndian.Uint32(rest[end:]))
				if name != "" {
					if s.classes == nil {
						s.classes = make(map[game.Entity]string)
					}
					s.classes[game.Entity(id)] = name
					return
				}
			}
		}
		_, rest, ok = readVar(rest)
		if !ok {
			return
		}
	}
}
