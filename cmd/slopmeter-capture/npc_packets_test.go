package main

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
)

func maskedBossPacket(width int, name string) []byte {
	p := binary.AppendUvarint(nil, 20)
	mask := make([]byte, width)
	mask[0] = 0x0c
	p = append(p, mask...)
	if name != "" {
		p = append(p, 1)
		p = binary.AppendUvarint(p, uint64(len(name)))
		p = append(p, []byte(name)...)
	} else {
		p = append(p, 0)
	}
	p = binary.LittleEndian.AppendUint32(p, bossCode())
	p = append(p, make([]byte, 31)...)
	p = binary.AppendUvarint(p, 700)
	p = binary.AppendUvarint(p, 2000)
	return append(p, 100, 0, 0, 0, 100, 0, 0, 0)
}
func embeddedHP(id uint64, hp uint32) []byte {
	p := []byte{0x8d}
	p = binary.AppendUvarint(p, id)
	p = append(p, 2, 1, 0)
	p = binary.LittleEndian.AppendUint32(p, hp)
	return append(p, 0, 0, 0, 0)
}
func TestMaskedBossSpawnLayouts(t *testing.T) {
	for _, width := range []int{2, 4} {
		for _, name := range []string{"", "Named NPC"} {
			e, now := encounterFixture()
			e.observe(serverMessage(0x3641, maskedBossPacket(width, name), now))
			e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 100})
			s := e.snapshot(identities{}, now)
			if s.Boss == nil || s.Boss.Name != "Fediv Wraith" || s.Boss.HP != 700 || s.Boss.Percent != 35 {
				t.Fatalf("width=%d name=%q: %+v", width, name, s.Boss)
			}
		}
	}
}
func TestEmbeddedHealthBeforeIdentityAndValidation(t *testing.T) {
	e, now := encounterFixture()
	p := append([]byte{5, 6, 7}, embeddedHP(20, 600)...)
	e.observe(serverMessage(0x3804, p, now))
	e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 100})
	if e.snapshot(identities{}, now).Boss != nil {
		t.Fatal("HP alone declared an ordinary target a boss")
	}
	e.npc(20, bossCode())
	if e.snapshot(identities{}, now).Boss.HP != 600 {
		t.Fatal("early embedded HP discarded")
	}
	e.health(now, 20, 600, 2000)
	for _, p := range [][]byte{
		embeddedHP(20, 400)[:10],
		append(embeddedHP(20, 400)[:9], 1, 0, 0, 0),
		embeddedHP(20, 3000),
	} {
		e.observe(serverMessage(0x3804, p, now.Add(time.Second)))
	}
	if e.npcs[20].hp != 600 {
		t.Fatal("bad embedded record changed HP")
	}
	m := serverMessage(0x3804, embeddedHP(20, 400), now.Add(time.Second))
	m.flags = wire.FromClient
	e.observe(m)
	if e.npcs[20].hp != 600 {
		t.Fatal("client HP accepted")
	}
	e.observe(serverMessage(0x3804, embeddedHP(20, 400), now.Add(2*time.Second)))
	if e.npcs[20].hp != 400 {
		t.Fatal("live embedded HP missing")
	}
	e.observe(serverMessage(0x3804, embeddedHP(20, 0), now.Add(3*time.Second)))
	if e.active || e.status != "Boss defeated" {
		t.Fatal("embedded defeat did not end fight")
	}
}
func TestLateBossPromotionPreservesSessionAndSkills(t *testing.T) {
	e, now := encounterFixture()
	e.cast(now, game.Cast{Actor: 1, Target: 20, Skill: 10})
	e.hit(now, game.Hit{Actor: 1, Target: 20, Skill: 10, Damage: 100})
	e.hit(now.Add(time.Second), game.Hit{Actor: 1, Target: 21, Skill: 11, Damage: 900})
	e.npc(20, bossCode())
	if e.boss != 20 || e.number != 1 || e.meter.actors[1].damage != 100 || e.meter.actors[1].skills[11] != 0 {
		t.Fatal("late promotion reset pull or retained trash")
	}
	if e.meter.actors[1].breakdown[10].casts != 1 {
		t.Fatal("pre-pull cast lost")
	}
	e.hit(now.Add(2*time.Second), game.Hit{Actor: 1, Target: 20, Skill: 10, Damage: 50})
	if e.meter.actors[1].damage != 150 {
		t.Fatal("promoted damage counted twice")
	}
	e.cast(now.Add(3*time.Second), game.Cast{Actor: 1, Target: 20, Skill: 10})
	if e.meter.actors[1].breakdown[10].casts != 2 {
		t.Fatal("promoted cast counted twice")
	}
	if e.tick(now.Add(10*time.Minute)) || !e.active {
		t.Fatal("downtime ended confirmed boss fight")
	}
	e.observe(aMessage{event: game.Death{Entity: 20, Flag: 1}, flags: wire.FromServer, t: now.Add(11 * time.Minute)})
	if !e.active {
		t.Fatal("boss disappearing during mechanic ended fight")
	}
	e.hit(now.Add(12*time.Minute), game.Hit{Actor: 1, Target: 20, Skill: 10, Damage: 25})
	if e.number != 1 || e.meter.actors[1].damage != 175 {
		t.Fatal("damage after mechanic started new fight")
	}
}
func TestMalformedMaskedSpawnDoesNotExposeOrdinaryNPC(t *testing.T) {
	e, now := encounterFixture()
	p := maskedBossPacket(4, "Boss")
	p[6] = 255
	e.observe(serverMessage(0x3641, p, now))
	e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 100})
	if e.boss != 0 || e.snapshot(identities{}, now).Boss != nil {
		t.Fatal("invalid inline name accepted as boss identity")
	}
}
