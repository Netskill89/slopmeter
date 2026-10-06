package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

func validRefreshInterval(interval time.Duration) bool {
	return interval >= 50*time.Millisecond && interval <= time.Second && interval%(50*time.Millisecond) == 0
}

// UI commands never restart packet acquisition. Separate bounded mailboxes
// prevent slider coalescing from discarding a requested damage reset.
func readRefreshCommands(ctx context.Context, input io.Reader, changes chan time.Duration, resets chan struct{}, report io.Writer) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 1024), 4096)
	for scanner.Scan() {
		var command struct {
			IntervalMS int  `json:"intervalMs"`
			Reset      bool `json:"reset"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &command); err != nil {
			fmt.Fprintln(report, "Ignoring invalid refresh command")
			continue
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		if command.Reset {
			if command.IntervalMS != 0 {
				fmt.Fprintln(report, "Ignoring combined reset/refresh command")
				continue
			}
			select {
			case resets <- struct{}{}:
			default:
			}
			continue
		}
		interval := time.Duration(command.IntervalMS) * time.Millisecond
		if !validRefreshInterval(interval) {
			fmt.Fprintln(report, "Refresh interval must be 50–1000 ms in 50 ms steps")
			continue
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		select {
		case <-changes:
		default:
		}
		select {
		case changes <- interval:
		case <-ctx.Done():
			return
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		fmt.Fprintln(report, "Refresh control:", err)
	}
}
