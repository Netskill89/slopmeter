package main

import (
	"encoding/binary"
	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
	"testing"
	"time"
)

func TestClassIdentityAndFallback(t *testing.T) {
	now := time.Now()
	s := scope{identities: identities{self: 1, names: map[game.Entity]string{1: "Self"}}}
	// Entity varint, identity mask, name marker, length/name, class code.
	p := []byte{1, 1, 0x20, 0, 0, 7, 4, 'S', 'e', 'l', 'f', 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(p[11:], 9)
	s.observeMessage(aMessage{opcode: 0x3645, flags: wire.FromServer, payload: p, t: now})
	if s.classes[1] != "Templar" {
		t.Fatal("valid class identity not detected")
	}
	p[7] = 'X'
	binary.LittleEndian.PutUint32(p[11:], 25)
	s.observeMessage(aMessage{opcode: 0x3645, flags: wire.FromServer, payload: p, t: now})
	if s.classes[1] != "Templar" {
		t.Fatal("mismatched identity changed class")
	}
	var skill game.Skill
	for id, name := range skillClasses {
		if name == "Cleric" {
			skill = id
			break
		}
	}
	if skill == 0 {
		t.Fatal("missing class skill catalogue")
	}
	s.confirm(2, 1, "Party", now)
	s.observeMessage(aMessage{flags: wire.FromServer, event: game.Hit{Actor: 2, Skill: skill}, t: now})
	s.observeMessage(aMessage{flags: wire.FromServer, event: game.Hit{Actor: 3, Skill: skill}, t: now})
	if s.classes[2] != "Cleric" || s.classes[3] != "" {
		t.Fatal("skill fallback ignored party scope")
	}
	m := meter{}
	m.add(now, game.Hit{Actor: 2, Damage: 100})
	if m.snapshot(s.identities).Actors[0].Class != "Cleric" {
		t.Fatal("class missing from snapshot")
	}
	s.observeMessage(aMessage{opcode: 0x3615, flags: wire.FromServer, t: now})
	if len(s.classes) != 0 {
		t.Fatal("class retained across login")
	}
}

func TestClassCodes(t *testing.T) {
	for _, code := range []uint32{0, 4, 37, 44, 49, 0xffffffff} {
		if className(code) != "" {
			t.Fatalf("unknown code %d accepted", code)
		}
	}
	if className(5) != "Gladiator" || className(36) != "Chanter" || className(48) != "Brawler" {
		t.Fatal("class code boundaries")
	}
}
