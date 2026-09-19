//go:build windows

package desktop

import (
	"path/filepath"
	"strings"
	"unsafe"
)

type workspaceHistoryEntry struct {
	Remote bool
	Path   string
}

const workspaceHistoryLimit = 64

func normalizeWorkspaceHistoryPath(remote bool, value string) string {
	value = strings.TrimSpace(value)
	if remote {
		if value == "" {
			return "/"
		}
		return cleanRemote(value)
	}
	if value == "" {
		return ""
	}
	return filepath.Clean(value)
}

func sameWorkspaceHistoryEntry(left, right workspaceHistoryEntry) bool {
	return left.Remote == right.Remote &&
		normalizeWorkspaceHistoryPath(left.Remote, left.Path) ==
			normalizeWorkspaceHistoryPath(right.Remote, right.Path)
}

func appendWorkspaceHistory(history []workspaceHistoryEntry, entry workspaceHistoryEntry) []workspaceHistoryEntry {
	entry.Path = normalizeWorkspaceHistoryPath(entry.Remote, entry.Path)
	if entry.Path == "" {
		return history
	}
	if len(history) > 0 && sameWorkspaceHistoryEntry(history[len(history)-1], entry) {
		return history
	}
	history = append(history, entry)
	if len(history) > workspaceHistoryLimit {
		copy(history, history[len(history)-workspaceHistoryLimit:])
		history = history[:workspaceHistoryLimit]
	}
	return history
}

func (a *app) recordWorkspaceNavigation(remote bool, previous, next string) {
	if a == nil {
		return
	}
	oldEntry := workspaceHistoryEntry{Remote: remote, Path: normalizeWorkspaceHistoryPath(remote, previous)}
	newEntry := workspaceHistoryEntry{Remote: remote, Path: normalizeWorkspaceHistoryPath(remote, next)}
	if oldEntry.Path == "" || sameWorkspaceHistoryEntry(oldEntry, newEntry) {
		if a.workspaceReplayActive && sameWorkspaceHistoryEntry(a.workspaceReplayTarget, newEntry) {
			a.workspaceReplayActive = false
		}
		a.updateMasterToolbarState()
		return
	}
	if a.workspaceReplayActive && sameWorkspaceHistoryEntry(a.workspaceReplayTarget, newEntry) {
		a.workspaceReplayActive = false
		a.updateMasterToolbarState()
		return
	}
	a.workspaceBackHistory = appendWorkspaceHistory(a.workspaceBackHistory, oldEntry)
	a.workspaceForwardHistory = nil
	a.updateMasterToolbarState()
}

func (a *app) failWorkspaceReplay(remote bool, target string) {
	if a == nil || !a.workspaceReplayActive {
		return
	}
	entry := workspaceHistoryEntry{Remote: remote, Path: normalizeWorkspaceHistoryPath(remote, target)}
	if !sameWorkspaceHistoryEntry(a.workspaceReplayTarget, entry) {
		return
	}
	a.workspaceBackHistory = appendWorkspaceHistory(a.workspaceBackHistory, entry)
	if len(a.workspaceForwardHistory) > 0 {
		a.workspaceForwardHistory = a.workspaceForwardHistory[:len(a.workspaceForwardHistory)-1]
	}
	a.workspaceReplayActive = false
	a.updateMasterToolbarState()
}

func (a *app) currentWorkspaceHistoryEntry(remote bool) workspaceHistoryEntry {
	if remote {
		return workspaceHistoryEntry{Remote: true, Path: normalizeWorkspaceHistoryPath(true, a.remoteCurrent)}
	}
	return workspaceHistoryEntry{Path: normalizeWorkspaceHistoryPath(false, a.localCurrent)}
}

func (a *app) navigateWorkspaceHistory(back bool) {
	if a == nil || a.workspaceReplayActive {
		return
	}
	var source, destination *[]workspaceHistoryEntry
	if back {
		source = &a.workspaceBackHistory
		destination = &a.workspaceForwardHistory
	} else {
		source = &a.workspaceForwardHistory
		destination = &a.workspaceBackHistory
	}
	if len(*source) == 0 {
		return
	}
	target := (*source)[len(*source)-1]
	*source = (*source)[:len(*source)-1]
	current := a.currentWorkspaceHistoryEntry(target.Remote)
	if current.Path != "" {
		*destination = appendWorkspaceHistory(*destination, current)
	}
	a.workspaceReplayActive = true
	a.workspaceReplayTarget = target
	a.updateMasterToolbarState()
	if target.Remote {
		if !a.connected || a.connectionBusy {
			a.failWorkspaceReplay(true, target.Path)
			return
		}
		a.lastFilePaneRemote = true
		a.refreshRemote(target.Path)
		return
	}
	a.lastFilePaneRemote = false
	a.refreshLocal(target.Path)
}

