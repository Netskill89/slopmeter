package main

import (
	"encoding/binary"
	"unicode/utf8"

	"github.com/nuriland/a2kit/game"
)

// Promote late identity in place, keeping only this target's damage. Metadata
// must not create a new session, count trash against a boss, or rewind the clock.
func (e *encounter) promoteBoss(id game.Entity) {
	if !e.active || e.boss != 0 {
		return
	}
	target := e.targetMeters[id]
	if target == nil || target.first.IsZero() {
		return
	}
	e.boss = id
	e.meter = *target
	e.lastDamage = target.last
	e.status = "Boss fight"
}

// Current masked spawn: entity + u32 mask + name gate + optional UTF-8 name
// + u32 NPC code. Older clients use a u16 mask. Aion2Flow supplies the catalog;
// A2Tools documents the mask-width change and optional subtree names.
func (e *encounter) observeModernNPC(m aMessage) {
	if m.opcode != 0x3640 && m.opcode != 0x3641 {
		return
	}
	id, rest, ok := readVar(m.payload)
	if !ok || id == 0 {
		return
	}
	for _, width := range []int{4, 2} {
		if len(rest) < width+5 {
			continue
		}
		gate := rest[width]
		offset := width + 1
		if gate&1 != 0 {
			length, tail, valid := readVar(rest[offset:])
			if !valid || length == 0 || length > 128 || uint32(len(tail)) < length {
				continue
			}
			name := tail[:length]
			if !utf8.Valid(name) {
				continue
			}
			offset = len(rest) - len(tail) + int(length)
		}
		if offset+4 > len(rest) {
			continue
		}
		code := binary.LittleEndian.Uint32(rest[offset:])
		if _, confirmed := bossCatalog[code]; !confirmed {
			continue
		}
		e.npc(game.Entity(id), code)
		// Optional fields move the HP pair. Only accept a bounded pair followed
		// by BOTH fixed 100/100 resource gauges; never infer max from observed HP.
		for at := offset + 16; at < min(len(rest), offset+85); at++ {
			if current, maximum, valid := hpPair(rest, at); valid {
				e.health(m.t, game.Entity(id), current, maximum)
				break
			}
		}
		return
	}
}

// A2Tools' current-HP feed can be embedded inside another combat record rather
// than emitted as a standalone 00 8D packet. Require its complete discriminator
// and zero trailer to avoid treating other values as HP. Readings may precede
// identity; retaining HP never classifies or displays an ordinary enemy. The
// scan is linear in the framed payload and needs no temporary buffers.
func (e *encounter) observeEmbeddedHealth(m aMessage) {
	p := m.payload
	for at := 0; at+12 <= len(p); at++ {
		if p[at] != 0x8d {
			continue
		}
		id, rest, ok := readVar(p[at+1:])
		if !ok || id == 0 || id > 9_999_999 || len(rest) < 11 {
			continue
		}
		if rest[0] != 2 || rest[1] != 1 || rest[2] != 0 || binary.LittleEndian.Uint32(rest[7:]) != 0 {
			continue
		}
		current := binary.LittleEndian.Uint32(rest[3:])
		n := e.targets[game.Entity(id)]
		if n != nil && n.max > 0 && current > n.max || (n == nil || n.max == 0) && current > 100_000_000 {
			continue
		}
		e.health(m.t, game.Entity(id), current, 0)
		at = len(p) - len(rest) + 10
	}
}
