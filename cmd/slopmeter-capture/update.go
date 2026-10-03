package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var version = "0.1.0-alpha.1"
var releaseRepository = "https://github.com/Netskill89/slopmeter"
var stableVersion = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*))?$`)

type updateInfo struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"available"`
	Page      string `json:"page"`
	Asset     string `json:"asset"`
	URL       string `json:"url"`
	Checksums string `json:"checksums"`
	Path      string `json:"path,omitempty"`
}
type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

func newerVersion(candidate, current string) bool {
	a, b := stableVersion.FindStringSubmatch(candidate), stableVersion.FindStringSubmatch(current)
	if a == nil || b == nil {
		return false
	}
	for n := 1; n <= 3; n++ {
		x, err := strconv.ParseUint(a[n], 10, 64)
		if err != nil {
			return false
		}
		y, err := strconv.ParseUint(b[n], 10, 64)
		if err != nil {
			return false
		}
		if x != y {
			return x > y
		}
	}
	if a[4] == b[4] {
		return false
	}
	if a[4] == "" {
		return true
	}
	if b[4] == "" {
		return false
	}
	aa, bb := strings.Split(a[4], "."), strings.Split(b[4], ".")
	for n := 0; n < len(aa) && n < len(bb); n++ {
		if aa[n] == bb[n] {
			continue
		}
		x, xe := strconv.ParseUint(aa[n], 10, 64)
		y, ye := strconv.ParseUint(bb[n], 10, 64)
		if xe == nil && ye == nil {
			return x > y
		}
		if xe == nil {
			return false
		}
		if ye == nil {
			return true
		}
		return aa[n] > bb[n]
	}
	return len(aa) > len(bb)
}
func releaseEndpoint(repository string) (string, bool, error) {
	u, err := url.Parse(strings.TrimSuffix(repository, "/"))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", false, fmt.Errorf("use a public HTTPS GitHub/GitLab repository URL")
	}
	path := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || strings.Contains(path, "/-/") {
		return "", false, fmt.Errorf("use the repository URL, not its releases page")
	}
	if u.Host == "github.com" {
		if len(parts) != 2 {
			return "", false, fmt.Errorf("GitHub repository must be owner/project")
		}
		return "https://api.github.com/repos/" + path + "/releases?per_page=20", true, nil
	}
	return "https://" + u.Host + "/api/v4/projects/" + url.PathEscape(path) + "/releases?per_page=20", false, nil
}
func fetch(ctx context.Context, client *http.Client, address string) (*http.Response, error) {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return nil, fmt.Errorf("release download must use HTTPS")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "SlopMeter/"+version)
	req.Header.Set("Accept", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("release server returned %s", response.Status)
	}
	return response, nil
}
func checkUpdate(ctx context.Context, client *http.Client, repository, format string) (updateInfo, error) {
	info := updateInfo{Current: version}
	endpoint, github, err := releaseEndpoint(repository)
	if err != nil {
		return info, err
	}
	response, err := fetch(ctx, client, endpoint)
	if err != nil {
		return info, err
	}
	defer response.Body.Close()
	type remoteRelease struct {
		Tag        string `json:"tag_name"`
		Page       string `json:"html_url"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Upcoming   bool   `json:"upcoming_release"`
		Links      struct {
			Self string `json:"self"`
		} `json:"_links"`
		Assets json.RawMessage `json:"assets"`
	}
	var releases []remoteRelease
	if err = json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&releases); err != nil {
		return info, err
	}
	var release remoteRelease
	alpha := strings.Contains(version, "-")
	for _, candidate := range releases {
		if candidate.Draft || candidate.Upcoming || !stableVersion.MatchString(candidate.Tag) || !alpha && (candidate.Prerelease || strings.Contains(candidate.Tag, "-")) {
			continue
		}
		if release.Tag == "" || newerVersion(candidate.Tag, release.Tag) {
			release = candidate
		}
	}
	if release.Tag == "" {
		return info, fmt.Errorf("no compatible versioned release found")
	}
	info.Latest = strings.TrimPrefix(release.Tag, "v")
	info.Available = newerVersion(info.Latest, version)
	info.Page = release.Page
	if info.Page == "" {
		info.Page = release.Links.Self
	}
	var assets []releaseAsset
	if len(release.Assets) == 0 {
		// An up-to-date release need not contain this architecture’s assets.
	} else if github {
		err = json.Unmarshal(release.Assets, &assets)
	} else {
		var nested struct {
			Links []struct {
				Name   string `json:"name"`
				URL    string `json:"url"`
				Direct string `json:"direct_asset_url"`
			} `json:"links"`
		}
		err = json.Unmarshal(release.Assets, &nested)
		for _, a := range nested.Links {
			address := a.URL
			if a.Direct != "" {
				address = a.Direct
			}
			assets = append(assets, releaseAsset{a.Name, address})
		}
	}
	if err != nil {
		return info, err
	}
	if format != "AppImage" && format != "tar.gz" {
		return info, fmt.Errorf("format must be AppImage or tar.gz")
	}
	arch := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[runtime.GOARCH]
	if arch == "" {
		return info, fmt.Errorf("unsupported release architecture %s", runtime.GOARCH)
	}
	info.Asset = "SlopMeter-" + info.Latest + "-" + arch + "." + format
	for _, a := range assets {
		if a.Name == info.Asset {
			info.URL = a.URL
		}
		if a.Name == "SHA256SUMS" {
			info.Checksums = a.URL
		}
	}
	if info.Available && (info.URL == "" || info.Checksums == "") {
		return info, fmt.Errorf("release is missing %s or SHA256SUMS", info.Asset)
	}
	return info, nil
}
func downloadUpdate(ctx context.Context, client *http.Client, info updateInfo, directory string) (string, error) {
	if !info.Available {
		return "", fmt.Errorf("already using the latest stable release")
	}
	sums, err := fetch(ctx, client, info.Checksums)
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(io.LimitReader(sums.Body, 1<<20))
	sums.Body.Close()
	if err != nil {
		return "", err
	}
	expected := ""
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == info.Asset {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	digest, err := hex.DecodeString(expected)
	if err != nil || len(digest) != sha256.Size {
		return "", fmt.Errorf("missing or invalid checksum for %s", info.Asset)
	}
	if filepath.Base(info.Asset) != info.Asset {
		return "", fmt.Errorf("invalid release filename")
	}
	if err = os.MkdirAll(directory, 0755); err != nil {
		return "", err
	}
	destination := filepath.Join(directory, info.Asset)
	if _, err = os.Lstat(destination); !os.IsNotExist(err) {
		return "", fmt.Errorf("destination already exists or cannot be accessed: %s", destination)
	}
	response, err := fetch(ctx, client, info.URL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	file, err := os.CreateTemp(directory, ".slopmeter-update-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, (512<<20)+1))
	if err != nil {
		return "", err
	}
	if size > 512<<20 {
		return "", fmt.Errorf("release exceeds 512 MiB limit")
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		return "", fmt.Errorf("download checksum mismatch")
	}
	mode := os.FileMode(0644)
	if strings.HasSuffix(info.Asset, ".AppImage") {
		mode = 0755
	}
	if err = file.Chmod(mode); err != nil {
		return "", err
	}
	if err = file.Sync(); err != nil {
		return "", err
	}
	if err = file.Close(); err != nil {
		return "", err
	}
	// Link atomically without replacing another file created during the download.
	if err = os.Link(file.Name(), destination); err != nil {
		return "", err
	}
	return destination, nil
}
func runUpdate(repository, format, directory string, download bool) error {
	if repository == "" {
		repository = os.Getenv("SLOPMETER_RELEASE_REPOSITORY")
	}
	if repository == "" {
		repository = releaseRepository
	}
	if repository == "" {
		return fmt.Errorf("no release repository configured; set it in Settings or SLOPMETER_RELEASE_REPOSITORY")
	}
	client := &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 || req.URL.Scheme != "https" {
			return fmt.Errorf("unsafe release redirect")
		}
		return nil
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	info, err := checkUpdate(ctx, client, repository, format)
	cancel()
	if err != nil {
		return err
	}
	if download {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		info.Path, err = downloadUpdate(ctx, client, info, directory)
		if err != nil {
			return err
		}
	}
	return json.NewEncoder(os.Stdout).Encode(info)
}
