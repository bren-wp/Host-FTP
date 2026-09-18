//go:build windows

package desktop

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/bren-wp/Host-FTP/internal/i18n"
	"github.com/bren-wp/Host-FTP/internal/model"
	"github.com/bren-wp/Host-FTP/internal/platform"
)

const (
	siteIDList           = 8101
	siteIDName           = 8102
	siteIDProtocol       = 8103
	siteIDHost           = 8104
	siteIDPort           = 8105
	siteIDUser           = 8106
	siteIDLocal          = 8107
	siteIDRemote         = 8108
	siteIDKey            = 8109
	siteIDSecurity       = 8110
	siteIDSave           = 8111
	siteIDDelete         = 8112
	siteIDConnect        = 8113
	siteIDClose          = 8114
	siteIDPassword       = 8115
	siteIDPassphrase     = 8116
	siteIDDuplicate      = 8117
	siteIDSettings       = 8118
	siteIDNavConnections = 8120
	siteIDNavTransfers   = 8121
	siteIDNavSync        = 8122
	siteIDNavRemote      = 8123
	siteIDNavLocal       = 8124
	siteIDNavSettings    = 8125
	siteIDNewSite        = 8126
	siteIDGlobalSearch   = 8127
	siteIDQuickConnect   = 8128
	siteIDSiteManagerTab = 8129
	siteIDImportExport   = 8130
	siteIDPresetsTab     = 8131
	siteIDSyncTab        = 8132
	siteIDAutomationTab  = 8133
	siteIDPresetStandard = 8134
	siteIDPresetWebsite  = 8135
	siteIDPresetBackup   = 8136
	siteIDPresetMedia    = 8137
	siteIDRecentList     = 8138
	siteIDSyncBackup     = 8139
	siteIDSyncSkip       = 8140
	siteIDSyncConfirm    = 8141
	siteIDTestConnection = 8142
	siteIDTitleMinimize   = 8143
	siteIDTitleMaximize   = 8144
	siteIDTitleClose      = 8145
	siteIDNavSiteManager  = 8146
	siteIDNavAutomation   = 8147
	siteIDSavedSearch     = 8148
	siteIDRecentClear     = 8149

	siteLBSNotify           = 0x0001
	siteLBSNoIntegralHeight = 0x0100
	siteLBAddString         = 0x0180
	siteLBResetContent      = 0x0184
	siteLBSetCurSel         = 0x0186
	siteLBGetCurSel         = 0x0188
	siteLBNSelChange        = 1
	siteLBNDblClk           = 2
	siteBSDefPushButton     = 0x00000001
	siteBSAutoCheckBox      = 0x00000003
	siteBMGetCheck          = 0x00F0
	siteBMSetCheck          = 0x00F1
	siteBSTChecked          = 1
	siteWindowStyle         = ghostWindowStyle
	siteWMCtlColorListBox   = 0x0134
)

var sitePathLabels = map[string][2]string{
	"en": {"Local path", "Remote path"},
	"hr": {"Lokalna putanja", "Udaljena putanja"},
	"de": {"Lokaler Pfad", "Remote-Pfad"},
	"fr": {"Chemin local", "Chemin distant"},
	"es": {"Ruta local", "Ruta remota"},
	"tr": {"Yerel yol", "Uzak yol"},
	"el": {"Τοπική διαδρομή", "Απομακρυσμένη διαδρομή"},
	"pt": {"Caminho local", "Caminho remoto"},
	"zh": {"本地路径", "远程路径"},
	"ru": {"Локальный путь", "Удалённый путь"},
	"hi": {"स्थानीय पथ", "दूरस्थ पथ"},
	"ja": {"ローカルパス", "リモートパス"},
	"it": {"Percorso locale", "Percorso remoto"},
	"pl": {"Ścieżka lokalna", "Ścieżka zdalna"},
	"nl": {"Lokaal pad", "Extern pad"},
	"cs": {"Místní cesta", "Vzdálená cesta"},
	"uk": {"Локальний шлях", "Віддалений шлях"},
	"sv": {"Lokal sökväg", "Fjärrsökväg"},
	"ro": {"Cale locală", "Cale la distanță"},
	"hu": {"Helyi elérési út", "Távoli elérési út"},
	"da": {"Lokal sti", "Fjernsti"},
	"fi": {"Paikallinen polku", "Etäpolku"},
	"no": {"Lokal sti", "Ekstern sti"},
	"ko": {"로컬 경로", "원격 경로"},
}

func sitePathLabel(language string, remote bool) string {
	pair, ok := sitePathLabels[i18n.Normalize(language)]
	if !ok {
		pair = sitePathLabels[i18n.DefaultLanguage]
	}
	if remote {
		return pair[1]
	}
	return pair[0]
}

func cleanConnectionSecurityTitle(value string) string {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{"GhostFTP —", "Ghost FTP —"} {
		value = strings.TrimSpace(strings.TrimPrefix(value, prefix))
	}
	// The Connections editor can represent FTP, FTPS and SFTP. Reuse the
	// maintained localized SFTP security noun, but remove the protocol token so
	// FTPS/FTP profiles are never shown under a misleading "SFTP security"
	// heading.
	value = strings.ReplaceAll(value, "SFTP", "")
	value = strings.TrimSpace(strings.Trim(value, "—-:"))
	if value == "" {
		return "Security"
	}
	return value
}

type siteManagerState struct {
	parent          *app
	hwnd            uintptr
	list            uintptr
	recentList      uintptr
	listBrush       uintptr
	name            uintptr
	protocol        uintptr
	host            uintptr
	port            uintptr
	user            uintptr
	password        uintptr
	localPath       uintptr
	remotePath      uintptr
	keyPath         uintptr
	passphrase      uintptr
	security        uintptr
	options         uintptr
	securityInfo    uintptr
	settings        uintptr
	duplicate       uintptr
	save            uintptr
	delete          uintptr
	connect         uintptr
	close           uintptr
	profiles        []model.PublicProfile
	selected        int
	closed          bool
	connectAfter    bool
	postAction      int
	navConnections  uintptr
	navTransfers    uintptr
	navSync         uintptr
	navRemote       uintptr
	navLocal        uintptr
	navSiteManager  uintptr
	navAutomation   uintptr
	navSettings     uintptr
	newSite         uintptr
	savedSearch     uintptr
	recentClear     uintptr
	globalSearch    uintptr
	brandIcon       uintptr
	brandHero       uintptr
	mottoPrimary    uintptr
	mottoAccent     uintptr
	privacyLabel    uintptr
	transferHeading uintptr
	syncHeading     uintptr
	savedHeading    uintptr
	recentHeading   uintptr
	activeLabel     uintptr
	securityLabel   uintptr
	savedLabel      uintptr
	recentLabel     uintptr
	footerReady     uintptr
	footerStats     uintptr
	quickConnectTab uintptr
	siteManagerTab  uintptr
	importExportTab uintptr
	presetsTab      uintptr
	syncTab         uintptr
	automationTab   uintptr
	presetStandard  uintptr
	presetWebsite   uintptr
	presetBackup    uintptr
	presetMedia     uintptr
	syncBackup      uintptr
	syncSkip        uintptr
	syncConfirm     uintptr
	testConnection  uintptr
	titleMinimize   uintptr
	titleMaximize   uintptr
	titleClose      uintptr
	testing         bool
	testCancel      context.CancelFunc
	closeAfterTest  bool
}

var (
	siteManagerStates sync.Map
	siteManagerOnce   sync.Once
	siteManagerClass  = "GhostFTP.SiteManager"
	siteManagerProc   = syscall.NewCallback(siteManagerWndProc)
)

func siteManagerWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) (result uintptr) {
	defer func() {
		if recover() == nil {
			return
		}
		if value, ok := siteManagerStates.Load(hwnd); ok {
			if state, ok := value.(*siteManagerState); ok && state != nil && state.parent != nil && !state.closed {
				state.parent.setStatus("Connections recovered from an internal UI error.")
			}
		}
		result = 0
	}()

	value, ok := siteManagerStates.Load(hwnd)
	if ok {
		state := value.(*siteManagerState)
		switch message {
		case wmPaint:
			state.paintReferenceConnections()
			return 0
		case wmNcHitTest:
			base, _, _ := defWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
			if base != htClient {
				return base
			}
			point := chromePoint{X: signedWord(lParam), Y: signedHighWord(lParam)}
			chromeScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&point)))
			if point.Y >= 0 && point.Y < int32(state.parent.scale(42)) && point.X >= 0 && point.X < int32(state.parent.scale(540)) {
				return htCaption
			}
			return htClient
		case wmSize:
			width := state.parent.unscale(int(lParam & 0xffff))
			state.layoutResponsive(width)
			return 0
		case wmCommand:
			id := int(wParam & 0xffff)
			notify := int((wParam >> 16) & 0xffff)
			if id == siteIDList && (notify == siteLBNSelChange || notify == siteLBNDblClk) {
				state.loadCurrentSelection()
				if notify == siteLBNDblClk && state.selected > 0 {
					state.connectAfter = true
					destroyWindow.Call(hwnd)
				}
				return 0
			}
			if id == siteIDRecentList && (notify == siteLBNSelChange || notify == siteLBNDblClk) {
				if state.loadRecentSelection() && notify == siteLBNDblClk && state.selected > 0 {
					state.connectAfter = true
					destroyWindow.Call(hwnd)
				}
				return 0
			}
			if id == siteIDProtocol && notify == cbnSelChange {
				state.syncProtocolPort()
				return 0
			}
			if notify == bnClicked {
				switch id {
				case siteIDTitleMinimize:
					showWindow.Call(hwnd, chromeSWMinimize)
					return 0
				case siteIDTitleMaximize:
					zoomed, _, _ := chromeIsZoomed.Call(hwnd)
					if zoomed != 0 {
						showWindow.Call(hwnd, chromeSWRestore)
					} else {
						showWindow.Call(hwnd, chromeSWMaximize)
					}
					return 0
				case siteIDTitleClose:
					if state.testing {
						state.closeAfterTest = true
						state.cancelConnectionTest()
						state.parent.setStatus("Cancelling connection test…")
						return 0
					}
					destroyWindow.Call(hwnd)
					return 0
				case siteIDNavConnections:
					return 0
				case siteIDNavTransfers:
					state.postAction = siteIDNavTransfers
					destroyWindow.Call(hwnd)
					return 0
				case siteIDNavSync:
					state.postAction = siteIDNavSync
					destroyWindow.Call(hwnd)
					return 0
				case siteIDNavRemote:
					state.postAction = siteIDNavRemote
					destroyWindow.Call(hwnd)
					return 0
				case siteIDNavLocal:
					state.postAction = siteIDNavLocal
					destroyWindow.Call(hwnd)
					return 0
				case siteIDNavSiteManager:
					sidebarSetFocus.Call(state.list)
					return 0
				case siteIDNavAutomation:
					state.parent.openSettings()
					state.refreshOptionsSummary()
					return 0
				case siteIDNavSettings:
					state.postAction = siteIDNavSettings
					destroyWindow.Call(hwnd)
					return 0
				case siteIDSavedSearch:
					state.searchSavedSite()
					return 0
				case siteIDRecentClear:
					state.clearRecentConnections()
					return 0
				case siteIDNewSite:
					sendMessageW.Call(state.list, siteLBSetCurSel, 0, 0)
					state.loadSelection(0)
					return 0
				case siteIDGlobalSearch:
					state.postAction = siteIDGlobalSearch
					destroyWindow.Call(hwnd)
					return 0
				case siteIDQuickConnect:
					sendMessageW.Call(state.list, siteLBSetCurSel, 0, 0)
					state.loadSelection(0)
					sidebarSetFocus.Call(state.host)
					return 0
				case siteIDSiteManagerTab:
					sidebarSetFocus.Call(state.list)
					return 0
				case siteIDImportExport:
					state.importExportProfiles()
					return 0
				case siteIDPresetsTab:
					return 0
				case siteIDSyncTab:
					state.postAction = siteIDNavSync
					destroyWindow.Call(hwnd)
					return 0
				case siteIDAutomationTab:
					state.parent.openSettings()
					state.refreshOptionsSummary()
					return 0
				case siteIDPresetStandard:
					state.applyTransferPreset("standard")
					return 0
				case siteIDPresetWebsite:
					state.applyTransferPreset("website")
					return 0
				case siteIDPresetBackup:
					state.applyTransferPreset("backup")
					return 0
				case siteIDPresetMedia:
					state.applyTransferPreset("media")
					return 0
				case siteIDSyncBackup, siteIDSyncSkip, siteIDSyncConfirm:
					state.saveSyncOptions()
					return 0
				case siteIDTestConnection:
					state.testCurrentConnection()
					return 0
				case siteIDDuplicate:
					state.duplicateCurrent()
					return 0
				case siteIDSettings:
					state.parent.openSettings()
					state.refreshOptionsSummary()
					return 0
				case siteIDSave:
					state.saveCurrent()
					return 0
				case siteIDDelete:
					state.deleteCurrent()
					return 0
				case siteIDConnect:
					if state.selected == 0 {
						state.applyQuickConnectToMain()
					}
					state.connectAfter = true
					destroyWindow.Call(hwnd)
					return 0
				case siteIDClose:
					destroyWindow.Call(hwnd)
					return 0
				}
			}
		case wmMeasureItem:
			if state.measureNavigationItem(lParam) {
				return 1
			}
		case wmDrawItem:
			if lParam != 0 {
				d := drawItemFromLParam(lParam)
				if state.drawNavigationItem(&d) {
					return 1
				}
				if state.parent.drawButton(&d) {
					return 1
				}
			}
		case siteWMCtlColorListBox:
			setTextColor.Call(wParam, textColor())
			setBkColor.Call(wParam, listColor())
			if state.listBrush != 0 {
				return state.listBrush
			}
			return state.parent.panelBrush
		case wmCtlColorStatic:
			color := textColor()
			if lParam == state.mottoPrimary {
				color = mutedColor()
			} else if lParam == state.mottoAccent {
				color = accentColor()
			}
			setTextColor.Call(wParam, color)
			setBkColor.Call(wParam, panelColor())
			return state.parent.panelBrush
		case wmCtlColorEdit, wmCtlColorBtn:
			setTextColor.Call(wParam, textColor())
			setBkColor.Call(wParam, panelColor())
			return state.parent.panelBrush
		case wmClose:
			if state.testing {
				state.closeAfterTest = true
				state.cancelConnectionTest()
				state.parent.setStatus("Cancelling connection test…")
				return 0
			}
			destroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
			state.cancelConnectionTest()
			for _, button := range []uintptr{
				state.duplicate, state.save, state.delete, state.connect, state.close, state.settings, state.newSite,
				state.navConnections, state.navTransfers, state.navSync, state.navRemote, state.navLocal, state.navSiteManager, state.navAutomation, state.navSettings,
				state.globalSearch, state.savedSearch, state.recentClear, state.quickConnectTab, state.siteManagerTab, state.importExportTab,
				state.presetsTab, state.syncTab, state.automationTab,
				state.presetStandard, state.presetWebsite, state.presetBackup, state.presetMedia,
				state.syncBackup, state.syncSkip, state.syncConfirm, state.testConnection,
				state.titleMinimize, state.titleMaximize, state.titleClose,
			} {
				delete(state.parent.buttons, button)
			}
			if state.listBrush != 0 {
				deleteObject.Call(state.listBrush)
				state.listBrush = 0
			}
			state.closed = true
			return 0
		}
	}
	result, _, _ = defWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func (state *siteManagerState) protocolValue() string {
	index := selectedComboIndex(state.protocol)
	if index < 0 || index >= len(protocolSpecs) {
		return protocolSpecs[0].Value
	}
	return protocolSpecs[index].Value
}

func (state *siteManagerState) setProtocol(protocol string) {
	sendMessageW.Call(state.protocol, cbSetCurSel, protocolIndex(protocol), 0)
	state.syncProtocolControls()
}

func (state *siteManagerState) syncProtocolControls() {
	sftp := state.protocolValue() == "sftp"
	setControlEnabled(state.keyPath, sftp)
	setControlEnabled(state.passphrase, sftp)
	if !sftp {
		setText(state.passphrase, "")
	}
}

func (state *siteManagerState) syncProtocolPort() {
	protocol := state.protocolValue()
	current := strings.TrimSpace(getText(state.port))
	for _, spec := range protocolSpecs {
		if current == spec.Port {
			setText(state.port, protocolSpecs[protocolIndex(protocol)].Port)
			state.syncProtocolControls()
			return
		}
	}
	if current == "" {
		setText(state.port, protocolSpecs[protocolIndex(protocol)].Port)
	}
	state.syncProtocolControls()
}

