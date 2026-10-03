package main

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type blockedSnapshotWriter struct {
	bytes.Buffer
	started, release chan struct{}
	once             sync.Once
}

func (w *blockedSnapshotWriter) Write(data []byte) (int, error) {
	w.once.Do(func() { close(w.started); <-w.release })
	return w.Buffer.Write(data)
}

func TestSlowDesktopDoesNotBlockAccountingOrLoseHistory(t *testing.T) {
	w := &blockedSnapshotWriter{started: make(chan struct{}), release: make(chan struct{})}
	p := newSnapshotPublisher(w)
	p.offer(snapshot{Session: 1})
	<-w.started
	history := []historyEntry{{Snapshot: snapshot{Session: 1}}}
	p.offer(snapshot{Session: 2, History: &history})
	done := make(chan struct{})
	go func() {
		for n := uint64(3); n <= 10000; n++ {
			p.offer(snapshot{Session: n})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		close(w.release)
		t.Fatal("snapshot queue blocked accounting")
	}
	close(w.release)
	if err := p.close(); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&w.Buffer)
	var first, last snapshot
	if err := decoder.Decode(&first); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&last); err != nil {
		t.Fatal(err)
	}
	if first.Session != 1 || last.Session != 10000 || last.History == nil || (*last.History)[0].Snapshot.Session != 1 {
		t.Fatal("latest frame/history lost")
	}
}
