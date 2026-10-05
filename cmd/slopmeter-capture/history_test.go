package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nuriland/a2kit/game"
)

func TestFinishedFightReceivesLateClass(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	h, err := loadHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	e, now := encounterFixture()
	i := identities{self: 1, name: "Self", names: map[game.Entity]string{1: "Self"}}
	e.hit(now, game.Hit{Actor: 1, Target: 20, Damage: 1803})
	e.finish(now, "Finished", false)
	if err := h.archive(e, i, now); err != nil {
		t.Fatal(err)
	}
	h.dirty = false
	i.classes = map[game.Entity]string{1: "Cleric"}
	i.names[1] = "Different player"
	if err := h.enrichClasses(i); err != nil || h.dirty {
		t.Fatal("reused entity ID enriched another player's fight", err)
	}
	i.names[1] = "Self"
	if err := h.enrichClasses(i); err != nil {
		t.Fatal(err)
	}
	r := h.entries[0].Snapshot.Actors[0]
	if !h.dirty || r.Class != "Cleric" || r.Name != "Self" || r.Damage != 1803 {
		t.Fatalf("late metadata failed or altered combat totals: %+v", r)
	}
	loaded, err := loadHistory(path)
	if err != nil || loaded.entries[0].Snapshot.Actors[0].Class != "Cleric" {
		t.Fatal("enriched class was not persisted", err)
	}
	h.dirty = false
	i.classes[1] = "Templar"
	if err := h.enrichClasses(i); err != nil || h.dirty || h.entries[0].Snapshot.Actors[0].Class != "Cleric" {
		t.Fatal("known historical class overwritten", err)
	}
}

func TestHistoryRolloverPersistenceAndImmutability(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "history.json")
	h, err := loadHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	i := identities{self: 1, name: "Self", names: map[game.Entity]string{1: "Self"}}
	e := newEncounter(0, 5*time.Second, 90*time.Second)
	e.onFinish = func(e *encounter, at time.Time) {
		if err := h.archive(e, i, at); err != nil {
			t.Fatal(err)
		}
	}
	start := time.Unix(100, 0)
	for n := 0; n < 12; n++ {
		at := start.Add(time.Duration(n) * 10 * time.Second)
		e.hit(at, game.Hit{Actor: 1, Target: 2, Skill: 4000000000, Damage: uint32(n + 1), Type: 3})
		e.tick(at.Add(5 * time.Second))
	}
	if len(h.entries) != 10 || h.entries[0].Snapshot.Session != 12 || h.entries[9].Snapshot.Session != 3 {
		t.Fatal("wrong retained sessions")
	}
	e.finish(start.Add(200*time.Second), "Already finished", false)
	if len(h.entries) != 10 || h.entries[0].Snapshot.Session != 12 {
		t.Fatal("duplicate archive")
	}
	loaded, err := loadHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.lastID() != 12 || loaded.entries[9].Snapshot.Actors[0].Damage != 3 {
		t.Fatal("history did not survive restart")
	}
	if loaded.entries[0].Snapshot.Actors[0].Skills[0].Damage != 12 || loaded.entries[0].Snapshot.Active {
		t.Fatal("mutable or active history entry")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatal("history should be private to the user")
	}
}

func TestBossTransitionArchivesBothFights(t *testing.T) {
	h, _ := loadHistory("")
	i := identities{names: map[game.Entity]string{1: "Self"}}
	e := newEncounter(0, 5*time.Second, 90*time.Second)
	e.onFinish = func(e *encounter, t time.Time) {
		if err := h.archive(e, i, t); err != nil {
			panic(err)
		}
	}
	t0 := time.Unix(100, 0)
	e.hit(t0, game.Hit{Actor: 1, Target: 2, Skill: 4000000000, Damage: 10})
	e.npcs[3] = &npcState{definition: bossDefinition{Name: "Boss", Divisor: 1}, hp: 100, max: 100, known: true}
	e.cast(t0.Add(time.Second), game.Cast{Actor: 1, Target: 3, Skill: 4000000000})
	e.hit(t0.Add(2*time.Second), game.Hit{Actor: 1, Target: 3, Skill: 4000000000, Damage: 20})
	e.finish(t0.Add(3*time.Second), "Boss defeated", false)
	if len(h.entries) != 2 || h.entries[0].Snapshot.Actors[0].Damage != 20 || h.entries[1].Snapshot.Actors[0].Damage != 10 {
		t.Fatal("boss transition lost a fight")
	}
	if h.entries[0].Snapshot.Actors[0].Skills[0].Uses != 1 {
		t.Fatal("lost cast before boss's first hit")
	}
	e.cast(t0.Add(time.Second), game.Cast{Actor: 1, Target: 3, Skill: 4000000000})
	e.hit(t0.Add(4*time.Second), game.Hit{Actor: 1, Target: 3, Skill: 4000000000, Damage: 30})
	if e.meter.actors[1].skill(4000000000).casts != 0 {
		t.Fatal("previous pull's cast leaked")
	}
}

func TestHistoryRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadHistory(path); err == nil {
		t.Fatal("corrupt history accepted")
	}
}
