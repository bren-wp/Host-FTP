//go:build windows

package desktop

import (
	"sync"
	"unsafe"
)

const (
	ssIcon      = 0x00000003
	stmSetImage = 0x0172
	imageIcon   = 1
	lrShared    = 0x00008000
)

var (
	brandLogoWindows sync.Map
	loadImageW       = user32.NewProc("LoadImageW")
)

// ensureBrandLogo uses dedicated PE icon group 2: the transparent Ghost mark
// designed to sit beside the wordmark. Group 1 remains the rounded-square
// Windows application icon used by Explorer and the taskbar.
func (a *app) ensureBrandLogo() uintptr {
	if a == nil || a.hwnd == 0 {
		return 0
	}
	if value, ok := brandLogoWindows.Load(a.hwnd); ok {
		if logo, ok := value.(uintptr); ok && logo != 0 {
			return logo
		}
	}

	hinst, _, _ := getModuleHandleW.Call(0)
	iconSize := a.scale(48)
	icon, _, _ := loadImageW.Call(hinst, 2, imageIcon, uintptr(iconSize), uintptr(iconSize), lrShared)
	if icon == 0 {
		// LoadIcon returns a shared icon resource too, so neither path requires
		// Ghost FTP to destroy a handle that Windows owns.
		icon, _, _ = loadIconW.Call(hinst, 2)
	}
	if icon == 0 {
		return 0
	}

	logo, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr("STATIC"))),
		0,
		uintptr(wsChild|wsVisible|ssIcon),
		0, 0, uintptr(iconSize), uintptr(iconSize),
		a.hwnd, 0, hinst, 0,
	)
	if logo == 0 {
		return 0
	}
	sendMessageW.Call(logo, stmSetImage, imageIcon, icon)
	brandLogoWindows.Store(a.hwnd, logo)
	return logo
}

func (a *app) refineBrandHeader() {
	logo := a.ensureBrandLogo()
	if logo == 0 {
		return
	}
	a.move(logo, 14, 10, 48, 48)

	// Reserve a fixed wordmark gutter large enough for the real Segoe UI bold
	// rendering seen on Windows runners. Keep the subtitle responsive inside
	// the remaining header space rather than allowing it to overlap the title.
	const titleX, ghostWidth, ftpWidth, wordmarkGap, subtitleX = 72, 92, 54, 2, 230
	a.move(a.brandTitle, titleX, 10, ghostWidth, 35)
	a.move(a.brandFTP, titleX+ghostWidth+wordmarkGap, 10, ftpWidth, 35)
	var client rect
	if ok, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); ok != 0 {
		logicalWidth := a.unscale(int(client.Right - client.Left))
		subtitleWidth := logicalWidth - subtitleX - 320
		if subtitleWidth < 120 {
			subtitleWidth = 120
		}
		a.move(a.brandSubtitle, subtitleX, 17, subtitleWidth, 20)
	}
}

func styleWorkspaceCombos(combos ...uintptr) {
	for _, combo := range combos {
		applyDarkControl(combo, "COMBOBOX")
	}
}

func (a *app) roundWorkspaceControl(hwnd uintptr, radius int) {
	if a == nil || hwnd == 0 || radius <= 0 {
		return
	}
	var bounds rect
	if ok, _, _ := getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&bounds))); ok == 0 {
		return
	}
	width := bounds.Right - bounds.Left
	height := bounds.Bottom - bounds.Top
	if width <= 1 || height <= 1 {
		return
	}
	r := a.scale(radius)
	region, _, _ := createRoundRectRgn.Call(
		0, 0,
		uintptr(width+1), uintptr(height+1),
		uintptr(r), uintptr(r),
	)
	if region == 0 {
		return
	}
	// SetWindowRgn owns the region after a successful call.
	if ok, _, _ := setWindowRgn.Call(hwnd, region, 1); ok == 0 {
		deleteObject.Call(region)
	}
}

func (a *app) roundReferenceWorkspaceControls() {
	if a == nil {
		return
	}
	for _, hwnd := range []uintptr{
		a.localPath, a.remotePath,
		a.profilesCombo, a.protocol, a.host, a.port, a.user, a.pass,
		a.keyPath, a.passphrase,
	} {
		a.roundWorkspaceControl(hwnd, 9)
	}
	for _, hwnd := range []uintptr{a.localList, a.remoteList, a.transferList} {
		a.roundWorkspaceControl(hwnd, 10)
	}
}

// stabilizeWorkspaceChrome is intentionally idempotent because state changes
// and resizes both flow through refineWorkspaceLayout. It keeps native child
// controls aligned with the active Ghost FTP appearance on both light and dark
// Windows desktops without duplicating theme state per control.
func (a *app) stabilizeWorkspaceChrome() {
	if a == nil {
		return
	}
	styleWorkspaceCombos(a.languageCombo, a.profilesCombo, a.protocol)
	for _, list := range []uintptr{a.localList, a.remoteList, a.transferList} {
		// Native Header controls notify their immediate ListView parent. Install
		// that route before applying the list theme so header text and fill stay
		// under Ghost FTP control in either supported appearance.
		installWorkspaceHeaderDraw(a, list)
		styleWorkspaceList(list)
	}
	// Remote operations remain disabled by updateActionControls. The list itself
	// stays enabled so Windows does not substitute its disabled-control palette.
	setControlEnabled(a.remoteList, true)
	a.refineBrandHeader()
	a.roundReferenceWorkspaceControls()
}
