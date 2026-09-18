# Ghost FTP update service

Ghost FTP desktop updates use a dedicated first-party origin:

`https://update.ghostftp.com/`

The Windows application checks `/windows/latest.json` only when the user explicitly asks to check for updates. No server credentials, saved sites, paths, file names or transfer metadata are included in the request.

## Manifest

Production manifest:

`https://update.ghostftp.com/windows/latest.json`

Example:

```json
{
  "schema": 1,
  "version": "0.3.0",
  "setup_url": "https://update.ghostftp.com/windows/Ghost-FTP-0.3.0-Setup.exe",
  "setup_sha256": "<64 lowercase hex characters>",
  "portable_url": "https://update.ghostftp.com/windows/Ghost-FTP-0.3.0-Portable.exe",
  "portable_sha256": "<64 lowercase hex characters>",
  "updater_url": "https://update.ghostftp.com/windows/Ghost-FTP-0.3.0-Update.exe",
  "updater_sha256": "<64 lowercase hex characters>"
}
```

The desktop client rejects artifact URLs outside `update.ghostftp.com`, rejects non-HTTPS URLs and rejects malformed SHA-256 values.

## Deployment order

1. Upload the new Setup, Portable and Update executables to `/windows/`.
2. Verify their SHA-256 values.
3. Upload `latest.json` last.
4. Keep previous signed/verified packages available while clients migrate.

Publishing the manifest last prevents clients from discovering a version whose binaries are not yet available.
