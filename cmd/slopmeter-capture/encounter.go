package main

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
)

//go:embed data/bosses.json
var bossCatalogJSON []byte

type bossDefinition struct {
	Code    uint32 `json:"code"`
	Name    string `json:"name"`
	Divisor uint32 `json:"divisor"`
}

var bossCatalog = func() map[uint32]bossDefinition {
	var definitions []bossDefinition
	if err := json.Unmarshal(bossCatalogJSON, &definitions); err != nil {
		panic(err)
	}
	out := make(map[uint32]bossDefinition, len(definitions))
	for _, d := range definitions {
		if d.Divisor == 0 {
			d.Divisor = 1
		}
		out[d.Code] = d
	}
	return out
}()

type npcState struct {
	definition bossDefinition
	hp, max    uint32
	known      bool
	observed   time.Time
}
type bossDisplay struct {
	Entity   game.Entity `json:"entity"`
	Name     string      `json:"name"`
	HP       uint32      `json:"hp"`
	Max      uint32      `json:"max"`
	Percent  float64     `json:"percent"`
	Known    bool        `json:"known"`
	MaxKnown bool        `json:"maxKnown"`
}
type timedCast struct {
	t    time.Time
	cast game.Cast
}
type encounter struct {
	targets           map[game.Entity]*npcState
	onFinish          func(*encounter, time.Time)
	pendingCasts      []timedCast
	meter             meter
	filter            game.Entity
	npcs              map[game.Entity]*npcState
	boss              game.Entity
	lastTarget        game.Entity
	active            bool
	number            uint64
	lastDamage, ended time.Time
	status            string
	idle, bossIdle    time.Duration
}

