package main

import (
	"fmt"
	"github.com/nuriland/a2kit/game"
)

func (e *encounter) healthStatus(boss *bossDisplay) string {
	if boss != nil {
		if !boss.Known {
			return "Boss identified; waiting for captured HP updates."
		}
		if !boss.MaxKnown {
			return "Current boss HP captured; maximum HP missing. Re-enter the area to refresh spawn data."
		}
		return "Current and maximum boss HP captured."
	}
	if e.lastTarget == 0 {
		return "Waiting for your group to attack a boss."
	}
	if n := e.targets[e.lastTarget]; n != nil && n.definition.Code != 0 {
		return fmt.Sprintf("Attacked NPC %d is not classified as a boss in the catalogue.", n.definition.Code)
	}
	return "Attacked target identity missing. Re-enter the area with capture running."
}

func (e *encounter) bossSnapshot(id game.Entity, i identities) *bossDisplay {
	if id == 0 {
		return nil
	}
	d := &bossDisplay{Entity: id, Name: fmt.Sprintf("Target %d", id)}
	if name := i.names[id]; name != "" {
		d.Name = name
	}
	n := e.targets[id]
	if n == nil {
		n = e.npcs[id]
	}
	if n == nil {
		return d
	}
	if n.definition.Name != "" {
		d.Name = n.definition.Name
	}
	divisor := max(uint32(1), n.definition.Divisor)
	d.HP = n.hp / divisor
	d.Max = n.max / divisor
	d.Known = n.known
	d.MaxKnown = n.max > 0
	if n.max > 0 {
		d.Percent = min(100, float64(n.hp)/float64(n.max)*100)
	}
	return d
}
