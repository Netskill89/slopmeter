package main

import (
	"bytes"
	"context"
	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
	"strings"
	"testing"
	"time"
)

func TestRememberedCharacterRequiresFreshIdentity(t *testing.T) {
	s := scope{}
	s.setCharacterHint("Self")
	now := time.Now()
	if s.accepts(1, now) {
		t.Fatal("hint alone accepted damage")
	}
	observe := func(p game.Player) { s.observeMessage(aMessage{event: p, flags: wire.FromServer, t: now}) }
	observe(game.Player{Entity: 2, Name: "Nearby"})
	if s.self != 0 {
		t.Fatal("nearby player became self")
	}
	observe(game.Player{Entity: 1, Name: "Self"})
	if s.self != 1 || !s.hintedSelf || !s.accepts(1, now) {
		t.Fatal("fresh name match failed")
	}
	s.observeMessage(aMessage{opcode: 0x3623, flags: wire.FromServer, t: now})
	if s.self != 0 || s.characterHint != "Self" || s.accepts(1, now) {
		t.Fatal("zone retained old entity identity")
	}
	observe(game.Player{Entity: 9, Name: "Self"})
	if s.self != 9 {
		t.Fatal("new zone entity was not rebound")
	}
	observe(game.Player{Entity: 10, Name: "OtherCharacter", Self: true})
	if s.self != 10 || s.hintedSelf || s.characterHint != "OtherCharacter" {
		t.Fatal("confirmed self did not override hint")
	}
	s.observeMessage(aMessage{opcode: 0x3611, flags: wire.FromServer, t: now})
	if s.self != 0 || s.characterHint != "OtherCharacter" {
		t.Fatal("logout cache invalidation failed")
	}
}
func TestAmbiguousCharacterHint(t *testing.T) {
	s := scope{}
	s.setCharacterHint("Same")
	for _, id := range []game.Entity{1, 2} {
		s.observeMessage(aMessage{flags: wire.FromServer, event: game.Player{Entity: id, Name: "Same"}})
	}
	if s.self != 0 {
		t.Fatal("duplicate player names established identity")
	}
}
func TestCharacterHintControl(t *testing.T) {
	names := make(chan string, 1)
	var report bytes.Buffer
	readRefreshCommands(context.Background(), strings.NewReader("{\"characterName\":\"First\"}\n{\"characterName\":\"Second\"}\n"), nil, nil, &report, names)
	if <-names != "Second" || report.Len() != 0 {
		t.Fatal("name command not applied/coalesced")
	}
	for _, bad := range []string{strings.Repeat("x", 73), "Name\nBad", string([]byte{0xff})} {
		if validCharacterHint(bad) {
			t.Fatal("invalid hint accepted")
		}
	}
}