func (state *siteManagerState) searchSavedSite() {
	if state == nil || state.parent == nil {
		return
	}
	query, ok := platform.PromptDialog(
		"Ghost FTP — Saved Sites",
		"Search saved connections",
		"",
	)
	if !ok {
		return
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return
	}
	for index, profile := range state.profiles {
		material := strings.ToLower(profile.Name + "\n" + profile.Host + "\n" + profile.Username + "\n" + profile.Protocol)
		if strings.Contains(material, query) {
			sendMessageW.Call(state.list, siteLBSetCurSel, uintptr(index+1), 0)
			state.loadSelection(index + 1)
			sidebarSetFocus.Call(state.list)
			state.parent.setStatus("Saved site found: " + profile.Name)
			return
		}
	}
	platform.InfoDialog(
		"Ghost FTP — Saved Sites",
		"No matching saved site",
		"No saved connection matched that search. Try a profile name, host, username or protocol.",
	)
}

func (state *siteManagerState) clearRecentConnections() {
	if state == nil || state.parent == nil {
		return
	}
	state.parent.recentConnections = nil
	state.refillRecentConnections()
	state.refreshFooter()
	state.parent.setStatus("Recent connection history cleared")
}

func siteManagerRate(bytesPerSecond float64) string {
	if bytesPerSecond <= 0 {
		return "0 B/s"
	}
	const (
		kiB = 1024.0
		miB = 1024.0 * 1024.0
	)
	switch {
	case bytesPerSecond >= miB:
		return fmt.Sprintf("%.1f MB/s", bytesPerSecond/miB)
	case bytesPerSecond >= kiB:
		return fmt.Sprintf("%.1f KB/s", bytesPerSecond/kiB)
	default:
		return fmt.Sprintf("%.0f B/s", bytesPerSecond)
	}
}

func (state *siteManagerState) refreshFooter() {
	if state == nil || state.parent == nil {
		return
	}
	active := 0
	uploadRate := 0.0
	downloadRate := 0.0
	for _, job := range state.parent.transferJobs {
		if job.Status != "running" {
			continue
		}
		active++
		if strings.EqualFold(job.Direction, "upload") {
			uploadRate += job.BytesPerSecond
		} else if strings.EqualFold(job.Direction, "download") {
			downloadRate += job.BytesPerSecond
		}
	}
	ready := "●  Ready"
	if state.testing {
		ready = "●  Testing connection…"
	}
	activity := "No active transfers"
	if active == 1 {
		activity = "1 active transfer"
	} else if active > 1 {
		activity = fmt.Sprintf("%d active transfers", active)
	}
	setText(state.footerReady, ready+"     "+activity)
	setText(
		state.footerStats,
		fmt.Sprintf("%d connections saved     %d active transfers     ↓ %s     ↑ %s", len(state.profiles), active, siteManagerRate(downloadRate), siteManagerRate(uploadRate)),
	)
}

func (state *siteManagerState) refillProfiles(selectedID string) {
	sendMessageW.Call(state.list, siteLBResetContent, 0, 0)
	quick := "+  " + state.parent.tr("profile.quick")
	sendMessageW.Call(state.list, siteLBAddString, 0, uintptr(unsafe.Pointer(wstr(quick))))
	selected := 0
	for index, profile := range state.profiles {
		label := profile.Name + "   ·   " + strings.ToUpper(profile.Protocol) + "   ·   " + profile.Host
		sendMessageW.Call(state.list, siteLBAddString, 0, uintptr(unsafe.Pointer(wstr(label))))
		if profile.ID == selectedID {
			selected = index + 1
		}
	}
	state.selected = selected
	sendMessageW.Call(state.list, siteLBSetCurSel, uintptr(selected), 0)
	state.loadSelection(selected)
}

func (state *siteManagerState) refillRecentConnections() {
	if state == nil || state.recentList == 0 || state.parent == nil {
		return
	}
	sendMessageW.Call(state.recentList, siteLBResetContent, 0, 0)
	now := time.Now()
	for _, entry := range state.parent.recentConnections {
		label := recentConnectionLabel(entry, now)
		sendMessageW.Call(state.recentList, siteLBAddString, 0, uintptr(unsafe.Pointer(wstr(label))))
	}
	state.refreshFooter()
}

func (state *siteManagerState) loadRecentSelection() bool {
	if state == nil || state.recentList == 0 || state.parent == nil {
		return false
	}
	index, _, _ := sendMessageW.Call(state.recentList, siteLBGetCurSel, 0, 0)
	if int32(index) < 0 || int(index) >= len(state.parent.recentConnections) {
		return false
	}
	entry := state.parent.recentConnections[int(index)]
	if entry.ProfileID != "" {
		for profileIndex, profile := range state.profiles {
			if profile.ID == entry.ProfileID {
				sendMessageW.Call(state.list, siteLBSetCurSel, uintptr(profileIndex+1), 0)
				state.loadSelection(profileIndex + 1)
				return true
			}
		}
	}

	state.selected = 0
	sendMessageW.Call(state.list, siteLBSetCurSel, 0, 0)
	setText(state.name, entry.Name)
	state.setProtocol(entry.Protocol)
	setText(state.host, entry.Host)
	if entry.Port > 0 {
		setText(state.port, strconv.Itoa(entry.Port))
	} else {
		setText(state.port, protocolSpecs[protocolIndex(entry.Protocol)].Port)
	}
	setText(state.user, entry.Username)
	setText(state.password, "")
	setText(state.passphrase, "")
	setText(state.localPath, state.parent.localCurrent)
	setText(state.remotePath, "/")
	setText(state.keyPath, "")
	setText(state.security, "Recent session · credentials are never stored in recent history")
	setControlEnabled(state.duplicate, false)
	setControlEnabled(state.delete, false)
	return true
}

func (state *siteManagerState) loadCurrentSelection() {
	index, _, _ := sendMessageW.Call(state.list, siteLBGetCurSel, 0, 0)
	if int32(index) < 0 {
		return
	}
	state.loadSelection(int(index))
}

func (state *siteManagerState) loadSelection(index int) {
	state.selected = index
	setText(state.password, "")
	setText(state.passphrase, "")
	if index <= 0 || index > len(state.profiles) {
		state.selected = 0
		setText(state.name, "")
		state.setProtocol(defaultConnectionProtocol)
		setText(state.host, "")
		setText(state.port, protocolSpecs[0].Port)
		setText(state.user, "")
		setText(state.localPath, state.parent.localCurrent)
		setText(state.remotePath, "/")
		setText(state.keyPath, "")
		setText(state.security, state.parent.tr("profile.quick")+" · "+state.parent.tr("cue.password"))
		setControlEnabled(state.duplicate, false)
		setControlEnabled(state.delete, false)
		return
	}
	profile := state.profiles[index-1]
	setText(state.name, profile.Name)
	state.setProtocol(profile.Protocol)
	setText(state.host, profile.Host)
	setText(state.port, strconv.Itoa(profile.Port))
	setText(state.user, profile.Username)
	setText(state.localPath, profile.LocalPath)
	setText(state.remotePath, profile.RemotePath)
	setText(state.keyPath, profile.PrivateKeyPath)
	setText(state.security, state.securitySummary(profile))
	setControlEnabled(state.duplicate, true)
	setControlEnabled(state.delete, true)
}

func (state *siteManagerState) securitySummary(profile model.PublicProfile) string {
	parts := make([]string, 0, 4)
	if profile.HasPassword {
		parts = append(parts, "● "+state.parent.tr("terminal.password"))
	}
	if profile.PrivateKeyPath != "" {
		parts = append(parts, "● "+state.parent.tr("terminal.private_key"))
	}
	if profile.HasPassphrase {
		parts = append(parts, "● "+state.parent.tr("cue.passphrase"))
	}
	if profile.Fingerprint != "" {
		parts = append(parts, "● "+strings.TrimSuffix(state.parent.tr("terminal.fingerprint"), ":")+" "+profile.Fingerprint)
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, "    ")
}

