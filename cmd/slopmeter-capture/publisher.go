package main

import (
	"encoding/json"
	"io"
)

// Packet accounting never waits for the desktop to consume a snapshot. One
// queued snapshot bounds memory; superseded UI frames keep their history update.
type snapshotPublisher struct {
	pending chan snapshot
	done    chan struct{}
	err     error
}

func newSnapshotPublisher(output io.Writer) *snapshotPublisher {
	p := &snapshotPublisher{pending: make(chan snapshot, 1), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		encoder := json.NewEncoder(output)
		for state := range p.pending {
			if err := encoder.Encode(state); err != nil {
				p.err = err
				return
			}
		}
	}()
	return p
}

// offer is called only by the combat-accounting goroutine.
func (p *snapshotPublisher) offer(state snapshot) {
	select {
	case p.pending <- state:
		return
	default:
	}
	select {
	case previous := <-p.pending:
		if state.History == nil {
			state.History = previous.History
		}
	default:
	}
	p.pending <- state
}

func (p *snapshotPublisher) close() error {
	close(p.pending)
	<-p.done
	return p.err
}
