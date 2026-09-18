package updatecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bren-wp/Host-FTP/internal/brand"
)

var (
	ErrInvalidCurrentVersion = errors.New("invalid current version")
	ErrInvalidManifest       = errors.New("invalid update manifest")
	ErrUntrustedUpdateURL    = errors.New("untrusted update URL")
)

const (
	manifestSchema     = 1
	maxManifestBytes   = 64 << 10
	defaultHTTPTimeout = 12 * time.Second
)

type Manifest struct {
	Schema         int    `json:"schema"`
	Version        string `json:"version"`
	SetupURL       string `json:"setup_url"`
	SetupSHA256    string `json:"setup_sha256"`
	PortableURL    string `json:"portable_url"`
	PortableSHA256 string `json:"portable_sha256"`
	UpdaterURL     string `json:"updater_url,omitempty"`
	UpdaterSHA256  string `json:"updater_sha256,omitempty"`
}

type Result struct {
	CurrentVersion string
	LatestVersion  string
	DisplayVersion string
	UpdateURL      string
	SetupURL       string
	SetupSHA256    string
	PortableURL    string
	PortableSHA256 string
	UpdaterURL     string
	UpdaterSHA256  string
	Available      bool
}

func Check(ctx context.Context, currentVersion string) (Result, error) {
	client := &http.Client{Timeout: defaultHTTPTimeout}
	return checkURL(ctx, client, brand.UpdateManifestURL, currentVersion, true)
}

func checkURL(ctx context.Context, client *http.Client, manifestURL, currentVersion string, requireTrusted bool) (Result, error) {
	current, err := parseVersion(currentVersion)
	if err != nil {
		return Result{}, ErrInvalidCurrentVersion
	}
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	if requireTrusted && !TrustedURL(manifestURL) {
		return Result{}, ErrUntrustedUpdateURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Ghost-FTP/"+current.String())

	resp, err := client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("check for updates: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("check for updates: unexpected HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes+1))
	if err != nil {
		return Result{}, fmt.Errorf("read update manifest: %w", err)
	}
	if len(body) == 0 || len(body) > maxManifestBytes {
		return Result{}, ErrInvalidManifest
	}

	var manifest Manifest
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return Result{}, ErrInvalidManifest
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return Result{}, ErrInvalidManifest
	}
	latest, err := validateManifest(manifest, requireTrusted)
	if err != nil {
		return Result{}, err
	}

	return Result{
		CurrentVersion: current.String(),
		LatestVersion:  latest.String(),
		DisplayVersion: latest.String(),
		UpdateURL:      brand.UpdateURL,
		SetupURL:       manifest.SetupURL,
		SetupSHA256:    strings.ToLower(manifest.SetupSHA256),
		PortableURL:    manifest.PortableURL,
		PortableSHA256: strings.ToLower(manifest.PortableSHA256),
		UpdaterURL:     manifest.UpdaterURL,
		UpdaterSHA256:  strings.ToLower(manifest.UpdaterSHA256),
		Available:      latest.compare(current) > 0,
	}, nil
}

func validateManifest(manifest Manifest, requireTrusted bool) (version, error) {
	if manifest.Schema != manifestSchema {
		return version{}, ErrInvalidManifest
	}
	latest, err := parseVersion(manifest.Version)
	if err != nil {
		return version{}, ErrInvalidManifest
	}
	if !validSHA256(manifest.SetupSHA256) || !validSHA256(manifest.PortableSHA256) {
		return version{}, ErrInvalidManifest
	}
	if strings.TrimSpace(manifest.UpdaterURL) != "" && !validSHA256(manifest.UpdaterSHA256) {
		return version{}, ErrInvalidManifest
	}
	if requireTrusted {
		for _, value := range []string{manifest.SetupURL, manifest.PortableURL} {
			if !TrustedURL(value) {
				return version{}, ErrUntrustedUpdateURL
			}
		}
		if manifest.UpdaterURL != "" && !TrustedURL(manifest.UpdaterURL) {
			return version{}, ErrUntrustedUpdateURL
		}
	}
	if strings.TrimSpace(manifest.SetupURL) == "" || strings.TrimSpace(manifest.PortableURL) == "" {
		return version{}, ErrInvalidManifest
	}
	return latest, nil
}

func TrustedURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || parsed.User != nil {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), "update.ghostftp.com")
}

func validSHA256(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 64 {
		return false
	}
	for _, ch := range value {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}

type version struct {
	major int
	minor int
	patch int
}

func parseVersion(value string) (version, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return version{}, errors.New("version must contain three numeric parts")
	}
	values := [3]int{}
	for i, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0' && part != "0") {
			return version{}, errors.New("invalid numeric version part")
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return version{}, errors.New("invalid numeric version part")
			}
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || n > 999999 {
			return version{}, errors.New("invalid numeric version part")
		}
		values[i] = n
	}
	return version{major: values[0], minor: values[1], patch: values[2]}, nil
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

func (v version) compare(other version) int {
	if v.major != other.major {
		if v.major < other.major { return -1 }
		return 1
	}
	if v.minor != other.minor {
		if v.minor < other.minor { return -1 }
		return 1
	}
	if v.patch < other.patch { return -1 }
	if v.patch > other.patch { return 1 }
	return 0
}
