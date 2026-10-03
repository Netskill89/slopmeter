package main

import (
	"encoding/binary"
	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
	"testing"
	"time"
)

func directParty(id uint32, slot byte, name string) []byte {
	p := make([]byte, 56+len(name))
	p[0] = 3
	p[1] = slot
	binary.LittleEndian.PutUint32(p[2:], id)
	binary.LittleEndian.PutUint16(p[6:], 1001)
	p[10] = 36
	copy(p[11:], "01234567-89ab-cdef-0123-456789abcdef")
	binary.LittleEndian.PutUint16(p[53:], 1001)
	p[55] = byte(len(name))
	copy(p[56:], name)
	return p
}
func TestPartyScope(t *testing.T) {
	t0 := time.Unix(100, 0)
	s := scope{}
	if s.accepts(1, t0) {
		t.Fatal("accepted before self identification")
	}
	s.observeMessage(aMessage{flags: wire.FromServer, event: game.Player{Entity: 1, Name: "Self", Self: true}, t: t0})
	s.observeMessage(aMessage{flags: wire.FromServer, event: game.Player{Entity: 2, Name: "Nearby"}, t: t0})
	if !s.accepts(1, t0) || s.accepts(2, t0) {
		t.Fatal("nearby player included")
	}
	s.observeMessage(aMessage{opcode: 0x920d, flags: wire.FromServer, payload: directParty(3, 2, "Party"), t: t0})
	if !s.accepts(3, t0) || s.names[3] != "Party" {
		t.Fatal("confirmed party excluded")
	}
	s.observeMessage(aMessage{opcode: 0x920d, flags: wire.FromServer, payload: directParty(4, 2, "Replacement"), t: t0})
	if s.accepts(3, t0) || !s.accepts(4, t0) {
		t.Fatal("slot replacement not applied")
	}
	if s.accepts(4, t0.Add(memberLease+time.Millisecond)) {
		t.Fatal("expired member retained")
	}
	s.observeMessage(aMessage{opcode: 0x3615, flags: wire.FromServer, t: t0})
	if s.accepts(1, t0) || s.accepts(4, t0) || s.name != "" {
		t.Fatal("login retained stale identity or group")
	}
}
func TestPartyStatusValidation(t *testing.T) {
	t0 := time.Unix(100, 0)
	s := scope{identities: identities{self: 1}}
	p := []byte{2, 50, 100}
	p = append(p, make([]byte, 25)...)
	s.observeMessage(aMessage{opcode: 0x921b, flags: wire.FromServer, payload: p, t: t0})
	if !s.accepts(2, t0) {
		t.Fatal("valid party status excluded")
	}
	for _, payload := range [][]byte{{3, 50, 100}, append([]byte{3, 101, 100}, make([]byte, 25)...), append([]byte{3, 50, 0}, make([]byte, 25)...)} {
		s.observeMessage(aMessage{opcode: 0x921b, flags: wire.FromServer, payload: payload, t: t0})
		if s.accepts(3, t0) {
			t.Fatal("malformed party status accepted")
		}
	}
	s.observeMessage(aMessage{opcode: 0x920d, flags: wire.FromClient, payload: directParty(4, 1, "Wrong direction"), t: t0})
	if s.accepts(4, t0) {
		t.Fatal("client-side roster accepted")
	}
}
func TestRosterReplacesParty(t *testing.T) {
	t0 := time.Unix(100, 0)
	s := scope{identities: identities{self: 1}}
	s.confirm(3, 1, "Old", t0)
	row := directParty(4, 2, "New")
	row = append([]byte{0}, row...)
	row[0] = 5
	row[1] = 0
	p := append(make([]byte, 25), row...)
	s.observeMessage(aMessage{opcode: 0x9200, flags: wire.FromServer, payload: p, t: t0})
	if s.accepts(3, t0) || !s.accepts(4, t0) {
		t.Fatal("full roster did not replace membership")
	}
}
func TestExcludedDamageDoesNotChangeTotals(t *testing.T) {
	t0 := time.Unix(100, 0)
	s := scope{identities: identities{self: 1}}
	s.confirm(2, 1, "Party", t0)
	m := meter{}
	for _, h := range []game.Hit{{Actor: 1, Damage: 100}, {Actor: 2, Damage: 200}, {Actor: 3, Damage: 9000}} {
		if s.accepts(h.Actor, t0) {
			m.add(t0, h)
		}
	}
	if len(m.actors) != 2 || m.actors[3] != nil {
		t.Fatal("nearby damage was counted")
	}
}
