package main

import (
	"encoding/binary"
	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
	"strings"
	"testing"
	"time"
)

func TestBossHealthBeforeMetadata(t *testing.T) {
	e, now := encounterFixture()
	e.health(now, 20, 700, 2000)
	e.npc(20, bossCode())
	e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 100})
	s := e.snapshot(identities{}, now)
	if s.Boss == nil || s.Boss.HP != 700 || s.Boss.Percent != 35 || s.Boss.Name != "Fediv Wraith" {
		t.Fatalf("earlier health lost: %+v", s.Boss)
	}
}

func TestBossMetadataAfterLastHit(t *testing.T) {
	e, now := encounterFixture()
	e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 100})
	e.health(now.Add(time.Second), 20, 700, 2000)
	if e.snapshot(identities{}, now).Boss != nil {
		t.Fatal("unknown target displayed as boss")
	}
	e.npc(20, bossCode())
	s := e.snapshot(identities{}, now.Add(time.Second))
	if s.Boss == nil || s.Boss.HP != 700 || s.Boss.Percent != 35 {
		t.Fatalf("late boss identity requires another hit: %+v", s.Boss)
	}
	if s.Actors[0].Damage != 100 {
		t.Fatal("metadata discarded damage")
	}
	e.hit(now.Add(2*time.Second), game.Hit{Actor: 1, Target: 21, Damage: 50})
	if e.boss != 20 || e.meter.actors[1].damage != 100 {
		t.Fatal("late identity did not promote boss or trash entered boss totals")
	}
}

func TestExtendedBossHealthWithAdditionalTail(t *testing.T) {
	e, now := encounterFixture()
	rest := make([]byte, 42)
	rest[0], rest[1] = 0x85, 0x21
	binary.LittleEndian.PutUint32(rest[5:], bossCode())
	rest = binary.AppendUvarint(rest, 600)
	rest = binary.AppendUvarint(rest, 2000)
	rest = append(rest, 100, 0, 0, 0, 100, 0, 0, 0)
	rest = append(rest, make([]byte, 80)...)
	p := append(binary.AppendUvarint(nil, 20), rest...)
	e.observe(aMessage{opcode: 0x3641, flags: wire.FromServer, payload: p, t: now})
	e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 100})
	s := e.snapshot(identities{}, now)
	if s.Boss == nil || !s.Boss.MaxKnown || s.Boss.HP != 600 || s.Boss.Percent != 30 {
		t.Fatalf("extra tail hid valid extended HP: %+v", s.Boss)
	}
	rest[46] = 99 // Corrupt the resource-gauge sentinel; do not accept arbitrary numbers.
	e2, _ := encounterFixture()
	e2.observe(aMessage{opcode: 0x3641, flags: wire.FromServer, payload: append(binary.AppendUvarint(nil, 20), rest...), t: now})
	e2.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 100})
	if e2.snapshot(identities{}, now).Boss.Known {
		t.Fatal("invalid HP sentinel accepted")
	}
}

func TestHealthDiagnosticsPreserveUnknownNPCObservation(t *testing.T) {
	e, now := encounterFixture()
	e.health(now, 20, 600, 2000)
	e.observe(aMessage{event: game.Spawn{Entity: 20, NPC: 2000001}, flags: wire.FromServer, t: now})
	e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 100})
	s := e.snapshot(identities{}, now)
	if s.Boss != nil || !strings.Contains(s.HealthStatus, "2000001") {
		t.Fatalf("unknown NPC incorrectly displayed: %+v", s)
	}
	if e.targets[20].hp != 600 || !e.targets[20].known {
		t.Fatal("NPC metadata erased observed HP")
	}
	e.npc(20, bossCode())
	if e.snapshot(identities{}, now).Boss.Known {
		t.Fatal("new NPC code reused previous entity HP")
	}
}
