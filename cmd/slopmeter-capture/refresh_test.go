package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestRefreshCommandsKeepLatestValidRate(t *testing.T) {
	changes := make(chan time.Duration, 1)
	var errors bytes.Buffer
	readRefreshCommands(context.Background(), strings.NewReader("{\"intervalMs\":50}\n{\"intervalMs\":125}\n{\"intervalMs\":200}\n{\"intervalMs\":1000}\n{\"intervalMs\":1001}\ninvalid\n"), changes, nil, &errors)
	if len(changes) != 1 || <-changes != time.Second {
		t.Fatal("control mailbox did not retain the latest valid interval")
	}
	if errors.Len() == 0 {
		t.Fatal("invalid commands were silently accepted")
	}
}

func TestRefreshBoundsAndCancellation(t *testing.T) {
	for n := 50; n <= 1000; n += 50 {
		if !validRefreshInterval(time.Duration(n) * time.Millisecond) {
			t.Fatalf("rejected %d ms", n)
		}
	}
	for _, n := range []int{0, 1, 49, 51, 125, 1001} {
		if validRefreshInterval(time.Duration(n) * time.Millisecond) {
			t.Fatalf("accepted %d ms", n)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	changes := make(chan time.Duration, 1)
	readRefreshCommands(ctx, strings.NewReader("{\"intervalMs\":50}\n"), changes, nil, &bytes.Buffer{})
	if len(changes) != 0 {
		t.Fatal("command applied after cancellation")
	}
}
