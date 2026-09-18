//go:build windows

package desktop

import "unsafe"

// The approved Ghost FTP boards use application-owned chrome rather than the
// tall legacy Win32 caption. Keep the standard resizable frame and system
// behaviors, but let the client own the top band and window buttons.
const (
	ghostWindowStyle = 0x820F0000 // WS_POPUP | WS_CLIPCHILDREN | THICKFRAME | SYSMENU | MIN/MAX boxes

	idTitleMinimize = 9901
	idTitleMaximize = 9902
	idTitleClose    = 9903

	wmNcHitTest = 0x0084
	htClient    = 1
	htCaption   = 2
	chromeSWMinimize = 6
	chromeSWMaximize = 3
	chromeSWRestore  = 9
)

var (
	chromeIsZoomed      = user32.NewProc("IsZoomed")
	chromeScreenToClient = user32.NewProc("ScreenToClient")
)

type chromePoint struct {
	X int32
	Y int32
}

func signedWord(value uintptr) int32 {
	return int32(int16(value & 0xffff))
}

func signedHighWord(value uintptr) int32 {
	return int32(int16((value >> 16) & 0xffff))
}

func (a *app) ensureWindowChrome() {
	if a == nil || a.hwnd == 0 || a.titleClose != 0 {
		return
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	create := func(id int, label string, variant buttonVariant) uintptr {
		hwnd, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(wstr("BUTTON"))),
			uintptr(unsafe.Pointer(wstr(label))),
			uintptr(wsChild|wsVisible|bsOwnerDraw),
			0, 0, 1, 1,
			a.hwnd, uintptr(id), hinst, 0,
		)
		if hwnd != 0 && a.font != 0 {
			sendMessageW.Call(hwnd, wmSetFont, a.font, 1)
		}
		applyDarkControl(hwnd, "BUTTON")
		return a.registerButton(hwnd, "", label, variant)
	}
	a.titleMinimize = create(idTitleMinimize, "—", buttonSubtle)
	a.titleMaximize = create(idTitleMaximize, "□", buttonSubtle)
	a.titleClose = create(idTitleClose, "×", buttonSubtle)
}

func (a *app) layoutWindowChrome(width int) {
	if a == nil {
		return
	}
	a.ensureWindowChrome()
	const buttonW, buttonH = 42, 32
	x := width - buttonW*3 - 8
	if x < applicationContentLeft+260 {
		x = applicationContentLeft + 260
	}
	a.move(a.titleMinimize, x, 4, buttonW, buttonH)
	a.move(a.titleMaximize, x+buttonW, 4, buttonW, buttonH)
	a.move(a.titleClose, x+buttonW*2, 4, buttonW, buttonH)
}

func (a *app) windowChromeCommand(id int) bool {
	if a == nil || a.hwnd == 0 {
		return false
	}
	switch id {
	case idTitleMinimize:
		showWindow.Call(a.hwnd, chromeSWMinimize)
		return true
	case idTitleMaximize:
		zoomed, _, _ := chromeIsZoomed.Call(a.hwnd)
		if zoomed != 0 {
			showWindow.Call(a.hwnd, chromeSWRestore)
		} else {
			showWindow.Call(a.hwnd, chromeSWMaximize)
		}
		return true
	case idTitleClose:
		postMessageW.Call(a.hwnd, wmClose, 0, 0)
		return true
	}
	return false
}

// chromeHitTest preserves the standard resize border returned by DefWindowProc
// and turns only the empty top-left product band into a draggable caption. The
// search field and command buttons remain normal client controls.
func (a *app) chromeHitTest(lParam uintptr) uintptr {
	if a == nil || a.hwnd == 0 {
		return htClient
	}
	result, _, _ := defWindowProcW.Call(a.hwnd, wmNcHitTest, 0, lParam)
	if result != htClient {
		return result
	}
	point := chromePoint{X: signedWord(lParam), Y: signedHighWord(lParam)}
	chromeScreenToClient.Call(a.hwnd, uintptr(unsafe.Pointer(&point)))
	if point.Y >= 0 && point.Y < int32(a.scale(40)) && point.X >= 0 && point.X < int32(a.scale(applicationContentLeft)) {
		return htCaption
	}
	return htClient
}

func (a *app) cleanupWindowChrome() {
	if a == nil {
		return
	}
	for _, hwnd := range []uintptr{a.titleMinimize, a.titleMaximize, a.titleClose} {
		delete(a.buttons, hwnd)
	}
	a.titleMinimize = 0
	a.titleMaximize = 0
	a.titleClose = 0
}
