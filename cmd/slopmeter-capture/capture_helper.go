package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nuriland/a2kit"
)

// Keep capture privileges on the capture helper, never on the decoder or UI.
type captureHelper struct {
	cmd    *exec.Cmd
	pipe   io.ReadCloser
	cancel context.CancelFunc
	once   sync.Once
	err    error
}

func dumpcapCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	path, err := exec.LookPath("dumpcap")
	if bundled := os.Getenv("SLOPMETER_DUMPCAP"); bundled != "" {
		path = bundled
		err = nil
	}
	if err != nil {
		return nil, fmt.Errorf("live capture requires dumpcap; install your distribution's Wireshark capture package and enable non-root capture: %w", err)
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stderr = os.Stderr
	return cmd, nil
}

// A packaged helper lives outside an AppImage's FUSE mount, so pkexec can read it.
// Only dumpcap is elevated; the decoder and UI retain the desktop user's identity.
func withoutLibraryOverride() []string {
	env := []string{}
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "LD_LIBRARY_PATH=") {
			env = append(env, value)
		}
	}
	return env
}
func capturePermitted(ctx context.Context, path, adapter string, system bool) bool {
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	args := []string{"-L"}
	if adapter != "" {
		args = append(args, "-i", adapter)
	}
	probe := exec.CommandContext(probeCtx, path, args...)
	if system {
		probe.Env = withoutLibraryOverride()
	}
	return probe.Run() == nil
}
func liveDumpcapCommand(ctx context.Context, adapter string, args ...string) (*exec.Cmd, error) {
	bundled := os.Getenv("SLOPMETER_DUMPCAP")
	if bundled == "" {
		return dumpcapCommand(ctx, args...)
	}
	if system, err := exec.LookPath("dumpcap"); err == nil && system != bundled && capturePermitted(ctx, system, adapter, true) {
		cmd := exec.CommandContext(ctx, system, args...)
		cmd.Env = withoutLibraryOverride()
		cmd.Stderr = os.Stderr
		return cmd, nil
	}
	if capturePermitted(ctx, bundled, adapter, false) {
		return dumpcapCommand(ctx, args...)
	}
	authority, err := exec.LookPath("pkexec")
	if err != nil {
		return nil, fmt.Errorf("your desktop's Polkit permission service (pkexec) is unavailable; live capture needs administrator authorization")
	}
	fmt.Fprintln(os.Stderr, "Approve the Linux permission dialog to start packet capture. Only the bundled capture helper needs administrator access.")
	launcher := filepath.Join(filepath.Dir(bundled), "setpriv")
	if _, err := os.Stat(launcher); err != nil {
		return nil, fmt.Errorf("bundled capture launcher is missing: %w", err)
	}
	// Drop back to the calling user before capture, retaining only network
	// capabilities. The backend can then terminate its own helper on exit.
	authorized := []string{launcher, "--reuid=" + strconv.Itoa(os.Getuid()), "--regid=" + strconv.Itoa(os.Getgid()), "--clear-groups", "--inh-caps=+net_raw,+net_admin", "--ambient-caps=+net_raw,+net_admin", "--bounding-set=-all,+net_raw,+net_admin", "--no-new-privs", bundled}
	cmd := exec.CommandContext(ctx, authority, append(authorized, args...)...)
	cmd.Stderr = os.Stderr
	return cmd, nil
}

func listCaptureInterfaces() error {
	cmd, err := dumpcapCommand(context.Background(), "-D")
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func openLiveCapture(ctx context.Context, adapter string) (*a2kit.Reader, *captureHelper, error) {
	ctx, cancel := context.WithCancel(ctx)
	args := []string{"-q", "-f", "tcp", "-w", "-"}
	if adapter != "" {
		args = append(args, "-i", adapter)
	}
	cmd, err := liveDumpcapCommand(ctx, adapter, args...)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		_ = pipe.Close()
		return nil, nil, fmt.Errorf("start capture helper: %w", err)
	}
	helper := &captureHelper{cmd: cmd, pipe: pipe, cancel: cancel}
	reader, err := a2kit.OpenReader(pipe, a2kit.Config{})
	if err != nil {
		helper.close()
		return nil, nil, fmt.Errorf("capture did not start: %w; approve the desktop permission dialog when prompted, or check the helper error above", err)
	}
	return reader, helper, nil
}

func (h *captureHelper) wait() error {
	h.once.Do(func() { h.err = h.cmd.Wait() })
	return h.err
}

func (h *captureHelper) close() {
	h.cancel()
	_ = h.pipe.Close()
	_ = h.wait()
}