func (state *siteManagerState) profileInput() (model.ProfileInput, error) {
	protocol := state.protocolValue()
	host := getText(state.host)
	username := getText(state.user)
	port, err := validateRawConnectionInput(protocol, host, getText(state.port), username)
	if err != nil {
		return model.ProfileInput{}, err
	}
	name := strings.TrimSpace(getText(state.name))
	if name == "" {
		return model.ProfileInput{}, fmt.Errorf("%s", state.parent.tr("column.name"))
	}
	id := ""
	if state.selected > 0 && state.selected <= len(state.profiles) {
		id = state.profiles[state.selected-1].ID
	}
	remotePath := optionalRemotePath(getText(state.remotePath))
	return model.ProfileInput{
		ID:             id,
		Name:           name,
		Protocol:       protocol,
		Host:           host,
		Port:           port,
		Username:       username,
		Password:       getText(state.password),
		PrivateKeyPath: getText(state.keyPath),
		Passphrase:     getText(state.passphrase),
		LocalPath:      getText(state.localPath),
		RemotePath:     remotePath,
	}, nil
}

func (state *siteManagerState) connectionDraft() (string, model.ConnectionConfig, error) {
	if state == nil || state.parent == nil {
		return "", model.ConnectionConfig{}, fmt.Errorf("connection editor is unavailable")
	}
	protocol := state.protocolValue()
	host := getText(state.host)
	username := getText(state.user)
	port, err := validateRawConnectionInput(protocol, host, getText(state.port), username)
	if err != nil {
		return "", model.ConnectionConfig{}, err
	}
	profileID := ""
	if state.selected > 0 && state.selected <= len(state.profiles) {
		profileID = state.profiles[state.selected-1].ID
	}
	return profileID, model.ConnectionConfig{
		Protocol:       protocol,
		Host:           host,
		Port:           port,
		Username:       username,
		Password:       getText(state.password),
		PrivateKeyPath: strings.TrimSpace(getText(state.keyPath)),
		Passphrase:     getText(state.passphrase),
	}, nil
}

func (state *siteManagerState) cancelConnectionTest() {
	if state == nil {
		return
	}
	if state.testCancel != nil {
		state.testCancel()
		state.testCancel = nil
	}
}

func (state *siteManagerState) setTesting(testing bool) {
	if state == nil {
		return
	}
	state.testing = testing
	label := "Test Connection"
	if testing {
		label = "Testing…"
	}
	if state.testConnection != 0 {
		state.parent.setButtonLabel(state.testConnection, label)
		state.parent.registerButtonVisual(state.testConnection, iconConnect, label, buttonDefault, false)
		setControlEnabled(state.testConnection, !testing)
		invalidateRect.Call(state.testConnection, 0, 0)
	}
	setControlEnabled(state.connect, !testing)
	setControlEnabled(state.save, !testing)
	state.refreshFooter()
}

func (state *siteManagerState) finishConnectionTest(host string, err error) {
	if state == nil || state.parent == nil || state.closed {
		return
	}
	state.testCancel = nil
	state.setTesting(false)
	if state.closeAfterTest {
		state.closeAfterTest = false
		destroyWindow.Call(state.hwnd)
		return
	}
	if err != nil {
		state.parent.setStatus("Connection test failed")
		platform.ErrorDialog(
			"Ghost FTP — Connection Test",
			"Connection test failed",
			state.parent.userMessage(err, "connection.failed_body"),
		)
		return
	}
	state.parent.setStatus("Connection test passed: " + host)
	platform.InfoDialog(
		"Ghost FTP — Connection Test",
		"Connection verified",
		"Ghost FTP successfully authenticated, verified the remote session and closed the temporary test connection. No profile or credential was changed.",
	)
}

func (state *siteManagerState) runConnectionTest(profileID string, cfg model.ConnectionConfig, fingerprint string) {
	if state == nil || state.parent == nil {
		return
	}
	host := cfg.Host
	timeout := connectionTimeoutDuration(state.parent.settings)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	state.cancelConnectionTest()
	state.testCancel = cancel
	state.parent.goSafe(func() {
		defer cancel()
		result, err := state.parent.engine.Connect(ctx, profileID, cfg, fingerprint, false)
		if err == nil && result.RequiresTrust {
			state.parent.engine.CancelPendingTrust()
			err = fmt.Errorf("server identity still requires confirmation")
		}
		if err == nil {
			err = state.parent.engine.Probe(ctx)
		}
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), timeout)
		_ = state.parent.engine.Disconnect(disconnectCtx)
		disconnectCancel()
		cfg.Password = ""
		cfg.Passphrase = ""
		state.parent.dispatch(func() {
			state.finishConnectionTest(host, err)
		})
	})
}

func (state *siteManagerState) testCurrentConnection() {
	if state == nil || state.parent == nil || state.testing {
		return
	}
	if state.parent.connected || state.parent.connectionBusy {
		platform.InfoDialog(
			"Ghost FTP — Connection Test",
			"Disconnect the active session first",
			"Test Connection uses an isolated temporary login through the maintained Ghost FTP transfer engine. Disconnect the current server before starting a test.",
		)
		return
	}

	profileID, cfg, err := state.connectionDraft()
	if err != nil {
		platform.ErrorDialog(
			"Ghost FTP — Connection Test",
			"Connection details are incomplete",
			state.parent.userMessage(err, "connection.invalid_data_body"),
		)
		return
	}
	state.setTesting(true)

	timeout := connectionTimeoutDuration(state.parent.settings)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	state.cancelConnectionTest()
	state.testCancel = cancel
	state.parent.goSafe(func() {
		defer cancel()
		result, connectErr := state.parent.engine.Connect(ctx, profileID, cfg, "", false)
		if connectErr != nil {
			cfg.Password = ""
			cfg.Passphrase = ""
			state.parent.dispatch(func() {
				state.finishConnectionTest(cfg.Host, connectErr)
			})
			return
		}
		if result.RequiresTrust {
			fingerprint := result.Fingerprint
			state.parent.dispatch(func() {
				if state.closed {
					state.parent.engine.CancelPendingTrust()
					return
				}
				if !platform.ConfirmDialog(
					state.parent.tr("sftp.security"),
					state.parent.tr("sftp.new_key"),
					state.parent.tr("sftp.trust_body", fingerprint),
				) {
					state.parent.engine.CancelPendingTrust()
					state.finishConnectionTest(cfg.Host, fmt.Errorf("server identity was not trusted"))
					return
				}
				state.runConnectionTest(profileID, cfg, fingerprint)
			})
			return
		}

		probeErr := state.parent.engine.Probe(ctx)
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), timeout)
		_ = state.parent.engine.Disconnect(disconnectCtx)
		disconnectCancel()
		cfg.Password = ""
		cfg.Passphrase = ""
		state.parent.dispatch(func() {
			state.finishConnectionTest(cfg.Host, probeErr)
		})
	})
}

func (state *siteManagerState) importExportProfiles() {
	if state == nil || state.parent == nil {
		return
	}
	export, chosen := platform.ChoiceDialog(
		"Ghost FTP — Sites",
		"Import or export saved sites",
		"Export creates a metadata-only Ghost FTP site bundle. Stored passwords, private-key passphrases and trusted host fingerprints are never written to the bundle. Import adds valid sites without importing secrets.",
		"Export",
		"Import",
	)
	if !chosen {
		return
	}

	if export {
		path, err := platform.ChooseProfileExportFile()
		if err != nil {
			platform.ErrorDialog("Ghost FTP — Sites", "Export failed", state.parent.userMessage(err, "error.generic"))
			return
		}
		if strings.TrimSpace(path) == "" {
			return
		}
		count, err := state.parent.engine.ExportProfiles(path)
		if err != nil {
			platform.ErrorDialog("Ghost FTP — Sites", "Export failed", state.parent.userMessage(err, "error.generic"))
			return
		}
		state.parent.setStatus(fmt.Sprintf("Exported %d saved sites", count))
		platform.InfoDialog(
			"Ghost FTP — Sites",
			"Saved sites exported",
			fmt.Sprintf("%d saved sites were exported without stored passwords, passphrases or trusted host fingerprints.", count),
		)
		return
	}

	path, err := platform.ChooseProfileImportFile()
	if err != nil {
		platform.ErrorDialog("Ghost FTP — Sites", "Import failed", state.parent.userMessage(err, "error.generic"))
		return
	}
	if strings.TrimSpace(path) == "" {
		return
	}
	result, err := state.parent.engine.ImportProfiles(path)
	if err != nil {
		platform.ErrorDialog("Ghost FTP — Sites", "Import failed", state.parent.userMessage(err, "error.generic"))
		return
	}
	profiles, err := state.parent.engine.Profiles()
	if err != nil {
		platform.ErrorDialog("Ghost FTP — Sites", state.parent.tr("profile.load_failed"), state.parent.userMessage(err, "error.generic"))
		return
	}
	state.profiles = profiles
	state.parent.applyProfiles(profiles, nil)
	state.refillProfiles(state.parent.selectedProfileID)
	state.parent.setStatus(fmt.Sprintf("Imported %d saved sites · %d already present", result.Imported, result.Skipped))
	platform.InfoDialog(
		"Ghost FTP — Sites",
		"Site import complete",
		fmt.Sprintf("%d sites imported. %d matching sites were already present. Credentials must be entered or saved again on this Windows account.", result.Imported, result.Skipped),
	)
}

