package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/nuriland/a2kit"
	"github.com/nuriland/a2kit/game"
)

func TestDamageAccounting(t *testing.T) {
	m := meter{target: 7}
	t0 := time.Unix(100, 0)
	m.add(t0, game.Hit{Actor: 1, Target: 7, Skill: 10, Damage: 100, Extra: []uint32{500}})
	m.add(t0.Add(2*time.Second), game.Hit{Actor: 2, Target: 7, Skill: 11, Damage: 300})
	m.add(t0.Add(time.Second), game.Hit{Actor: 1, Target: 7, Skill: 10, Damage: 100})
	m.add(t0.Add(30*time.Second), game.Hit{Actor: 1, Target: 8, Damage: 999})
	if m.actors[1].damage != 200 || m.actors[1].skills[10] != 200 || m.actors[1].hits != 2 {
		t.Fatal("incorrect hit accounting")
	}
	if m.last.Sub(m.first) != 2*time.Second {
		t.Fatal("filtered hits or out-of-order hits changed duration")
	}
	var b bytes.Buffer
	m.print(&b, true)
	if !strings.Contains(b.String(), "60.0%") || !strings.Contains(b.String(), "150.0") {
		t.Fatal(b.String())
	}
}

func TestDecoderFixture(t *testing.T) {
	// The upstream README's a2log sample: one 36-damage hit and a death.
	const log = `{"schema":"a2log/v0.1","decoder":"github.com/nuriland/a2kit@v0.3.0","source":{"kind":"feed"},"t0":"2026-09-23T18:00:00.123Z"}
{"t":0,"opcode":"04 38","flags":["server"],"src":"10.0.0.2:13328","dst":"10.0.0.1:10000","payload":"x3wEAPWjAuAmqAAAAkvtn0EBAAAAkE4kAQA="}
{"t":51,"opcode":"42 36","flags":["server"],"src":"10.0.0.2:13328","dst":"10.0.0.1:10000","payload":"x3wAAw=="}
`
	r, err := a2kit.OpenReader(strings.NewReader(log), a2kit.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	m := meter{}
	for msg, err := range r.Messages() {
		if err != nil {
			t.Fatal(err)
		}
		if h, ok := msg.Event.(game.Hit); ok {
			m.add(msg.Time, h)
		}
	}
	if len(m.actors) != 1 || m.actors[37365] == nil || m.actors[37365].damage != 36 {
		t.Fatalf("unexpected decoded hit: %+v", m.actors)
	}
}

func TestSelfIdentity(t *testing.T) {
	i := identities{}
	i.observe(game.Player{Entity: 2, Name: "Other"})
	if i.name != "" {
		t.Fatal("another player was identified as self")
	}
	i.observe(game.Player{Entity: 1, Name: "My character", Self: true})
	m := meter{}
	m.add(time.Unix(100, 0), game.Hit{Actor: 1, Damage: 200})
	s := m.snapshot(i)
	if s.Character != "My character" || s.Actors[0].Name != "My character" {
		t.Fatal(s)
	}
	i.observe(game.Player{Entity: 3, Self: true})
	if i.name != "" {
		t.Fatal("stale name retained on character change")
	}
}
