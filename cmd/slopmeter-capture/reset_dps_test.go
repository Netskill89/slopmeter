package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
)

func TestManualResetKeepsCaptureMetadataAndHistory(t *testing.T) {
	e, now := encounterFixture()
	e.scene.sceneDisplay = sceneDisplay{ID: 600021, Name: "Fire Temple", Instance: 42}
	e.npc(20, bossCode())
	e.health(now, 20, 700, 2000)
	e.hit(now, game.Hit{Actor: 1, Target: 20, Skill: 10, Damage: 100})
	h, _ := loadHistory("")
	ids := identities{name: "Self", names: map[game.Entity]string{1: "Self"}}
	e.onFinish = func(e *encounter, timestamp time.Time) {
		if err := h.archive(e, ids, timestamp); err != nil {
			t.Fatal(err)
		}
	}
	cutoff := now.Add(2 * time.Second)
	e.resetDamage(cutoff)
	s := e.snapshot(ids, cutoff)
	if !s.Cleared || s.Active || len(s.Actors) != 0 || s.Duration != 0 || s.Character != "Self" || s.Scene.Instance != 42 || s.Boss.HP != 700 {
		t.Fatalf("reset lost metadata or retained totals: %+v", s)
	}
	if len(h.entries) != 1 || h.entries[0].Snapshot.Actors[0].Damage != 100 {
		t.Fatal("interrupted fight not archived")
	}
	e.cast(now.Add(time.Second), game.Cast{Actor: 1, Target: 20, Skill: 10})
	e.hit(cutoff, game.Hit{Actor: 1, Target: 20, Skill: 10, Damage: 900})
	if len(e.meter.actors) != 0 || len(e.pendingCasts) != 0 {
		t.Fatal("queued pre-reset damage/cast reused")
	}
	e.hit(cutoff.Add(time.Second), game.Hit{Actor: 1, Target: 20, Skill: 10, Damage: 50})
	s = e.snapshot(ids, cutoff.Add(time.Second))
	if s.Cleared || s.Session != 2 || s.Actors[0].Damage != 50 || s.Actors[0].DPS != 50 {
		t.Fatalf("fresh calculation failed: %+v", s)
	}
}

func TestResetCommandIsNotLostWhenRefreshCommandsCoalesce(t *testing.T) {
	changes := make(chan time.Duration, 1)
	resets := make(chan struct{}, 1)
	var report bytes.Buffer
	readRefreshCommands(context.Background(), strings.NewReader("{\"intervalMs\":50}\n{\"reset\":true}\n{\"intervalMs\":1000}\n{\"reset\":true}\n"), changes, resets, &report)
	if len(resets) != 1 || len(changes) != 1 || <-changes != time.Second || report.Len() != 0 {
		t.Fatal("reset was lost or affected refresh rate", report.String())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	empty := make(chan struct{}, 1)
	readRefreshCommands(ctx, strings.NewReader("{\"reset\":true}\n"), changes, empty, &report)
	if len(empty) != 0 {
		t.Fatal("reset delivered after cancellation")
	}
}

func periodicPacket(kind byte, damage uint64) []byte {
	p := binary.AppendUvarint(nil, 20)
	p = append(p, kind)
	p = binary.AppendUvarint(p, 1)
	p = binary.AppendUvarint(p, 0)
	p = binary.LittleEndian.AppendUint32(p, 15020000) // A2Tools allowlist: Firebomb.
	return binary.AppendUvarint(p, damage)
}
func TestPeriodicDamageExcludesHealingBuffsAndInvalidLayouts(t *testing.T) {
	for _, kind := range []byte{2, 10} {
		h, ok := combatHit(aMessage{opcode: 0x3805, flags: wire.FromServer, payload: periodicPacket(kind, 250)})
		if !ok || h.Actor != 1 || h.Target != 20 || h.Skill != 15020000 || h.Damage != 250 || h.Type != 0 {
			t.Fatalf("damage tick missing: %+v", h)
		}
	}
	for _, kind := range []byte{0, 1, 3, 8, 9, 11, 48} {
		if _, ok := combatHit(aMessage{opcode: 0x3805, flags: wire.FromServer, payload: periodicPacket(kind, 250)}); ok {
			t.Fatalf("non-damage effect %d counted", kind)
		}
	}
	for _, p := range [][]byte{periodicPacket(2, 0), periodicPacket(2, 100_000_000), periodicPacket(2, 250)[:5], append(periodicPacket(2, 250), 128)} {
		if _, ok := combatHit(aMessage{opcode: 0x3805, flags: wire.FromServer, payload: p}); ok {
			t.Fatal("invalid periodic packet counted")
		}
	}
	if _, ok := combatHit(aMessage{opcode: 0x3805, flags: wire.FromClient, payload: periodicPacket(2, 250)}); ok {
		t.Fatal("client damage accepted")
	}
}

func TestDPSMatchesReferenceArithmetic(t *testing.T) {
	// Reference: A2Tools dps_calculator.rs (amount / max(battle_ms,1000)*1000)
	// and Aion2Flow SceneCombatSnapshotAdapter.cs (amount / encounter_ms*1000).
	// Use the same common encounter window, not each player's first-hit time.
	e, now := encounterFixture()
	e.npc(20, bossCode())
	e.hit(now, game.Hit{Actor: 1, Target: 20, Skill: 10, Damage: 5000})
	e.hit(now.Add(5*time.Second), game.Hit{Actor: 2, Target: 20, Skill: 11, Damage: 2000})
	dot, ok := combatHit(aMessage{opcode: 0x3805, flags: wire.FromServer, payload: periodicPacket(2, 1000)})
	if !ok {
		t.Fatal("validated DoT missing")
	}
	e.hit(now.Add(8*time.Second), dot)
	e.hit(now.Add(10*time.Second), game.Hit{Actor: 1, Target: 20, Skill: 10, Damage: 4000})
	e.finish(now.Add(10*time.Second), "Boss defeated", false)
	s := e.snapshot(identities{}, now.Add(time.Hour))
	if s.Duration != 10 || len(s.Actors) != 2 || s.Actors[0].Damage != 10000 || s.Actors[0].DPS != 1000 || s.Actors[1].DPS != 2000/10.0 {
		t.Fatalf("reference totals/DPS differ: %+v", s)
	}
	if math.Abs(s.Actors[0].Share-83.33333333333333) > 1e-9 || s.Actors[1].Fill != 20 {
		t.Fatal("contributions or relative bars incorrect")
	}
	var skills uint64
	for _, skill := range s.Actors[0].Skills {
		skills += skill.Damage
	}
	if skills != s.Actors[0].Damage {
		t.Fatal("skill totals do not reconcile")
	}
}
func TestLiveBossDPSIncludesDowntimeAndSingleHitFloor(t *testing.T) {
	e, now := encounterFixture()
	e.npc(20, bossCode())
	e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 10000})
	if e.snapshot(identities{}, now).Actors[0].DPS != 10000 {
		t.Fatal("single hit did not use one-second minimum")
	}
	if e.snapshot(identities{}, now.Add(20*time.Second)).Actors[0].DPS != 500 {
		t.Fatal("live boss downtime omitted from denominator")
	}
}