func newEncounter(target game.Entity, idle, bossIdle time.Duration) *encounter {
	return &encounter{meter: meter{target: target}, filter: target, targets: make(map[game.Entity]*npcState), npcs: make(map[game.Entity]*npcState), status: "Waiting for combat", idle: idle, bossIdle: bossIdle}
}
func (e *encounter) start(t time.Time, boss game.Entity) {
	if e.active {
		// Preserve only casts for the upcoming pull; completed pulls never leak uses.
		pending := append([]timedCast(nil), e.pendingCasts...)
		last := e.lastDamage
		e.finish(last, "Open world ended · boss engaged", false)
		for _, c := range pending {
			if c.t.After(last) {
				e.pendingCasts = append(e.pendingCasts, c)
			}
		}
	}
	target := e.filter
	if boss != 0 {
		target = boss
	}
	e.meter = meter{target: target}
	e.boss = boss
	e.lastTarget = 0
	e.active = true
	e.ended = time.Time{}
	e.number++
	e.lastDamage = t
	for _, c := range e.pendingCasts {
		if !c.t.After(t) && t.Sub(c.t) <= 5*time.Second {
			e.meter.cast(c.cast)
		}
	}
	e.pendingCasts = nil
	e.status = "Open world"
	if boss != 0 {
		e.status = "Boss fight"
	}
}
func (e *encounter) finish(t time.Time, status string, clear bool) {
	wasActive := e.active
	e.active = false
	e.ended = t
	e.status = status
	if wasActive && e.onFinish != nil {
		e.onFinish(e, t)
	}
	e.pendingCasts = nil
	if clear {
		e.meter = meter{target: e.filter}
		e.boss = 0
		e.lastTarget = 0
		e.lastDamage = time.Time{}
		e.pendingCasts = nil
	}
}
func (e *encounter) tick(t time.Time) bool {
	if !e.active {
		return false
	}
	elapsed := t.Sub(e.lastDamage)
	if e.boss == 0 && elapsed >= e.idle {
		e.finish(e.lastDamage, "Waiting for combat · previous fight ended", true)
		return true
	}
	if e.boss != 0 && elapsed >= e.bossIdle {
		e.finish(e.lastDamage, "Boss fight ended · inactive", false)
		return true
	}
	return false
}
func (e *encounter) hit(t time.Time, h game.Hit) {
	if h.Damage == 0 || (e.filter != 0 && h.Target != e.filter) {
		return
	}
	n := e.targets[h.Target]
	if n == nil {
		n = e.npcs[h.Target]
	}
	if n != nil && n.known && n.hp == 0 {
		return
	}
	// A backwards/out-of-order message must not create a fresh pull or move clocks back.
	if !e.lastDamage.IsZero() && t.Before(e.lastDamage) && !e.active {
		return
	}
	e.tick(t)
	boss := game.Entity(0)
	if e.npcs[h.Target] != nil {
		boss = h.Target
	}
	if e.active && e.boss != 0 && h.Target != e.boss {
		return
	}
	if !e.active || (e.boss == 0 && boss != 0) {
		e.start(t, boss)
	}
	if e.meter.target != 0 && h.Target != e.meter.target {
		return
	}
	e.meter.add(t, h)
	if !t.Before(e.lastDamage) {
		e.lastTarget = h.Target
	}
	if t.After(e.lastDamage) {
		e.lastDamage = t
	}
}
func (e *encounter) cast(t time.Time, c game.Cast) {
	if !e.ended.IsZero() && !t.After(e.ended) {
		return
	}
	if e.filter != 0 && c.Target != e.filter {
		return
	}
	e.tick(t)
	if e.active {
		e.meter.cast(c)
	}
	fresh := e.pendingCasts[:0]
	for _, old := range e.pendingCasts {
		if t.Sub(old.t) <= 5*time.Second {
			fresh = append(fresh, old)
		}
	}
	e.pendingCasts = append(fresh, timedCast{t, c})
	if len(e.pendingCasts) > 256 {
		e.pendingCasts = e.pendingCasts[len(e.pendingCasts)-256:]
	}
}
func (e *encounter) npc(id game.Entity, code uint32) *npcState {
	d, ok := bossCatalog[code]
	if !ok {
		return nil
	}
	n := e.npcs[id]
	if n == nil {
		n = e.targets[id]
	}
	if n != nil && (n.definition.Code == 0 || n.definition.Code == code) {
		n.definition = d
	} else if n == nil || n.definition.Code != code {
		n = &npcState{definition: d}
	}
	e.npcs[id] = n
	e.targets[id] = n
	return n
}
func hpPair(p []byte, offset int) (uint32, uint32, bool) {
	if offset < 0 || offset >= len(p) {
		return 0, 0, false
	}
	current, rest, ok := readVar(p[offset:])
	if !ok {
		return 0, 0, false
	}
	max, rest, ok := readVar(rest)
	if !ok || max == 0 || current > max || len(rest) < 8 || !bytes.Equal(rest[:8], []byte{100, 0, 0, 0, 100, 0, 0, 0}) {
		return 0, 0, false
	}
	return current, max, true
}
func (e *encounter) targetState(id game.Entity) *npcState {
	n := e.targets[id]
	if n == nil {
		n = e.npcs[id]
	}
	if n == nil {
		// HP can precede a spawn. Retain the observed number without guessing max HP.
		if len(e.targets) >= 4096 {
			for other := range e.targets {
				if other != e.boss && e.npcs[other] == nil {
					delete(e.targets, other)
					break
				}
			}
		}
		n = &npcState{}
		e.targets[id] = n
	}
	return n
}
func (e *encounter) health(t time.Time, id game.Entity, current, max uint32) {
	n := e.targetState(id)
	if t.Before(n.observed) {
		if n.max == 0 && max > 0 {
			n.max = max
		}
		return
	}
	n.observed = t
	old := n.hp
	wasKnown := n.known
	if max > 0 {
		n.max = max
	}
	n.hp = current
	n.known = true
	if e.active && e.boss == id {
		if current == 0 {
			e.finish(t, "Boss defeated", false)
		} else if wasKnown && old > 0 && old < n.max && current == n.max {
			e.finish(t, "Boss reset · ready for next pull", false)
		}
	}
}

