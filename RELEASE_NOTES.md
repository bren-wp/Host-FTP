# Ghost FTP 0.3.0

Ghost FTP 0.3.0 continues the reference-UI rebuild around the approved Ghost FTP brand and Windows desktop layouts.

## Highlights

- English remains the primary and default product language.
- Main Windows workspace is aligned more closely with the approved dual-pane Local Files / Remote Files reference.
- Connections uses the approved dark three-column composition for saved sites, connection details and transfer/safety settings.
- Ghost FTP branding uses the cyan → electric-blue → violet identity throughout navigation, buttons, status states and product assets.
- About and Settings content has been rewritten for clearer product, security, privacy and update information.
- Update discovery now uses the dedicated first-party service at `update.ghostftp.com`.
- New `Ghost-FTP-0.3.0-Update.exe` verifies the update manifest, downloads the official Setup package and validates SHA-256 before launching it.
- GitHub Actions now builds and publishes Setup, Portable, Update, source ZIP, SHA-256 checksums and an update manifest for every production release.

## Windows downloads

- `Ghost-FTP-0.3.0-Setup.exe`
- `Ghost-FTP-0.3.0-Portable.exe`
- `Ghost-FTP-0.3.0-Update.exe`
- `Ghost-FTP-0.3.0-Source.zip`
- `SHA256.txt`
- `latest.json`

## Update service

The desktop client checks:

`https://update.ghostftp.com/windows/latest.json`

Update binaries referenced by that manifest must also be hosted on `https://update.ghostftp.com/` and are verified by SHA-256 before execution.

## Privacy

The update request contains no FTP/SFTP connection profiles, credentials, remote paths, local file names or transfer history.
