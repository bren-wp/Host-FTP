//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bren-wp/Host-FTP/internal/brand"
	"github.com/bren-wp/Host-FTP/internal/platform"
)

func registerIntegratedUninstall(appPath, currentVersion string) error {
	digest, err := platform.VerifiedRegularFileSHA256(appPath)
	if err != nil {
		return fmt.Errorf("installed executable ownership digest could not be verified: %w", err)
	}

	quoted := fmt.Sprintf("\"%s\" --uninstall", appPath)
	info, err := os.Stat(appPath)
	if err != nil {
		return fmt.Errorf("installed executable size could not be read: %w", err)
	}
	estimatedKiB := uint32((info.Size() + 1023) / 1024)
	installDate := time.Now().Format("20060102")

	values := []struct {
		name  string
		value string
	}{
		{"DisplayName", brand.ProductFull},
		{"DisplayVersion", currentVersion},
		{"Publisher", brand.Company},
		{"InstallLocation", filepath.Dir(appPath)},
		{"DisplayIcon", appPath + ",0"},
		{"UninstallString", quoted},
		{installedExecutableDigestValue, digest},
		{"URLInfoAbout", brand.Website},
		{"InstallDate", installDate},
	}
	for _, item := range values {
		if err := platform.SetRegistryString(uninstallKey, item.name, item.value); err != nil {
			return err
		}
	}

	// The integrated uninstaller is intentionally interactive: it requires
	// confirmation and reports completion to the user. Do not advertise that
	// same command as QuietUninstallString. Remove the stale value left by
	// earlier installers; the surrounding registry snapshot restores it if the
	// installation transaction later rolls back.
	if err := platform.DeleteRegistryValue(uninstallKey, "QuietUninstallString"); err != nil {
		return err
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "EstimatedSize", estimatedKiB); err != nil {
		return err
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoModify", 1); err != nil {
		return err
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoRepair", 1); err != nil {
		return err
	}
	if err := registerBrowserProtocol(appPath); err != nil {
		return fmt.Errorf("browser launch protocol registration failed: %w", err)
	}
	return nil
}
