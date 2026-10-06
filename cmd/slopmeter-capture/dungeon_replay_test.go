package main

import (
	"testing"
	"time"

	"github.com/nuriland/a2kit"
)

func TestDungeonReplayFromRawPackets(t *testing.T) {
	// Synthetic public fixture: current masked spawn, map state, two group hits
	// ten minutes apart, a damage tick, an excluded healing tick and a HP record embedded in another opcode's payload.
	r, err := a2kit.Open("../../tests/fixtures/dungeon.jsonl", a2kit.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	e := newEncounter(0, 5*time.Second, 90*time.Second)
	group := scope{}
	var end time.Time
	for msg, err := range r.Messages() {
		if err != nil {
			t.Fatal(err)
		}
		end = msg.Time
		am := aMessage{msg.Opcode, msg.Flags, msg.Payload, msg.Event, msg.Time}
		e.observe(am)
		e.tick(msg.Time)
		group.observeMessage(am)
		if h, ok := combatHit(am); ok && group.accepts(h.Actor, msg.Time) && !group.accepts(h.Target, msg.Time) {
			e.hit(msg.Time, h)
		}
	}
	s := e.snapshot(group.identities, end)
	if s.Character != "Play" || s.Scene.Name != "Fire Temple" || s.Scene.ID != 600021 || s.Session != 1 || !s.Active {
		t.Fatalf("dungeon session not identified: %+v", s)
	}
	if s.Boss == nil || s.Boss.Name != "Fediv Wraith" || s.Boss.HP != 600 || s.Boss.Percent != 30 {
		t.Fatalf("raw boss metadata or HP missing: %+v", s.Boss)
	}
	if len(s.Actors) != 1 || s.Actors[0].Damage != 550 || s.Duration != 599 {
		t.Fatalf("downtime split totals: %+v", s)
	}
}
