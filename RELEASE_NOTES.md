# Ghost FTP 0.4.0

Ghost FTP 0.4.0 is a deeper UI/UX and quality pass aligned to the approved Ghost FTP brand, file-manager and Connections references.

## Interface

- Reworked the Ghost FTP wordmark so the transparent Ghost symbol, white **Ghost** text and accent **FTP** text form one integrated product lockup.
- Added Windows 11 Segoe UI Variable typography when available, with Windows-compatible fallback.
- Added flat navigation styling with a cyan active rail and cyan → electric-blue → violet primary-action gradients.
- Main navigation now follows the approved Sites / Transfers / Queue / Sync / Settings structure.
- Saved sites appear directly in the main left rail for faster connection selection.
- Main workspace keeps Local Files and Remote Files as the dominant dual-pane working area.
- Global search, connection state, file actions and navigation spacing were retuned toward the supplied reference.
- Transfer Queue now has functional All / Uploading / Downloading / Completed filters.
- Queue columns show real name, direction, progress, size, speed, status and time remaining data.

## Connections

- Added Quick Connect, Site Manager and Import / Export reference tabs.
- Added Presets / Sync / Automation controls in Transfer & Sync Options.
- Added real Standard Upload, Website Deployment, Backup (Incremental) and Media Transfer presets backed by persisted Ghost FTP settings.
- Improved compact-display behavior so optional side cards collapse instead of clipping connection actions.
- Import / Export explicitly preserves the no-clear-text-secrets contract; encrypted profile bundle import/export is not falsely exposed as completed functionality.

## Code quality

- Added `gofmt`, `go vet` and the complete Go test suite as release gates.
- Fixed transfer-column localization drift that could overwrite the new queue metric headings.
- Fixed queue selection/action mapping so filtered transfer views still cancel, retry and reorder the intended job.
- Kept English as the primary and default UI language.

## Windows downloads

- `Ghost-FTP-0.4.0-Setup.exe`
- `Ghost-FTP-0.4.0-Portable.exe`
- `Ghost-FTP-0.4.0-Update.exe`
- `Ghost-FTP-0.4.0-Source.zip`
- `SHA256.txt`
- `latest.json`

## Update service

The desktop client checks:

`https://update.ghostftp.com/windows/latest.json`

The updater accepts update packages only from `https://update.ghostftp.com/` and validates SHA-256 before launching Setup.

## Privacy

Update checks do not include FTP/SFTP connection profiles, credentials, local/remote paths, file names or transfer history.