func (state *siteManagerState) duplicateCurrent() {
	if state.selected <= 0 || state.selected > len(state.profiles) {
		return
	}
	draft := duplicateProfileDraft(state.profiles[state.selected-1])

	// Move to the unsaved/quick slot before populating the controls. Programmatic
	// LB_SETCURSEL does not emit LBN_SELCHANGE, so it cannot clear the draft.
	state.selected = 0
	sendMessageW.Call(state.list, siteLBSetCurSel, 0, 0)
	setText(state.name, draft.Name)
	state.setProtocol(draft.Protocol)
	setText(state.host, draft.Host)
	setText(state.port, strconv.Itoa(draft.Port))
	setText(state.user, draft.Username)
	setText(state.password, "")
	setText(state.localPath, draft.LocalPath)
	setText(state.remotePath, draft.RemotePath)
	setText(state.keyPath, draft.PrivateKeyPath)
	setText(state.passphrase, "")
	setText(state.security, "—")
	setControlEnabled(state.duplicate, false)
	setControlEnabled(state.delete, false)
}

func (state *siteManagerState) saveCurrent() {
	input, err := state.profileInput()
	if err != nil {
		platform.ErrorDialog("Ghost FTP — "+nativeMenuWords(state.parent.languageCode())[5], state.parent.tr("settings.invalid_value"), state.parent.userMessage(err, "error.generic"))
		return
	}
	if input.Password != "" || input.Passphrase != "" {
		words := credentialConsentText(state.parent.languageCode())
		if !platform.ConfirmDialog(words.Title, words.Question, words.Body) {
			input.Password = ""
			input.Passphrase = ""
			input.ClearPassword = true
			input.ClearPassphrase = true
		}
	}
	saved, err := state.parent.engine.SaveProfile(input)
	input.Password = ""
	input.Passphrase = ""
	setText(state.password, "")
	setText(state.passphrase, "")
	if err != nil {
		platform.ErrorDialog("Ghost FTP — "+nativeMenuWords(state.parent.languageCode())[5], state.parent.tr("settings.save_failed"), state.parent.userMessage(err, "settings.save_failed_body"))
		return
	}
	profiles, err := state.parent.engine.Profiles()
	if err != nil {
		platform.ErrorDialog("Ghost FTP", state.parent.tr("profile.load_failed"), state.parent.userMessage(err, "error.generic"))
		return
	}
	state.profiles = profiles
	state.parent.selectedProfileID = saved.ID
	state.parent.applyProfiles(profiles, nil)
	state.refillProfiles(saved.ID)
	state.parent.setStatus(state.parent.tr("profile.save") + ": " + saved.Name)
}

func (state *siteManagerState) deleteCurrent() {
	if state.selected <= 0 || state.selected > len(state.profiles) {
		return
	}
	profile := state.profiles[state.selected-1]
	if !platform.ConfirmDialog("Ghost FTP — "+nativeMenuWords(state.parent.languageCode())[5], state.parent.tr("profile.delete"), profile.Name) {
		return
	}
	if err := state.parent.engine.RemoveProfile(profile.ID); err != nil {
		platform.ErrorDialog("Ghost FTP", state.parent.tr("profile.delete"), state.parent.userMessage(err, "error.generic"))
		return
	}
	profiles, err := state.parent.engine.Profiles()
	if err != nil {
		platform.ErrorDialog("Ghost FTP", state.parent.tr("profile.load_failed"), state.parent.userMessage(err, "error.generic"))
		return
	}
	state.profiles = profiles
	if state.parent.selectedProfileID == profile.ID {
		state.parent.selectedProfileID = ""
	}
	state.parent.applyProfiles(profiles, nil)
	state.refillProfiles("")
	state.parent.setStatus(state.parent.tr("profile.delete"))
}

func (state *siteManagerState) applyQuickConnectToMain() {
	state.parent.selectedProfileID = ""
	sendMessageW.Call(state.parent.profilesCombo, cbSetCurSel, 0, 0)
	state.parent.setProtocolValue(state.protocolValue())
	setText(state.parent.host, getText(state.host))
	setText(state.parent.port, getText(state.port))
	setText(state.parent.user, getText(state.user))
	setText(state.parent.pass, getText(state.password))
	setText(state.parent.keyPath, getText(state.keyPath))
	setText(state.parent.passphrase, getText(state.passphrase))
	setText(state.parent.localPath, getText(state.localPath))
	setText(state.parent.remotePath, getText(state.remotePath))
	state.parent.resetProfileCredentialCues()
	state.parent.updateActionControls()
}

func siteManagerLimitText(value int) string {
	if value <= 0 {
		return "Unlimited"
	}
	return fmt.Sprintf("%d KiB/s", value)
}

func (state *siteManagerState) applyTransferPreset(name string) {
	if state == nil || state.parent == nil {
		return
	}
	next := state.parent.settings
	label := "Standard Upload"
	switch name {
	case "website":
		label = "Website Deployment"
		next.Parallelism = 4
		next.ConflictPolicy = model.ConflictPolicyReplaceBackup
		next.BackupBeforeOverwrite = true
		next.SkipExisting = false
		next.ConfirmDelete = true
		next.AutoRetryCount = 2
		next.RetryDelaySeconds = 3
	case "backup":
		label = "Backup (Incremental)"
		next.Parallelism = 2
		next.ConflictPolicy = model.ConflictPolicySkip
		next.BackupBeforeOverwrite = true
		next.SkipExisting = true
		next.ConfirmDelete = true
		next.AutoRetryCount = 2
		next.RetryDelaySeconds = 5
	case "media":
		label = "Media Transfer"
		next.Parallelism = 2
		next.ConflictPolicy = model.ConflictPolicyReplace
		next.BackupBeforeOverwrite = false
		next.SkipExisting = false
		next.ConfirmDelete = true
		next.AutoRetryCount = 1
		next.RetryDelaySeconds = 3
	default:
		next.Parallelism = 3
		next.ConflictPolicy = model.ConflictPolicyReplace
		next.BackupBeforeOverwrite = false
		next.SkipExisting = false
		next.ConfirmDelete = true
		next.AutoRetryCount = 1
		next.RetryDelaySeconds = 3
	}
	next.UploadLimitKiBPerSecond = 0
	next.DownloadLimitKiBPerSecond = 0
	saved, err := state.parent.engine.SetSettings(next)
	if err != nil {
		platform.ErrorDialog("Ghost FTP", "Transfer preset", state.parent.userMessage(err, "settings.save_failed_body"))
		return
	}
	state.parent.settings = saved
	state.refreshOptionsSummary()
	state.refreshSyncOptions()
	state.parent.setStatus("Transfer preset applied: " + label)
}

func siteChecked(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	checked, _, _ := sendMessageW.Call(hwnd, siteBMGetCheck, 0, 0)
	return checked == siteBSTChecked
}

func siteSetChecked(hwnd uintptr, checked bool) {
	if hwnd == 0 {
		return
	}
	value := uintptr(0)
	if checked {
		value = siteBSTChecked
	}
	sendMessageW.Call(hwnd, siteBMSetCheck, value, 0)
}

func (state *siteManagerState) refreshSyncOptions() {
	if state == nil || state.parent == nil {
		return
	}
	settings := state.parent.settings
	siteSetChecked(state.syncBackup, settings.BackupBeforeOverwrite)
	siteSetChecked(state.syncSkip, settings.SkipExisting)
	siteSetChecked(state.syncConfirm, settings.ConfirmDelete)
}

