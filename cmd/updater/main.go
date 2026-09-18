package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bren-wp/Host-FTP/internal/brand"
	"github.com/bren-wp/Host-FTP/internal/platform"
	"github.com/bren-wp/Host-FTP/internal/updatecheck"
)

var version = "0.3.0"

const maxInstallerBytes = 256 << 20

func expectedDigest(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return nil, errors.New("invalid update SHA-256 digest")
	}
	return decoded, nil
}

func downloadInstaller(ctx context.Context, client *http.Client, result updatecheck.Result) (string, error) {
	if !result.Available {
		return "", errors.New("no update is available")
	}
	if !updatecheck.TrustedURL(result.SetupURL) {
		return "", updatecheck.ErrUntrustedUpdateURL
	}
	expected, err := expectedDigest(result.SetupSHA256)
	if err != nil {
		return "", err
	}
	if client == nil {
		client = &http.Client{Timeout: 4 * time.Minute}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, result.SetupURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Ghost-FTP-Updater/"+version)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download update: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download update: unexpected HTTP status %d", resp.StatusCode)
	}
	if resp.ContentLength > maxInstallerBytes {
		return "", errors.New("update package is unexpectedly large")
	}

	file, err := os.CreateTemp("", "Ghost-FTP-"+result.LatestVersion+"-Setup-*.exe")
	if err != nil {
		return "", err
	}
	path := file.Name()
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(path)
	}

	hash := sha256.New()
	limited := io.LimitReader(resp.Body, maxInstallerBytes+1)
	written, err := io.Copy(io.MultiWriter(file, hash), limited)
	if err != nil {
		cleanup()
		return "", err
	}
	if written <= 0 || written > maxInstallerBytes {
		cleanup()
		return "", errors.New("update package has an invalid size")
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), hex.EncodeToString(expected)) {
		_ = os.Remove(path)
		return "", errors.New("update package integrity verification failed")
	}
	return path, nil
}

func run() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := updatecheck.Check(ctx, version)
	if err != nil {
		platform.ErrorDialog(
			brand.ProductName+" Update",
			"Unable to check for updates",
			"Ghost FTP could not reach "+brand.UpdateBaseURL+"\n\nCheck your connection and try again.",
		)
		return 1
	}
	if !result.Available {
		platform.InfoDialog(
			brand.ProductName+" Update",
			"You're up to date",
			"Ghost FTP "+result.CurrentVersion+" is the latest available version.",
		)
		return 0
	}

	if !platform.ConfirmDialog(
		brand.ProductName+" Update",
		"Install Ghost FTP "+result.LatestVersion+"?",
		"Current version: "+result.CurrentVersion+"\nLatest version: "+result.LatestVersion+"\n\nThe verified Setup package will be downloaded from update.ghostftp.com.",
	) {
		return 0
	}

	path, err := downloadInstaller(ctx, nil, result)
	if err != nil {
		platform.ErrorDialog(
			brand.ProductName+" Update",
			"Update download failed",
			err.Error(),
		)
		return 1
	}

	command := exec.Command(path)
	if err := command.Start(); err != nil {
		_ = os.Remove(path)
		platform.ErrorDialog(
			brand.ProductName+" Update",
			"Setup could not be started",
			"Close Ghost FTP and try again.\n\n"+err.Error(),
		)
		return 1
	}
	return 0
}

func main() {
	platform.HardenProcessPrivacy()
	os.Exit(run())
}
