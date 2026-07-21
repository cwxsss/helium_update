# helium_updater

Portable Helium updater for Windows x64, adapted from `libsgh/chrome_updater` and keeping Chrome++ Next integration from `Bush2021/chrome_plus`.

## Features

- Windows x64 only.
- Downloads the latest x64 Helium package from `imputnet/helium-windows`.
- Extracts and normalizes the official package into a portable layout whose target is the `Application` folder containing `chrome.exe`.
- Downloads only `version.dll` from the Helium++ package and writes it as `version.dll.new` next to `chrome.exe`; it never changes `chrome++.ini`.
- Supports GitHub proxy settings, self-update checks, and tray auto-checks.

## Usage

1. For a first install, choose an empty folder. The updater creates `Application`, `Cache`, and `Data`, then installs Helium into `Application`.
2. For an existing official install, select the `Application` folder, for example `%LOCALAPPDATA%\imput\Helium\Application`.
3. Configuration is stored at `%APPDATA%\helium_updater\config.json`, or next to `helium_updater.exe` when a same-folder config is present.
4. Helium and Helium++ can be updated independently. After a Helium++ download, close Helium and this updater, then manually replace `version.dll` with `version.dll.new` in the `Application` directory.

## Notes

The main `imputnet/helium` repository does not publish Windows binaries directly in its release assets. Windows packages are published in `imputnet/helium-windows`.

## Build And Release

### Local Build

This is a Go + Fyne desktop app. A full local Windows build requires:

- Go 1.24 or newer.
- A C compiler required by Fyne, for example MSYS2/MinGW `gcc`.
- Access to `proxy.golang.org`, or a working `GOPROXY`.

Basic check:

```powershell
go test ./...
```

If `gcc` is missing, Fyne/OpenGL dependencies fail with an error like:

```text
cgo: C compiler "gcc" not found
```

That is a local toolchain issue, not necessarily an app code issue. Use GitHub Actions for packaging when the local machine does not have the full Fyne build environment.

### GitHub Actions Packaging

The workflow is defined at:

```text
.github/workflows/build.yml
```

Triggers:

- Push to `main` or `master`: builds and uploads an Actions artifact.
- `workflow_dispatch`: builds and uploads an Actions artifact.
- GitHub Release creation: builds and attaches the package to that Release.

The workflow builds Windows x64 only:

```text
fyne-cross windows -arch=amd64
```

### Release Publishing

Recommended release flow:

1. Update `Version` in `FyneApp.toml`.
2. Commit and push the code.
3. Create a GitHub Release:

```powershell
gh release create v0.1.18 --repo cwxsss/helium_update --target <commit> --title "helium_updater v0.1.18" --notes "Release notes"
```

After the Release is created, GitHub Actions builds and uploads:

```text
helium_updater-windows-amd64.zip
```

To build without publishing a Release, push a branch or run the workflow manually, then download the artifact from the Actions page.

### Proxy Trap: 127.0.0.1:9

Local builds previously failed because dependency downloads were routed through a dead proxy:

```text
proxyconnect tcp: dial tcp 127.0.0.1:9: connectex: No connection could be made because the target machine actively refused it.
```

Check for invalid proxy environment variables:

```text
HTTP_PROXY=127.0.0.1:9
HTTPS_PROXY=127.0.0.1:9
ALL_PROXY=127.0.0.1:9
```

Temporary PowerShell fix for the current session:

```powershell
$env:HTTP_PROXY=""
$env:HTTPS_PROXY=""
$env:ALL_PROXY=""
$env:http_proxy=""
$env:https_proxy=""
$env:all_proxy=""
go test ./...
```

If a proxy is required, set it to a real local proxy such as `127.0.0.1:7890`. Do not keep `127.0.0.1:9`; it can break Go dependency downloads, GitHub API requests, and local packaging.

## Credits

- Updater reference: [libsgh/chrome_updater](https://github.com/libsgh/chrome_updater)
- Enhancement component: [Bush2021/chrome_plus](https://github.com/Bush2021/chrome_plus)
- Helium Windows: [imputnet/helium-windows](https://github.com/imputnet/helium-windows)
