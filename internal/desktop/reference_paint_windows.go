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

func (a *app) drawReferenceDivider(hdc uintptr, x, y, width int, color uintptr) {
	if a == nil || hdc == 0 || width <= 0 {
		return
	}
	line := rect{
		Left:   int32(a.scale(x)),
		Top:    int32(a.scale(y)),
		Right:  int32(a.scale(x + width)),
		Bottom: int32(a.scale(y + 1)),
	}
	brush, _, _ := createSolidBrush.Call(color)
	fillRectW.Call(hdc, uintptr(unsafe.Pointer(&line)), brush)
	if brush != 0 {
		deleteObject.Call(brush)
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
		14, 10,
		applicationSidebarWidth+18,
		height-35,
		18,
		panelColor(),
		borderColor(),
	)
	a.drawReferenceDivider(hdc, applicationSidebarControlX, 392, applicationSidebarControlW, borderColor())

	contentLeft := applicationContentLeft
	contentRight := width - premiumOuterGap
	contentWidth := contentRight - contentLeft
	if contentWidth < 600 {
		return
	}

	// The approved main board keeps the title/search band open rather than
	// wrapping the whole header and toolbar in a large card. Two restrained
	// dividers preserve structure without introducing a legacy container.
	a.drawReferenceDivider(hdc, contentLeft, 78, contentWidth, borderColor())
	a.drawReferenceDivider(hdc, contentLeft, 142, contentWidth, borderColor())

	paneGap := 32
	paneW := (contentWidth - paneGap) / 2
	statusY, _ := statusBandGeometry(height)
	queueH := clampInt(height/7, 124, 136)
	queueY := statusY - queueH - 14
	queueLabelY := queueY - 42
	paneTop := 148
	paneBottom := queueLabelY - 9
	paneH := paneBottom - paneTop
	if paneH > 120 {
		a.drawReferenceCard(hdc, contentLeft-2, paneTop, paneW+2, paneH, 16, panelColor(), borderColor())
		a.drawReferenceCard(hdc, contentLeft+paneW+paneGap-2, paneTop, paneW+2, paneH, 16, panelColor(), borderColor())
	}
	if paneH > 180 {
		footerLineY := paneBottom - 34
		a.drawReferenceDivider(hdc, contentLeft+4, footerLineY, paneW-8, borderColor())
		a.drawReferenceDivider(hdc, contentLeft+paneW+paneGap+4, footerLineY, paneW-8, borderColor())
	}

	queueCardY := queueLabelY - 10
	queueCardH := statusY - queueCardY - 6
	if queueCardH > 80 {
		a.drawReferenceCard(hdc, contentLeft-2, queueCardY, contentWidth+2, queueCardH, 16, panelColor(), borderColor())
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
	// always present. Optional reference side cards render only when both width
	// and height can contain the full board without clipping.
	logicalWidth := state.parent.unscale(int(client.Right - client.Left))
	logicalHeight := state.parent.unscale(int(client.Bottom - client.Top))
	compact := logicalWidth < 1580 || logicalHeight < 820

	railHeight := logicalHeight - 91
	if railHeight < 620 {
		railHeight = 620
	}
	const contentTop = 90
	contentBottom := logicalHeight - 92
	editorHeight := contentBottom - contentTop
	if editorHeight < 620 {
		editorHeight = 620
	}
	state.parent.drawReferenceCard(hdc, 16, 12, 220, railHeight, 18, panelColor(), borderColor())
	state.parent.drawReferenceCard(hdc, 248, contentTop, 660, editorHeight, 16, panelColor(), borderColor())

	if !compact {
		state.parent.drawReferenceCard(hdc, 918, contentTop, 306, 464, 16, panelColor(), borderColor())
		state.parent.drawReferenceCard(hdc, 918, 564, 306, 285, 16, panelColor(), borderColor())
		state.parent.drawReferenceCard(hdc, 1236, contentTop, 352, 464, 16, panelColor(), borderColor())
		state.parent.drawReferenceCard(hdc, 1236, 564, 352, 285, 16, panelColor(), borderColor())
		footerY := logicalHeight - 77
		state.parent.drawReferenceCard(hdc, 16, footerY, 1572, 42, 12, panelColor(), borderColor())
	}
}
