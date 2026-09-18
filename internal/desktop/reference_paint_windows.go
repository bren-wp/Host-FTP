//go:build windows

package desktop

import "unsafe"

// drawReferenceCard renders the low-contrast rounded containers used throughout
// the approved Ghost FTP mockups. Child controls remain native Windows controls;
// the parent paints only the structural surfaces behind them.
func (a *app) drawReferenceCard(hdc uintptr, x, y, width, height, radius int, fill, border uintptr) {
	if a == nil || hdc == 0 || width <= 0 || height <= 0 {
		return
	}
	brush, _, _ := createSolidBrush.Call(fill)
	pen, _, _ := createPen.Call(psSolid, 1, border)
	oldBrush, _, _ := selectObject.Call(hdc, brush)
	oldPen, _, _ := selectObject.Call(hdc, pen)
	left := a.scale(x)
	top := a.scale(y)
	right := a.scale(x + width)
	bottom := a.scale(y + height)
	r := a.scale(radius)
	roundRect.Call(hdc, uintptr(left), uintptr(top), uintptr(right), uintptr(bottom), uintptr(r), uintptr(r))
	selectObject.Call(hdc, oldBrush)
	selectObject.Call(hdc, oldPen)
	if brush != 0 {
		deleteObject.Call(brush)
	}
	if pen != 0 {
		deleteObject.Call(pen)
	}
}

func (a *app) paintReferenceWorkspace() {
	if a == nil || a.hwnd == 0 {
		return
	}
	var ps paintStruct
	hdc, _, _ := beginPaint.Call(a.hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer endPaint.Call(a.hwnd, uintptr(unsafe.Pointer(&ps)))

	var client rect
	if ok, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		return
	}
	background, _, _ := createSolidBrush.Call(windowColor())
	fillRectW.Call(hdc, uintptr(unsafe.Pointer(&client)), background)
	if background != 0 {
		deleteObject.Call(background)
	}

	width := a.unscale(int(client.Right - client.Left))
	height := a.unscale(int(client.Bottom - client.Top))
	if width <= 0 || height <= 0 {
		return
	}

	// Integrated application rail.
	a.drawReferenceCard(
		hdc,
		10, 10,
		applicationSidebarWidth+18,
		height-20,
		18,
		panelColor(),
		borderColor(),
	)

	contentLeft := applicationContentLeft
	contentRight := width - premiumOuterGap
	contentWidth := contentRight - contentLeft
	if contentWidth < 600 {
		return
	}

	// Global search + action strip.
	a.drawReferenceCard(hdc, contentLeft-8, 10, contentWidth+8, 100, 16, panelColor(), borderColor())

	paneGap := 14
	paneW := (contentWidth - paneGap) / 2
	statusY, _ := statusBandGeometry(height)
	queueH := clampInt(height/5, 128, 184)
	queueY := statusY - queueH - 10
	queueButtonsY := queueY - 38
	queueLabelY := queueButtonsY - 25
	paneTop := 118
	paneBottom := queueLabelY - 9
	paneH := paneBottom - paneTop
	if paneH > 120 {
		a.drawReferenceCard(hdc, contentLeft-8, paneTop, paneW+8, paneH, 16, panelColor(), borderColor())
		a.drawReferenceCard(hdc, contentLeft+paneW+paneGap-8, paneTop, paneW+8, paneH, 16, panelColor(), borderColor())
	}

	queueCardY := queueLabelY - 10
	queueCardH := statusY - queueCardY - 6
	if queueCardH > 80 {
		a.drawReferenceCard(hdc, contentLeft-8, queueCardY, contentWidth+8, queueCardH, 16, panelColor(), borderColor())
	}
}

func (state *siteManagerState) paintReferenceConnections() {
	if state == nil || state.hwnd == 0 || state.parent == nil {
		return
	}
	var ps paintStruct
	hdc, _, _ := beginPaint.Call(state.hwnd, uintptr(unsafe.Pointer(&ps)))
	if hdc == 0 {
		return
	}
	defer endPaint.Call(state.hwnd, uintptr(unsafe.Pointer(&ps)))

	var client rect
	if ok, _, _ := getClientRect.Call(state.hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		return
	}
	background, _, _ := createSolidBrush.Call(windowColor())
	fillRectW.Call(hdc, uintptr(unsafe.Pointer(&client)), background)
	if background != 0 {
		deleteObject.Call(background)
	}

	// Approved Connections composition: product rail and connection editor are
	// always present. Optional reference side cards render only when the logical
	// client width can contain them without clipping.
	state.parent.drawReferenceCard(hdc, 12, 12, 220, 818, 18, panelColor(), borderColor())
	state.parent.drawReferenceCard(hdc, 244, 44, 654, 770, 16, panelColor(), borderColor())

	logicalWidth := state.parent.unscale(int(client.Right - client.Left))
	if logicalWidth >= 1580 {
		state.parent.drawReferenceCard(hdc, 908, 44, 300, 510, 16, panelColor(), borderColor())
		state.parent.drawReferenceCard(hdc, 908, 564, 300, 250, 16, panelColor(), borderColor())
		state.parent.drawReferenceCard(hdc, 1222, 44, 354, 462, 16, panelColor(), borderColor())
		state.parent.drawReferenceCard(hdc, 1222, 516, 354, 298, 16, panelColor(), borderColor())
	}
}
