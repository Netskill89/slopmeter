package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/nuriland/a2kit"
	"github.com/nuriland/a2kit/game"
)

// Set by scripts/build.sh from VERSION.
var version = "development"

type total struct {
	damage    uint64
	hits      uint64
	skills    map[game.Skill]uint64
	breakdown map[game.Skill]*skillTotal
}
type meter struct {
	first, last time.Time
	actors      map[game.Entity]*total
	target      game.Entity
}

func (m *meter) add(t time.Time, h game.Hit) {
	if m.target != 0 && h.Target != m.target {
		return
	}
	if m.first.IsZero() || t.Before(m.first) {
		m.first = t
	}
	if t.After(m.last) {
		m.last = t
	}
	a := m.actor(h.Actor)
	a.damage += uint64(h.Damage)
	a.hits++
	a.skills[h.Skill] += uint64(h.Damage)
	skill := a.skill(h.Skill)
	skill.damage += uint64(h.Damage)
	if h.Damage > 0 {
		if skill.hits == 0 || h.Damage < skill.min {
			skill.min = h.Damage
		}
		skill.max = max(skill.max, h.Damage)
		skill.hits++
		if h.Type == 2 || h.Type == 3 {
			skill.typed++
		}
		if h.Type == 3 {
			skill.critical++
		}
	}
}

func (m *meter) print(w io.Writer, details bool) {
	seconds := m.last.Sub(m.first).Seconds()
	if seconds < 1 {
		seconds = 1
	}
	fmt.Fprintf(w, "\nAION 2 fight totals — duration %.1fs (DPS denominator at least 1s)\n", m.last.Sub(m.first).Seconds())
	fmt.Fprintln(w, "ACTOR          DAMAGE          DPS       HITS     SHARE")
	ids := make([]game.Entity, 0, len(m.actors))
	var sum uint64
	for id, a := range m.actors {
		ids = append(ids, id)
		sum += a.damage
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := m.actors[ids[i]], m.actors[ids[j]]
		if a.damage == b.damage {
			return ids[i] < ids[j]
		}
		return a.damage > b.damage
	})
	for _, id := range ids {
		a := m.actors[id]
		share := float64(0)
		if sum > 0 {
			share = float64(a.damage) / float64(sum) * 100
		}
		fmt.Fprintf(w, "%-10d %12d %12.1f %10d %8.1f%%\n", id, a.damage, float64(a.damage)/seconds, a.hits, share)
		if details {
			skills := make([]game.Skill, 0, len(a.skills))
			for s := range a.skills {
				skills = append(skills, s)
			}
			sort.Slice(skills, func(i, j int) bool {
				if a.skills[skills[i]] == a.skills[skills[j]] {
					return skills[i] < skills[j]
				}
				return a.skills[skills[i]] > a.skills[skills[j]]
			})
			for _, s := range skills {
				fmt.Fprintf(w, "  skill %-10d %12d damage\n", s, a.skills[s])
			}
		}
	}
	if len(ids) == 0 {
		fmt.Fprintln(w, "Waiting for decoded damage events.")
	}
}

