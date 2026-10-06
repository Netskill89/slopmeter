package main

import (
	"encoding/binary"
	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
	"testing"
	"time"
)

func rosterStatsFixture() []byte {
	p := make([]byte, 4)
	p = append(p, 5)
	p = append(p, []byte("Party")...)
	p = append(p, 2)
	p = append(p, make([]byte, 17)...)
	p = append(p, 2)
	for index, name := range []string{"Self", "Friend"} {
		p = append(p, 1, byte(index+1))
		id := make([]byte, 8)
		binary.LittleEndian.PutUint64(id, uint64(1304)<<48|uint64(index+1))
		p = append(p, id...)
		p = append(p, byte(len(name)))
		p = append(p, []byte(name)...)
		fields := make([]byte, 12)
		binary.LittleEndian.PutUint32(fields, 32)
		binary.LittleEndian.PutUint32(fields[4:], 45)
		binary.LittleEndian.PutUint32(fields[8:], uint32(3000+index))
		p = append(p, fields...)
		// Extra byte before the server anchor, variable record tails.
		p = append(p, 0xff)
		tail := make([]byte, 13)
		binary.LittleEndian.PutUint16(tail, 1304)
		binary.LittleEndian.PutUint64(tail[5:], uint64(39000+index))
		p = append(p, tail...)
		p = append(p, 0, 0, 0)
	}
	return p
}
func TestRosterStats(t *testing.T) {
	p := rosterStatsFixture()
	stats, ok := parseRosterStats(p)
	if !ok || stats["Self"].GearScore != 3000 || stats["Friend"].CombatPower != 39001 {
		t.Fatalf("bad stats: %+v %v", stats, ok)
	}
	for n := 0; n < len(p)-3; n++ {
		if _, ok := parseRosterStats(p[:n]); ok {
			t.Fatalf("accepted truncated roster at %d", n)
		}
	}
	s := scope{identities: identities{names: map[game.Entity]string{1: "Self"}, self: 1, name: "Self"}}
	s.observeMessage(aMessage{opcode: 0x9702, flags: wire.FromServer, payload: p, t: time.Now()})
	if s.stats["Self"].CombatPower != 39000 || s.classRevision == 0 {
		t.Fatal("metadata not published")
	}
	s.observeMessage(aMessage{opcode: 0x3611, flags: wire.FromServer})
	if s.stats != nil {
		t.Fatal("stats survived logout")
	}
	s.observeMessage(aMessage{opcode: 0x9702, payload: p})
	if s.stats != nil {
		t.Fatal("accepted client roster")
	}
	s.observeMessage(aMessage{opcode: 0x3801, flags: wire.FromServer, payload: append([]byte{9, 2, 0x97}, p...)})
	if s.stats["Friend"].GearScore != 3001 {
		t.Fatal("embedded roster not decoded")
	}
}
