import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class WindowsVisualRegressionTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_workspace_refinement_does_not_erase_entire_parent(self):
        source = self.read("internal/desktop/workspace_layout_windows.go")
        self.assertNotIn("invalidateRect.Call(a.hwnd, 0, 1)", source)
        self.assertIn("a.stabilizeWorkspaceChrome()", source)
        connection = self.read("internal/desktop/connection_profiles_windows.go")
        self.assertNotIn("invalidateRect.Call(a.hwnd, 0, 1)", connection)

    def test_workspace_uses_real_pe_brand_icon_and_full_title_gutter(self):
        chrome = self.read("internal/desktop/chrome_windows.go")
        ui = self.read("internal/desktop/ui_windows.go")
        self.assertIn('loadImageW.Call(hinst, 2, imageIcon', chrome)
        self.assertIn("stmSetImage", chrome)
        self.assertIn("const titleX, ghostWidth, ftpWidth, wordmarkGap, subtitleX = 72, 92, 54, 2, 230", chrome)
        self.assertIn("a.move(a.brandTitle, titleX, 10, ghostWidth, 35)", chrome)
        self.assertIn("a.move(a.brandFTP, titleX+ghostWidth+wordmarkGap, 10, ftpWidth, 35)", chrome)

    def test_sidebar_brand_and_connection_status_cannot_overlap_profile_actions(self):
        sidebar = self.read("internal/desktop/sidebar_windows.go")
        self.assertIn("sendMessageW.Call(a.brandTitle, wmSetFont, a.titleFont, 1)", sidebar)
        self.assertIn("connectionY := height - applicationSidebarBottomInset - 28", sidebar)
        self.assertIn("footerY := connectionY - 68", sidebar)
        self.assertIn("mottoY := footerY - 112", sidebar)
        self.assertIn("a.move(a.sidebarConnectionStatus", sidebar)

    def test_native_dark_chrome_and_site_manager_use_premium_controls(self):
        dark = self.read("internal/desktop/dark_mode_windows.go")
        ui = self.read("internal/desktop/ui_windows.go")
        site = self.read("internal/desktop/site_manager_windows.go")
        self.assertIn("enableImmersiveDarkMode(hwnd)", ui)
        self.assertIn("SetPreferredAppMode", dark)
        self.assertIn('case "COMBOBOX", "EDIT":', ui)
        self.assertIn("case wmDrawItem:", site)
        self.assertIn("bsOwnerDraw", site)
        self.assertIn("IconSm:     icon", site)

    def test_resize_is_batched_without_background_erase(self):
        source = self.read("internal/desktop/layout_batch_windows.go")
        windows = self.read("internal/desktop/windows.go")
        self.assertIn("wmSetRedraw", source)
        self.assertIn("rdwAllChildren", source)
        self.assertNotIn("rdwErase", source)
        self.assertIn("a.reflowWorkspace(w, h)", windows)

    def test_full_wordmark_dark_headers_and_sidebar_are_guarded(self):
        chrome = self.read("internal/desktop/chrome_windows.go")
        header = self.read("internal/desktop/header_draw_windows.go")
        sidebar = self.read("internal/desktop/sidebar_windows.go")
        navigation = self.read("internal/desktop/navigation_labels.go")
        wnd = self.read("internal/desktop/windows.go")
        self.assertIn("const titleX, ghostWidth, ftpWidth, wordmarkGap, subtitleX = 72, 92, 54, 2, 230", chrome)
        self.assertIn("installWorkspaceHeaderDraw(a, list)", chrome)
        self.assertIn("SetWindowSubclass", header)
        self.assertIn("workspaceListSubclass", header)
        self.assertIn("nmCustomDraw", header)
        self.assertIn("fillRectHeader.Call", header)
        self.assertIn("setTextColor.Call(d.HDC, textColor())", header)
        self.assertIn("cdrfSkipDefault", header)
        self.assertIn("applyApplicationSidebar", sidebar)
        self.assertIn("applicationSidebarWidth", sidebar)
        self.assertIn("applicationContentLeft", sidebar)
        self.assertIn("setSidebarButtonVisual", sidebar)
        self.assertIn("buttonNavActive", sidebar)
        self.assertIn("updateSidebarTransferBadge", sidebar)
        self.assertIn("navigationLabelsForLanguage", navigation)
        self.assertIn('"en": {"Files", "Connections", "Transfer Queue", "Site Manager", "Connection info"}', navigation)
        self.assertIn('"ko": {"파일", "연결", "전송", "서버 관리자", "연결 정보"}', navigation)
        self.assertFalse((ROOT / "internal/desktop/menu_draw_windows.go").exists())
        self.assertFalse((ROOT / "internal/desktop/menu_windows.go").exists())
        self.assertNotIn("a.measureMenuItem(lParam)", wnd)
        self.assertNotIn("a.drawMenuItem(&d)", wnd)

    def test_workspace_list_rows_use_reference_palette_without_stock_borders(self):
        ui = self.read("internal/desktop/ui_windows.go")
        rows = self.read("internal/desktop/list_draw_windows.go")
        wnd = self.read("internal/desktop/windows.go")
        self.assertIn("drawWorkspaceList", wnd)
        self.assertIn("selectionColor()", rows)
        self.assertIn("workspaceListHotColor()", rows)
        self.assertIn("cdrfNotifyItemDraw", rows)
        self.assertNotIn('wsBorder|wsTabStop|lvsReport|lvsShowSelAlways, idLocalList', ui)
        self.assertNotIn('wsBorder|wsTabStop|lvsReport|lvsShowSelAlways, idRemoteList', ui)
        self.assertNotIn('wsBorder|wsTabStop|lvsReport|lvsShowSelAlways, idTransferList', ui)

    def test_reference_workspace_has_real_file_pane_footers_and_open_header(self):
        windows = self.read("internal/desktop/windows.go")
        ui = self.read("internal/desktop/ui_windows.go")
        workspace = self.read("internal/desktop/master_workspace_windows.go")
        paint = self.read("internal/desktop/reference_paint_windows.go")
        self.assertIn("localPaneSummary", windows)
        self.assertIn("remotePanePath", windows)
        self.assertIn('a.localPaneSummary = mk("STATIC", "", 0, 0)', ui)
        self.assertIn('a.localPanePath = mk("STATIC", "", ssRight, 0)', ui)
        self.assertIn('a.remotePanePath = mk("STATIC", "", ssRight, 0)', ui)
        self.assertIn("func (a *app) refreshPaneFooters()", workspace)
        self.assertIn("paneFooterText(a.localItems)", workspace)
        self.assertNotIn("drawReferenceCard(hdc, contentLeft-2, 10, contentWidth+2, 132", paint)
        self.assertIn("drawReferenceDivider(hdc, contentLeft, 78, contentWidth", paint)
        self.assertIn("footerLineY := paneBottom - 34", paint)

    def test_dark_theme_uses_the_supplied_brand_board_palette(self):
        palette = self.read("internal/uipalette/palette.go")
        self.assertIn("Window:       RGB{0x0B, 0x0E, 0x14}", palette)
        self.assertIn("Panel:        RGB{0x16, 0x1B, 0x24}", palette)
        self.assertIn("Accent:       RGB{0x00, 0xE5, 0xFF}", palette)
        self.assertIn("AccentStrong: RGB{0x3B, 0x82, 0xF6}", palette)
        self.assertIn("Text:         RGB{0xE5, 0xE7, 0xEB}", palette)

    def test_idle_queue_hides_non_actionable_controls(self):
        state = self.read("internal/desktop/action_state_windows.go")
        self.assertIn("showControls(transferState.Pause, a.pauseQueue)", state)
        self.assertIn("showControls(transferState.Resume, a.resumeQueue)", state)
        self.assertIn("showControls(transferState.Cancel, a.cancelJob)", state)
        self.assertIn("showControls(transferState.Retry, a.retryJob)", state)
        self.assertIn("showControls(true, a.clearQueue)", state)

    def test_connections_cards_start_below_the_top_search_band(self):
        paint = self.read("internal/desktop/reference_paint_windows.go")
        site = self.read("internal/desktop/site_manager_windows.go")
        self.assertIn("const contentTop = 90", paint)
        self.assertIn("railHeight := logicalHeight - 91", paint)
        self.assertIn("footerY := logicalHeight - 77", paint)
        self.assertIn("drawReferenceCard(hdc, 16, footerY, 1572, 42", paint)
        self.assertIn("footerY := height - 60", site)
        self.assertIn("state.parent.move(state.footerReady, 32, footerY", site)

    def test_disconnected_remote_list_keeps_dark_enabled_surface(self):
        source = self.read("internal/desktop/chrome_windows.go")
        self.assertIn("setControlEnabled(a.remoteList, true)", source)
        self.assertIn("styleWorkspaceList(list)", source)

    def test_startup_settings_skip_redundant_language_rebuild(self):
        source = self.read("internal/desktop/settings_windows.go")
        self.assertIn("previousLanguage := a.languageCode()", source)
        self.assertIn("if a.languageCode() != previousLanguage", source)

    def test_connect_timeout_and_cancel_are_real_ui_behaviors(self):
        source = self.read("internal/desktop/connection_profiles_windows.go")
        self.assertIn("connectionTimeoutDuration(a.settings)", source)
        self.assertIn("a.connectionBusy && !a.connected", source)
        self.assertIn("a.cancelConnectionAttempt()", source)

    def test_x86_ftp_and_sftp_have_secure_sysnative_fallbacks(self):
        ftp = self.read("internal/remote/tools.go")
        sftp = self.read("internal/remote/sftp.go")
        self.assertIn('arch == "386"', ftp)
        self.assertIn('"Sysnative", "curl.exe"', ftp)
        self.assertNotIn('exec.LookPath("curl.exe")', ftp)
        self.assertIn("windowsOpenSSHCandidates", sftp)
        self.assertIn('"Sysnative", "OpenSSH", name', sftp)

    def test_list_selection_uses_real_listview_state(self):
        defs = self.read("internal/desktop/win32_defs_windows.go")
        rows = self.read("internal/desktop/list_draw_windows.go")
        self.assertIn("lvmGetItemState             = lvmFirst + 44", defs)
        self.assertIn("workspaceListActualDrawState", rows)
        self.assertIn("lvmGetItemState", rows)
        self.assertIn("selected&lvisSelected", rows)

    def test_reference_typography_is_stable_on_windows_runners(self):
        ui = self.read("internal/desktop/ui_windows.go")
        dialogs = self.read("internal/platform/dialog_premium_windows.go")
        self.assertIn('return createNamedUIFont("Segoe UI", height, weight, false)', ui)
        self.assertNotIn('name = "Segoe UI Variable Text"', ui)
        self.assertIn('a.font = createUIFont(int32(-a.scale(14)), 400)', ui)
        self.assertIn('a.titleFont = createUIFont(int32(-a.scale(26)), 700)', ui)
        self.assertIn('a.smallFont = createUIFont(int32(-a.scale(12)), 400)', ui)
        self.assertIn('promptWstr("Segoe UI")', dialogs)
        self.assertNotIn('promptWstr("Segoe UI Variable Text")', dialogs)

    def test_main_reference_panes_keep_icon_title_composition(self):
        workspace = self.read("internal/desktop/master_workspace_windows.go")
        paint = self.read("internal/desktop/reference_paint_windows.go")
        self.assertIn("drawReferencePaneGlyph", paint)
        self.assertIn("iconOpenLocal", paint)
        self.assertIn("iconCloud", paint)
        self.assertIn("a.move(a.sectionLocal, leftX+40", workspace)
        self.assertIn("a.move(a.sectionRemote, rightX+40", workspace)
        self.assertIn("searchY, searchH := 23, 40", workspace)
        self.assertIn("toolbarY, toolbarH := 91, 42", workspace)

    def test_windows_build_captures_and_packages_authentic_reference_ui(self):
        workflow = self.read(".github/workflows/windows-build.yml")
        self.assertIn("- 'work/**'", workflow)
        self.assertIn("- name: Capture authentic reference UI", workflow)
        self.assertIn("capture_reference_windows.ps1", workflow)
        self.assertIn("- name: Refresh README screenshots from the real Windows build", workflow)
        self.assertIn("ui-screenshots/*.png", workflow)
        self.assertIn("- name: Upload Windows deliverables", workflow)
        self.assertIn("Ghost-FTP-*-Setup.exe", workflow)
        self.assertIn("Ghost-FTP-*-Portable.exe", workflow)
        self.assertIn("Ghost-FTP-*-Update.exe", workflow)
        self.assertIn("SHA256.txt", workflow)
        self.assertIn("latest.json", workflow)


if __name__ == "__main__":
    unittest.main()