func run() error {
	showVersion := flag.Bool("version", false, "Print SlopMeter version")
	file := flag.String("read", "", "Replay a pcap, pcapng or a2log JSONL file")
	adapter := flag.String("interface", "", "Live interface (default: dumpcap-selected interface)")
	devices := flag.Bool("interfaces", false, "List capture interfaces")
	target := flag.Uint64("target", 0, "Only count hits on this target entity ID")
	interval := flag.Duration("interval", 200*time.Millisecond, "Live refresh interval: 50ms–1s in 50ms steps")
	idle := flag.Duration("idle", 5*time.Second, "Open-world encounter timeout")
	bossIdle := flag.Duration("boss-idle", 90*time.Second, "Boss encounter timeout without damage")
	jsonOutput := flag.Bool("json", false, "Stream UI snapshots as JSON lines")
	historyFile := flag.String("history", "", "Combat history file (default: XDG state directory for live capture; memory for replay)")
	flag.Parse()
	if *showVersion {
		fmt.Println("SlopMeter", version)
		return nil
	}
	if *target > 0xffffffff {
		return fmt.Errorf("target must fit a 32-bit entity ID")
	}
	if !validRefreshInterval(*interval) {
		return fmt.Errorf("interval must be 50ms–1s in 50ms steps")
	}
	if *idle <= 0 || *bossIdle <= 0 {
		return fmt.Errorf("encounter timeouts must be positive")
	}
	if *devices {
		if *jsonOutput {
			return listNetworkInterfaces()
		}
		return listCaptureInterfaces()
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var r *a2kit.Reader
	var err error
	var live *captureHelper
	if *file != "" {
		r, err = a2kit.Open(*file, a2kit.Config{})
	} else {
		r, live, err = openLiveCapture(ctx, *adapter)
		if live != nil {
			defer live.close()
		}
	}
	if err != nil {
		return err
	}
	var closeOnce sync.Once
	closeReader := func() { closeOnce.Do(func() { _ = r.Close() }) }
	defer closeReader()
	historyPath := *historyFile
	if historyPath == "" && *file == "" {
		historyPath, err = defaultHistoryPath()
		if err != nil {
			return err
		}
	}
	history, err := loadHistory(historyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "History unavailable:", err)
		history, _ = loadHistory("")
	}
	fight := newEncounter(game.Entity(*target), *idle, *bossIdle)
	fight.number = history.lastID()
	group := scope{identities: identities{names: make(map[game.Entity]string)}}
	archive := func(e *encounter, t time.Time) {
		if err := history.archive(e, group.identities, t); err != nil {
			fmt.Fprintln(os.Stderr, "Could not save combat history:", err)
		}
	}
	fight.onFinish = archive
	logicalTime := time.Now()
	defer func() {
		if fight.active {
			end := logicalTime
			if *file == "" {
				end = time.Now()
			}
			fight.finish(end, "Capture ended", false)
		}
	}()
	var publisher *snapshotPublisher
	if *jsonOutput {
		publisher = newSnapshotPublisher(os.Stdout)
		defer publisher.close()
	}
	emit := func(t time.Time) {
		state := fight.snapshot(group.identities, t)
		if history.dirty {
			entries := append([]historyEntry{}, history.entries...)
			state.History = &entries
			history.dirty = false
		}
		if *jsonOutput {
			publisher.offer(state)
		} else {
			fmt.Fprintf(os.Stdout, "\n%s · encounter %d\n", state.Encounter, state.Session)
			m := fight.meter
			if !m.first.IsZero() {
				m.last = m.first.Add(time.Duration(state.Duration * float64(time.Second)))
			}
			m.print(os.Stdout, false)
		}
	}
	if *jsonOutput {
		emit(logicalTime)
	}
	type incoming struct {
		message a2kit.Message
		err     error
	}
	messages := make(chan incoming, 256)
	go func() {
		defer close(messages)
		for msg, err := range r.Messages() {
			select {
			case messages <- incoming{msg, err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	timer := time.NewTicker(*interval)
	defer timer.Stop()
	refreshChanges := make(chan time.Duration, 1)
	go readRefreshCommands(ctx, os.Stdin, refreshChanges, os.Stderr)
	dirty := false
	var readError error
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case interval := <-refreshChanges:
			timer.Reset(interval)
		case now := <-timer.C:
			if *file != "" {
				continue
			}
			changed := fight.tick(now)
			if changed || fight.active || dirty || history.dirty {
				emit(now)
				dirty = false
			}
		case result, open := <-messages:
			if !open {
				break loop
			}
			if result.err != nil {
				readError = result.err
				break loop
			}
			msg := result.message
			if msg.Time.After(logicalTime) || *file != "" {
				logicalTime = msg.Time
			}
			oldSelf, oldName := group.self, group.name
			oldSession, oldStatus, oldBoss := fight.number, fight.status, fight.boss
			am := aMessage{msg.Opcode, msg.Flags, msg.Payload, msg.Event, msg.Time}
			fight.tick(msg.Time)
			fight.observe(am)
			if p, ok := msg.Event.(game.Player); ok && p.Self && oldSelf != 0 && p.Entity != oldSelf {
				fight.finish(msg.Time, "Character changed", true)
			}
			group.observeMessage(am)
			if oldSelf != 0 && group.self != oldSelf {
				number := fight.number
				fight = newEncounter(game.Entity(*target), *idle, *bossIdle)
				fight.number = number
				fight.onFinish = archive
			}
			if c, ok := msg.Event.(game.Cast); ok && group.accepts(c.Actor, msg.Time) {
				fight.cast(msg.Time, c)
				dirty = true
			}
			if h, ok := msg.Event.(game.Hit); ok && group.accepts(h.Actor, msg.Time) {
				fight.hit(msg.Time, h)
				dirty = true
			}
			if oldSelf != group.self || oldName != group.name || oldSession != fight.number || oldStatus != fight.status || oldBoss != fight.boss {
				dirty = true
			}
			if p, ok := msg.Event.(game.Player); ok && fight.meter.actors[p.Entity] != nil {
				dirty = true
			}
			if fight.boss != 0 && (msg.Opcode == 0x8d00 || msg.Opcode == 0x3640 || msg.Opcode == 0x3641) {
				dirty = true
			}
		}
	}
	if live != nil && ctx.Err() == nil && readError == nil {
		if err := live.wait(); err != nil {
			return fmt.Errorf("dumpcap stopped: %w (see capture error above)", err)
		}
	}
	if ctx.Err() == nil && readError != nil {
		return readError
	}
	// Stop the reader before inspecting its counters, including on Ctrl+C.
	stop()
	closeReader()
	for range messages {
	}
	end := logicalTime
	if *file == "" {
		end = time.Now()
	}
	if fight.active {
		fight.finish(end, "Capture ended", false)
	}
	emit(end)
	s := r.Summary()
	fmt.Fprintf(os.Stderr, "Decoded %d/%d messages; unsupported %d; layout failures %d; truncated segments %d\n", s.Decoded, s.Messages, s.Unread, s.Failed, s.CutShort)
	if s.Lost != nil {
		fmt.Fprintf(os.Stderr, "Capture loss: %v\n", s.Lost)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "slopmeter:", err)
		os.Exit(1)
	}
}
