param(
    [Parameter(Mandatory = $true)]
    [string]$Executable,
    [string]$OutputDirectory = "ui-screenshots"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

Add-Type -AssemblyName System.Drawing
Add-Type @"
using System;
using System.Text;
using System.Runtime.InteropServices;

public static class GhostReferenceCapture
{
    [StructLayout(LayoutKind.Sequential)]
    public struct RECT { public int Left; public int Top; public int Right; public int Bottom; }
    public delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);
    [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr hWnd, IntPtr hdcBlt, uint flags);
    [DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern bool EnumWindows(EnumWindowsProc callback, IntPtr data);
    [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern int GetWindowText(IntPtr hWnd, StringBuilder text, int maxCount);
    [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr hWnd, uint message, IntPtr wParam, IntPtr lParam);
    [DllImport("user32.dll")] public static extern bool MoveWindow(IntPtr hWnd, int x, int y, int width, int height, bool repaint);
}
"@

function Wait-MainWindow {
    param([System.Diagnostics.Process]$Process)
    $deadline = [DateTime]::UtcNow.AddSeconds(20)
    do {
        if ($Process.HasExited) { throw "Ghost FTP exited early with $($Process.ExitCode)." }
        $Process.Refresh()
        if ($Process.MainWindowHandle -ne [IntPtr]::Zero) { return $Process.MainWindowHandle }
        Start-Sleep -Milliseconds 200
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Timed out waiting for Ghost FTP."
}

function Find-Window {
    param([int]$ProcessId, [string]$TitleContains)
    $deadline = [DateTime]::UtcNow.AddSeconds(15)
    do {
        $script:refFound = [IntPtr]::Zero
        $script:refPid = [uint32]$ProcessId
        $script:refTitle = $TitleContains
        $callback = [GhostReferenceCapture+EnumWindowsProc]{
            param([IntPtr]$hWnd, [IntPtr]$data)
            [uint32]$windowPid = 0
            [GhostReferenceCapture]::GetWindowThreadProcessId($hWnd, [ref]$windowPid) | Out-Null
            if ($windowPid -ne $script:refPid -or -not [GhostReferenceCapture]::IsWindowVisible($hWnd)) { return $true }
            $buffer = New-Object System.Text.StringBuilder 512
            [GhostReferenceCapture]::GetWindowText($hWnd, $buffer, $buffer.Capacity) | Out-Null
            if ($buffer.ToString() -like "*$($script:refTitle)*") {
                $script:refFound = $hWnd
                return $false
            }
            return $true
        }
        [GhostReferenceCapture]::EnumWindows($callback, [IntPtr]::Zero) | Out-Null
        if ($script:refFound -ne [IntPtr]::Zero) { return $script:refFound }
        Start-Sleep -Milliseconds 200
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Timed out waiting for window containing '$TitleContains'."
}

function Set-ReferenceWindowBounds {
    param([IntPtr]$Window, [int]$Width, [int]$Height, [string]$Name)
    if (-not [GhostReferenceCapture]::MoveWindow($Window, 0, 0, $Width, $Height, $true)) {
        throw "MoveWindow failed for $Name."
    }
    Start-Sleep -Milliseconds 350
    $rect = New-Object GhostReferenceCapture+RECT
    if (-not [GhostReferenceCapture]::GetWindowRect($Window, [ref]$rect)) {
        throw "GetWindowRect failed for $Name."
    }
    $actualWidth = $rect.Right - $rect.Left
    $actualHeight = $rect.Bottom - $rect.Top
    if ($actualWidth -ne $Width -or $actualHeight -ne $Height) {
        throw "$Name did not reach the required reference canvas. Expected ${Width}x${Height}, got ${actualWidth}x${actualHeight}."
    }
}

function Save-Window {
    param([IntPtr]$Window, [string]$Path)
    $rect = New-Object GhostReferenceCapture+RECT
    if (-not [GhostReferenceCapture]::GetWindowRect($Window, [ref]$rect)) { throw "GetWindowRect failed." }
    $width = $rect.Right - $rect.Left
    $height = $rect.Bottom - $rect.Top
    if ($width -lt 600 -or $height -lt 400 -or $width -gt 5000 -or $height -gt 3000) {
        throw "Unexpected screenshot geometry: ${width}x${height}"
    }
    $bitmap = New-Object System.Drawing.Bitmap $width, $height, ([System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
    try {
        $hdc = $graphics.GetHdc()
        try {
            if (-not [GhostReferenceCapture]::PrintWindow($Window, $hdc, 2)) { throw "PrintWindow failed." }
        } finally {
            $graphics.ReleaseHdc($hdc)
        }
        $full = [IO.Path]::GetFullPath($Path)
        [IO.Directory]::CreateDirectory([IO.Path]::GetDirectoryName($full)) | Out-Null
        $bitmap.Save($full, [System.Drawing.Imaging.ImageFormat]::Png)
    } finally {
        $graphics.Dispose()
        $bitmap.Dispose()
    }
    if ((Get-Item -LiteralPath $Path).Length -lt 4096) { throw "Screenshot is unexpectedly small." }
}

$exe = (Resolve-Path -LiteralPath $Executable).Path
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$previousReferenceCapture = $env:GHOSTFTP_REFERENCE_CAPTURE
$env:GHOSTFTP_REFERENCE_CAPTURE = "1"
try {
    $process = Start-Process -FilePath $exe -PassThru
} finally {
    $env:GHOSTFTP_REFERENCE_CAPTURE = $previousReferenceCapture
}
try {
    $main = Wait-MainWindow $process
    Set-ReferenceWindowBounds $main 1664 960 "Main workspace"
    Start-Sleep -Milliseconds 550
    Save-Window $main (Join-Path $OutputDirectory "Ghost-FTP-main-reference.png")

    if (-not [GhostReferenceCapture]::PostMessage($main, 0x0111, [IntPtr]701, [IntPtr]::Zero)) {
        throw "Could not open Connections."
    }
    $connections = Find-Window $process.Id "Connections"
    Set-ReferenceWindowBounds $connections 1590 880 "Connections"
    Start-Sleep -Milliseconds 500
    Save-Window $connections (Join-Path $OutputDirectory "Ghost-FTP-connections-reference.png")
    [GhostReferenceCapture]::PostMessage($connections, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero) | Out-Null
    Start-Sleep -Milliseconds 350

    # Settings is an application-owned modal and must stay visually aligned with
    # the main Ghost FTP reference UI. Capture it on every Windows release so
    # regressions in spacing, branding or button layout are visible in CI.
    if (-not [GhostReferenceCapture]::PostMessage($main, 0x0111, [IntPtr]94, [IntPtr]::Zero)) {
        throw "Could not open Settings."
    }
    $settings = Find-Window $process.Id "Settings"
    Start-Sleep -Milliseconds 500
    Save-Window $settings (Join-Path $OutputDirectory "Ghost-FTP-settings-reference.png")
    [GhostReferenceCapture]::PostMessage($settings, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero) | Out-Null
} finally {
    if (-not $process.HasExited) {
        $process.CloseMainWindow() | Out-Null
        Start-Sleep -Milliseconds 400
        if (-not $process.HasExited) { $process.Kill() }
    }
}
Write-Host "REFERENCE_SCREENSHOT_CAPTURE=PASS"
