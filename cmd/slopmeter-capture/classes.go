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
	// Class-specific casts also identify healers/buffers before their first hit.
	// Shared/unmapped skills and nearby players cannot provide evidence.
	var actor game.Entity
	var skill game.Skill
	switch event := m.event.(type) {
	case game.Hit:
		actor, skill = event.Actor, event.Skill
	case game.Cast:
		actor, skill = event.Actor, event.Skill
	case game.CastEnd:
		actor, skill = event.Actor, event.Skill
	}
	if actor != 0 && s.accepts(actor, m.t) && s.classes[actor] == "" {
		s.setClass(actor, skillClasses[skill])
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
					s.setClass(game.Entity(id), name)
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

func (s *scope) setClass(id game.Entity, name string) {
	if name == "" || s.classes[id] == name {
		return
	}
	if s.classes == nil {
		s.classes = make(map[game.Entity]string)
	}
	s.classes[id] = name
	s.classRevision++
}
