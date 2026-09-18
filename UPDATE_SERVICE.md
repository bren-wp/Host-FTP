# Ghost FTP update service

Ghost FTP desktop updates use a dedicated first-party origin:

`https://update.ghostftp.com/`

The Windows application checks `/windows/latest.json` only when the user explicitly asks to check for updates. No server credentials, saved sites, paths, file names or transfer metadata are included in the request.

## Manifest

Production manifest:

`https://update.ghostftp.com/windows/latest.json`

Example for the current source version:

```json
{
  "schema": 1,
  "version": "0.5.0",
  "setup_url": "https://update.ghostftp.com/windows/Ghost-FTP-0.5.0-Setup.exe",
  "setup_sha256": "<64 lowercase hex characters>",
  "portable_url": "https://update.ghostftp.com/windows/Ghost-FTP-0.5.0-Portable.exe",
  "portable_sha256": "<64 lowercase hex characters>",
  "updater_url": "https://update.ghostftp.com/windows/Ghost-FTP-0.5.0-Update.exe",
  "updater_sha256": "<64 lowercase hex characters>"
}
```

The desktop client rejects artifact URLs outside `update.ghostftp.com`, rejects non-HTTPS URLs and rejects malformed SHA-256 values.

## Deployment order

1. Upload the new Setup, Portable and Update executables to `/windows/`.
2. Verify their SHA-256 values against `SHA256.txt`.
3. Upload `latest.json` last.
4. Keep previous verified packages available while clients migrate.

Publishing the manifest last prevents clients from discovering a version whose binaries are not yet available.

## GitHub release parity

The GitHub production release and `update.ghostftp.com` should carry the same versioned Windows executables and checksums. GitHub Actions generates the Windows binaries, `SHA256.txt` and `latest.json`; deployment to the update host must preserve those exact bytes.
