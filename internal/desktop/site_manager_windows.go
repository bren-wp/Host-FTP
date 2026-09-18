//go:build windows

package desktop

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/bren-wp/Host-FTP/internal/i18n"
	"github.com/bren-wp/Host-FTP/internal/model"
	"github.com/bren-wp/Host-FTP/internal/platform"
)

const (
	siteIDList       = 8101
	siteIDName       = 8102
	siteIDProtocol   = 8103
	siteIDHost       = 8104
	siteIDPort       = 8105
	siteIDUser       = 8106
	siteIDLocal      = 8107
	siteIDRemote     = 8108
	siteIDKey        = 8109
	siteIDSecurity   = 8110
	siteIDSave       = 8111
	siteIDDelete     = 8112
	siteIDConnect    = 8113
	siteIDClose      = 8114
	siteIDPassword   = 8115
	siteIDPassphrase = 8116
	siteIDDuplicate  = 8117
	siteIDSettings       = 8118
	siteIDNavConnections = 8120
	siteIDNavTransfers   = 8121
	siteIDNavSync        = 8122
	siteIDNavRemote      = 8123
	siteIDNavLocal       = 8124
	siteIDNavSettings    = 8125
	siteIDNewSite        = 8126
	siteIDGlobalSearch   = 8127
	siteIDQuickConnect    = 8128
	siteIDSiteManagerTab  = 8129
	siteIDImportExport    = 8130
	siteIDPresetsTab      = 8131
	siteIDSyncTab         = 8132
	siteIDAutomationTab   = 8133
	siteIDPresetStandard  = 8134
	siteIDPresetWebsite   = 8135
	siteIDPresetBackup    = 8136
	siteIDPresetMedia     = 8137

	siteLBSNotify           = 0x0001
	siteLBSNoIntegralHeight = 0x0100
	siteLBAddString         = 0x0180
	siteLBResetContent      = 0x0184
	siteLBSetCurSel         = 0x0186
	siteLBGetCurSel         = 0x0188
	siteLBNSelChange        = 1
	siteLBNDblClk           = 2
	siteBSDefPushButton     = 0x00000001
	siteWindowStyle         = 0x00C80000 // WS_CAPTION | WS_SYSMENU
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
	parent       *app
	hwnd         uintptr
	list         uintptr
	listBrush    uintptr
	name         uintptr
	protocol     uintptr
	host         uintptr
	port         uintptr
	user         uintptr
	password     uintptr
	localPath    uintptr
	remotePath   uintptr
	keyPath      uintptr
	passphrase   uintptr
	security     uintptr
	options      uintptr
	securityInfo uintptr
	settings     uintptr
	duplicate    uintptr
	save         uintptr
	delete       uintptr
	connect      uintptr
	close        uintptr
	profiles     []model.PublicProfile
	selected     int
	closed       bool
	connectAfter  bool
	postAction    int
	navConnections uintptr
	navTransfers   uintptr
	navSync        uintptr
	navRemote      uintptr
	navLocal       uintptr
	navSettings    uintptr
	newSite        uintptr
	globalSearch   uintptr
	brandIcon      uintptr
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
}

var (
	siteManagerStates sync.Map
	siteManagerOnce   sync.Once
	siteManagerClass  = "GhostFTP.SiteManager"
	siteManagerProc   = syscall.NewCallback(siteManagerWndProc)
)

func siteManagerWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	value, ok := siteManagerStates.Load(hwnd)
	if ok {
		state := value.(*siteManagerState)
		switch message {
		case wmPaint:
			state.paintReferenceConnections()
			return 0
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
			if id == siteIDProtocol && notify == cbnSelChange {
				state.syncProtocolPort()
				return 0
			}
			if notify == bnClicked {
				switch id {
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
				case siteIDNavSettings:
					state.postAction = siteIDNavSettings
					destroyWindow.Call(hwnd)
					return 0
				case siteIDNewSite:
					sendMessageW.Call(state.list, siteLBSetCurSel, 0, 0)
					state.loadSelection(0)
					return 0
				case siteIDGlobalSearch:
					state.postAction = siteIDGlobalSearch
					destroyWindow.Call(hwnd)
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
			setTextColor.Call(wParam, textColor())
			setBkColor.Call(wParam, panelColor())
			return state.parent.panelBrush
		case wmCtlColorEdit, wmCtlColorBtn:
			setTextColor.Call(wParam, textColor())
			setBkColor.Call(wParam, panelColor())
			return state.parent.panelBrush
		case wmClose:
			destroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
			for _, button := range []uintptr{
				state.duplicate, state.save, state.delete, state.connect, state.close, state.settings, state.newSite,
				state.navConnections, state.navTransfers, state.navSync, state.navRemote, state.navLocal, state.navSettings,
				state.globalSearch,
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
	result, _, _ := defWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
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
		"Parallel transfers\r\n%d simultaneous jobs\r\n\r\nUpload limit\r\n%s\r\n\r\nDownload limit\r\n%s\r\n\r\nRetry policy\r\n%d retries · %ds delay\r\n\r\nConflict policy\r\n%s\r\n\r\nDelete confirmation\r\n%s",
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
	if width > 0 && width < 1380 {
		// The approved four-column view needs desktop width. On smaller work areas
		// keep all connection editing actions reachable and collapse only the
		// read-only transfer/safety and saved-site side cards.
		showControls(false, state.options, state.securityInfo, state.newSite, state.list, state.duplicate, state.delete)
		state.parent.move(state.settings, 500, 690, 160, 38)
		return
	}
	showControls(true, state.options, state.securityInfo, state.newSite, state.list, state.duplicate, state.delete)
	state.parent.move(state.settings, 926, 738, 264, 42)
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
		variant := buttonSubtle
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
	mk("STATIC", "Private desktop · No account required", 0, 1280, 24, 280, 22, 0)
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
	motto := mk("STATIC", "FILES\r\nMOVE\r\nFREELY.\r\n\r\nYOU STAY\r\nIN CONTROL.", 0, 34, 610, 176, 148, 0)
	if motto != 0 && parent.smallFont != 0 {
		sendMessageW.Call(motto, wmSetFont, parent.smallFont, 1)
	}

	// Main connection card.
	heading("New Connection", 262, 54, 420)
	label("Connection Name", 262, 98, 220)
	state.name = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 262, 120, 618, 34, siteIDName)

	label(parent.tr("terminal.protocol"), 262, 170, 150)
	label("Host / Address", 502, 170, 190)
	label(parent.tr("terminal.port"), 796, 170, 70)
	state.protocol = mk("COMBOBOX", "", cbsDropDownList|wsTabStop|wsVScroll, 262, 192, 224, 240, siteIDProtocol)
	for _, spec := range protocolSpecs {
		sendMessageW.Call(state.protocol, cbAddString, 0, uintptr(unsafe.Pointer(wstr(protocolLabel(parent.languageCode(), spec.Value)))))
	}
	state.host = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 502, 192, 278, 34, siteIDHost)
	state.port = mk("EDIT", protocolSpecs[0].Port, wsBorder|wsTabStop|esAutoHScroll, 796, 192, 84, 34, siteIDPort)

	label(parent.tr("terminal.username"), 262, 244, 260)
	label(parent.tr("terminal.password"), 578, 244, 260)
	state.user = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 262, 266, 300, 34, siteIDUser)
	state.password = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll|esPassword, 578, 266, 302, 34, siteIDPassword)

	label(sitePathLabel(parent.languageCode(), false), 262, 318, 260)
	label(sitePathLabel(parent.languageCode(), true), 578, 318, 260)
	state.localPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 262, 340, 300, 34, siteIDLocal)
	state.remotePath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 578, 340, 302, 34, siteIDRemote)

	label(parent.tr("terminal.private_key"), 262, 392, 260)
	label(parent.tr("cue.passphrase"), 646, 392, 210)
	state.keyPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 262, 414, 368, 34, siteIDKey)
	state.passphrase = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll|esPassword, 646, 414, 234, 34, siteIDPassphrase)

	label(cleanConnectionSecurityTitle(parent.tr("sftp.security")), 262, 466, 618)
	state.security = mk("STATIC", "", wsBorder, 262, 488, 618, 82, siteIDSecurity)

	label("PROFILE BEHAVIOR", 262, 590, 618)
	profileNote := "Quick Connect uses credentials only for this session. Saving a profile asks before secrets are stored in the protected Windows credential layer."
	mk("STATIC", profileNote, 0, 262, 614, 618, 52, 0)

	state.save = parent.registerButton(mk("BUTTON", "Save as Profile", wsTabStop|bsOwnerDraw, 262, 738, 160, 42, siteIDSave), iconSave, "Save as Profile", buttonDefault)
	state.connect = parent.registerButton(mk("BUTTON", "Connect to Server", wsTabStop|siteBSDefPushButton|bsOwnerDraw, 646, 738, 234, 42, siteIDConnect), iconConnect, "Connect to Server", buttonAccent)
	state.close = parent.registerButton(mk("BUTTON", parent.tr("common.cancel"), wsTabStop|bsOwnerDraw, 500, 738, 132, 42, siteIDClose), iconCancel, parent.tr("common.cancel"), buttonSubtle)

	// Transfer & Sync settings card uses real persisted settings.
	heading("Transfer & Sync Options", 924, 54, 270)
	label("ACTIVE TRANSFER SETTINGS", 926, 100, 262)
	state.options = mk("STATIC", "", wsBorder, 926, 124, 264, 330, 0)
	label("SECURITY", 926, 472, 264)
	securityText := "SFTP host-key verification and FTPS certificate verification are enforced by the protocol engine. Saved secrets use the protected Windows credential layer."
	state.securityInfo = mk("STATIC", securityText, wsBorder, 926, 496, 264, 142, 0)
	state.settings = parent.registerButton(mk("BUTTON", "Open Transfer Settings", wsTabStop|bsOwnerDraw, 926, 738, 264, 42, siteIDSettings), iconSettings, "Open Transfer Settings", buttonDefault)

	// Saved Sites card on the right, matching the approved reference.
	heading("Saved Sites", 1240, 54, 190)
	state.newSite = parent.registerButton(mk("BUTTON", "New Site", wsTabStop|bsOwnerDraw, 1460, 50, 104, 38, siteIDNewSite), iconNewFolder, "New Site", buttonAccent)
	label("SAVED CONNECTIONS", 1240, 100, 300)
	state.list = mk("LISTBOX", "", wsBorder|wsTabStop|wsVScroll|siteLBSNotify|siteLBSNoIntegralHeight|siteLBSOwnerDrawFixed|siteLBSHasStrings, 1240, 124, 324, 514, siteIDList)
	if state.list != 0 {
		applySiteManagerNavigationTheme(state.list)
		state.listBrush, _, _ = createSolidBrush.Call(listColor())
	}
	duplicateLabel := siteManagerDuplicateLabel(parent.languageCode())
	state.duplicate = parent.registerButton(mk("BUTTON", duplicateLabel, wsTabStop|bsOwnerDraw, 1240, 650, 154, 38, siteIDDuplicate), iconCopy, duplicateLabel, buttonDefault)
	state.delete = parent.registerButton(mk("BUTTON", parent.tr("profile.delete"), wsTabStop|bsOwnerDraw, 1404, 650, 160, 38, siteIDDelete), iconDelete, parent.tr("profile.delete"), buttonDanger)

	for _, control := range []uintptr{
		state.list, state.duplicate, state.name, state.protocol, state.host, state.port, state.user, state.password,
		state.localPath, state.remotePath, state.keyPath, state.passphrase, state.security, state.options, state.securityInfo,
		state.settings, state.save, state.delete, state.connect, state.close, state.newSite,
		state.navConnections, state.navTransfers, state.navSync, state.navRemote, state.navLocal, state.navSettings,
		state.globalSearch,
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
	cue(state.host, "example.com")
	cue(state.password, parent.tr("terminal.password"))
	cue(state.passphrase, parent.tr("cue.passphrase"))
	state.refreshOptionsSummary()
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
	pixelW, pixelH := a.scale(logicalW), a.scale(logicalH)
	screenW, _, _ := getSystemMetrics.Call(smCxScreen)
	screenH, _, _ := getSystemMetrics.Call(smCyScreen)
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
	enableWindow.Call(a.hwnd, 0)
	showWindow.Call(hwnd, swShow)
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
