package main

// Group payload layouts were checked against Aion2Flow's PacketPlayerGroupParser
// (GPL-3.0). See THIRD_PARTY.md. Unknown layouts fail closed.
import (
	"encoding/binary"
	"time"
	"unicode/utf8"

	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
)

type membership struct {
	seen time.Time
	slot byte
}
type scope struct {
	characterHint string
	hintedSelf    bool
	identities
	members       map[game.Entity]membership
	classRevision uint64
}

const memberLease = 10 * time.Second

func readVar(p []byte) (uint32, []byte, bool) {
	v, n := binary.Uvarint(p)
	if n <= 0 || n > 5 || v > 0xffffffff {
		return 0, nil, false
	}
	return uint32(v), p[n:], true
}
func uuidOK(p []byte) bool {
	if len(p) != 36 {
		return false
	}
	for i, c := range p {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}
func partyRow(p []byte, list bool) (game.Entity, byte, string, int, bool) {
	base := 0
	nameOffset := 55
	if list {
		base = 1
		nameOffset = 56
	}
	if len(p) <= nameOffset {
		return 0, 0, "", 0, false
	}
	if list {
		if p[0] > 5 || p[1] > 7 {
			return 0, 0, "", 0, false
		}
	} else if p[0] != 3 {
		return 0, 0, "", 0, false
	}
	slot := p[1+base]
	id := binary.LittleEndian.Uint32(p[2+base:])
	server := binary.LittleEndian.Uint16(p[6+base:])
	if slot < 1 || slot > 6 || id == 0 || id > 0x7fffffff || server < 1000 || server > 2999 || p[10+base] != 36 || !uuidOK(p[11+base:47+base]) || binary.LittleEndian.Uint16(p[53+base:]) != server {
		return 0, 0, "", 0, false
	}
	n := int(p[nameOffset])
	end := nameOffset + 1 + n
	if n == 0 || end > len(p) || !utf8.Valid(p[nameOffset+1:end]) {
		return 0, 0, "", 0, false
	}
	return game.Entity(id), slot, string(p[nameOffset+1 : end]), end, true
}
func (s *scope) confirm(id game.Entity, slot byte, name string, t time.Time) {
	if s.members == nil {
		s.members = make(map[game.Entity]membership)
	}
	if s.names == nil {
		s.names = make(map[game.Entity]string)
	}
	if slot != 0 {
		for other, m := range s.members {
			if other != id && m.slot == slot {
				delete(s.members, other)
			}
		}
	}
	if old, ok := s.members[id]; ok && slot == 0 {
		slot = old.slot
	}
	s.members[id] = membership{t, slot}
	if name != "" {
		s.names[id] = name
	}
}
func (s *scope) observeMessage(m aMessage) {
	if m.flags&wire.FromServer == 0 {
		return
	}
	if p, ok := m.event.(game.Player); ok && p.Self && s.self != 0 && s.self != p.Entity {
		s.members = nil
		s.names = nil
		s.classes = nil
		s.stats = nil
	}
	if m.opcode == 0x3623 && s.hintedSelf {
		s.self = 0
		s.name = ""
		s.names = nil
		s.classes = nil
		s.members = nil
		s.stats = nil
		s.hintedSelf = false
	}
	s.identities.observe(m.event)
	s.observeCharacterHint(m.event)
	s.observeClass(m)
	if m.opcode == 0x3611 || m.opcode == 0x3615 {
		s.identities = identities{}
		s.members = nil
		s.hintedSelf = false
		return
	}
	s.observeRosterStats(m)
	p := m.payload
	switch m.opcode {
	case 0x921b: // Party resource status: entity/current/max varints + fixed 25-byte tail.
		id, rest, ok := readVar(p)
		if !ok || id == 0 {
			return
		}
		current, rest, ok := readVar(rest)
		if !ok {
			return
		}
		max, rest, ok := readVar(rest)
		if !ok || max == 0 || current > max || len(rest) != 25 || rest[24] > 1 {
			return
		}
		s.confirm(game.Entity(id), 0, "", m.t)
	case 0x920d:
		id, slot, name, _, ok := partyRow(p, false)
		if ok {
			s.confirm(id, slot, name, m.t)
		}
	case 0x9200:
		offset := 25
		if len(p) > 34 && p[0] == 1 {
			offset = 34
		}
		roster := make(map[game.Entity]membership)
		names := make(map[game.Entity]string)
		for offset+57 <= len(p) && len(roster) < 6 {
			id, slot, name, end, ok := partyRow(p[offset:], true)
			if !ok {
				offset++
				continue
			}
			roster[id] = membership{m.t, slot}
			names[id] = name
			offset += max(end+32, 112)
		}
		// Never infer an empty roster from an unknown payload.
		if len(roster) > 0 {
			s.members = roster
			for id, name := range names {
				if s.names == nil {
					s.names = make(map[game.Entity]string)
				}
				s.names[id] = name
			}
		}
	}
}

type aMessage struct {
	opcode  wire.Opcode
	flags   wire.Flags
	payload []byte
	event   game.Event
	t       time.Time
}

func (s *scope) accepts(id game.Entity, t time.Time) bool {
	if s.self == 0 {
		return false
	}
	if id == s.self {
		return true
	}
	m, ok := s.members[id]
	if !ok {
		return false
	}
	age := t.Sub(m.seen)
	return age >= 0 && age <= memberLease
}
