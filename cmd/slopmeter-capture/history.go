package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type historyEntry struct {
	Snapshot snapshot  `json:"snapshot"`
	Started  time.Time `json:"started"`
	Ended    time.Time `json:"ended"`
}

type combatHistory struct {
	entries []historyEntry
	path    string
	dirty   bool
}

func defaultHistoryPath() (string, error) {
	root := os.Getenv("XDG_STATE_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(root, "aiondps", "history.json"), nil
}

func loadHistory(path string) (*combatHistory, error) {
	h := &combatHistory{path: path, entries: []historyEntry{}, dirty: true}
	if path == "" {
		return h, nil
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return h, nil
	}
	if err != nil {
		return nil, err
	}
	var file struct {
		Version int            `json:"version"`
		Entries []historyEntry `json:"entries"`
	}
	if err := json.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	if file.Version != 1 {
		return nil, fmt.Errorf("unsupported combat history version %d", file.Version)
	}
	h.entries = file.Entries
	if len(h.entries) > 10 {
		h.entries = h.entries[:10]
	}
	if h.entries == nil {
		h.entries = []historyEntry{}
	}
	return h, nil
}

func (h *combatHistory) lastID() uint64 {
	var id uint64
	for _, entry := range h.entries {
		id = max(id, entry.Snapshot.Session)
	}
	return id
}

func (h *combatHistory) archive(e *encounter, i identities, t time.Time) error {
	state := e.snapshot(i, t)
	if len(state.Actors) == 0 {
		return nil
	}
	state.History = nil
	h.entries = append([]historyEntry{{Snapshot: state, Started: e.meter.first, Ended: t}}, h.entries...)
	if len(h.entries) > 10 {
		h.entries = h.entries[:10]
	}
	h.dirty = true
	return h.save()
}

func (h *combatHistory) save() error {
	if h.path == "" {
		return nil
	}
	b, err := json.Marshal(struct {
		Version int            `json:"version"`
		Entries []historyEntry `json:"entries"`
	}{1, h.entries})
	if err != nil {
		return err
	}
	dir := filepath.Dir(h.path)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".history-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), h.path)
}
