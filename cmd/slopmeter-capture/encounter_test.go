package main

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
)

func bossCode() uint32 { return 2090175 }
func encounterFixture() (*encounter, time.Time) {
	return newEncounter(0, 5*time.Second, 90*time.Second), time.Unix(100, 0)
}
func TestOpenWorldIdleAndNextPull(t *testing.T) {
	e, t0 := encounterFixture()
	e.hit(t0, game.Hit{Actor: 1, Target: 7, Damage: 100})
	e.hit(t0.Add(time.Second), game.Hit{Actor: 1, Target: 8, Damage: 200})
	if e.tick(t0.Add(5 * time.Second)) {
		t.Fatal("idle counted from first hit")
	}
	if !e.tick(t0.Add(6*time.Second)) || e.active || len(e.meter.actors) != 0 {
		t.Fatal("idle did not clear current encounter")
	}
	e.hit(t0.Add(7*time.Second), game.Hit{Actor: 1, Target: 9, Damage: 50})
	if e.number != 2 || e.meter.actors[1].damage != 50 {
		t.Fatal("new pull retained damage")
	}
}
func TestBossIsolationAndLongMechanic(t *testing.T) {
	e, t0 := encounterFixture()
	e.npc(20, bossCode())
	e.health(t0, 20, 1000, 1000)
	e.hit(t0, game.Hit{Actor: 1, Target: 9, Damage: 500})
	e.hit(t0.Add(time.Second), game.Hit{Actor: 1, Target: 20, Damage: 100})
	e.hit(t0.Add(2*time.Second), game.Hit{Actor: 1, Target: 9, Damage: 9000})
	if e.meter.actors[1].damage != 100 || e.boss != 20 {
		t.Fatal("trash included in boss totals")
	}
	if e.tick(t0.Add(20*time.Second)) || !e.active {
		t.Fatal("boss mechanic split encounter")
	}
	s := e.snapshot(identities{}, t0.Add(21*time.Second))
	if s.Duration != 20 || s.Actors[0].DPS != 5 {
		t.Fatal("boss DPS must include mechanics", s)
	}
	e.health(t0.Add(22*time.Second), 20, 0, 0)
	if e.active || e.status != "Boss defeated" {
		t.Fatal("boss death did not finish encounter")
	}
	s = e.snapshot(identities{}, t0.Add(100*time.Second))
	if s.Duration != 21 {
		t.Fatal("completed duration was not frozen")
	}
	e.health(t0.Add(101*time.Second), 20, 1000, 1000)
	e.hit(t0.Add(102*time.Second), game.Hit{Actor: 1, Target: 20, Damage: 80})
	if e.meter.actors[1].damage != 80 || e.number != 3 {
		t.Fatal("repull retained totals")
	}
}
func TestBossResetAndOpenWorldAfterKill(t *testing.T) {
	e, t0 := encounterFixture()
	e.npc(20, bossCode())
	e.health(t0, 20, 1000, 1000)
	e.hit(t0, game.Hit{Actor: 1, Target: 20, Damage: 100})
	e.health(t0.Add(time.Second), 20, 800, 0)
	e.health(t0.Add(2*time.Second), 20, 1000, 0)
	if e.active {
		t.Fatal("reset to full health did not finish fight")
	}
	e.hit(t0.Add(3*time.Second), game.Hit{Actor: 1, Target: 9, Damage: 50})
	if e.boss != 0 || e.meter.target != 0 || e.meter.actors[1].damage != 50 {
		t.Fatal("boss filter leaked into open world")
	}
}
func TestBossPacketHealthAndUnknownMaximum(t *testing.T) {
	e, t0 := encounterFixture()
	p := binary.AppendUvarint(nil, 20)
	rest := make([]byte, 30)
	rest[0] = 5
	rest[1] = 0x20
	binary.LittleEndian.PutUint32(rest[5:], bossCode())
	p = append(p, rest...)
	p = binary.AppendUvarint(p, 1000)
	p = binary.AppendUvarint(p, 2000)
	p = append(p, 100, 0, 0, 0, 100, 0, 0, 0)
	e.observe(aMessage{opcode: 0x3641, flags: wire.FromServer, payload: p, t: t0})
	e.hit(t0, game.Hit{Actor: 1, Target: 20, Damage: 100})
	hp := binary.AppendUvarint(nil, 20)
	hp = append(hp, 2, 1, 0)
	hp = binary.LittleEndian.AppendUint32(hp, 600)
	e.observe(aMessage{opcode: 0x8d00, flags: wire.FromServer, payload: hp, t: t0.Add(time.Second)})
	s := e.snapshot(identities{}, t0.Add(time.Second))
	if s.Boss == nil || !s.Boss.MaxKnown || s.Boss.HP != 600 || s.Boss.Percent != 30 || s.Boss.Name != "Fediv Wraith" {
		t.Fatal("incorrect packet HP", s.Boss)
	}
	hp[1] = 3 // A non-health value must not overwrite HP.
	e.observe(aMessage{opcode: 0x8d00, flags: wire.FromServer, payload: hp, t: t0})
	if e.npcs[20].hp != 600 {
		t.Fatal("non-health resource accepted")
	}
	e2, _ := encounterFixture()
	e2.npc(20, bossCode())
	e2.health(t0, 20, 700, 0)
	e2.hit(t0, game.Hit{Actor: 1, Target: 20, Damage: 10})
	if e2.snapshot(identities{}, t0).Boss.MaxKnown {
		t.Fatal("invented maximum HP")
	}
}
func TestRelativeBars(t *testing.T) {
	m := meter{}
	t0 := time.Unix(100, 0)
	m.add(t0, game.Hit{Actor: 1, Damage: 100})
	m.add(t0, game.Hit{Actor: 2, Damage: 50})
	s := m.snapshot(identities{})
	if s.Actors[0].Fill != 100 || s.Actors[1].Fill != 50 {
		t.Fatal("bars are not relative to leader")
	}
}
func TestBossIdleDeathAndZone(t *testing.T) {
	e, t0 := encounterFixture()
	e.npc(20, bossCode())
	e.hit(t0, game.Hit{Actor: 1, Target: 20, Damage: 10})
	e.observe(aMessage{event: game.Death{Entity: 20, Flag: 3}, flags: wire.FromServer, t: t0.Add(time.Second)})
	if e.active {
		t.Fatal("death packet did not stop boss fight")
	}
	e.scene.ID = 600021
	e.scene.Name = mapCatalog[600021]
	e.scene.candidate = 600031
	e.observe(aMessage{opcode: 0x3623, payload: make([]byte, 20), flags: wire.FromServer, t: t0.Add(2 * time.Second)})
	if e.active || e.boss != 0 || len(e.npcs) != 0 {
		t.Fatal("zone retained old boss")
	}
	e.npc(21, bossCode())
	e.hit(t0.Add(3*time.Second), game.Hit{Actor: 1, Target: 21, Damage: 20})
	if e.tick(t0.Add(194*time.Second)) || !e.active {
		t.Fatal("long boss downtime split encounter")
	}
}

func TestOutOfOrderHitsKeepTotalsAndClock(t *testing.T) {
	e, t0 := encounterFixture()
	e.hit(t0, game.Hit{Actor: 1, Target: 7, Damage: 100})
	e.hit(t0.Add(2*time.Second), game.Hit{Actor: 1, Target: 7, Damage: 200})
	e.hit(t0.Add(time.Second), game.Hit{Actor: 1, Target: 7, Damage: 50})
	if e.meter.actors[1].damage != 350 || e.lastDamage != t0.Add(2*time.Second) {
		t.Fatal("out-of-order hit lost or clock rewound")
	}
}

func TestDelayedHealthCannotResetFight(t *testing.T) {
	e, t0 := encounterFixture()
	e.npc(20, bossCode())
	e.hit(t0, game.Hit{Actor: 1, Target: 20, Damage: 100})
	e.health(t0.Add(2*time.Second), 20, 700, 0)
	e.health(t0, 20, 1000, 1000)
	if !e.active || e.npcs[20].hp != 700 || e.npcs[20].max != 1000 {
		t.Fatal("delayed spawn HP rewound current health or reset fight")
	}
}
