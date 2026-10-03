package main

import (
	"fmt"
	"github.com/nuriland/a2kit/game"
)

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
