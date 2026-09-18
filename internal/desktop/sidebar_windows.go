//go:build windows

package desktop

import (
	"strings"
	"sync"
	"unsafe"
)

const (
	applicationSidebarX           = 16
	applicationSidebarWidth       = 258
	applicationSidebarControlX    = 30
	applicationSidebarControlW    = 244
	applicationContentLeft        = 304
	applicationSidebarCardH       = 48
	applicationSidebarCardGap     = 8
	applicationSidebarUtilityH    = 38
	applicationSidebarUtilityGap  = 7
	applicationSidebarPrimaryTop  = 86
	applicationSidebarBrandIcon   = 54
	applicationSidebarBrandGap    = 8
	applicationSidebarBottomInset = 16
)

var (
	sidebarFiles           sync.Map
	sidebarTransfers       sync.Map
	sidebarQueue           sync.Map
	sidebarSync            sync.Map
	sidebarDiagnostics     sync.Map
	sidebarBookmarks       sync.Map
	sidebarGetWindowRect   = user32.NewProc("GetWindowRect")
	sidebarMapWindowPoints = user32.NewProc("MapWindowPoints")
	sidebarSetFocus        = user32.NewProc("SetFocus")
)

func (a *app) ensureSidebarButton(store *sync.Map, id int, label, icon string) uintptr {
	if a == nil || a.hwnd == 0 || store == nil {
		return 0
	}
	if value, ok := store.Load(a.hwnd); ok {
		if hwnd, ok := value.(uintptr); ok && hwnd != 0 {
			return hwnd
		}
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr("BUTTON"))),
		uintptr(unsafe.Pointer(wstr(label))),
		uintptr(wsChild|wsVisible|wsTabStop|bsOwnerDraw),
		0, 0, 1, 1,
		a.hwnd, uintptr(id), hinst, 0,
	)
	if hwnd == 0 {
		return 0
	}
	if a.font != 0 {
		sendMessageW.Call(hwnd, wmSetFont, a.font, 1)
	}
	applyDarkControl(hwnd, "BUTTON")
	a.registerButton(hwnd, icon, label, buttonSubtle)
	store.Store(a.hwnd, hwnd)
	return hwnd
}

func (a *app) ensureSidebarFiles() uintptr {
	labels := navigationLabelsForLanguage(a.languageCode())
	return a.ensureSidebarButton(&sidebarFiles, idFilesNav, labels.Files, iconOpenLocal)
}

func (a *app) ensureSidebarTransfers() uintptr {
	label := "Transfers"
	if a.languageCode() != "en" {
		label = navigationLabelsForLanguage(a.languageCode()).TransferQueue
	}
	return a.ensureSidebarButton(&sidebarTransfers, idTransferQueueNav, label, iconUpload)
}

func (a *app) ensureSidebarQueue() uintptr {
	label := "Queue"
	return a.ensureSidebarButton(&sidebarQueue, idQueueNav, label, iconSync)
}

func (a *app) ensureSidebarSync() uintptr {
	label := "Sync"
	if a.languageCode() == "hr" {
		label = "Sinkronizacija"
	}
	return a.ensureSidebarButton(&sidebarSync, idDirectoryCompare, label, iconSync)
}

func (a *app) ensureSidebarDiagnostics() uintptr {
	labels := navigationLabelsForLanguage(a.languageCode())
	return a.ensureSidebarButton(&sidebarDiagnostics, idDiagnostics, labels.Diagnostics, iconDiagnostics)
}

func (a *app) ensureSidebarBookmarks() uintptr {
	label := bookmarkWordsForLanguage(a.languageCode()).Title
	return a.ensureSidebarButton(&sidebarBookmarks, idBookmarks, label, iconOpenLocal)
}

