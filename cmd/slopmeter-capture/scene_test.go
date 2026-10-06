package main

import (
	"encoding/binary"
	"net/netip"
	"testing"
	"time"

	"github.com/nuriland/a2kit/game"
	"github.com/nuriland/a2kit/wire"
)

func scenePacket(id uint32) []byte {
	p := make([]byte, 36)
	binary.LittleEndian.PutUint32(p[4:], id)
	return p
}
func serverMessage(op wire.Opcode, p []byte, t time.Time) aMessage {
	return aMessage{opcode: op, payload: p, flags: wire.FromServer, t: t}
}
func TestSceneMetadataAndConfirmedTransition(t *testing.T) {
	e, now := encounterFixture()
	e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 100})
	e.observe(serverMessage(0x3621, scenePacket(600021), now.Add(time.Second)))
	s := e.snapshot(identities{}, now)
	if s.Scene.Name != "Fire Temple" || e.number != 1 || e.meter.actors[1].damage != 100 {
		t.Fatalf("metadata reset session: %+v", s)
	}
	e.observe(serverMessage(0x3623, make([]byte, 20), now.Add(2*time.Second)))
	if !e.active {
		t.Fatal("duplicate arrival ended fight")
	}
	e.observe(serverMessage(0x3621, scenePacket(600031), now.Add(3*time.Second)))
	if e.active || e.scene.Name != "Draupnir" {
		t.Fatal("arrival-before-map transition not confirmed")
	}
	if e.snapshot(identities{}, now).Scene.Name != "Fire Temple" {
		t.Fatal("completed fight area overwritten")
	}
	e.hit(now.Add(4*time.Second), game.Hit{Actor: 1, Target: 30, Damage: 10})
	e.observe(serverMessage(0x3621, scenePacket(600021), now.Add(5*time.Second)))
	if !e.active || e.scene.Name != "Draupnir" {
		t.Fatal("candidate committed before arrival")
	}
	e.observe(serverMessage(0x3623, make([]byte, 20), now.Add(6*time.Second)))
	if e.active || e.scene.Name != "Fire Temple" {
		t.Fatal("candidate-before-arrival transition missing")
	}
}
func TestSceneRejectsUnknownClientTruncatedAndStale(t *testing.T) {
	e, now := encounterFixture()
	e.observe(serverMessage(0x3621, scenePacket(600021), now))
	for _, m := range []aMessage{
		{opcode: 0x3621, payload: scenePacket(600031), flags: wire.FromClient, t: now.Add(time.Second)},
		serverMessage(0x3621, scenePacket(123456789), now.Add(time.Second)),
		serverMessage(0x3621, scenePacket(600031)[:8], now.Add(time.Second)),
		serverMessage(0x3621, scenePacket(600031), now.Add(-time.Second)),
		serverMessage(0x3623, nil, now.Add(time.Second)),
	} {
		e.observe(m)
	}
	if e.scene.ID != 600021 || e.scene.candidate != 0 || e.scene.arrival {
		t.Fatalf("invalid metadata accepted: %+v", e.scene)
	}
}
func TestInstanceRegistrationAndSameMapReentry(t *testing.T) {
	e, now := encounterFixture()
	e.observe(serverMessage(0x3621, scenePacket(600021), now))
	key := []byte("map-event")
	p := binary.LittleEndian.AppendUint32(nil, 42)
	p = append(p, byte(len(key)))
	p = append(p, key...)
	e.observe(serverMessage(0x922e, p, now))
	e.npc(20, bossCode())
	e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 10})
	if e.snapshot(identities{}, now).Scene.Instance != 42 {
		t.Fatal("instance ID missing")
	}
	e.observe(serverMessage(0x922f, binary.LittleEndian.AppendUint32(nil, 99), now))
	e.observe(serverMessage(0x3623, make([]byte, 20), now))
	if !e.active {
		t.Fatal("unrelated event unregister split fight")
	}
	e.observe(serverMessage(0x922f, binary.LittleEndian.AppendUint32(nil, 42), now))
	e.observe(serverMessage(0x3623, make([]byte, 20), now))
	if e.active || len(e.npcs) != 0 {
		t.Fatal("confirmed same-map reentry did not end old fight")
	}
}
func TestSceneOpcodeSurvivesWireFraming(t *testing.T) {
	// Exercise the actual dependency, not just handcrafted aMessage values.
	d := wire.NewDecoder(wire.Config{EmitUnlocked: true})
	src := netip.MustParseAddrPort("192.0.2.1:13328")
	dst := netip.MustParseAddrPort("192.0.2.2:10000")
	p := scenePacket(600021)
	frame := binary.AppendUvarint(nil, uint64(len(p)+6))
	frame = append(frame, 0x21, 0x36)
	frame = append(frame, p...)
	d.Feed(time.Unix(100, 0), src, dst, frame)
	found := false
	for f := range d.Frames() {
		if f.Opcode == 0x3621 {
			found = true
			if binary.LittleEndian.Uint32(f.Payload[4:]) != 600021 {
				t.Fatal("wire payload offset mismatch")
			}
		}
	}
	if !found {
		t.Fatal("scene packet discarded by capture dependency")
	}
}

func TestCharacterExitArchivesBeforeClearingMetadata(t *testing.T) {
	e, now := encounterFixture()
	e.npc(20, bossCode())
	e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 100})
	calls := 0
	e.onFinish = func(completed *encounter, _ time.Time) {
		calls++
		if completed.snapshot(identities{}, now).Boss == nil || completed.meter.actors[1].damage != 100 {
			t.Fatal("exit cleared metadata before archiving")
		}
	}
	e.observe(serverMessage(0x3615, nil, now.Add(time.Second)))
	if e.active || calls != 1 || len(e.npcs) != 0 || e.scene.ID != 0 {
		t.Fatal("character exit retained stale session")
	}
}
