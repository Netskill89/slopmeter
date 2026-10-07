package main

import (
	"fmt"
	"github.com/nuriland/a2kit/game"
	"sort"
)

type identities struct {
	names   map[game.Entity]string
	classes map[game.Entity]string
	stats   map[string]playerStats
	self    game.Entity
	name    string
}

func (i *identities) observe(event game.Event) {
	if p, ok := event.(game.Player); ok {
		if i.names == nil {
			i.names = make(map[game.Entity]string)
		}
		if p.Name != "" {
			i.names[p.Entity] = p.Name
		}
		if p.Self {
			i.self = p.Entity
			i.name = p.Name
		}
	}
}

type row struct {
	Timeline          []skillUse  `json:"timeline,omitempty"`
	TimelineTruncated bool        `json:"timelineTruncated,omitempty"`
	GearScore         uint32      `json:"gearScore"`
	CombatPower       uint64      `json:"combatPower"`
	ID                game.Entity `json:"id"`
	Name              string      `json:"name"`
	Class             string      `json:"class"`
	Damage            uint64      `json:"damage"`
	DPS               float64     `json:"dps"`
	Share             float64     `json:"share"`
	Fill              float64     `json:"fill"`
	Skills            []skillRow  `json:"skills"`
}
type snapshot struct {
	Cleared      bool            `json:"cleared,omitempty"`
	Scene        sceneDisplay    `json:"scene"`
	Character    string          `json:"character"`
	Actors       []row           `json:"actors"`
	Duration     float64         `json:"duration"`
	Session      uint64          `json:"session"`
	Encounter    string          `json:"encounter"`
	Active       bool            `json:"active"`
	Boss         *bossDisplay    `json:"boss"`
	HealthStatus string          `json:"healthStatus"`
	History      *[]historyEntry `json:"history,omitempty"`
}

func (m *meter) snapshot(i identities) snapshot {
	s := snapshot{Character: i.name, Actors: []row{}, Duration: m.last.Sub(m.first).Seconds()}
	seconds := s.Duration
	if seconds < 1 {
		seconds = 1
	}
	var sum, largest uint64
	for _, a := range m.actors {
		sum += a.damage
		largest = max(largest, a.damage)
	}
	for id, a := range m.actors {
		if a.damage == 0 {
			continue
		}
		name := i.names[id]
		if name == "" {
			name = fmt.Sprintf("Entity %d", id)
		}
		share := 0.0
		if sum > 0 {
			share = float64(a.damage) / float64(sum) * 100
		}
		fill := 0.0
		if largest > 0 {
			fill = float64(a.damage) / float64(largest) * 100
		}
		s.Actors = append(s.Actors, row{Timeline: a.timelineRows(m.first), TimelineTruncated: a.timelineTruncated, GearScore: i.stats[name].GearScore, CombatPower: i.stats[name].CombatPower, ID: id, Name: name, Class: i.classes[id], Damage: a.damage, DPS: float64(a.damage) / seconds, Share: share, Fill: fill, Skills: a.skillRows(seconds, sum)})
	}
	sort.Slice(s.Actors, func(a, b int) bool {
		if s.Actors[a].Damage == s.Actors[b].Damage {
			return s.Actors[a].ID < s.Actors[b].ID
		}
		return s.Actors[a].Damage > s.Actors[b].Damage
	})
	return s
}