func (a *app) ensureSidebarProfileControls() {
	if a == nil || a.hwnd == 0 {
		return
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	if a.sidebarMotto == 0 {
		motto, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(wstr("STATIC"))),
			uintptr(unsafe.Pointer(wstr("Move More\r\nDo More"))),
			uintptr(wsChild|wsVisible),
			0, 0, 1, 1,
			a.hwnd, 0, hinst, 0,
		)
		if motto != 0 && a.scriptFont != 0 {
			sendMessageW.Call(motto, wmSetFont, a.scriptFont, 1)
		}
		a.sidebarMotto = motto
	}
	if a.sidebarConnectionStatus == 0 {
		status, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(wstr("STATIC"))),
			uintptr(unsafe.Pointer(wstr("●  Disconnected"))),
			uintptr(wsChild|wsVisible),
			0, 0, 1, 1,
			a.hwnd, 0, hinst, 0,
		)
		if status != 0 && a.smallFont != 0 {
			sendMessageW.Call(status, wmSetFont, a.smallFont, 1)
		}
		a.sidebarConnectionStatus = status
	}
	if a.sidebarBookmarkHeading == 0 {
		heading, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(wstr("STATIC"))),
			uintptr(unsafe.Pointer(wstr("Bookmarks"))),
			uintptr(wsChild|wsVisible),
			0, 0, 1, 1,
			a.hwnd, 0, hinst, 0,
		)
		if heading != 0 && a.smallFont != 0 {
			sendMessageW.Call(heading, wmSetFont, a.smallFont, 1)
		}
		a.sidebarBookmarkHeading = heading
	}
	for len(a.sidebarProfileButtons) < maxSidebarProfiles {
		index := len(a.sidebarProfileButtons)
		id := idSidebarProfileBase + index
		hwnd, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(wstr("BUTTON"))),
			uintptr(unsafe.Pointer(wstr(""))),
			uintptr(wsChild|wsVisible|wsTabStop|bsOwnerDraw),
			0, 0, 1, 1,
			a.hwnd, uintptr(id), hinst, 0,
		)
		if hwnd == 0 {
			break
		}
		if a.font != 0 {
			sendMessageW.Call(hwnd, wmSetFont, a.font, 1)
		}
		applyDarkControl(hwnd, "BUTTON")
		a.registerButton(hwnd, iconOpenLocal, "", buttonNav)
		a.sidebarProfileButtons = append(a.sidebarProfileButtons, hwnd)
	}
}

func (a *app) refreshSidebarProfileControls() {
	if a == nil {
		return
	}
	a.ensureSidebarProfileControls()
	setText(a.sidebarBookmarkHeading, "Bookmarks")
	for index, hwnd := range a.sidebarProfileButtons {
		if index >= len(a.profiles) {
			showControls(false, hwnd)
			continue
		}
		profile := a.profiles[index]
		label := profile.Name
		if label == "" {
			label = profile.Host
		}
		variant := buttonNav
		if profile.ID != "" && profile.ID == a.selectedProfileID {
			variant = buttonNavActive
		}
		a.setSidebarButtonVisual(hwnd, iconOpenLocal, label, variant)
		setControlEnabled(hwnd, !a.connected && !a.connectionBusy && !a.profileMutationBusy)
		showControls(true, hwnd)
	}
}

func (a *app) selectSidebarProfile(index int) {
	if a == nil || index < 0 || index >= len(a.profiles) || index >= maxSidebarProfiles {
		return
	}
	if a.connected || a.connectionBusy || a.profileMutationBusy {
		a.openSiteManager()
		return
	}
	sendMessageW.Call(a.profilesCombo, cbSetCurSel, uintptr(index+1), 0)
	a.selectProfile()
	a.refreshSidebarProfileControls()
	a.setStatus("Selected site: " + a.profiles[index].Name)
}

func sidebarControl(store *sync.Map, owner uintptr) uintptr {
	if store == nil || owner == 0 {
		return 0
	}
	value, ok := store.Load(owner)
	if !ok {
		return 0
	}
	hwnd, _ := value.(uintptr)
	return hwnd
}

func (a *app) sidebarFilesButton() uintptr {
	if a == nil {
		return 0
	}
	return sidebarControl(&sidebarFiles, a.hwnd)
}

func (a *app) sidebarTransferButton() uintptr {
	if a == nil {
		return 0
	}
	return sidebarControl(&sidebarTransfers, a.hwnd)
}

func (a *app) sidebarQueueButton() uintptr {
	if a == nil {
		return 0
	}
	return sidebarControl(&sidebarQueue, a.hwnd)
}

func (a *app) sidebarSyncButton() uintptr {
	if a == nil {
		return 0
	}
	return sidebarControl(&sidebarSync, a.hwnd)
}

