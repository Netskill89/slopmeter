package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fakeDumpcap(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dumpcap"), []byte("#!/bin/sh\n"+body), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestCaptureHelperRejectsFailedCapture(t *testing.T) {
	fakeDumpcap(t, "exit 7\n")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, _, err := openLiveCapture(ctx, "test0"); err == nil {
		t.Fatal("failed capture accepted")
	}
}

func TestCaptureHelperStreamsAndCloses(t *testing.T) {
	// An empty little-endian pcap followed by a still-running capture process.
	fakeDumpcap(t, "printf '\\324\\303\\262\\241\\002\\000\\004\\000\\000\\000\\000\\000\\000\\000\\000\\000\\377\\377\\000\\000\\001\\000\\000\\000'\nwhile :; do :; done\n")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	r, helper, err := openLiveCapture(ctx, "test0")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	done := make(chan struct{})
	go func() { helper.close(); helper.close(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("capture helper did not stop")
	}
}

func TestPackagedCaptureUsesAuthorizationWithoutSystemDumpcap(t *testing.T) {
	dir := t.TempDir()
	bundled := filepath.Join(dir, "bundled-dumpcap")
	if err := os.WriteFile(bundled, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "setpriv"), []byte("#!/bin/sh\nexit 0\n"), 0755)
	authority := filepath.Join(dir, "pkexec")
	if err := os.WriteFile(authority, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("SLOPMETER_DUMPCAP", bundled)
	cmd, err := liveDumpcapCommand(context.Background(), "test0", "-q", "-f", "tcp", "-w", "-")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != authority || len(cmd.Args) != 15 || cmd.Args[9] != bundled || cmd.Args[14] != "-" {
		t.Fatalf("unexpected authorized helper: %v", cmd.Args)
	}
}

func TestPackagedCaptureReusesPermittedSystemHelper(t *testing.T) {
	fakeDumpcap(t, "exit 0\n")
	t.Setenv("SLOPMETER_DUMPCAP", "/unused/bundled-dumpcap")
	t.Setenv("LD_LIBRARY_PATH", "/unused/bundled-libraries")
	cmd, err := liveDumpcapCommand(context.Background(), "test0", "-q")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(cmd.Path) != "dumpcap" {
		t.Fatalf("system helper not reused: %s", cmd.Path)
	}
	for _, entry := range cmd.Env {
		if len(entry) > 16 && entry[:16] == "LD_LIBRARY_PATH=" {
			t.Fatal("system helper inherited bundled libraries")
		}
	}
}
