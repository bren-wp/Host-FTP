//go:build windows

package desktop

import (
	"fmt"
	"unsafe"
)

const (
	cdisSelected = 0x0001
	cdisDisabled = 0x0004
	cdisHot      = 0x0040

	cddsSubItem         = 0x00020000
	cddsSubItemPrepaint = cddsItemPrepaint | cddsSubItem
	lvmGetSubItemRect   = lvmFirst + 56
	lvirBounds          = 0
	dtRightText         = 0x00000002
)

// nmListViewCustomDraw is the Win32 NMLVCUSTOMDRAW prefix used by the report
// views in the main Ghost FTP workspace. Owning the row palette removes the
// stock Explorer selection/background treatment that visibly diverged from the
// approved 1672x941 reference board while preserving the native ListView's
// accessibility, keyboard navigation, sorting and real file/queue behavior.
type nmListViewCustomDraw struct {
	Draw       nmCustomDrawStruct
	ClrText    uint32
	ClrTextBk  uint32
	SubItem    int32
	ItemType   uint32
	ClrFace    uint32
	IconEffect int32
	IconPhase  int32
	PartID     int32
	StateID    int32
	TextRect   rect
	Align      uint32
}

func listViewCustomDrawFromLParam(p uintptr) nmListViewCustomDraw {
	var draw nmListViewCustomDraw
	if p != 0 {
		rtlMoveMemory.Call(uintptr(unsafe.Pointer(&draw)), p, unsafe.Sizeof(draw))
	}
	return draw
}

func listViewCustomDrawToLParam(p uintptr, draw nmListViewCustomDraw) {
	if p != 0 {
		rtlMoveMemory.Call(p, uintptr(unsafe.Pointer(&draw)), unsafe.Sizeof(draw))
	}
}

func (a *app) isWorkspaceList(hwnd uintptr) bool {
	if a == nil || hwnd == 0 {
		return false
	}
	return hwnd == a.localList || hwnd == a.remoteList || hwnd == a.transferList
}

func workspaceListHotColor() uintptr {
	if activeThemeIsDark() {
		return rgb(0x0E, 0x23, 0x35)
	}
	return rgb(0xE4, 0xEE, 0xF8)
}

func workspaceListRowColors(state uint32) (background, foreground uintptr) {
	background = listColor()
	foreground = textColor()
	switch {
	case state&cdisDisabled != 0:
		foreground = mutedColor()
	case state&cdisSelected != 0:
		background = selectionColor()
	case state&cdisHot != 0:
		background = workspaceListHotColor()
	}
	return
}

func workspaceListActualDrawState(list uintptr, item uintptr, drawState uint32) uint32 {
	if list == 0 {
		return drawState
	}
	// NMLVCUSTOMDRAW can report CDIS_SELECTED for an entire full-row paint pass
	// under PrintWindow/remote-session rendering. Ask the ListView for the real
	// LVIS_SELECTED bit so only genuinely selected rows receive the blue reference
	// highlight in screenshots and on users' desktops.
	drawState &^= cdisSelected
	selected, _, _ := sendMessageW.Call(list, lvmGetItemState, item, lvisSelected)
	if selected&lvisSelected != 0 {
		drawState |= cdisSelected
	}
	return drawState
}

