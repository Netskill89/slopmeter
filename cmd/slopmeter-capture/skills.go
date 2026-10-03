package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/nuriland/a2kit/game"
)

//go:embed data/skills.json
var skillData []byte

type skillDefinition struct {
	Base game.Skill `json:"base"`
	Name string     `json:"name"`
	Icon string     `json:"icon"`
}

var skillCatalog = func() map[game.Skill]skillDefinition {
	var definitions map[game.Skill]skillDefinition
	if err := json.Unmarshal(skillData, &definitions); err != nil {
		panic(err)
	}
	return definitions
}()

func canonicalSkill(id game.Skill) game.Skill {
	if d := skillCatalog[id]; d.Base != 0 {
		return d.Base
	}
	return id
}

type skillTotal struct {
	damage, hits, critical, typed, casts uint64
	min, max                             uint32
}

func (m *meter) actor(id game.Entity) *total {
	if m.actors == nil {
		m.actors = make(map[game.Entity]*total)
	}
	a := m.actors[id]
	if a == nil {
		a = &total{skills: make(map[game.Skill]uint64), breakdown: make(map[game.Skill]*skillTotal)}
		m.actors[id] = a
	}
	return a
}

func (a *total) skill(id game.Skill) *skillTotal {
	if a.breakdown == nil {
		a.breakdown = make(map[game.Skill]*skillTotal)
	}
	id = canonicalSkill(id)
	s := a.breakdown[id]
	if s == nil {
		s = &skillTotal{}
		a.breakdown[id] = s
	}
	return s
}

func (m *meter) cast(c game.Cast) {
	if m.target != 0 && c.Target != m.target {
		return
	}
	m.actor(c.Actor).skill(c.Skill).casts++
}

type skillRow struct {
	ID            game.Skill `json:"id"`
	Name          string     `json:"name"`
	Icon          string     `json:"icon"`
	Damage        uint64     `json:"damage"`
	Share         float64    `json:"share"`
	FightShare    float64    `json:"fightShare"`
	Hits          uint64     `json:"hits"`
	Uses          uint64     `json:"uses"`
	UsesKnown     bool       `json:"usesKnown"`
	Critical      uint64     `json:"critical"`
	CriticalRate  float64    `json:"criticalRate"`
	CriticalKnown bool       `json:"criticalKnown"`
	Min           uint32     `json:"min"`
	Max           uint32     `json:"max"`
	Average       float64    `json:"average"`
	HitsPerSecond float64    `json:"hitsPerSecond"`
}

func (a *total) skillRows(seconds float64, fightDamage uint64) []skillRow {
	rows := []skillRow{}
	for id, s := range a.breakdown {
		d := skillCatalog[id]
		if d.Name == "" {
			d.Name = fmt.Sprintf("Skill %d", id)
		}
		r := skillRow{ID: id, Name: d.Name, Icon: d.Icon, Damage: s.damage, Hits: s.hits, Uses: s.casts, UsesKnown: s.casts > 0,
			Critical: s.critical, CriticalKnown: s.typed > 0, Min: s.min, Max: s.max, HitsPerSecond: float64(s.hits) / seconds}
		if a.damage > 0 {
			r.Share = float64(s.damage) / float64(a.damage) * 100
		}
		if fightDamage > 0 {
			r.FightShare = float64(s.damage) / float64(fightDamage) * 100
		}
		if s.typed > 0 {
			r.CriticalRate = float64(s.critical) / float64(s.typed) * 100
		}
		if s.hits > 0 {
			r.Average = float64(s.damage) / float64(s.hits)
		}
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Damage == rows[j].Damage {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].Damage > rows[j].Damage
	})
	return rows
}
