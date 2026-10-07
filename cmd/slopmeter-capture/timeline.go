package main

import (
	"fmt"
	"github.com/nuriland/a2kit/game"
	"sort"
	"time"
)

const maxTimelineCasts = 5000

type timedSkillUse struct {
	at    time.Time
	skill game.Skill
}
type skillUse struct {
	Time  float64    `json:"time"`
	Skill game.Skill `json:"skill"`
	Name  string     `json:"name"`
	Icon  string     `json:"icon"`
}

func (a *total) timelineRows(start time.Time) []skillUse {
	result := make([]skillUse, 0, len(a.timeline))
	if start.IsZero() {
		return result
	}
	for _, use := range a.timeline {
		d := skillCatalog[use.skill]
		if d.Name == "" {
			d.Name = fmt.Sprintf("Skill %d", use.skill)
		}
		result = append(result, skillUse{use.at.Sub(start).Seconds(), use.skill, d.Name, d.Icon})
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Time < result[j].Time })
	return result
}