func (a *app) masterNewFolderAction() {
	if a == nil {
		return
	}
	if a.lastFilePaneRemote && a.connected && !a.connectionBusy && !a.remoteMutationBusy {
		a.remoteMkdirAction()
		return
	}
	a.localMkdirAction()
}

func (a *app) masterMoreAction() {
	if a == nil {
		return
	}
	// The approved master workspace exposes search directly in the action bar.
	// Reuse Ghost FTP's real per-pane filter rather than introducing a decorative
	// search field that cannot affect the file model.
	if a.lastFilePaneRemote && a.connected && !a.connectionBusy {
		a.openFileFilter(a.remoteList)
		return
	}
	a.openFileFilter(a.localList)
}

func (a *app) masterConnectAction() {
	if a == nil {
		return
	}
	if a.connectionBusy || a.connected {
		a.disconnectNow()
		return
	}
	a.openSiteManager()
}

func (a *app) ensureMasterWorkspaceButton(hinst uintptr, id int, label, icon string, variant buttonVariant) uintptr {
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
	return a.registerButton(hwnd, icon, label, variant)
}

func (a *app) ensureMasterWorkspaceControls() {
	if a == nil || a.hwnd == 0 {
		return
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	if a.masterBack == 0 {
		a.masterBack = a.ensureMasterWorkspaceButton(hinst, idWorkspaceBack, "Back", iconBack, buttonSubtle)
		a.masterForward = a.ensureMasterWorkspaceButton(hinst, idWorkspaceForward, "Forward", iconForward, buttonSubtle)
		a.remoteBack = a.ensureMasterWorkspaceButton(hinst, idWorkspaceBack, "Back", iconBack, buttonSubtle)
		a.remoteForward = a.ensureMasterWorkspaceButton(hinst, idWorkspaceForward, "Forward", iconForward, buttonSubtle)
		a.masterRefresh = a.ensureMasterWorkspaceButton(hinst, idRefreshAll, a.tr("common.refresh"), iconRefresh, buttonSubtle)
		a.masterNewFolder = a.ensureMasterWorkspaceButton(hinst, idWorkspaceNewFolder, a.tr("common.new_folder"), iconNewFolder, buttonDefault)
		a.masterBookmarks = a.ensureMasterWorkspaceButton(hinst, idBookmarks, bookmarkWordsForLanguage(a.languageCode()).Title, iconOpenLocal, buttonDefault)
		a.masterMore = a.ensureMasterWorkspaceButton(hinst, idWorkspaceMore, "Search files, folders, or sites…    Ctrl+K", iconSearch, buttonSubtle)
		a.queueTabAll = a.ensureMasterWorkspaceButton(hinst, idTransferTabAll, "All (0)", "", buttonNavActive)
		a.queueTabUploading = a.ensureMasterWorkspaceButton(hinst, idTransferTabUploading, "Uploading (0)", "", buttonNav)
		a.queueTabDownloading = a.ensureMasterWorkspaceButton(hinst, idTransferTabDownload, "Downloading (0)", "", buttonNav)
		a.queueTabCompleted = a.ensureMasterWorkspaceButton(hinst, idTransferTabCompleted, "Completed (0)", "", buttonNav)
	}
	a.setButtonLabel(a.masterRefresh, a.tr("common.refresh"))
	a.setButtonLabel(a.masterNewFolder, a.tr("common.new_folder"))
	a.setButtonLabel(a.masterBookmarks, bookmarkWordsForLanguage(a.languageCode()).Title)
	a.setButtonLabel(a.masterMore, "Search files, folders, or sites…    Ctrl+K")
	a.registerButtonVisual(a.masterMore, iconSearch, "Search files, folders, or sites…    Ctrl+K", buttonSubtle, false)
	if a.transferViewFilter == "" {
		a.transferViewFilter = "all"
	}
	a.updateTransferTabLabels()
	a.updateMasterToolbarState()
}

func (a *app) updateMasterConnectVisual() {
	if a == nil || a.connect == 0 {
		return
	}
	label := a.tr("common.connect")
	icon := iconConnect
	variant := buttonAccent
	if a.connectionBusy {
		label = a.tr("common.cancel")
		icon = iconCancel
		variant = buttonDanger
	}
	a.setButtonLabel(a.connect, label)
	a.registerButtonVisual(a.connect, icon, label, variant, false)
	setControlEnabled(a.connect, !a.profileMutationBusy && !a.connected)
	if a.disconnect != 0 {
		disconnectLabel := a.tr("common.disconnect")
		a.setButtonLabel(a.disconnect, disconnectLabel)
		a.registerButtonVisual(a.disconnect, iconDisconnect, disconnectLabel, buttonSubtle, false)
		setControlEnabled(a.disconnect, a.connected || a.connectionBusy)
		invalidateRect.Call(a.disconnect, 0, 0)
	}
	invalidateRect.Call(a.connect, 0, 0)
}

func (a *app) updateMasterToolbarState() {
	if a == nil {
		return
	}
	historyBack := len(a.workspaceBackHistory) > 0 && !a.workspaceReplayActive
	historyForward := len(a.workspaceForwardHistory) > 0 && !a.workspaceReplayActive
	setControlEnabled(a.masterBack, historyBack)
	setControlEnabled(a.masterForward, historyForward)
	setControlEnabled(a.remoteBack, historyBack)
	setControlEnabled(a.remoteForward, historyForward)
	setControlEnabled(a.masterRefresh, !a.closing)
	setControlEnabled(a.masterNewFolder, !a.closing && !a.localMutationBusy &&
		(!a.lastFilePaneRemote || (a.connected && !a.connectionBusy && !a.remoteMutationBusy)))
	setControlEnabled(a.masterBookmarks, !a.closing)
	setControlEnabled(a.masterMore, !a.closing)
	a.updateMasterConnectVisual()
}

func (a *app) layoutMasterWorkspaceChrome() {
	if a == nil || a.hwnd == 0 {
		return
	}
	a.ensureMasterWorkspaceControls()
	filters := a.fileFilterState()

	var client rect
	if ok, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		return
	}
	width := a.unscale(int(client.Right - client.Left))
	height := a.unscale(int(client.Bottom - client.Top))
	contentLeft := applicationContentLeft
	contentRight := width - premiumOuterGap
	contentWidth := contentRight - contentLeft
	a.layoutWindowChrome(width)
	if contentWidth < 620 {
		return
	}

	// The file workspace follows the approved reference: connection editing stays
	// in Connections, while this screen is dedicated to navigation and transfers.
	showControls(false,
		a.profilesCombo,
		a.protocol, a.host, a.port, a.user, a.pass,
		a.keyPath, a.chooseKey, a.passphrase,
		a.saveProfile, a.removeProfile,
		a.localChoose, a.localRefresh, a.remoteRefresh,
		a.localMkdir, a.localRename, a.localDelete,
		a.remoteMkdir, a.remoteRename, a.remoteDelete, remoteEditButton(a), a.remoteChmod,
	)
	showControls(true,
		a.connectionBadge, a.connect, a.disconnect,
		a.masterBack, a.masterForward, a.remoteBack, a.remoteForward,
		a.localUp, a.remoteUp, a.localPath, a.remotePath,
		a.localList, a.remoteList, a.transferList,
		a.queueTabAll, a.queueTabUploading, a.queueTabDownloading, a.queueTabCompleted,
	)

	if a.languageCode() == "en" {
		setText(a.sectionLocal, "Local Files")
		setText(a.sectionRemote, "Remote Files")
		setText(a.sectionTransfers, "Transfer Queue")
	}

	// Global search / command field.
	searchY, searchH := 14, 48
	badgeW := 224
	searchW := contentWidth - badgeW - 18
	if searchW > 640 {
		searchW = 640
	}
	if searchW < 340 {
		searchW = 340
	}
	a.move(a.masterMore, contentLeft, searchY, searchW, searchH)
	a.move(a.connectionBadge, contentRight-badgeW, searchY+3, badgeW, 34)

	// Primary command row. At desktop widths the right side becomes the remote
	// file search field exactly where it appears in the approved layout.
	toolbarY, toolbarH := 80, 48
	gap := 8
	connectW, disconnectW, folderW, uploadW, downloadW, refreshW := 146, 140, 132, 110, 120, 108
	if contentWidth < 1050 {
		gap = 6
		connectW, disconnectW, folderW = 108, 108, 112
		uploadW, downloadW, refreshW = 88, 92, 84
	}
	actions := []struct {
		control uintptr
		width   int
	}{
		{a.connect, connectW},
		{a.disconnect, disconnectW},
		{a.masterNewFolder, folderW},
		{a.upload, uploadW},
		{a.download, downloadW},
		{a.masterRefresh, refreshW},
	}
	x := contentLeft
	for _, item := range actions {
		a.move(item.control, x, toolbarY, item.width, toolbarH)
		x += item.width + gap
	}

	if filters != nil {
		showControls(false, filters.localButton)
		remoteSearchW := contentRight - x
		if remoteSearchW >= 150 {
			label := "Search remote files…"
			a.setButtonLabel(filters.remoteButton, label)
			a.registerButtonVisual(filters.remoteButton, iconSearch, label, buttonSubtle, false)
			a.move(filters.remoteButton, x, toolbarY, remoteSearchW, toolbarH)
			showControls(true, filters.remoteButton)
		} else {
			showControls(false, filters.remoteButton)
		}
	}

	paneGap := 14
	paneW := (contentWidth - paneGap) / 2
	leftX := contentLeft
	rightX := leftX + paneW + paneGap
	sectionY, pathY := 156, 191
	a.move(a.sectionLocal, leftX+12, sectionY, paneW-24, 26)
	a.move(a.sectionRemote, rightX+12, sectionY, paneW-24, 26)

	// Breadcrumb-style local path row.
	miniW, miniGap := 40, 6
	lx := leftX + 8
	for _, control := range []uintptr{a.masterBack, a.masterForward, a.localUp} {
		a.move(control, lx, pathY, miniW, 38)
		lx += miniW + miniGap
	}
	localPathW := leftX + paneW - 8 - lx
	if localPathW < 120 {
		localPathW = 120
	}
	a.move(a.localPath, lx, pathY, localPathW, 38)

	// Independent remote visual navigation controls use the same maintained
	// cross-pane history stack, so both reference arrows remain functional.
	rx := rightX + 8
	for _, control := range []uintptr{a.remoteBack, a.remoteForward, a.remoteUp} {
		a.move(control, rx, pathY, miniW, 38)
		rx += miniW + miniGap
	}
	remotePathW := rightX + paneW - 8 - rx
	if remotePathW < 120 {
		remotePathW = 120
	}
	a.move(a.remotePath, rx, pathY, remotePathW, 38)

	statusY, _ := statusBandGeometry(height)
	queueH := clampInt(height/6, 132, 152)
	queueY := statusY - queueH - 10
	queueToolbarY := queueY - 38
	queueLabelY := queueToolbarY - 25

	listY := pathY + 46
	listBottom := queueLabelY - 12
	listH := listBottom - listY
	if listH < 150 {
		listH = 150
	}
	a.move(a.localList, leftX+8, listY, paneW-16, listH)
	a.move(a.remoteList, rightX+8, listY, paneW-16, listH)

	// Persistent transfer queue. Filter tabs mirror the approved All /
	// Uploading / Downloading / Completed row and filter the real queue model.
	a.move(a.sectionTransfers, contentLeft+12, queueLabelY-2, 180, 26)
	tabY := queueLabelY - 5
	tabX := contentLeft + 194
	tabGap := 4
	tabWidths := []int{82, 112, 128, 118}
	for index, control := range []uintptr{a.queueTabAll, a.queueTabUploading, a.queueTabDownloading, a.queueTabCompleted} {
		a.move(control, tabX, tabY, tabWidths[index], 30)
		tabX += tabWidths[index] + tabGap
	}
	summaryX := tabX + 8
	summaryW := contentRight - summaryX - 8
	if summaryW > 360 {
		summaryW = 360
	}
	if summaryW < 80 {
		summaryW = 80
	}
	a.move(a.transferSummary, summaryX, queueLabelY+1, summaryW, 22)

	queueControls := []struct {
		control uintptr
		width   int
	}{
		{a.pauseQueue, 94},
		{a.resumeQueue, 94},
		{a.cancelJob, 90},
		{a.retryJob, 86},
		{a.clearQueue, 156},
	}
	qx := contentRight - 8
	for i := len(queueControls) - 1; i >= 0; i-- {
		qx -= queueControls[i].width
		a.move(queueControls[i].control, qx, queueToolbarY, queueControls[i].width, 31)
		qx -= 6
	}
	a.move(a.transferList, contentLeft+8, queueY, contentWidth-16, queueH)
	a.move(a.status, contentLeft+12, statusY, contentWidth-318, statusBandHeight)
	a.move(a.statusVersion, contentRight-296, statusY, 288, statusBandHeight)

	a.updateMasterToolbarState()
}

func (a *app) cleanupMasterWorkspaceControls() {
	if a == nil {
		return
	}
	for _, control := range []uintptr{
		a.masterBack, a.masterForward, a.remoteBack, a.remoteForward, a.masterRefresh,
		a.masterNewFolder, a.masterBookmarks, a.masterMore,
		a.queueTabAll, a.queueTabUploading, a.queueTabDownloading, a.queueTabCompleted,
	} {
		delete(a.buttons, control)
	}
}