func (a *app) sidebarDiagnosticsButton() uintptr {
	if a == nil {
		return 0
	}
	return sidebarControl(&sidebarDiagnostics, a.hwnd)
}

func (a *app) sidebarBookmarksButton() uintptr {
	if a == nil {
		return 0
	}
	return sidebarControl(&sidebarBookmarks, a.hwnd)
}

func (a *app) setSidebarButtonVisual(hwnd uintptr, icon, label string, variant buttonVariant) {
	if a == nil || hwnd == 0 {
		return
	}
	setText(hwnd, label)
	a.registerButtonVisual(hwnd, icon, label, variant, false)
	invalidateRect.Call(hwnd, 0, 0)
}

func (a *app) updateSidebarTransferBadge() {
	if a == nil {
		return
	}
	running, queued := 0, 0
	for _, job := range a.transferJobs {
		switch job.Status {
		case "running":
			running++
		case "queued":
			queued++
		}
	}
	a.setButtonBadge(a.sidebarTransferButton(), running)
	a.setButtonBadge(a.sidebarQueueButton(), queued)
}

func (a *app) focusFilesWorkspace() {
	if a == nil {
		return
	}
	target := a.localList
	if target == 0 {
		target = a.localPath
	}
	if target != 0 {
		sidebarSetFocus.Call(target)
	}
}

func (a *app) focusTransferQueue() {
	if a == nil || a.transferList == 0 {
		return
	}
	sidebarSetFocus.Call(a.transferList)
}

func (a *app) refreshSidebarConnectionStatus() {
	if a == nil || a.sidebarConnectionStatus == 0 {
		return
	}
	label := "●  Disconnected"
	if a.connectionBusy {
		label = "●  Connecting…"
	} else if a.connected {
		host := strings.TrimSpace(getText(a.host))
		if host == "" {
			label = "●  Connected"
		} else {
			label = "●  Connected to " + host
		}
	}
	setText(a.sidebarConnectionStatus, label)
	invalidateRect.Call(a.sidebarConnectionStatus, 0, 0)
}

func (a *app) cleanupSidebarControls() {
	if a == nil || a.hwnd == 0 {
		return
	}
	for _, store := range []*sync.Map{&sidebarFiles, &sidebarTransfers, &sidebarQueue, &sidebarSync, &sidebarBookmarks, &sidebarDiagnostics} {
		if hwnd := sidebarControl(store, a.hwnd); hwnd != 0 {
			delete(a.buttons, hwnd)
		}
		store.Delete(a.hwnd)
	}
	for _, hwnd := range a.sidebarProfileButtons {
		delete(a.buttons, hwnd)
	}
	a.sidebarProfileButtons = nil
	a.sidebarBookmarkHeading = 0
	a.sidebarMotto = 0
	a.sidebarConnectionStatus = 0
}

func (a *app) sidebarLogicalRect(hwnd uintptr) (rect, bool) {
	if a == nil || a.hwnd == 0 || hwnd == 0 {
		return rect{}, false
	}
	var value rect
	if ok, _, _ := sidebarGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&value))); ok == 0 {
		return rect{}, false
	}
	// A RECT is two POINT structures, so MapWindowPoints converts both corners
	// from screen coordinates to the main client coordinate system in one call.
	sidebarMapWindowPoints.Call(0, a.hwnd, uintptr(unsafe.Pointer(&value)), 2)
	return rect{
		Left:   int32(a.unscale(int(value.Left))),
		Top:    int32(a.unscale(int(value.Top))),
		Right:  int32(a.unscale(int(value.Right))),
		Bottom: int32(a.unscale(int(value.Bottom))),
	}, true
}

func (a *app) transformSidebarContent(hwnd uintptr, oldLeft, oldRight, newLeft, newRight int) {
	if hwnd == 0 || oldRight <= oldLeft || newRight <= newLeft {
		return
	}
	r, ok := a.sidebarLogicalRect(hwnd)
	if !ok {
		return
	}
	x := int(r.Left)
	w := int(r.Right - r.Left)
	oldSpan := oldRight - oldLeft
	newSpan := newRight - newLeft
	nextX := newLeft + (x-oldLeft)*newSpan/oldSpan
	nextW := w * newSpan / oldSpan
	if nextW < 1 {
		nextW = 1
	}
	a.move(hwnd, nextX, int(r.Top), nextW, int(r.Bottom-r.Top))
}

