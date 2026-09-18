package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bren-wp/Host-FTP/internal/model"
)

func newBundleTestEngine(t *testing.T) *Engine {
	t.Helper()
	dir := t.TempDir()
	engine, err := New(dir, filepath.Join(dir, "GhostFTP.exe"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(engine.Close)
	return engine
}

func TestProfileBundleNeverExportsSecrets(t *testing.T) {
	engine := newBundleTestEngine(t)
	_, err := engine.SaveProfile(model.ProfileInput{
		Name:       "Primary Site",
		Protocol:   "ftp",
		Host:       "files.acme.test",
		Port:       21,
		Username:   "deploy",
		Password:   "super-secret-password",
		RemotePath: "/public",
	})
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "sites.ghostftp.json")
	count, err := engine.ExportProfiles(path)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("exported %d profiles, want 1", count)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(payload)
	if strings.Contains(text, "super-secret-password") || strings.Contains(strings.ToLower(text), "passwordblob") {
		t.Fatal("profile bundle exposed a saved secret")
	}
	if !strings.Contains(text, ""product": "Ghost FTP"") {
		t.Fatal("profile bundle product identity is missing")
	}
}

func TestProfileBundleImportIsSecretFreeAndIdempotent(t *testing.T) {
	source := newBundleTestEngine(t)
	_, err := source.SaveProfile(model.ProfileInput{
		Name:       "SFTP Site",
		Protocol:   "sftp",
		Host:       "sftp.acme.test",
		Port:       22,
		Username:   "operator",
		Password:   "do-not-export",
		RemotePath: "/srv/www",
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sites.ghostftp.json")
	if _, err := source.ExportProfiles(path); err != nil {
		t.Fatal(err)
	}

	target := newBundleTestEngine(t)
	result, err := target.ImportProfiles(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 1 || result.Skipped != 0 {
		t.Fatalf("first import = %#v, want 1 imported / 0 skipped", result)
	}
	profiles, err := target.Profiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatalf("profiles = %d, want 1", len(profiles))
	}
	if profiles[0].HasPassword || profiles[0].HasPassphrase {
		t.Fatal("imported metadata-only profile unexpectedly contains a secret")
	}

	result, err = target.ImportProfiles(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 0 || result.Skipped != 1 {
		t.Fatalf("second import = %#v, want 0 imported / 1 skipped", result)
	}
}
