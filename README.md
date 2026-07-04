# helium_updater

Portable Helium updater for Windows x64, adapted from `libsgh/chrome_updater` and keeping Chrome++ Next integration from `Bush2021/chrome_plus`.

## Features

- Windows x64 only.
- Downloads the latest x64 Helium package from `imputnet/helium-windows`.
- Extracts and normalizes the official package into a portable layout whose target is the `Application` folder containing `helium.exe`.
- Installs Chrome++ Next next to `helium.exe` as `version.dll` plus `chrome++.ini`.
- Supports GitHub proxy settings, self-update checks, and tray auto-checks.

## Usage

1. For a first install, choose an empty folder. The updater creates `Application`, `Cache`, and `Data`, then installs Helium into `Application`.
2. For an existing official install, select the `Application` folder, for example `%LOCALAPPDATA%\imput\Helium\Application`.
3. Configuration is stored at `%APPDATA%\helium_updater\config.json`, or next to `helium_updater.exe` when a same-folder config is present.
4. Helium and Chrome++ can be updated independently.

## Notes

The main `imputnet/helium` repository does not publish Windows binaries directly in its release assets. Windows packages are published in `imputnet/helium-windows`.

## Credits

- Updater reference: [libsgh/chrome_updater](https://github.com/libsgh/chrome_updater)
- Enhancement component: [Bush2021/chrome_plus](https://github.com/Bush2021/chrome_plus)
- Helium Windows: [imputnet/helium-windows](https://github.com/imputnet/helium-windows)
