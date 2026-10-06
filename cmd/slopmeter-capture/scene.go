package main

import (
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
)

// Map names and scene layouts derive from Aion2Flow (see THIRD_PARTY.md).
//
//go:embed data/maps.json
var mapCatalogJSON []byte
var mapCatalog = func() map[uint32]string {
	var out map[uint32]string
	if err := json.Unmarshal(mapCatalogJSON, &out); err != nil {
		panic(err)
	}
	return out
}()

type sceneDisplay struct {
	ID       uint32 `json:"id"`
	Name     string `json:"name"`
	Instance uint32 `json:"instance"`
}

type sceneState struct {
	sceneDisplay
	candidate uint32
	arrival   bool
	departed  bool
	observed  time.Time
}

func (e *encounter) enterMap(t time.Time, id uint32, boundary bool) {
	if boundary {
		e.finish(t, "Area changed", false)
		e.boss = 0
		e.lastTarget = 0
		e.npcs = make(map[game.Entity]*npcState)
		e.targets = make(map[game.Entity]*npcState)
		e.scene.Instance = 0
	}
	e.scene.ID = id
	e.scene.Name = mapCatalog[id]
	e.scene.candidate = 0
	e.scene.arrival = false
	e.scene.departed = false
	if e.active && !boundary {
		e.fightScene = e.scene.sceneDisplay
	}
	if e.active && e.boss == 0 {
		e.status = e.combatLabel()
	}
}

func (e *encounter) combatLabel() string {
	if e.scene.Name != "" {
		return e.scene.Name + " · Combat"
	}
	return "Combat · Area unknown"
}

func (e *encounter) observeScene(m aMessage) bool {
	p := m.payload
	switch m.opcode {
	case 0x3621: // 21 36: sequence + map code + composite scene state.
		if len(p) < 30 {
			return false
		}
		id := binary.LittleEndian.Uint32(p[4:])
		if mapCatalog[id] == "" {
			return false
		}
		if e.scene.ID == 0 {
			// First metadata refines the existing session; it is not a new pull.
			e.enterMap(m.t, id, false)
		} else if id == e.scene.ID {
			e.scene.candidate = 0
		} else if e.scene.arrival {
			e.enterMap(m.t, id, true)
		} else {
			e.scene.candidate = id
		}
	case 0x3623: // 23 36: destination arrival, not an unconditional combat reset.
		if len(p) < 20 {
			return false
		}
		e.scene.arrival = true
		if e.scene.candidate != 0 || e.scene.departed {
			id := e.scene.ID
			if e.scene.candidate != 0 {
				id = e.scene.candidate
			}
			e.enterMap(m.t, id, true)
		}
	case 0x922e: // 2E 92: map event registration, LE instance ID + printable key.
		if len(p) < 6 || p[4] == 0 || p[4] > 64 || len(p) < 5+int(p[4]) {
			return false
		}
		for _, b := range p[5 : 5+int(p[4])] {
			if b < 0x20 || b > 0x7e {
				return false
			}
		}
		id := binary.LittleEndian.Uint32(p)
		if id == 0 {
			return false
		}
		e.scene.Instance = id
		if e.active {
			e.fightScene.Instance = id
		}
	case 0x922f: // Match a known registration before declaring departure.
		if len(p) != 4 {
			return false
		}
		id := binary.LittleEndian.Uint32(p)
		if id == 0 || id != e.scene.Instance {
			return false
		}
		e.scene.Instance = 0
		e.scene.departed = true
	default:
		return false
	}
	e.scene.observed = m.t
	return true
}

// Direction and ordering are checked independently from damage timestamps.
func (e *encounter) sceneMessage(m aMessage) bool {
	if m.flags&wire.FromServer == 0 || m.t.Before(e.scene.observed) {
		return false
	}
	return e.observeScene(m)
}
