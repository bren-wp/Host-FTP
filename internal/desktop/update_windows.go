//go:build windows

package desktop

import (
	"context"
	"fmt"
	"time"

	"github.com/bren-wp/Host-FTP/internal/brand"
	"github.com/bren-wp/Host-FTP/internal/external"
	"github.com/bren-wp/Host-FTP/internal/platform"
	"github.com/bren-wp/Host-FTP/internal/updatecheck"
)

func (a *app) checkForUpdates() {
	if a == nil || a.connectionBusy || a.profileMutationBusy {
		return
	}
	a.setStatus("Checking update.ghostftp.com…")
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		result, err := updatecheck.Check(ctx, a.version)
		a.dispatch(func() {
			if err != nil {
				a.setStatus("Update check failed.")
				platform.ErrorDialog(
					brand.ProductName,
					"Unable to check for updates",
					"Ghost FTP could not reach the official update service. Check your connection and try again.\n\nUpdate service: "+brand.UpdateBaseURL,
				)
				return
			}
			if !result.Available {
				a.setStatus("Ghost FTP "+result.CurrentVersion+" is up to date.")
				platform.InfoDialog(
					brand.ProductName,
					"You're up to date",
					fmt.Sprintf("Ghost FTP %s is the latest available version.\n\nUpdates are delivered from %s", result.CurrentVersion, brand.UpdateBaseURL),
				)
				return
			}

			a.setStatus("Ghost FTP "+result.LatestVersion+" is available.")
			platform.InfoDialog(
				brand.ProductName,
				"Update available",
				fmt.Sprintf(
					"Ghost FTP %s is available. You are currently running %s.\n\nUse Download latest to get the verified Windows package from %s",
					result.LatestVersion,
					result.CurrentVersion,
					brand.UpdateBaseURL,
				),
			)
		})
	})
}

func (a *app) openUpdateDownload() {
	if a == nil {
		return
	}
	if err := external.OpenUpdatePage(); err != nil {
		platform.ErrorDialog(brand.ProductName, "Unable to open updates", "Open "+brand.UpdateURL+" in your browser.")
		return
	}
	a.setStatus("Official Ghost FTP update service opened in your browser.")
}

func (a *app) openPremiumDownload() {
	if a == nil {
		return
	}
	if err := external.OpenPremiumPage(); err != nil {
		platform.ErrorDialog(brand.ProductName, "Unable to open Premium", "Open "+brand.PremiumURL+" in your browser.")
		return
	}
	a.setStatus("Premium download page opened in your browser.")
}

func (a *app) openOfficialWebsite() {
	if a == nil {
		return
	}
	if err := external.OpenWebsite(); err != nil {
		platform.ErrorDialog(brand.ProductName, "Unable to open website", "Open "+brand.WebsiteURL+" in your browser.")
		return
	}
	a.setStatus("Official Ghost FTP website opened in your browser.")
}
