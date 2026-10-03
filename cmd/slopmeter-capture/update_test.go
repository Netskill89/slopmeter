package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type updateTransport struct {
	base        http.RoundTripper
	destination *url.URL
}

func (tr updateTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	u := *req.URL
	u.Host = tr.destination.Host
	u.Scheme = tr.destination.Scheme
	copy.URL = &u
	return tr.base.RoundTrip(copy)
}
func TestVersionOrdering(t *testing.T) {
	for _, test := range []struct {
		a, b string
		want bool
	}{
		{"0.1.0-alpha.2", "0.1.0-alpha.1", true}, {"0.1.0-alpha.10", "0.1.0-alpha.2", true}, {"0.1.0", "0.1.0-alpha.10", true},
		{"0.1.1", "0.1.0", true}, {"v1.0.0", "0.9.9", true}, {"0.1.0-alpha.1", "0.1.0", false}, {"0.1.0", "0.1.0", false},
		{"garbage", "0.1.0", false}, {"01.0.0", "0.1.0", false},
	} {
		if got := newerVersion(test.a, test.b); got != test.want {
			t.Fatalf("%s > %s: %v", test.a, test.b, got)
		}
	}
}
func TestReleaseEndpoints(t *testing.T) {
	endpoint, github, err := releaseEndpoint("https://github.com/Netskill89/slopmeter.git")
	if err != nil || !github || endpoint != "https://api.github.com/repos/Netskill89/slopmeter/releases?per_page=20" {
		t.Fatal(endpoint, err)
	}
	endpoint, github, err = releaseEndpoint("https://gitlab.com/group/nested/slopmeter")
	if err != nil || github || !strings.Contains(endpoint, "group%2Fnested%2Fslopmeter") {
		t.Fatal(endpoint, err)
	}
	for _, bad := range []string{"http://github.com/owner/project", "https://token@github.com/owner/project", "https://github.com/owner/project/releases", "https://gitlab.com/group/project/-/releases"} {
		if _, _, err := releaseEndpoint(bad); err == nil {
			t.Fatal("unsafe repository accepted", bad)
		}
	}
}
func TestUpdateSelectionDownloadAndChecksum(t *testing.T) {
	old := version
	defer func() { version = old }()
	version = "0.1.0-alpha.1"
	payload := []byte("synthetic AppImage release")
	hash := sha256.Sum256(payload)
	checksum := hex.EncodeToString(hash[:])
	var asset string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases"):
			fmt.Fprintf(w, `[{"tag_name":"v0.1.0-alpha.2","prerelease":true,"assets":[{"name":%q,"browser_download_url":"https://release.example/image"},{"name":"SHA256SUMS","browser_download_url":"https://release.example/sums"}]},{"tag_name":"v9.0.0","draft":true},{"tag_name":"v0.1.0"}]`, asset)
		case r.URL.Path == "/sums":
			fmt.Fprintf(w, "%s  %s\n", checksum, asset)
		case r.URL.Path == "/image":
			w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	destination, _ := url.Parse(server.URL)
	client := server.Client()
	client.Transport = updateTransport{client.Transport, destination}
	// First test stable selection separately; stable 0.1.0 supersedes its alphas.
	version = "0.1.0"
	info, err := checkUpdate(context.Background(), client, releaseRepository, "AppImage")
	if err != nil || info.Latest != "0.1.0" || info.Available {
		t.Fatal(info, err)
	}
	version = "0.0.1-alpha.1"
	// The newest stable release lacks assets in this fixture: fail rather than download an unrelated file.
	if _, err = checkUpdate(context.Background(), client, releaseRepository, "AppImage"); err == nil {
		t.Fatal("missing release assets accepted")
	}
	info = updateInfo{Available: true, Asset: "SlopMeter-0.1.0-alpha.2-x86_64.AppImage", URL: "https://release.example/image", Checksums: "https://release.example/sums"}
	asset = info.Asset
	folder := t.TempDir()
	path, err := downloadUpdate(context.Background(), client, info, folder)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != string(payload) {
		t.Fatal("download damaged", err)
	}
	if stat, _ := os.Stat(path); stat.Mode().Perm() != 0755 {
		t.Fatal("AppImage is not executable")
	}
	if _, err = downloadUpdate(context.Background(), client, info, folder); err == nil {
		t.Fatal("existing release overwritten")
	}
	checksum = strings.Repeat("0", 64)
	badFolder := t.TempDir()
	if _, err = downloadUpdate(context.Background(), client, info, badFolder); err == nil {
		t.Fatal("checksum mismatch accepted")
	}
	if _, err = os.Stat(filepath.Join(badFolder, asset)); !os.IsNotExist(err) {
		t.Fatal("unverified artifact left behind")
	}
	files, _ := os.ReadDir(badFolder)
	if len(files) != 0 {
		t.Fatal("temporary files left behind")
	}
}