func (state *siteManagerState) saveSyncOptions() {
	if state == nil || state.parent == nil {
		return
	}
	next := state.parent.settings
	next.BackupBeforeOverwrite = siteChecked(state.syncBackup)
	next.SkipExisting = siteChecked(state.syncSkip)
	next.ConfirmDelete = siteChecked(state.syncConfirm)
	saved, err := state.parent.engine.SetSettings(next)
	if err != nil {
		state.refreshSyncOptions()
		platform.ErrorDialog("Ghost FTP", "Sync Options", state.parent.userMessage(err, "settings.save_failed_body"))
		return
	}
	state.parent.settings = saved
	state.refreshOptionsSummary()
	state.parent.setStatus("Sync options updated")
}

func (state *siteManagerState) refreshOptionsSummary() {
	if state == nil || state.parent == nil || state.options == 0 {
		return
	}
	settings := normalizeSettingsForPrompt(state.parent.settings)
	conflict := conflictPolicyText(state.parent.languageCode())
	policy := conflict.Replace
	switch conflictPolicyIndex(settings) {
	case 0:
		policy = conflict.Skip
	case 2:
		policy = conflict.ReplaceBackup
	}
	confirmDelete := "On"
	if !settings.ConfirmDelete {
		confirmDelete = "Off"
	}
	text := fmt.Sprintf(
		"Parallel %d · Upload %s · Download %s\r\nRetry %d × %ds · Conflict %s · Delete confirm %s",
		settings.Parallelism,
		siteManagerLimitText(settings.UploadLimitKiBPerSecond),
		siteManagerLimitText(settings.DownloadLimitKiBPerSecond),
		settings.AutoRetryCount,
		settings.RetryDelaySeconds,
		policy,
		confirmDelete,
	)
	setText(state.options, text)
}

func (state *siteManagerState) layoutResponsive(width int) {
	if state == nil || state.parent == nil {
		return
	}
	chromeX := width - 126
	if chromeX < 830 {
		chromeX = 830
	}
	state.parent.move(state.titleMinimize, chromeX, 4, 38, 32)
	state.parent.move(state.titleMaximize, chromeX+40, 4, 38, 32)
	state.parent.move(state.titleClose, chromeX+80, 4, 38, 32)
	var client rect
	height := 0
	if ok, _, _ := getClientRect.Call(state.hwnd, uintptr(unsafe.Pointer(&client))); ok != 0 {
		height = state.parent.unscale(int(client.Bottom - client.Top))
	}

	const (
		fullReferenceWidth  = 1580
		fullReferenceHeight = 820
	)
	compact := (width > 0 && width < fullReferenceWidth) || (height > 0 && height < fullReferenceHeight)
	if compact {
		// Preserve a complete, usable connection editor on smaller displays. Every
		// optional side-card control and heading is hidden together so no orphaned
		// labels, overlapping buttons or clipped reference chrome remain visible.
		showControls(false,
			state.privacyLabel, state.brandHero, state.settings,
			state.transferHeading, state.syncHeading, state.savedHeading, state.recentHeading,
			state.activeLabel, state.securityLabel, state.savedLabel, state.recentLabel,
			state.presetsTab, state.syncTab, state.automationTab,
			state.presetStandard, state.presetWebsite, state.presetBackup, state.presetMedia,
			state.syncBackup, state.syncSkip, state.syncConfirm,
			state.options, state.securityInfo, state.newSite, state.list, state.recentList, state.duplicate, state.delete,
		)
		state.parent.move(state.globalSearch, 560, 18, 320, 38)
		actionY := 738
		if height > 0 {
			actionY = height - 54
			if actionY < 690 {
				actionY = 690
			}
			if actionY > 738 {
				actionY = 738
			}
		}
		state.parent.move(state.testConnection, 262, actionY, 166, 42)
		state.parent.move(state.save, 742, 50, 138, 38)
		showControls(false, state.close)
		state.parent.move(state.connect, 646, actionY, 234, 42)
		invalidateRect.Call(state.hwnd, 0, 1)
		return
	}
	showControls(true,
		state.privacyLabel, state.brandHero, state.settings,
		state.transferHeading, state.syncHeading, state.savedHeading, state.recentHeading,
		state.activeLabel, state.securityLabel, state.savedLabel, state.recentLabel,
		state.presetsTab, state.syncTab, state.automationTab,
		state.presetStandard, state.presetWebsite, state.presetBackup, state.presetMedia,
		state.syncBackup, state.syncSkip, state.syncConfirm,
		state.options, state.securityInfo, state.newSite, state.list, state.recentList, state.duplicate, state.delete,
	)
	state.parent.move(state.globalSearch, 560, 18, 494, 38)
	state.parent.move(state.settings, 926, 738, 264, 42)
	state.parent.move(state.testConnection, 262, 738, 166, 42)
	state.parent.move(state.save, 742, 50, 138, 38)
	showControls(false, state.close)
	state.parent.move(state.connect, 646, 738, 234, 42)
	invalidateRect.Call(state.hwnd, 0, 1)
}

