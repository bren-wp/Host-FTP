//go:build windows

package platform

import (
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	// Use the standard Win32 dialog command IDs so IsDialogMessageW can route
	// Enter/Escape consistently even though this surface is application-owned.
	settingsIDApply      = 1 // IDOK
	settingsIDCancel     = 2 // IDCANCEL
	settingsIDReset      = 3
	settingsIDLanguage   = 4100
	settingsIDAppearance = 4101
	settingsIDNumberBase = 4110
	settingsIDConflict   = 4120
	settingsIDConfirm    = 4121
	settingsIDError      = 4124
	settingsIDUpdate     = 4125
	settingsIDDownload   = 4126
	settingsIDWebsite    = 4128

	settingsCBAdd       = 0x0143
	settingsCBGet       = 0x0147
	settingsCBSet       = 0x014E
	settingsBMGetCheck  = 0x00F0
	settingsBMSetCheck  = 0x00F1
	settingsBSTChecked  = 1
	settingsEMSetSel    = 0x00B1
	settingsESNumber    = 0x2000
	settingsAutoCheck   = 0x00000003
	settingsWSBorder    = 0x00800000
	settingsWSChild     = 0x40000000
	settingsWSVisible   = 0x10000000
	settingsWSTabStop   = 0x00010000
	settingsWSVScroll   = 0x00200000
	settingsCBSDropList = 0x0003
	settingsDefButton   = 0x00000001
	settingsEtchedHorz  = 0x00000010
	settingsSSIcon      = 0x00000003
	settingsSTMSetImage = 0x0172
	settingsImageIcon   = 1
)

// SettingsDialogNumber describes one bounded integer preference. InvalidText is
// caller-owned so validation remains localized without coupling platform code to
// Ghost FTP's translation catalog.
type SettingsDialogNumber struct {
	Label       string
	Value       int
	Min         int
	Max         int
	InvalidText string
}

// SettingsDialogConfig contains presentation text and current values for the
// application-owned Windows settings surface. The desktop layer owns all labels
// and business meaning; this package owns only native rendering and interaction.
type SettingsDialogConfig struct {
	Title                  string
	Heading                string
	Intro                  string
	LanguageLabel          string
	LanguageOptions        []string
	LanguageIndex          int
	AppearanceLabel        string
	AppearanceOptions      []string
	AppearanceIndex        int
	Numbers                []SettingsDialogNumber
	ConflictLabel          string
	ConflictOptions        []string
	ConflictIndex          int
	ConfirmDelete          string
	ConfirmDeleteOn        bool
	Footer                 string
	ApplyLabel             string
	CancelLabel            string
	ResetLabel             string
	UpdateLabel            string
	DownloadLabel          string
	WebsiteLabel           string
	DefaultLanguageIndex   int
	DefaultAppearanceIndex int
	DefaultNumbers         []int
	DefaultConflictIndex   int
	DefaultConfirmDelete   bool
}

type SettingsDialogResult struct {
	LanguageIndex   int
	AppearanceIndex int
	Action          string
	Numbers         []int
	ConflictIndex   int
	ConfirmDelete   bool
}

type settingsDialogState struct {
	language   uintptr
	appearance uintptr
	numbers    []uintptr
	conflict   uintptr
	confirm    uintptr
	errorLabel uintptr
	config     SettingsDialogConfig
	result     SettingsDialogResult
	accepted   bool
	closed     bool
}

var (
	settingsStates         sync.Map
	settingsOnce           sync.Once
	settingsClass          = "GhostFTP.SettingsDialog"
	settingsProc           = syscall.NewCallback(settingsWndProc)
	settingsSetWindowTextW = user32.NewProc("SetWindowTextW")
)

func settingsSetText(hwnd uintptr, text string) {
	if hwnd == 0 {
		return
	}
	settingsSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(promptWstr(text))))
}

func settingsComboIndex(hwnd uintptr, fallback int) int {
	if hwnd == 0 {
		return fallback
	}
	value, _, _ := promptSendMessageW.Call(hwnd, settingsCBGet, 0, 0)
	if int32(value) < 0 {
		return fallback
	}
	return int(value)
}

func settingsWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) (result uintptr) {
	// Do not let a rendering/validation panic escape a syscall callback and take
	// the entire desktop client down. Falling back to DefWindowProc keeps the
	// modal responsive and lets the user cancel/reopen it safely.
	defer func() {
		if recover() != nil {
			result, _, _ = promptDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
		}
	}()
	if v, ok := settingsStates.Load(hwnd); ok {
		state := v.(*settingsDialogState)
		switch message {
		case promptWMCommand:
			id := int(wParam & 0xffff)
			switch id {
			case settingsIDUpdate:
				state.result.Action = "update"
				promptDestroyWindow.Call(hwnd)
				return 0
			case settingsIDDownload:
				state.result.Action = "download"
				promptDestroyWindow.Call(hwnd)
				return 0
			case settingsIDWebsite:
				state.result.Action = "website"
				promptDestroyWindow.Call(hwnd)
				return 0
			case settingsIDApply:
				values := make([]int, len(state.numbers))
				for index, edit := range state.numbers {
					field := state.config.Numbers[index]
					value, err := strconv.Atoi(strings.TrimSpace(promptText(edit)))
					if err != nil || value < field.Min || value > field.Max {
						message := field.InvalidText
						if message == "" {
							message = field.Label
						}
						settingsSetText(state.errorLabel, message)
						promptSetFocus.Call(edit)
						promptSendMessageW.Call(edit, settingsEMSetSel, 0, ^uintptr(0))
						return 0
					}
					values[index] = value
				}

				state.result.LanguageIndex = settingsComboIndex(state.language, state.config.LanguageIndex)
				state.result.AppearanceIndex = settingsComboIndex(state.appearance, state.config.AppearanceIndex)
				state.result.Numbers = values
				state.result.ConflictIndex = settingsComboIndex(state.conflict, state.config.ConflictIndex)
				checked, _, _ := promptSendMessageW.Call(state.confirm, settingsBMGetCheck, 0, 0)
				state.result.ConfirmDelete = checked == settingsBSTChecked
				state.accepted = true
				promptDestroyWindow.Call(hwnd)
				return 0
			case settingsIDReset:
				if len(state.config.DefaultNumbers) != len(state.numbers) {
					return 0
				}
				promptSendMessageW.Call(
					state.language,
					settingsCBSet,
					uintptr(normalizedSettingsIndex(state.config.DefaultLanguageIndex, len(state.config.LanguageOptions))),
					0,
				)
				promptSendMessageW.Call(
					state.appearance,
					settingsCBSet,
					uintptr(normalizedSettingsIndex(state.config.DefaultAppearanceIndex, len(state.config.AppearanceOptions))),
					0,
				)
				for index, edit := range state.numbers {
					settingsSetText(edit, strconv.Itoa(state.config.DefaultNumbers[index]))
				}
				promptSendMessageW.Call(
					state.conflict,
					settingsCBSet,
					uintptr(normalizedSettingsIndex(state.config.DefaultConflictIndex, len(state.config.ConflictOptions))),
					0,
				)
				check := uintptr(0)
				if state.config.DefaultConfirmDelete {
					check = settingsBSTChecked
				}
				promptSendMessageW.Call(state.confirm, settingsBMSetCheck, check, 0)
				settingsSetText(state.errorLabel, "")
				return 0
			case settingsIDCancel:
				promptDestroyWindow.Call(hwnd)
				return 0
			}
		case premiumWMCtlColorEdit, premiumWMCtlColorListBox, premiumWMCtlColorBtn, premiumWMCtlColorStatic:
			return premiumDialogControlColor(wParam)
		case promptWMClose:
			promptDestroyWindow.Call(hwnd)
			return 0
		case promptWMDestroy:
			state.closed = true
			return 0
		}
	}
	r, _, _ := promptDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

func normalizedSettingsIndex(index, count int) int {
	if count <= 0 {
		return 0
	}
	if index < 0 || index >= count {
		return 0
	}
	return index
}

func settingsNumberInputLength(field SettingsDialogNumber) uintptr {
	maxText := strconv.Itoa(field.Max)
	minText := strconv.Itoa(field.Min)
	length := len(maxText)
	if len(minText) > length {
		length = len(minText)
	}
	if length < 1 {
		length = 1
	}
	return uintptr(length)
}

// SettingsDialog presents all frequently changed desktop preferences in one
// bounded modal instead of forcing the user through a chain of independent
// prompts. It is DPI-aware, owner-modal and follows the active Ghost FTP theme.
func SettingsDialog(config SettingsDialogConfig) (SettingsDialogResult, bool) {
	fallback := SettingsDialogResult{
		LanguageIndex:   normalizedSettingsIndex(config.LanguageIndex, len(config.LanguageOptions)),
		AppearanceIndex: normalizedSettingsIndex(config.AppearanceIndex, len(config.AppearanceOptions)),
		ConflictIndex:   normalizedSettingsIndex(config.ConflictIndex, len(config.ConflictOptions)),
		ConfirmDelete:   config.ConfirmDeleteOn,
		Numbers:         make([]int, len(config.Numbers)),
	}
	for index, field := range config.Numbers {
		fallback.Numbers[index] = field.Value
	}
	if len(config.LanguageOptions) == 0 || len(config.AppearanceOptions) == 0 || len(config.ConflictOptions) == 0 || len(config.Numbers) == 0 || len(config.Numbers) > 6 {
		return fallback, false
	}
	if config.ApplyLabel == "" {
		config.ApplyLabel = "OK"
	}
	if config.CancelLabel == "" {
		config.CancelLabel = "Cancel"
	}
	config.LanguageIndex = fallback.LanguageIndex
	config.AppearanceIndex = fallback.AppearanceIndex
	config.ConflictIndex = fallback.ConflictIndex
	if len(config.DefaultNumbers) != len(config.Numbers) {
		config.ResetLabel = ""
		config.DefaultNumbers = nil
	} else {
		config.DefaultLanguageIndex = normalizedSettingsIndex(config.DefaultLanguageIndex, len(config.LanguageOptions))
		config.DefaultAppearanceIndex = normalizedSettingsIndex(config.DefaultAppearanceIndex, len(config.AppearanceOptions))
		config.DefaultConflictIndex = normalizedSettingsIndex(config.DefaultConflictIndex, len(config.ConflictOptions))
	}

	const (
		wsOverlapped    = 0x00C80000
		clientWidth     = 860
		leftX           = 42
		rightX          = 438
		fieldWidth      = 380
		editWidth       = 170
		numberStartY    = 220
		numberLabelH    = 26
		numberRowHeight = 58
	)

	numberRows := (len(config.Numbers) + 1) / 2
	behaviorSeparatorY := numberStartY + numberRows*numberRowHeight + 4
	behaviorTitleY := behaviorSeparatorY + 12
	conflictLabelY := behaviorTitleY + 24
	confirmY := conflictLabelY + 58
	errorY := confirmY + 26
	footerSeparatorY := errorY + 28
	footerY := footerSeparatorY + 12
	utilityTitleY := footerY + 28
	utilityY := utilityTitleY + 24
	buttonY := utilityY + 46
	clientHeight := buttonY + 52

	hinst, _, _ := promptGetModuleHandleW.Call(0)
	settingsOnce.Do(func() {
		cursor, _, _ := promptLoadCursorW.Call(0, 32512)
		icon := premiumDialogIcon(hinst)
		wc := promptWndClassEx{
			CbSize:     uint32(unsafe.Sizeof(promptWndClassEx{})),
			WndProc:    settingsProc,
			Instance:   hinst,
			Cursor:     cursor,
			Icon:       icon,
			Background: premiumDialogBackgroundBrush(),
			ClassName:  promptWstr(settingsClass),
			IconSm:     icon,
		}
		promptRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	})

	owner := premiumDialogOwner()
	dpi := premiumDialogDPI(owner)
	windowWidth, windowHeight := premiumDialogOuterSize(clientWidth, clientHeight, wsOverlapped, 0, dpi)
	x, y := premiumDialogPosition(owner, windowWidth, windowHeight)
	hwnd, _, _ := promptCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(promptWstr(settingsClass))),
		uintptr(unsafe.Pointer(promptWstr(config.Title))),
		wsOverlapped,
		uintptr(x), uintptr(y), uintptr(windowWidth), uintptr(windowHeight),
		owner, 0, hinst, 0,
	)
	if hwnd == 0 {
		return fallback, false
	}
	applyPremiumDialogWindow(hwnd)

	state := &settingsDialogState{config: config, result: fallback}
	settingsStates.Store(hwnd, state)
	defer settingsStates.Delete(hwnd)
	restoreOwner := premiumModalOwner(owner)
	defer restoreOwner()

	font := premiumDialogFontForDPI(-15, dpi, 400)
	if font != 0 {
		defer promptDeleteObject.Call(font)
	}
	headingFont := premiumDialogFontForDPI(-25, dpi, 600)
	if headingFont != 0 {
		defer promptDeleteObject.Call(headingFont)
	}
	captionFont := premiumDialogFontForDPI(-13, dpi, 400)
	if captionFont != 0 {
		defer promptDeleteObject.Call(captionFont)
	}
	sectionFont := premiumDialogFontForDPI(-13, dpi, 600)
	if sectionFont != 0 {
		defer promptDeleteObject.Call(sectionFont)
	}

	scale := func(value int) uintptr { return uintptr(premiumScale(value, dpi)) }
	makeControl := func(class, text string, style uint32, x, y, w, h, id int, controlFont uintptr) uintptr {
		child, _, _ := promptCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(promptWstr(class))),
			uintptr(unsafe.Pointer(promptWstr(text))),
			uintptr(settingsWSChild|settingsWSVisible|style),
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

	brandIcon := premiumDialogBrandIcon(hinst, premiumScale(46, dpi))
	if brandIcon != 0 {
		brandControl := makeControl("STATIC", "", settingsSSIcon, 42, 22, 46, 46, 0, font)
		if brandControl != 0 {
			promptSendMessageW.Call(brandControl, settingsSTMSetImage, settingsImageIcon, brandIcon)
		}
	}
	makeControl("STATIC", config.Heading, 0, 100, 22, 718, 38, 0, headingFont)
	makeControl("STATIC", config.Intro, 0, 100, 63, 718, 32, 0, font)

	makeControl("STATIC", "GENERAL", 0, 42, 108, 776, 20, 0, sectionFont)
	makeControl("STATIC", config.LanguageLabel, 0, leftX, 136, fieldWidth, 22, 0, captionFont)
	state.language = makeControl("COMBOBOX", "", settingsWSTabStop|settingsWSVScroll|settingsCBSDropList, leftX, 160, fieldWidth, 240, settingsIDLanguage, font)
	for _, option := range config.LanguageOptions {
		promptSendMessageW.Call(state.language, settingsCBAdd, 0, uintptr(unsafe.Pointer(promptWstr(option))))
	}
	promptSendMessageW.Call(state.language, settingsCBSet, uintptr(config.LanguageIndex), 0)

	makeControl("STATIC", config.AppearanceLabel, 0, rightX, 136, fieldWidth, 22, 0, captionFont)
	state.appearance = makeControl("COMBOBOX", "", settingsWSTabStop|settingsWSVScroll|settingsCBSDropList, rightX, 160, fieldWidth, 240, settingsIDAppearance, font)
	for _, option := range config.AppearanceOptions {
		promptSendMessageW.Call(state.appearance, settingsCBAdd, 0, uintptr(unsafe.Pointer(promptWstr(option))))
	}
	promptSendMessageW.Call(state.appearance, settingsCBSet, uintptr(config.AppearanceIndex), 0)

	makeControl("STATIC", "", settingsEtchedHorz, 42, 210, 776, 2, 0, font)
	makeControl("STATIC", "TRANSFER PERFORMANCE", 0, 42, 202, 776, 20, 0, sectionFont)

	for index, field := range config.Numbers {
		row := index / 2
		column := index % 2
		xPos := leftX
		if column == 1 {
			xPos = rightX
		}
		yLabel := numberStartY + row*numberRowHeight
		makeControl("STATIC", field.Label, 0, xPos, yLabel, fieldWidth, numberLabelH, 0, captionFont)
		edit := makeControl(
			"EDIT",
			strconv.Itoa(field.Value),
			settingsWSBorder|settingsWSTabStop|settingsESNumber,
			xPos, yLabel+25, editWidth, 30,
			settingsIDNumberBase+index,
			font,
		)
		if edit != 0 {
			promptSendMessageW.Call(edit, promptEMSetLimitText, settingsNumberInputLength(field), 0)
		}
		state.numbers = append(state.numbers, edit)
	}

	makeControl("STATIC", "", settingsEtchedHorz, 42, behaviorSeparatorY, 776, 2, 0, font)
	makeControl("STATIC", "TRANSFER BEHAVIOR", 0, 42, behaviorTitleY, 776, 20, 0, sectionFont)
	makeControl("STATIC", config.ConflictLabel, 0, 42, conflictLabelY, 776, 22, 0, captionFont)
	state.conflict = makeControl("COMBOBOX", "", settingsWSTabStop|settingsWSVScroll|settingsCBSDropList, 42, conflictLabelY+24, 776, 240, settingsIDConflict, font)
	for _, option := range config.ConflictOptions {
		promptSendMessageW.Call(state.conflict, settingsCBAdd, 0, uintptr(unsafe.Pointer(promptWstr(option))))
	}
	promptSendMessageW.Call(state.conflict, settingsCBSet, uintptr(config.ConflictIndex), 0)

	state.confirm = makeControl("BUTTON", config.ConfirmDelete, settingsWSTabStop|settingsAutoCheck, 42, confirmY, 776, 28, settingsIDConfirm, font)
	if config.ConfirmDeleteOn {
		promptSendMessageW.Call(state.confirm, settingsBMSetCheck, settingsBSTChecked, 0)
	}
	state.errorLabel = makeControl("STATIC", "", 0, 42, errorY, 776, 22, settingsIDError, captionFont)

	makeControl("STATIC", "", settingsEtchedHorz, 42, footerSeparatorY, 776, 2, 0, font)
	makeControl("STATIC", config.Footer, 0, 42, footerY, 776, 30, 0, captionFont)
	makeControl("STATIC", "UPDATES && SUPPORT", 0, 42, utilityTitleY, 776, 20, 0, sectionFont)

	utilityLabels := []struct {
		text string
		id   int
		w    int
	}{
		{config.UpdateLabel, settingsIDUpdate, 190},
		{config.DownloadLabel, settingsIDDownload, 190},
		{config.WebsiteLabel, settingsIDWebsite, 170},
	}
	utilityX := 42
	for _, item := range utilityLabels {
		if item.text == "" {
			continue
		}
		makeControl("BUTTON", item.text, settingsWSTabStop, utilityX, utilityY, item.w, 36, item.id, font)
		utilityX += item.w + 10
	}

	if config.ResetLabel != "" {
		makeControl("BUTTON", config.ResetLabel, settingsWSTabStop, 42, buttonY, 170, 38, settingsIDReset, font)
	}
	applyButton := makeControl("BUTTON", config.ApplyLabel, settingsWSTabStop|settingsDefButton, 602, buttonY, 100, 38, settingsIDApply, font)
	makeControl("BUTTON", config.CancelLabel, settingsWSTabStop, 712, buttonY, 106, 38, settingsIDCancel, font)

	if state.language != 0 {
		promptSetFocus.Call(state.language)
	} else if state.appearance != 0 {
		promptSetFocus.Call(state.appearance)
	} else if applyButton != 0 {
		promptSetFocus.Call(applyButton)
	}
	promptShowWindow.Call(hwnd, 5)
	promptUpdateWindow.Call(hwnd)

	premiumRunDialogLoop(hwnd, func() bool { return state.closed })
	return state.result, state.accepted
}
