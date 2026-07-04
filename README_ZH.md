# helium_updater

Helium 便携更新器，基于 `libsgh/chrome_updater` 改造，并保留 `Bush2021/chrome_plus` 的增强组件安装能力。

## 功能

- 仅支持 Windows x64。
- 从 `imputnet/helium-windows` 获取最新 Helium x64 完整安装包。
- 将官方包解压并整理为便携目录，目标目录指向包含 `helium.exe` 的 `Application` 文件夹。
- 可单独检查、下载和安装 Chrome++ Next，将 `version.dll` 和 `chrome++.ini` 放到 `helium.exe` 同级目录。
- 支持 GitHub 代理、自身更新、托盘自动检查。

## 使用

1. 首次安装时选择一个空目录，程序会创建 `Application`、`Cache`、`Data` 目录，并把 Helium 安装到 `Application`。
2. 如果已经通过官方 exe 安装过 Helium，也可以直接选择 `C:\Users\chuai\AppData\Local\imput\Helium\Application`。
3. 配置默认保存到 `%APPDATA%\helium_updater\config.json`；如果 `helium_updater.exe` 同级存在配置文件，则优先使用同级配置。
4. 主程序和 Chrome++ 可以分别更新，互不影响。

## 说明

Helium 主源码仓库 `imputnet/helium` 的 release 不直接携带 Windows 安装包；Windows 包在 `imputnet/helium-windows` 仓库发布。

## 来源

- 更新器参考：[libsgh/chrome_updater](https://github.com/libsgh/chrome_updater)
- 增强组件：[Bush2021/chrome_plus](https://github.com/Bush2021/chrome_plus)
- Helium Windows：[imputnet/helium-windows](https://github.com/imputnet/helium-windows)
