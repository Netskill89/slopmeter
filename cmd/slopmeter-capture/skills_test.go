package main

import (
	"testing"
	"time"

	"github.com/nuriland/a2kit/game"
)

func TestSkillBreakdown(t *testing.T) {
	m := meter{target: 7}
	t0 := time.Unix(100, 0)
	const primary game.Skill = 4000000000
	const secondary game.Skill = 4000000001
	m.cast(game.Cast{Actor: 1, Target: 7, Skill: primary})
	m.add(t0, game.Hit{Actor: 1, Target: 7, Skill: primary, Damage: 100, Type: 2, Extra: []uint32{10}})
	m.add(t0.Add(2*time.Second), game.Hit{Actor: 1, Target: 7, Skill: primary, Damage: 300, Type: 3})
	m.add(t0, game.Hit{Actor: 1, Target: 7, Skill: secondary, Damage: 100, Type: 0})
	m.add(t0, game.Hit{Actor: 2, Target: 7, Skill: secondary, Damage: 500, Type: 2})
	m.add(t0, game.Hit{Actor: 1, Target: 8, Skill: primary, Damage: 999, Type: 3})
	rows := m.snapshot(identities{}).Actors[0].Skills
	r := rows[0]
	if r.ID != primary || r.Damage != 400 || r.Hits != 2 || r.Uses != 1 || r.CriticalRate != 50 || r.Min != 100 || r.Max != 300 || r.Share != 80 || r.FightShare != 40 || r.HitsPerSecond != 1 {
		t.Fatalf("incorrect breakdown: %+v", r)
	}
	if !r.CriticalKnown || !r.UsesKnown || r.Average != 200 {
		t.Fatal("incorrect availability or average")
	}
	if rows[1].CriticalKnown || rows[1].UsesKnown {
		t.Fatal("invented missing crit/cast data")
	}
}

func TestSkillVariantAggregation(t *testing.T) {
	var variant, base game.Skill
	for id, d := range skillCatalog {
		if d.Base != 0 && d.Base != id && skillCatalog[d.Base].Name != "" {
			variant = id
			base = d.Base
			break
		}
	}
	if variant == 0 {
		t.Fatal("no skill variants in catalogue")
	}
	m := meter{}
	m.cast(game.Cast{Actor: 1, Skill: base})
	m.add(time.Unix(100, 0), game.Hit{Actor: 1, Skill: variant, Damage: 10, Type: 2})
	m.add(time.Unix(101, 0), game.Hit{Actor: 1, Skill: base, Damage: 20, Type: 3})
	rows := m.snapshot(identities{}).Actors[0].Skills
	if len(rows) != 1 || rows[0].ID != base || rows[0].Damage != 30 || rows[0].Uses != 1 {
		t.Fatal("variants not grouped with casts")
	}
}
