# Ghost FTP

Ghost FTP is a privacy-first Windows FTP, FTPS and SFTP desktop client maintained in **bren-wp/Host-FTP**.

**Files move freely. You stay in control.**

## Source version

Version: **0.4.0**

English is the primary and default product language. Additional languages remain available from Settings.

Ghost FTP 0.4.0 is a deeper reference-UI rebuild based on the approved Ghost FTP product boards: charcoal/slate surfaces, cyan → electric-blue → violet interaction accents, an integrated Ghost wordmark, a persistent application rail, dual Local/Remote file panes and a metrics-rich transfer queue.

## Main workspace

- Sites, Transfers, Queue, Sync and Settings navigation
- saved-site shortcuts in the left rail
- global Ctrl+K file/site search entry point
- Connect / Disconnect / New Folder / Upload / Download / Refresh action row
- Local Files and Remote Files side by side
- independent pane navigation and remote search
- directory comparison / synchronization bridge
- transfer queue tabs: All, Uploading, Downloading and Completed
- transfer metrics: name, direction, progress, size, speed, status and ETA
- pause, resume, cancel, retry, clear and priority actions
- rename, delete, create folder, remote editing and CHMOD where supported
- recursive search, bookmarks and high-DPI-aware Windows layout

## Connections workspace

The Connections experience is structured around the approved reference layout:

- Quick Connect, Site Manager and Import / Export entry tabs
- FTP, FTPS and SFTP connection details
- private-key and passphrase fields for SFTP
- protected credential persistence
- Saved Sites management
- Transfer & Sync Options with functional presets
- Standard Upload, Website Deployment, Backup (Incremental) and Media Transfer presets
- Sync and Automation shortcuts into maintained application functionality

Import / Export is intentionally secret-safe: stored passwords and private-key passphrases are never exported as clear text. The current UI explains that contract while encrypted profile bundles remain a future capability.

## Security and privacy

Ghost FTP does not require a Ghost FTP account. The maintained desktop client does not add advertising or product telemetry.

Saved credentials use the maintained Windows protection layer. Sensitive credentials are not intentionally written to application logs. SFTP host-key and FTPS certificate validation remain enforced by the protocol engine.

## Updates

Update discovery and Windows update packages use the dedicated first-party service:

`https://update.ghostftp.com/`

The application checks `https://update.ghostftp.com/windows/latest.json` only when the user explicitly requests an update check.

Update downloads are restricted to the same HTTPS host and verified with SHA-256 before the standalone updater launches Setup.

See `UPDATE_SERVICE.md` for the deployment contract.

## Windows release artifacts

Every production GitHub release publishes:

- `Ghost-FTP-0.4.0-Setup.exe`
- `Ghost-FTP-0.4.0-Portable.exe`
- `Ghost-FTP-0.4.0-Update.exe`
- `Ghost-FTP-0.4.0-Source.zip`
- `SHA256.txt`
- `latest.json`

## Quality gates

The Windows release workflow runs formatting, `go vet`, the full Go test suite, deterministic brand-asset generation, executable builds, PE resource embedding, reference screenshot capture, source packaging and SHA-256 manifest generation before publishing release artifacts.

## Official website

`https://ghostftp.com/`

## Source repository

Canonical repository: **bren-wp/Host-FTP**

## License

This repository is distributed under **GNU GPL-3.0**. See `LICENSE`.

---

**Ghost FTP 0.4.0** — secure transfers, modern Windows workflow.
