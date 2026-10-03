package main

import (
	"github.com/nuriland/a2kit/game"
	"testing"
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