func (state *siteManagerState) createControls(hinst uintptr) error {
	parent := state.parent
	mk := func(class, text string, style uint32, x, y, width, height, id int) uintptr {
		hwnd, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(wstr(class))),
			uintptr(unsafe.Pointer(wstr(text))),
			uintptr(wsChild|wsVisible|style),
			uintptr(parent.scale(x)), uintptr(parent.scale(y)), uintptr(parent.scale(width)), uintptr(parent.scale(height)),
			state.hwnd, uintptr(id), hinst, 0,
		)
		if hwnd != 0 && parent.font != 0 {
			sendMessageW.Call(hwnd, wmSetFont, parent.font, 1)
		}
		if hwnd != 0 {
			applyDarkControl(hwnd, class)
		}
		return hwnd
	}
	label := func(text string, x, y, width int) uintptr {
		hwnd := mk("STATIC", text, 0, x, y, width, 20, 0)
		if hwnd != 0 && parent.smallFont != 0 {
			sendMessageW.Call(hwnd, wmSetFont, parent.smallFont, 1)
		}
		return hwnd
	}
	heading := func(text string, x, y, width int) uintptr {
		hwnd := mk("STATIC", text, 0, x, y, width, 28, 0)
		if hwnd != 0 && parent.titleFont != 0 {
			sendMessageW.Call(hwnd, wmSetFont, parent.titleFont, 1)
		}
		return hwnd
	}
	nav := func(id int, text, icon string, active bool) uintptr {
		variant := buttonNav
		if active {
			variant = buttonNavActive
		}
		return parent.registerButton(mk("BUTTON", text, wsTabStop|bsOwnerDraw, 28, 0, 184, 42, id), icon, text, variant)
	}

	// Left product rail. Resource group 2 is the transparent in-app Ghost mark.
	iconSize := parent.scale(42)
	brandIcon, _, _ := loadImageW.Call(hinst, 2, imageIcon, uintptr(iconSize), uintptr(iconSize), lrShared)
	if brandIcon != 0 {
		state.brandIcon = mk("STATIC", "", ssIcon, 28, 20, 42, 42, 0)
		if state.brandIcon != 0 {
			sendMessageW.Call(state.brandIcon, stmSetImage, imageIcon, brandIcon)
		}
	}
	heading("Ghost FTP", 78, 24, 136)
	label("SECURE TRANSFERS. WITHOUT A TRACE.", 30, 66, 184)

	// Global command/search field across the top, matching the reference while
	// remaining functional by delegating to the maintained workspace search.
	state.globalSearch = parent.registerButton(
		mk("BUTTON", "Search sites, history, or files…    Ctrl+K", wsTabStop|bsOwnerDraw, 560, 18, 494, 38, siteIDGlobalSearch),
		iconSearch, "Search sites, history, or files…    Ctrl+K", buttonSubtle,
	)
	state.titleMinimize = parent.registerButton(mk("BUTTON", "—", bsOwnerDraw, 1448, 4, 38, 32, siteIDTitleMinimize), "", "—", buttonSubtle)
	state.titleMaximize = parent.registerButton(mk("BUTTON", "□", bsOwnerDraw, 1488, 4, 38, 32, siteIDTitleMaximize), "", "□", buttonSubtle)
	state.titleClose = parent.registerButton(mk("BUTTON", "×", bsOwnerDraw, 1528, 4, 38, 32, siteIDTitleClose), "", "×", buttonDanger)
	state.privacyLabel = mk("STATIC", "Private desktop · No account required", 0, 1280, 24, 280, 22, 0)
	state.navConnections = nav(siteIDNavConnections, "Connections", iconConnect, true)
	state.navTransfers = nav(siteIDNavTransfers, "Transfers", iconUpload, false)
	state.navSync = nav(siteIDNavSync, "Synchronize", iconSync, false)
	state.navRemote = nav(siteIDNavRemote, "Remote Files", iconDownload, false)
	state.navLocal = nav(siteIDNavLocal, "Local Files", iconOpenLocal, false)
	state.navSettings = nav(siteIDNavSettings, "Settings", iconSettings, false)
	navY := 102
	for _, control := range []uintptr{state.navConnections, state.navTransfers, state.navSync, state.navRemote, state.navLocal, state.navSettings} {
		parent.move(control, 28, navY, 184, 42)
		navY += 50
	}
	state.mottoPrimary = mk("STATIC", "FILES\r\nMOVE\r\nFREELY.", 0, 34, 554, 176, 76, 0)
	state.mottoAccent = mk("STATIC", "YOU STAY\r\nIN CONTROL.", 0, 34, 640, 176, 58, 0)
	for _, motto := range []uintptr{state.mottoPrimary, state.mottoAccent} {
		if motto != 0 && parent.smallFont != 0 {
			sendMessageW.Call(motto, wmSetFont, parent.smallFont, 1)
		}
	}
	heroSize := parent.scale(116)
	heroIcon, _, _ := loadImageW.Call(hinst, 2, imageIcon, uintptr(heroSize), uintptr(heroSize), lrShared)
	if heroIcon != 0 {
		state.brandHero = mk("STATIC", "", ssIcon, 52, 704, 116, 116, 0)
		if state.brandHero != 0 {
			sendMessageW.Call(state.brandHero, stmSetImage, imageIcon, heroIcon)
		}
	}

	// Main connection card.
	heading("New Connection", 262, 54, 420)
	state.quickConnectTab = parent.registerButton(mk("BUTTON", "Quick Connect", wsTabStop|bsOwnerDraw, 262, 96, 150, 40, siteIDQuickConnect), iconConnect, "Quick Connect", buttonNavActive)
	state.siteManagerTab = parent.registerButton(mk("BUTTON", "Site Manager", wsTabStop|bsOwnerDraw, 420, 96, 150, 40, siteIDSiteManagerTab), iconOpenLocal, "Site Manager", buttonNav)
	state.importExportTab = parent.registerButton(mk("BUTTON", "Import / Export", wsTabStop|bsOwnerDraw, 578, 96, 164, 40, siteIDImportExport), iconUpload, "Import / Export", buttonNav)

	label("Connection Name", 262, 148, 220)
	state.name = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 262, 170, 618, 34, siteIDName)

	label(parent.tr("terminal.protocol"), 262, 216, 150)
	label("Host / Address", 502, 216, 190)
	label(parent.tr("terminal.port"), 796, 216, 70)
	state.protocol = mk("COMBOBOX", "", cbsDropDownList|wsTabStop|wsVScroll, 262, 238, 224, 240, siteIDProtocol)
	for _, spec := range protocolSpecs {
		sendMessageW.Call(state.protocol, cbAddString, 0, uintptr(unsafe.Pointer(wstr(protocolLabel(parent.languageCode(), spec.Value)))))
	}
	state.host = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 502, 238, 278, 34, siteIDHost)
	state.port = mk("EDIT", protocolSpecs[0].Port, wsBorder|wsTabStop|esAutoHScroll, 796, 238, 84, 34, siteIDPort)

	label(parent.tr("terminal.username"), 262, 290, 260)
	label(parent.tr("terminal.password"), 578, 290, 260)
	state.user = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 262, 312, 300, 34, siteIDUser)
	state.password = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll|esPassword, 578, 312, 302, 34, siteIDPassword)

	label(sitePathLabel(parent.languageCode(), false), 262, 364, 260)
	label(sitePathLabel(parent.languageCode(), true), 578, 364, 260)
	state.localPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 262, 386, 300, 34, siteIDLocal)
	state.remotePath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 578, 386, 302, 34, siteIDRemote)

	label(parent.tr("terminal.private_key"), 262, 438, 260)
	label(parent.tr("cue.passphrase"), 646, 438, 210)
	state.keyPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 262, 460, 368, 34, siteIDKey)
	state.passphrase = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll|esPassword, 646, 460, 234, 34, siteIDPassphrase)

	label(cleanConnectionSecurityTitle(parent.tr("sftp.security")), 262, 512, 618)
	state.security = mk("STATIC", "", wsBorder, 262, 534, 618, 72, siteIDSecurity)

	label("PROFILE BEHAVIOR", 262, 618, 618)
	profileNote := "Quick Connect uses credentials only for this session. Saving a profile asks before secrets are stored in the protected Windows credential layer."
	mk("STATIC", profileNote, 0, 262, 642, 618, 48, 0)

	state.save = parent.registerButton(mk("BUTTON", "Save as Profile", wsTabStop|bsOwnerDraw, 742, 50, 138, 38, siteIDSave), iconSave, "Save as Profile", buttonDefault)
	state.testConnection = parent.registerButton(mk("BUTTON", "Test Connection", wsTabStop|bsOwnerDraw, 262, 738, 166, 42, siteIDTestConnection), iconConnect, "Test Connection", buttonDefault)
	state.connect = parent.registerButton(mk("BUTTON", "Connect to Server", wsTabStop|siteBSDefPushButton|bsOwnerDraw, 646, 738, 234, 42, siteIDConnect), iconConnect, "Connect to Server", buttonAccent)
	state.close = parent.registerButton(mk("BUTTON", parent.tr("common.cancel"), wsTabStop|bsOwnerDraw, 500, 738, 132, 42, siteIDClose), iconCancel, parent.tr("common.cancel"), buttonSubtle)
	showControls(false, state.close)

	// Transfer & Sync settings card uses real persisted settings and real presets.
	state.transferHeading = heading("Transfer & Sync Options", 924, 54, 270)
	state.presetsTab = parent.registerButton(mk("BUTTON", "Presets", wsTabStop|bsOwnerDraw, 926, 96, 86, 38, siteIDPresetsTab), iconSave, "Presets", buttonNavActive)
	state.syncTab = parent.registerButton(mk("BUTTON", "Sync", wsTabStop|bsOwnerDraw, 1018, 96, 82, 38, siteIDSyncTab), iconSync, "Sync", buttonNav)
	state.automationTab = parent.registerButton(mk("BUTTON", "Automation", wsTabStop|bsOwnerDraw, 1106, 96, 84, 38, siteIDAutomationTab), iconSettings, "Automation", buttonNav)
	state.presetStandard = parent.registerButtonWithSubtitle(mk("BUTTON", "Standard Upload", wsTabStop|bsOwnerDraw, 926, 150, 264, 56, siteIDPresetStandard), iconUpload, "Standard Upload", "3 parallel · replace existing", buttonNavActive)
	state.presetWebsite = parent.registerButtonWithSubtitle(mk("BUTTON", "Website Deployment", wsTabStop|bsOwnerDraw, 926, 216, 264, 56, siteIDPresetWebsite), iconSync, "Website Deployment", "Backup before overwrite · retry twice", buttonNav)
	state.presetBackup = parent.registerButtonWithSubtitle(mk("BUTTON", "Backup (Incremental)", wsTabStop|bsOwnerDraw, 926, 282, 264, 56, siteIDPresetBackup), iconSave, "Backup (Incremental)", "Skip existing · preserve current files", buttonNav)
	state.presetMedia = parent.registerButtonWithSubtitle(mk("BUTTON", "Media Transfer", wsTabStop|bsOwnerDraw, 926, 348, 264, 56, siteIDPresetMedia), iconUpload, "Media Transfer", "2 parallel · replace existing", buttonNav)
	state.syncHeading = heading("Sync Options", 926, 422, 264)
	state.syncBackup = mk("BUTTON", "Backup before overwrite", wsTabStop|siteBSAutoCheckBox, 926, 458, 264, 28, siteIDSyncBackup)
	state.syncSkip = mk("BUTTON", "Skip existing files", wsTabStop|siteBSAutoCheckBox, 926, 492, 264, 28, siteIDSyncSkip)
	state.syncConfirm = mk("BUTTON", "Confirm destructive actions", wsTabStop|siteBSAutoCheckBox, 926, 526, 264, 28, siteIDSyncConfirm)
	state.activeLabel = label("ACTIVE TRANSFER SETTINGS", 926, 568, 262)
	state.options = mk("STATIC", "", wsBorder, 926, 592, 264, 72, 0)
	state.securityLabel = label("SECURITY", 926, 674, 264)
	securityText := "Host-key and certificate verification stay enabled. Saved secrets remain protected by Windows."
	state.securityInfo = mk("STATIC", securityText, wsBorder, 926, 696, 264, 34, 0)
	state.settings = parent.registerButton(mk("BUTTON", "Open Transfer Settings", wsTabStop|bsOwnerDraw, 926, 738, 264, 42, siteIDSettings), iconSettings, "Open Transfer Settings", buttonDefault)

	// Saved Sites and private session history share the right reference column.
	state.savedHeading = heading("Saved Sites", 1240, 54, 190)
	state.newSite = parent.registerButton(mk("BUTTON", "New Site", wsTabStop|bsOwnerDraw, 1460, 50, 104, 38, siteIDNewSite), iconNewFolder, "New Site", buttonAccent)
	state.savedLabel = label("SAVED CONNECTIONS", 1240, 100, 300)
	state.list = mk("LISTBOX", "", wsBorder|wsTabStop|wsVScroll|siteLBSNotify|siteLBSNoIntegralHeight|siteLBSOwnerDrawFixed|siteLBSHasStrings, 1240, 124, 324, 314, siteIDList)
	if state.list != 0 {
		applySiteManagerNavigationTheme(state.list)
		state.listBrush, _, _ = createSolidBrush.Call(listColor())
	}
	duplicateLabel := siteManagerDuplicateLabel(parent.languageCode())
	state.duplicate = parent.registerButton(mk("BUTTON", duplicateLabel, wsTabStop|bsOwnerDraw, 1240, 450, 154, 36, siteIDDuplicate), iconCopy, duplicateLabel, buttonDefault)
	state.delete = parent.registerButton(mk("BUTTON", parent.tr("profile.delete"), wsTabStop|bsOwnerDraw, 1404, 450, 160, 36, siteIDDelete), iconDelete, parent.tr("profile.delete"), buttonDanger)

	state.recentHeading = heading("Recent Connections", 1240, 514, 250)
	state.recentLabel = label("SESSION HISTORY · NO PASSWORDS", 1240, 550, 310)
	state.recentList = mk("LISTBOX", "", wsBorder|wsTabStop|wsVScroll|siteLBSNotify|siteLBSNoIntegralHeight, 1240, 576, 324, 148, siteIDRecentList)
	if state.recentList != 0 {
		applySiteManagerNavigationTheme(state.recentList)
	}

	for _, control := range []uintptr{
		state.list, state.recentList, state.duplicate, state.name, state.protocol, state.host, state.port, state.user, state.password,
		state.localPath, state.remotePath, state.keyPath, state.passphrase, state.security, state.options, state.securityInfo,
		state.settings, state.save, state.testConnection, state.delete, state.connect, state.close, state.newSite,
		state.titleMinimize, state.titleMaximize, state.titleClose,
		state.navConnections, state.navTransfers, state.navSync, state.navRemote, state.navLocal, state.navSettings,
		state.globalSearch, state.quickConnectTab, state.siteManagerTab, state.importExportTab,
		state.presetsTab, state.syncTab, state.automationTab, state.presetStandard, state.presetWebsite, state.presetBackup, state.presetMedia,
		state.syncBackup, state.syncSkip, state.syncConfirm,
	} {
		if control == 0 {
			return fmt.Errorf("Connections control initialization failed")
		}
	}
	limitEdit(state.name, 120)
	limitEdit(state.host, 253)
	limitEdit(state.port, 5)
	limitEdit(state.user, 1024)
	limitEdit(state.password, 8192)
	limitEdit(state.localPath, 32767)
	limitEdit(state.remotePath, 4096)
	limitEdit(state.keyPath, 32767)
	limitEdit(state.passphrase, 8192)
	cue(state.host, "server.yourdomain.com")
	cue(state.password, parent.tr("terminal.password"))
	cue(state.passphrase, parent.tr("cue.passphrase"))
	state.refreshOptionsSummary()
	state.refreshSyncOptions()
	state.refillRecentConnections()
	return nil
}

