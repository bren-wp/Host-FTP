//go:build windows

package desktop

import "unsafe"

const (
	cdisSelected = 0x0001
	cdisDisabled = 0x0004
	cdisHot      = 0x0040
)

// nmListViewCustomDraw is the Win32 NMLVCUSTOMDRAW prefix used by the report
// views in the main Ghost FTP workspace. Owning the row palette removes the
// stock Explorer selection/background treatment that visibly diverged from the
// approved 1672x941 reference board while preserving the native ListView's
// accessibility, keyboard navigation, sorting and real file/queue behavior.
type nmListViewCustomDraw struct {
	Draw        nmCustomDrawStruct
	ClrText     uint32
	ClrTextBk   uint32
	SubItem     int32
	ItemType    uint32
	ClrFace     uint32
	IconEffect  int32
	IconPhase   int32
	PartID      int32
	StateID     int32
	TextRect    rect
	Align       uint32
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

// drawWorkspaceList keeps the native report view functional but owns its
// high-value visual states. The result is the same deep list surface, cyan/blue
// hover and selected-row language used by the supplied Ghost FTP boards instead
// of a system-theme-dependent ListView highlight.
func (a *app) drawWorkspaceList(lParam uintptr) uintptr {
	draw := listViewCustomDrawFromLParam(lParam)
	switch draw.Draw.DrawStage {
	case cddsPrepaint:
		return cdrfNotifyItemDraw
	case cddsItemPrepaint:
		background := listColor()
		foreground := textColor()
		switch {
		case draw.Draw.ItemState&cdisDisabled != 0:
			foreground = mutedColor()
		case draw.Draw.ItemState&cdisSelected != 0:
			background = selectionColor()
		case draw.Draw.ItemState&cdisHot != 0:
			background = workspaceListHotColor()
		}
		draw.ClrText = uint32(foreground)
		draw.ClrTextBk = uint32(background)
		listViewCustomDrawToLParam(lParam, draw)
		return cdrfNewFont
	default:
		return 0
	}
}
