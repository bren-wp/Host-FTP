# Ghost FTP

Ghost FTP is a privacy-first Windows FTP, FTPS and SFTP desktop client maintained in **bren-wp/Host-FTP**.

**Files move freely. You stay in control.**

## Current release

Version: **0.3.0**

English is the primary and default product language. Additional languages remain available from Settings.

Ghost FTP 0.3.0 continues the approved premium dark interface: deep charcoal/slate surfaces, cyan → electric-blue → violet accents, a wide application rail, dual Local/Remote file panes and a persistent transfer queue.

## Core workspace

- saved sites and connection profiles
- FTP, FTPS and SFTP
- Local Files and Remote Files side by side
- upload / download / refresh / new-folder actions
- persistent transfer queue
- pause, resume, cancel, retry and clear actions
- rename, delete, create folder and CHMOD where supported
- remote editing
- bookmarks
- recursive search
- directory comparison / synchronization helpers
- high-DPI-aware Windows layout

## Security and privacy

Ghost FTP does not require a Ghost FTP account. The maintained desktop client does not add advertising or product telemetry.

Saved credentials use the maintained platform protection layer. Sensitive credentials are not intentionally written to application logs.

## Updates

Update discovery and Windows update packages use the dedicated first-party service:

`https://update.ghostftp.com/`

The application checks `https://update.ghostftp.com/windows/latest.json` only when the user explicitly requests an update check.

Update downloads are restricted to the same HTTPS host and verified with SHA-256 before the standalone updater launches Setup.

See `UPDATE_SERVICE.md` for the deployment contract.

## Windows release artifacts

Every production GitHub release publishes:

- `Ghost-FTP-0.3.0-Setup.exe`
- `Ghost-FTP-0.3.0-Portable.exe`
- `Ghost-FTP-0.3.0-Update.exe`
- `Ghost-FTP-0.3.0-Source.zip`
- `SHA256.txt`
- `latest.json`

## Official website

`https://ghostftp.com/`

## Source repository

Canonical repository: **bren-wp/Host-FTP**

## License

This repository is distributed under **GNU GPL-3.0**. See `LICENSE`.

---

**Ghost FTP 0.3.0** — secure transfers, modern Windows workflow.