func (a *app) transferProgressPercent(index int) (float64, string, bool, bool) {
	if a == nil || index < 0 || index >= len(a.transferViewJobs) {
		return 0, "", false, false
	}
	job := a.transferViewJobs[index]
	progress := job.Progress
	if job.BytesTotal > 0 {
		progress = float64(job.BytesTransferred) * 100 / float64(job.BytesTotal)
	} else if progress > 0 && progress <= 1 {
		progress *= 100
	}
	completed := job.Status == "done" || job.Status == "skipped"
	if completed {
		progress = 100
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	return progress, fmt.Sprintf("%.0f%%", progress), completed, true
}

func (a *app) transferProgressCellRect(item, subItem int, fallback rect) rect {
	if a == nil || a.transferList == 0 {
		return fallback
	}
	r := rect{Left: lvirBounds, Top: int32(subItem)}
	if ok, _, _ := sendMessageW.Call(
		a.transferList,
		lvmGetSubItemRect,
		uintptr(item),
		uintptr(unsafe.Pointer(&r)),
	); ok == 0 {
		return fallback
	}
	return r
}

func (a *app) drawTransferProgressCell(hdc uintptr, item int, rowState uint32, fallback rect) bool {
	progress, label, completed, ok := a.transferProgressPercent(item)
	if !ok || hdc == 0 {
		return false
	}
	cell := a.transferProgressCellRect(item, 2, fallback)
	background, _ := workspaceListRowColors(rowState)
	backgroundBrush, _, _ := createSolidBrush.Call(background)
	fillRectW.Call(hdc, uintptr(unsafe.Pointer(&cell)), backgroundBrush)
	if backgroundBrush != 0 {
		deleteObject.Call(backgroundBrush)
	}

	height := cell.Bottom - cell.Top
	if height <= 0 {
		return false
	}
	bar := cell
	bar.Left += int32(a.scale(9))
	bar.Right -= int32(a.scale(48))
	bar.Top += (height - int32(a.scale(10))) / 2
	bar.Bottom = bar.Top + int32(a.scale(10))
	if bar.Right <= bar.Left {
		return false
	}

	trackBrush, _, _ := createSolidBrush.Call(panelColor())
	trackPen, _, _ := createPen.Call(psSolid, 1, borderColor())
	oldBrush, _, _ := selectObject.Call(hdc, trackBrush)
	oldPen, _, _ := selectObject.Call(hdc, trackPen)
	radius := uintptr(a.scale(10))
	roundRect.Call(hdc, uintptr(bar.Left), uintptr(bar.Top), uintptr(bar.Right), uintptr(bar.Bottom), radius, radius)
	selectObject.Call(hdc, oldBrush)
	selectObject.Call(hdc, oldPen)
	if trackBrush != 0 {
		deleteObject.Call(trackBrush)
	}
	if trackPen != 0 {
		deleteObject.Call(trackPen)
	}

	if progress > 0 {
		fill := bar
		width := float64(bar.Right - bar.Left)
		fill.Right = fill.Left + int32(width*progress/100)
		minWidth := int32(a.scale(6))
		if fill.Right-fill.Left < minWidth {
			fill.Right = fill.Left + minWidth
		}
		if fill.Right > bar.Right {
			fill.Right = bar.Right
		}
		fillBrushColor := accentStrongColor()
		if completed {
			fillBrushColor = successColor()
		}
		fillBrush, _, _ := createSolidBrush.Call(fillBrushColor)
		old, _, _ := selectObject.Call(hdc, fillBrush)
		roundRect.Call(hdc, uintptr(fill.Left), uintptr(fill.Top), uintptr(fill.Right), uintptr(fill.Bottom), radius, radius)
		selectObject.Call(hdc, old)
		if fillBrush != 0 {
			deleteObject.Call(fillBrush)
		}
		if !completed {
			a.fillAccentGradient(hdc, fill, false)
		}
	}

	textRect := cell
	textRect.Left = bar.Right + int32(a.scale(7))
	textRect.Right -= int32(a.scale(5))
	setBkMode.Call(hdc, transparentBkMode)
	setTextColor.Call(hdc, textColor())
	oldFont := uintptr(0)
	if a.smallFont != 0 {
		oldFont, _, _ = selectObject.Call(hdc, a.smallFont)
	}
	text := wstr(label)
	drawTextW.Call(
		hdc,
		uintptr(unsafe.Pointer(text)),
		uintptr(len([]rune(label))),
		uintptr(unsafe.Pointer(&textRect)),
		dtRightText|dtVCenter|dtSingleLine|dtNoPrefix,
	)
	if oldFont != 0 {
		selectObject.Call(hdc, oldFont)
	}
	return true
}

// drawWorkspaceList keeps the native report view functional but owns its
// high-value visual states. The result is the same deep list surface, cyan/blue
// hover and selected-row language used by the supplied Ghost FTP boards instead
// of a system-theme-dependent ListView highlight. Transfer progress is rendered
// as a real data-driven bar inside the Progress column.
func (a *app) drawWorkspaceList(lParam uintptr) uintptr {
	draw := listViewCustomDrawFromLParam(lParam)
	switch draw.Draw.DrawStage {
	case cddsPrepaint:
		return cdrfNotifyItemDraw
	case cddsItemPrepaint:
		draw.Draw.ItemState = workspaceListActualDrawState(draw.Draw.Hdr.HwndFrom, draw.Draw.ItemSpec, draw.Draw.ItemState)
		background, foreground := workspaceListRowColors(draw.Draw.ItemState)
		draw.ClrText = uint32(foreground)
		draw.ClrTextBk = uint32(background)
		listViewCustomDrawToLParam(lParam, draw)
		if draw.Draw.Hdr.HwndFrom == a.transferList {
			return cdrfNotifyItemDraw
		}
		return cdrfNewFont
	case cddsSubItemPrepaint:
		draw.Draw.ItemState = workspaceListActualDrawState(draw.Draw.Hdr.HwndFrom, draw.Draw.ItemSpec, draw.Draw.ItemState)
		background, foreground := workspaceListRowColors(draw.Draw.ItemState)
		draw.ClrText = uint32(foreground)
		draw.ClrTextBk = uint32(background)
		listViewCustomDrawToLParam(lParam, draw)
		if draw.Draw.Hdr.HwndFrom == a.transferList && draw.SubItem == 2 {
			if a.drawTransferProgressCell(draw.Draw.HDC, int(draw.Draw.ItemSpec), draw.Draw.ItemState, draw.Draw.Rc) {
				return cdrfSkipDefault
			}
		}
		return cdrfNewFont
	default:
		return 0
	}
}
