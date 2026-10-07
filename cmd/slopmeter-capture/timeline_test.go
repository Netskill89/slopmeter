package main

import (
	"encoding/json"
	"github.com/nuriland/a2kit/game"
	"testing"
	"time"
)

func TestCastTimelineTimingAndHistory(t *testing.T) {
	start := time.Unix(1000, 0)
	m := meter{target: 9}
	m.cast(game.Cast{Actor: 1, Target: 9, Skill: 123}, start.Add(-time.Second))
	m.add(start, game.Hit{Actor: 1, Target: 9, Skill: 123, Damage: 100})
	m.cast(game.Cast{Actor: 1, Target: 9, Skill: 456}, start.Add(4*time.Second))
	m.cast(game.Cast{Actor: 1, Target: 8, Skill: 789}, start.Add(5*time.Second))
	m.add(start.Add(10*time.Second), game.Hit{Actor: 1, Target: 9, Skill: 123, Damage: 100})
	snapshot := m.snapshot(identities{})
	uses := snapshot.Actors[0].Timeline
	if len(uses) != 2 || uses[0].Time != -1 || uses[1].Time != 4 || uses[1].Skill != 456 {
		t.Fatalf("incorrect timeline: %+v", uses)
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var restored snapshotAlias
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if len(restored.Actors[0].Timeline) != 2 {
		t.Fatal("timeline lost in history serialization")
	}
	if snapshot.Actors[0].Damage != 200 || snapshot.Duration != 10 {
		t.Fatal("cast timing changed damage or fight duration")
	}
	m = meter{}
	m.add(start, game.Hit{Actor: 1, Target: 9, Damage: 1})
	if len(m.snapshot(identities{}).Actors[0].Timeline) != 0 {
		t.Fatal("damage hits invented casts")
	}
}

type snapshotAlias snapshot

func TestCastTimelineBound(t *testing.T) {
	m := meter{}
	for n := 0; n < maxTimelineCasts+5; n++ {
		m.cast(game.Cast{Actor: 1, Skill: 123}, time.Unix(int64(n+1), 0))
	}
	a := m.actors[1]
	if len(a.timeline) != maxTimelineCasts || !a.timelineTruncated || a.skill(123).casts != maxTimelineCasts+5 {
		t.Fatal("timeline cap changed usage totals")
	}
}
