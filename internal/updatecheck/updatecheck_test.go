package updatecheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckURLDetectsNewVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"schema":1,"version":"0.3.1","setup_url":"https://update.ghostftp.com/windows/Ghost-FTP-0.3.1-Setup.exe","setup_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","portable_url":"https://update.ghostftp.com/windows/Ghost-FTP-0.3.1-Portable.exe","portable_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","updater_url":"https://update.ghostftp.com/windows/Ghost-FTP-0.3.1-Update.exe","updater_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}`))
	}))
	defer server.Close()

	result, err := checkURL(context.Background(), server.Client(), server.URL, "0.3.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Available || result.LatestVersion != "0.3.1" || result.CurrentVersion != "0.3.0" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestCheckURLReportsCurrentVersionAsUpToDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"schema":1,"version":"0.3.0","setup_url":"https://update.ghostftp.com/windows/setup.exe","setup_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","portable_url":"https://update.ghostftp.com/windows/portable.exe","portable_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}`))
	}))
	defer server.Close()

	result, err := checkURL(context.Background(), server.Client(), server.URL, "0.3.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Available {
		t.Fatalf("unexpected update: %#v", result)
	}
}

func TestTrustedURLAcceptsOnlyDedicatedUpdateHost(t *testing.T) {
	for _, value := range []string{
		"https://update.ghostftp.com/",
		"https://update.ghostftp.com/windows/latest.json",
	} {
		if !TrustedURL(value) {
			t.Fatalf("expected trusted URL: %s", value)
		}
	}
	for _, value := range []string{
		"http://update.ghostftp.com/windows/latest.json",
		"https://ghostftp.com/download",
		"https://update.ghostftp.com.evil.example/file.exe",
		"https://user@update.ghostftp.com/file.exe",
	} {
		if TrustedURL(value) {
			t.Fatalf("unexpected trusted URL: %s", value)
		}
	}
}

func TestCheckRejectsMalformedVersion(t *testing.T) {
	for _, value := range []string{"", "0.8", "v0.0.8", "00.0.8", "0.00.8", "0.0.-1", "+0.0.8", "0.+0.8", "0.0.+8", "0.０.8", "0.0.8.1"} {
		if _, err := parseVersion(value); err == nil {
			t.Fatalf("expected invalid version: %q", value)
		}
	}
}
