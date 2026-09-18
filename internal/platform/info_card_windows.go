//go:build windows

package platform

import (
	"sync"
	"syscall"
	"unsafe"
)

const infoCardIDClose = 1 // IDOK: the visible default Close/OK button.

type infoCardState struct {
	closed bool
}

type infoCardLayout struct {
	clientWidth   int
	clientHeight  int
	headingY      int
	headingHeight int
	bodyY         int
	bodyHeight    int
	dividerY      int
	buttonY       int
}

func infoCardLayoutForText(heading, body string, compact bool) infoCardLayout {
	if compact {
		return infoCardLayout{
			clientWidth: 560, clientHeight: 300,
			headingY: 24, headingHeight: 44,
			bodyY: 82, bodyHeight: 126,
			dividerY: 220, buttonY: 238,
		}
	}
	headingHeight := decisionCardClamp(decisionCardEstimatedLines(heading, 34)*31, 62, 112)
	bodyHeight := decisionCardClamp(decisionCardEstimatedLines(body, 70)*22, 250, 500)
	bodyY := 30 + headingHeight + 14
	dividerY := bodyY + bodyHeight + 18
	buttonY := dividerY + 16
	return infoCardLayout{
		clientWidth: 820, clientHeight: buttonY + 62,
		headingY: 30, headingHeight: headingHeight,
		bodyY: bodyY, bodyHeight: bodyHeight,
		dividerY: dividerY, buttonY: buttonY,
	}
}

var (
	infoCardStates sync.Map
	infoCardOnce   sync.Once
	infoCardClass  = "GhostFTP.InfoCardDialog"
	infoCardProc   = syscall.NewCallback(infoCardWndProc)
)

func infoCardWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if v, ok := infoCardStates.Load(hwnd); ok {
		state := v.(*infoCardState)
		switch message {
		case promptWMCommand:
			id := int(wParam & 0xffff)
			// Enter activates the visible IDOK button; Escape is translated by the
			// shared dialog manager into IDCANCEL even though no second button is
			// required on an information-only surface.
			if id == infoCardIDClose || id == promptIDCancel {
				promptDestroyWindow.Call(hwnd)
				return 0
			}
		case premiumWMCtlColorEdit, premiumWMCtlColorListBox, premiumWMCtlColorBtn, premiumWMCtlColorStatic:
			return premiumDialogControlColor(wParam)
		case promptWMClose:
			promptDestroyWindow.Call(hwnd)
			return 0
		case promptWMDestroy:
			// Only the main desktop window owns WM_QUIT. Closing About or another
			// informational card must return to the main message loop, not end it.
			state.closed = true
			return 0
		}
	}
	r, _, _ := promptDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

// InfoCardDialog displays application-owned informational content in the same
// Light/Dark native shell as the rest of Ghost FTP. Unlike TaskDialog, it does
// not depend on the current Windows system theme to choose its client surface.
func InfoCardDialog(title, heading, body, closeLabel string) {
	infoCardDialog(title, heading, body, closeLabel, false)
}

// CompactInfoDialog is used for concise application-owned diagnostics/status
// content. It keeps the same themed shell without forcing short information into
// the larger About-card geometry.
func CompactInfoDialog(title, heading, body, closeLabel string) {
	infoCardDialog(title, heading, body, closeLabel, true)
}

func infoCardDialog(title, heading, body, closeLabel string, compact bool) {
	if closeLabel == "" {
		closeLabel = "OK"
	}
	hinst, _, _ := promptGetModuleHandleW.Call(0)
	infoCardOnce.Do(func() {
		cursor, _, _ := promptLoadCursorW.Call(0, 32512)
		wc := promptWndClassEx{
			CbSize:     uint32(unsafe.Sizeof(promptWndClassEx{})),
			WndProc:    infoCardProc,
			Instance:   hinst,
			Cursor:     cursor,
			Background: premiumDialogBackgroundBrush(),
			ClassName:  promptWstr(infoCardClass),
		}
		promptRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	})

	const (
		wsOverlapped    = 0x00C80000
		wsChild         = 0x40000000
		wsVisible       = 0x10000000
		wsTabStop       = 0x00010000
		bsDefPushButton = 0x00000001
		ssEtchedHorz    = 0x00000010
	)

	layout := infoCardLayoutForText(heading, body, compact)
	clientWidth := layout.clientWidth
	clientHeight := layout.clientHeight
	owner := premiumDialogOwner()
	dpi := premiumDialogDPI(owner)
	outerWidth, outerHeight := premiumDialogOuterSize(clientWidth, clientHeight, wsOverlapped, 0, dpi)
	x, y := premiumDialogPosition(owner, outerWidth, outerHeight)
	hwnd, _, _ := promptCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(promptWstr(infoCardClass))),
		uintptr(unsafe.Pointer(promptWstr(title))),
		wsOverlapped,
		uintptr(x), uintptr(y), uintptr(outerWidth), uintptr(outerHeight),
		owner, 0, hinst, 0,
	)
	if hwnd == 0 {
		return
	}
	applyPremiumDialogWindow(hwnd)
	state := &infoCardState{}
	infoCardStates.Store(hwnd, state)
	defer infoCardStates.Delete(hwnd)
	restoreOwner := premiumModalOwner(owner)
	defer restoreOwner()

	bodyFont := premiumDialogFontForDPI(-15, dpi, 400)
	if bodyFont != 0 {
		defer promptDeleteObject.Call(bodyFont)
	}
	headingHeight := int32(-24)
	if compact {
		headingHeight = -20
	}
	headingFont := premiumDialogFontForDPI(headingHeight, dpi, 600)
	if headingFont != 0 {
		defer promptDeleteObject.Call(headingFont)
	}

	scale := func(value int) uintptr { return uintptr(premiumScale(value, dpi)) }
	makeControl := func(class, text string, style uint32, x, y, w, h, id int, controlFont uintptr) uintptr {
		child, _, _ := promptCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(promptWstr(class))),
			uintptr(unsafe.Pointer(promptWstr(text))),
			uintptr(wsChild|wsVisible|style),
			scale(x), scale(y), scale(w), scale(h),
			hwnd, uintptr(id), hinst, 0,
		)
		if child != 0 && controlFont != 0 {
			promptSendMessageW.Call(child, promptWMSetFont, controlFont, 1)
		}
		if child != 0 {
			applyPremiumDialogControl(child, class)
		}
		return child
	}

	contentX := 36
	contentW := layout.clientWidth - 72
	makeControl("STATIC", heading, 0, contentX, layout.headingY, contentW, layout.headingHeight, 0, headingFont)
	makeControl("STATIC", body, 0, contentX, layout.bodyY, contentW, layout.bodyHeight, 0, bodyFont)
	makeControl("STATIC", "", ssEtchedHorz, contentX, layout.dividerY, contentW, 2, 0, bodyFont)
	closeButton := makeControl(
		"BUTTON",
		closeLabel,
		wsTabStop|bsDefPushButton,
		layout.clientWidth-140,
		layout.buttonY,
		104,
		38,
		infoCardIDClose,
		bodyFont,
	)
	if closeButton != 0 {
		promptSetFocus.Call(closeButton)
	}

	promptShowWindow.Call(hwnd, 5)
	promptUpdateWindow.Call(hwnd)
	premiumRunDialogLoop(hwnd, func() bool { return state.closed })
}
