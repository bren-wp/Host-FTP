# Ghost FTP build and audit scripts

This directory contains the maintained build, packaging, privacy, security and release helpers used by Ghost FTP **0.6.0**.

## Production release path

The active production workflow is:

`.github/workflows/windows-build.yml`

Ghost FTP 0.6.0 is a Windows 10/11 desktop release. The retired macOS application source and macOS-only distribution contracts are intentionally not part of this branch.

The workflow performs, in order:

1. source checkout and version validation;
2. `gofmt`, `go vet` and the complete Go test suite;
3. production UI-copy audit;
4. deterministic Ghost FTP PNG/ICO brand generation from the approved transfer mark;
5. Setup, Portable and Update executable builds;
6. integrated-uninstall verification — no permanent `Uninstall.exe`;
7. authentic 1672×941 screenshots from the built Windows executable;
8. README screenshot refresh;
9. exact source ZIP packaging;
10. SHA-256 and `latest.json` generation;
11. generated-source persistence and artifact upload;
12. GitHub Release publication when the workflow runs on `main`.

## Windows artifacts

For version 0.6.0 the workflow produces:

```text
Ghost-FTP-0.6.0-Setup.exe
Ghost-FTP-0.6.0-Portable.exe
Ghost-FTP-0.6.0-Update.exe
Ghost-FTP-0.6.0-Source.zip
SHA256.txt
latest.json
```

The installer registers uninstall through the installed Ghost FTP executable and Windows Installed Apps; release packaging rejects a separate permanent `Uninstall.exe`.

## Reference UI evidence

`capture_reference_windows.ps1` captures the real executable rather than a mockup. The preferred reference canvases are 1672×941 for the main workspace and 1672×941 for Connections. Settings is captured at its real application-owned modal size.

The generated screenshots are written to `docs/screenshots/` and are included in the Windows workflow artifact so visual regressions can be inspected against the approved Ghost FTP reference boards.

## Branding

`generate_brand_assets.py` is dependency-free and deterministic. It owns the Windows PNG/ICO outputs used by the executable, installer metadata and README. The embedded transfer mark is derived from the approved Ghost FTP brand reference supplied for this release; do not replace it with sample, third-party or approximate product artwork.

## Localization

The desktop registry contains 24 maintained languages. English is the default language and Croatian is fully supported alongside the additional locales. `audit_localization.py` and the Go localization tests guard catalog consistency.

## Privacy and product-copy rules

Production UI must not contain demo personas, sample servers, staging labels, development copy, subscription upsells or placeholder account data. `audit_product_copy.py` is the release gate for these strings.

Ghost FTP does not add product telemetry. Never place passwords, passphrases, private keys, signing secrets, customer data or production credentials in source, tests, screenshots or CI logs.

## Update service

The client update contract is documented in `../UPDATE_SERVICE.md`. Windows update metadata is published for:

`https://update.ghostftp.com/windows/latest.json`

The updater accepts first-party HTTPS update locations and validates the expected SHA-256 before launching Setup.

## Local verification

Before proposing a release change, run the same essential checks used by CI:

```text
go telemetry off
gofmt -w ./cmd ./internal
go vet ./...
go test ./...
python ./scripts/audit_product_copy.py
python ./scripts/generate_brand_assets.py --materialize
```

A change to release, security, privacy, updater, installer or brand-generation logic is a release-quality change and must not bypass these gates.
