# Ghost FTP 0.6.0

Ghost FTP 0.6.0 is the next reference-UI production pass built directly against the supplied 1672×941 Ghost FTP workspace and Connections boards.

## Reference UI

- Adopted the supplied **1672×941** application canvas as the exact Windows visual target for Main and Connections captures.
- Converted the top-level Windows frame to a full-client custom surface so title-bar spacing, search, status and window controls align with the reference instead of reserving stock non-client chrome.
- Added application-owned dark ListView headers and row-state drawing so file panes and the transfer queue no longer depend on the host Windows Explorer theme.
- Removed stock report-view borders from the core file and queue tables; the surrounding Ghost FTP cards now own those edges as in the reference.

- Refined the main Windows layout around the approved Sites / Transfers / Queue / Sync / Settings rail.
- Added the handwritten **Move More / Do More** rail signature and kept the secure-transfer tagline below it.
- Reworked the footer identity toward the approved **MORE ACCESS. A BRIGHTER TOMORROW.** treatment.
- Replaced the approximate procedural ghost with the approved Ghost FTP transfer-arrow mark extracted from the supplied brand reference, and use the same source for PNG/ICO generation and in-app branding.
- Uses Segoe UI Variable on supported Windows 11 systems with native Windows fallbacks.
- Primary actions keep the cyan → electric-blue → violet Ghost FTP gradient.
- The center compare/sync bridge is now a circular accent control as shown in the file-workspace reference.
- Fixed Local/Remote file metadata order and adaptive sizing so the visible table stays **Name / Size / Type / Modified / Permissions**.

## Connections

- Matched the Connections workspace to the supplied 1672×941 reference canvas and retained distinct Transfer & Sync, Sync Options, Saved Sites and Recent Connections regions.
- Added a large Ghost brand mark and the **FILES MOVE FREELY / YOU STAY IN CONTROL** rail composition.
- Recent Connections now reflects real successful sessions in memory only; passwords and passphrases are never recorded in recent history.
- Quick Connect, Site Manager, Presets, Sync and Automation controls remain connected to maintained application functions.
- Standard Upload, Website Deployment, Backup (Incremental) and Media Transfer presets change real persisted transfer settings.
- Sync Options now directly control backup-before-overwrite, skip-existing and destructive-action confirmation settings.

## Saved-site portability

- Import / Export is now functional rather than placeholder copy.
- Exported `.ghostftp.json` bundles contain connection metadata only.
- Saved passwords, private-key passphrases and trusted SFTP host fingerprints are intentionally excluded.
- Import validates Ghost FTP bundle structure, skips matching existing sites and requires credentials to be entered or saved again on the destination Windows account.

## Popups and product copy

- Branded prompts, decisions, errors, Settings and About surfaces now share the Ghost FTP icon, dark/light appearance, rounded Windows chrome, title-bar colors and Segoe UI Variable typography.
- About/info cards size to their real content instead of clipping longer product/security copy.
- Prompt spacing and hierarchy were retuned to the same product system as the main application.
- A production-copy CI audit blocks demo personas, sample servers, placeholder copy and development-only text from release UI strings.
- English remains the primary/default product language; Croatian plus 22 additional maintained desktop languages remain available.
- Removed the retired macOS application source and its macOS-only regression contracts from this Windows release branch.

## Repository presentation

- README is now a product-style landing page using the Ghost FTP wordmark, local feature icons and screenshots captured from the real Windows build.
- CI refreshes README screenshots from the executable it just built, so repository imagery cannot silently drift away from the release UI.

## Windows downloads

- `Ghost-FTP-0.6.0-Setup.exe`
- `Ghost-FTP-0.6.0-Portable.exe`
- `Ghost-FTP-0.6.0-Update.exe`
- `Ghost-FTP-0.6.0-Source.zip`
- `SHA256.txt`
- `latest.json`

## Update service

The desktop client checks:

`https://update.ghostftp.com/windows/latest.json`

Windows update packages referenced by that manifest are accepted only from `https://update.ghostftp.com/` and are SHA-256 verified before Setup is launched.

## Privacy

Ghost FTP does not require an account to connect to your servers and the maintained desktop client does not add advertising or product telemetry. Update checks do not include saved sites, credentials, paths, file names or transfer history.
