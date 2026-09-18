//go:build windows

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegratedUninstallCommandUsesInstalledApplication(t *testing.T) {
	appPath := filepath.Join(`C:\Users\Test\AppData\Local\Ghost FTP`, "GhostFTP.exe")
	command := integratedUninstallCommand(appPath)
	if !strings.Contains(command, "GhostFTP.exe") || !strings.Contains(command, "--uninstall") {
		t.Fatalf("unexpected uninstall command: %q", command)
	}
	if strings.Contains(strings.ToLower(command), "uninstall.exe") {
		t.Fatalf("separate uninstaller unexpectedly referenced: %q", command)
	}
}