func (a *app) openSiteManager() {
	if a == nil || a.hwnd == 0 || a.connectionBusy || a.connected {
		return
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	siteManagerOnce.Do(func() {
		cursor, _, _ := loadCursorW.Call(0, 32512)
		icon, _, _ := loadIconW.Call(hinst, 1)
		class := wndClassEx{
			CbSize:     uint32(unsafe.Sizeof(wndClassEx{})),
			WndProc:    siteManagerProc,
			Instance:   hinst,
			Icon:       icon,
			Cursor:     cursor,
			Background: a.brush,
			ClassName:  wstr(siteManagerClass),
			IconSm:     icon,
		}
		registerClassExW.Call(uintptr(unsafe.Pointer(&class)))
	})

	logicalW, logicalH := 1590, 850
	screenW, _, _ := getSystemMetrics.Call(smCxScreen)
	screenH, _, _ := getSystemMetrics.Call(smCyScreen)
	screenLogicalW := a.unscale(int(screenW))
	screenLogicalH := a.unscale(int(screenH))
	if screenLogicalW > 0 && logicalW > screenLogicalW-24 {
		logicalW = screenLogicalW - 24
	}
	if screenLogicalH > 0 && logicalH > screenLogicalH-48 {
		logicalH = screenLogicalH - 48
	}
	if logicalW < 960 {
		logicalW = 960
	}
	if logicalH < 700 {
		logicalH = 700
	}
	pixelW, pixelH := a.scale(logicalW), a.scale(logicalH)
	x := (int(screenW) - pixelW) / 2
	y := (int(screenH) - pixelH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	words := nativeMenuWords(a.languageCode())
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr(siteManagerClass))),
		uintptr(unsafe.Pointer(wstr("Ghost FTP — "+words[5]))),
		siteWindowStyle,
		uintptr(x), uintptr(y), uintptr(pixelW), uintptr(pixelH),
		a.hwnd, 0, hinst, 0,
	)
	if hwnd == 0 {
		platform.ErrorDialog("Ghost FTP", words[5], a.tr("error.generic"))
		return
	}
	state := &siteManagerState{parent: a, hwnd: hwnd, profiles: append([]model.PublicProfile(nil), a.profiles...)}
	siteManagerStates.Store(hwnd, state)
	defer siteManagerStates.Delete(hwnd)
	applyDarkTitleBar(hwnd)
	if err := state.createControls(hinst); err != nil {
		destroyWindow.Call(hwnd)
		platform.ErrorDialog("Ghost FTP", words[5], err.Error())
		return
	}
	selectedID := a.selectedProfileID
	state.refillProfiles(selectedID)
	state.refillRecentConnections()
	enableWindow.Call(a.hwnd, 0)
	showWindow.Call(hwnd, swShow)
	var siteClient rect
	if ok, _, _ := getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&siteClient))); ok != 0 {
		state.layoutResponsive(a.unscale(int(siteClient.Right - siteClient.Left)))
	}
	updateWindow.Call(hwnd)

	var message msg
	for !state.closed {
		result, _, callErr := getMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) == -1 {
			if callErr != nil && callErr != syscall.Errno(0) {
				a.setStatus(callErr.Error())
			}
			break
		}
		if result == 0 {
			postQuitMessage.Call(0)
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
	enableWindow.Call(a.hwnd, 1)
	showWindow.Call(a.hwnd, swShow)

	if !state.connectAfter {
		switch state.postAction {
		case siteIDNavTransfers:
			a.focusTransferQueue()
		case siteIDNavSync:
			a.directoryComparisonCommand()
		case siteIDNavRemote:
			if a.remoteList != 0 && a.connected {
				sidebarSetFocus.Call(a.remoteList)
			}
		case siteIDNavLocal:
			if a.localList != 0 {
				sidebarSetFocus.Call(a.localList)
			}
		case siteIDNavSettings:
			a.openSettings()
		case siteIDGlobalSearch:
			a.masterMoreAction()
		}
		return
	}
	if state.selected > 0 && state.selected <= len(state.profiles) {
		selectedID = state.profiles[state.selected-1].ID
		a.selectedProfileID = selectedID
		a.applyProfiles(state.profiles, nil)
		for index, profile := range a.profiles {
			if profile.ID == selectedID {
				sendMessageW.Call(a.profilesCombo, cbSetCurSel, uintptr(index+1), 0)
				a.selectProfile()
				break
			}
		}
	}
	a.connectNow()
}
