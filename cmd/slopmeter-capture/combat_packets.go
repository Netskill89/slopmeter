package main

import (
	_ "embed"
	"encoding/binary"
	"encoding/json"

	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
)

// Periodic damage allowlist and layout reference: A2Tools, see THIRD_PARTY.md.
//
//go:embed data/dot-skills.json
var dotSkillJSON []byte
var dotSkills = func() map[uint32]bool {
	var ids []uint32
	if err := json.Unmarshal(dotSkillJSON, &ids); err != nil {
		panic(err)
	}
	out := make(map[uint32]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out
}()

// Typed direct hits remain authoritative. Parse only validated periodic damage
// when a2kit has no typed event, never an alternative copy of the same hit.
func combatHit(m aMessage) (game.Hit, bool) {
	if m.flags&wire.FromServer == 0 {
		return game.Hit{}, false
	}
	if h, ok := m.event.(game.Hit); ok {
		return h, h.Actor != 0 && h.Target != 0 && h.Actor != h.Target && h.Damage > 0
	}
	if m.event != nil || m.opcode != 0x3805 {
		return game.Hit{}, false
	}
	target, rest, ok := readVar(m.payload)
	if !ok || target == 0 {
		return game.Hit{}, false
	}
	kind, rest, ok := readVar(rest)
	// Exact effects, not a bit mask: 0x0B is healing over time, not damage.
	if !ok || (kind != 2 && kind != 10) {
		return game.Hit{}, false
	}
	actor, rest, ok := readVar(rest)
	if !ok || actor == 0 || actor == target {
		return game.Hit{}, false
	}
	_, rest, ok = readVar(rest)
	if !ok || len(rest) < 4 {
		return game.Hit{}, false
	}
	skill := binary.LittleEndian.Uint32(rest)
	if !dotSkills[skill/100] {
		return game.Hit{}, false
	}
	amount, tail, ok := readVar(rest[4:])
	if !ok || amount == 0 || amount > 99_999_999 {
		return game.Hit{}, false
	}
	// Aion2Flow documents optional prefix/skill-reference tails. Keep them out
	// of the amount, and reject unrelated or truncated layouts.
	switch {
	case len(tail) == 0, len(tail) == 4:
	case len(tail) < 4:
		_, remaining, valid := readVar(tail)
		if !valid || len(remaining) != 0 {
			return game.Hit{}, false
		}
	default:
		_, remaining, valid := readVar(tail[:len(tail)-4])
		if !valid || len(remaining) != 0 {
			return game.Hit{}, false
		}
	}
	// Periodic ticks have no decoded critical flag. Type zero keeps their damage
	// and hit count without incorrectly treating them as critical/normal samples.
	return game.Hit{Actor: game.Entity(actor), Target: game.Entity(target), Skill: game.Skill(skill), Damage: amount}, true
}