func (a *app) resizeSidebarColumns() {
	// File panes have one canonical width policy. The old sidebar-specific
	// percentages overrode resizeListColumns and made Permissions too narrow.
	// Fit from each ListView's actual client width so sidebar and non-sidebar
	// passes cannot disagree about the same columns.
	a.fitFileColumnsToWorkspace()

	if a.transferList != 0 {
		var client rect
		if ok, _, _ := getClientRect.Call(a.transferList, uintptr(unsafe.Pointer(&client))); ok != 0 {
			width := int(client.Right - client.Left)
			if width >= a.scale(360) {
				parts := []int{20, 11, 16, 16, 12, 15, 10}
				for index, percent := range parts {
					sendMessageW.Call(a.transferList, lvmSetColumnWidth, uintptr(index), uintptr(width*percent/100))
				}
			}
		}
	}
}

func (a *app) layoutSidebarRail(height int) {
	files := a.ensureSidebarFiles()
	transfers := a.ensureSidebarTransfers()
	queue := a.ensureSidebarQueue()
	syncButton := a.ensureSidebarSync()
	diagnostics := a.ensureSidebarDiagnostics()
	bookmarks := a.ensureSidebarBookmarks()
	labels := navigationLabelsForLanguage(a.languageCode())
	a.ensureSidebarProfileControls()

	sitesLabel := labels.Connections
	if a.languageCode() == "en" {
		sitesLabel = "Sites"
	}
	a.setSidebarButtonVisual(a.siteManagerBtn, iconConnect, sitesLabel, buttonNavActive)
	showControls(false, files)
	a.setSidebarButtonVisual(transfers, iconUpload, "Transfers", buttonNav)
	a.setSidebarButtonVisual(queue, iconSync, "Queue", buttonNav)
	syncLabel := "Sync"
	if a.languageCode() == "hr" {
		syncLabel = "Sinkronizacija"
	}
	a.setSidebarButtonVisual(syncButton, iconSync, syncLabel, buttonNav)
	a.setSidebarButtonVisual(a.settingsBtn, iconSettings, a.tr("common.settings"), buttonNav)
	showControls(false, bookmarks, diagnostics, a.aboutBtn)
	a.updateSidebarTransferBadge()

	logo := a.ensureBrandLogo()
	if logo != 0 {
		a.move(logo, applicationSidebarControlX+2, 15, applicationSidebarBrandIcon, applicationSidebarBrandIcon)
	}
	if a.titleFont != 0 {
		sendMessageW.Call(a.brandTitle, wmSetFont, a.titleFont, 1)
		sendMessageW.Call(a.brandFTP, wmSetFont, a.titleFont, 1)
	}
	titleX := applicationSidebarControlX + applicationSidebarBrandIcon + applicationSidebarBrandGap + 2
	const ghostWordWidth = 72
	a.move(a.brandTitle, titleX, 20, ghostWordWidth, 36)
	a.move(a.brandFTP, titleX+ghostWordWidth+4, 20, 54, 36)

	y := applicationSidebarPrimaryTop
	for _, control := range []uintptr{a.siteManagerBtn, transfers, queue, syncButton, a.settingsBtn} {
		a.move(control, applicationSidebarControlX, y, applicationSidebarControlW, 44)
		y += 52
	}

	bookmarkY := y + 12
	if a.sidebarBookmarkHeading != 0 {
		a.move(a.sidebarBookmarkHeading, applicationSidebarControlX+10, bookmarkY, applicationSidebarControlW-20, 22)
		showControls(true, a.sidebarBookmarkHeading)
	}
	profileY := bookmarkY + 28
	a.refreshSidebarProfileControls()
	for index, hwnd := range a.sidebarProfileButtons {
		if hwnd == 0 || index >= len(a.profiles) {
			continue
		}
		a.move(hwnd, applicationSidebarControlX, profileY, applicationSidebarControlW, 36)
		profileY += 42
	}

	setText(a.brandSubtitle, "SECURE TRANSFERS.\r\nWITHOUT A TRACE.")
	if a.smallFont != 0 {
		sendMessageW.Call(a.brandSubtitle, wmSetFont, a.smallFont, 1)
	}
	connectionY := height - applicationSidebarBottomInset - 28
	footerY := connectionY - 68
	mottoY := footerY - 112
	if a.sidebarMotto != 0 && mottoY >= profileY+8 {
		if a.scriptFont != 0 {
			sendMessageW.Call(a.sidebarMotto, wmSetFont, a.scriptFont, 1)
		}
		a.move(a.sidebarMotto, applicationSidebarControlX+12, mottoY, applicationSidebarControlW-24, 92)
		showControls(true, a.sidebarMotto)
	} else {
		showControls(false, a.sidebarMotto)
	}
	if footerY < profileY+10 {
		footerY = profileY + 10
	}
	a.move(a.brandSubtitle, applicationSidebarControlX+10, footerY, applicationSidebarControlW-20, 48)
	showControls(true, a.brandSubtitle)
	if a.sidebarConnectionStatus != 0 {
		a.move(a.sidebarConnectionStatus, applicationSidebarControlX+10, connectionY, applicationSidebarControlW-20, 24)
		showControls(true, a.sidebarConnectionStatus)
		a.refreshSidebarConnectionStatus()
	}
	showControls(false, a.languageCombo)
}