// Layouts from Aion2Flow: 40/41 36 NPC states and typed 00 8D HP updates.
// HP is read from packets, never estimated by subtracting observed damage.
func (e *encounter) observe(m aMessage) {
	if m.flags&wire.FromServer == 0 {
		return
	}
	if spawn, ok := m.event.(game.Spawn); ok {
		if e.npc(spawn.Entity, uint32(spawn.NPC)) == nil && spawn.NPC != 0 {
			// Retain identity for diagnostics without declaring an unknown NPC a boss.
			n := e.targetState(spawn.Entity)
			n.definition.Code = uint32(spawn.NPC)
		}
	}
	if death, ok := m.event.(game.Death); ok && death.Flag == 3 && death.Entity != e.boss {
		e.health(m.t, death.Entity, 0, 0)
	}
	if death, ok := m.event.(game.Death); ok && death.Entity == e.boss && e.active {
		if death.Flag == 3 {
			e.health(m.t, death.Entity, 0, 0)
		} else {
			e.finish(m.t, "Boss left view", false)
		}
		return
	}
	if m.opcode == 0x3623 || m.opcode == 0x3611 || m.opcode == 0x3615 {
		e.finish(m.t, "Waiting for combat", true)
		e.npcs = make(map[game.Entity]*npcState)
		e.targets = make(map[game.Entity]*npcState)
		return
	}
	p := m.payload
	switch m.opcode {
	case 0x8d00:
		id, rest, ok := readVar(p)
		if !ok || id == 0 {
			return
		}
		a, rest, ok := readVar(rest)
		if !ok {
			return
		}
		b, rest, ok := readVar(rest)
		if !ok {
			return
		}
		c, rest, ok := readVar(rest)
		if !ok || a != 2 || b != 1 || c != 0 || len(rest) < 4 {
			return
		}
		e.health(m.t, game.Entity(id), binary.LittleEndian.Uint32(rest), 0)
	case 0x3640, 0x3641:
		id, rest, ok := readVar(p)
		if !ok || id == 0 || len(rest) < 7 {
			return
		}
		offsets := []int{5}
		if m.opcode == 0x3640 {
			tag := rest[:3]
			knownTag := ((tag[1] == 0x10 || tag[1] == 0x20 || tag[1] == 0x21 || tag[1] == 0x22 || tag[1] == 0x30 || tag[1] == 0x32) && tag[2] == 0 || tag[0] == 0x1c && tag[1] == 0 && tag[2] == 0)
			if !knownTag {
				// New state flags may still carry the same verified NPC/HP layout.
				if _, _, valid := hpPair(rest, 28); !valid {
					return
				}
			}
			offsets = []int{3}
		} else if rest[0] == 0x5f && rest[2] == 0 {
			offsets = []int{4, 5}
		}
		for _, offset := range offsets {
			if offset+4 > len(rest) {
				continue
			}
			code := binary.LittleEndian.Uint32(rest[offset:])
			n := e.npc(game.Entity(id), code)
			if n == nil {
				continue
			}
			hpOffset := offset + 25
			if m.opcode == 0x3641 && rest[2] == 0 && ((rest[0] == 0x85 && rest[1] == 0x21 && len(rest) == 116) || (rest[0] == 5 && rest[1] == 0x20 && len(rest) == 106)) {
				hpOffset += 12
			}
			current, max, ok := hpPair(rest, hpOffset)
			if !ok && m.opcode == 0x3641 {
				// Optional trailing fields change packet length. Validate both known
				// layouts instead of requiring one exact total length.
				alternative := offset + 37
				if hpOffset == alternative {
					alternative = offset + 25
				}
				current, max, ok = hpPair(rest, alternative)
			}
			if ok {
				e.health(m.t, game.Entity(id), current, max)
			}
			break
		}
	}
}
func (e *encounter) snapshot(i identities, t time.Time) snapshot {
	m := e.meter
	if !m.first.IsZero() {
		end := e.ended
		if e.active {
			end = t
		}
		if end.Before(m.last) {
			end = m.last
		}
		m.last = end
	}
	s := m.snapshot(i)
	s.Session = e.number
	s.Encounter = e.status
	s.Active = e.active
	if e.boss != 0 {
		s.Boss = e.bossSnapshot(e.boss, i)
	} else if e.lastTarget != 0 && e.npcs[e.lastTarget] != nil {
		// Identity can arrive after the last hit. Show its captured HP immediately
		// without waiting for another damage event or showing ordinary monsters.
		s.Boss = e.bossSnapshot(e.lastTarget, i)
	}
	s.HealthStatus = e.healthStatus(s.Boss)
	return s
}