// applyApplicationSidebar turns application-level navigation into one canonical
// left rail without rewriting the proven two-pane layout engine. The baseline
// layout computes the operational workspace first, then this function applies
// one affine horizontal transform and anchors the rail. State-only refreshes
// detect an already-transformed profile control and never compound the transform.
func (a *app) applyApplicationSidebar() {
	if a == nil || a.hwnd == 0 {
		return
	}

	var client rect
	if ok, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		return
	}
	width := a.unscale(int(client.Right - client.Left))
	height := a.unscale(int(client.Bottom - client.Top))
	a.layoutSidebarRail(height)

	// If the profile combo is already to the right of the rail then this layout
	// pass was transformed previously. Do not scale a transformed workspace twice.
	profileRect, ok := a.sidebarLogicalRect(a.profilesCombo)
	if ok && int(profileRect.Left) >= applicationContentLeft-2 {
		a.resizeSidebarColumns()
		return
	}

	oldLeft := premiumOuterGap
	oldRight := width - premiumOuterGap
	newLeft := applicationContentLeft
	newRight := oldRight
	if newRight-newLeft < 520 {
		return
	}

	for _, control := range []uintptr{
		a.protocol, a.host, a.port, a.user, a.pass, a.connect, a.disconnect,
		a.keyPath, a.chooseKey, a.passphrase,
		a.sectionLocal, a.sectionRemote,
		a.localPath, a.localUp, a.localChoose, a.localRefresh, a.localMkdir, a.localRename, a.localDelete, a.localList,
		a.remotePath, a.remoteUp, a.remoteRefresh, a.remoteMkdir, a.remoteRename, a.remoteDelete, a.remoteChmod, a.remoteList,
		a.upload, a.download,
		a.sectionTransfers, a.transferSummary, a.pauseQueue, a.resumeQueue, a.cancelJob, a.retryJob, a.clearQueue, a.transferList,
		a.status, a.statusVersion,
	} {
		a.transformSidebarContent(control, oldLeft, oldRight, newLeft, newRight)
	}

	// The saved-connection picker becomes the first content-row control. Keep the
	// live connection state visible in the same row and reserve independent bounds
	// for profile persistence actions; the previous layout left connectionBadge in
	// its legacy header position where it overlapped Delete profile.
	availableProfile := newRight - newLeft
	badgeW, buttonW, gap := 118, 116, 8
	profileW := availableProfile - badgeW - 2*buttonW - 3*gap
	if profileW < 220 {
		profileW = 220
	}
	a.move(a.profilesCombo, newLeft, 13, profileW, 31)
	badgeX := newLeft + profileW + gap
	a.move(a.connectionBadge, badgeX, 18, badgeW, 21)
	saveX := badgeX + badgeW + gap
	a.move(a.saveProfile, saveX, 13, buttonW, 31)
	a.move(a.removeProfile, saveX+buttonW+gap, 13, buttonW, 31)
	a.resizeSidebarColumns()
}
