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

## 构建和发布

### 本地构建

本项目是 Go + Fyne 桌面程序。Windows 本地完整构建需要：

- Go 1.24 或更新版本。
- Fyne 依赖的 C 编译环境，例如 MSYS2/MinGW 的 `gcc`。
- 可访问 `proxy.golang.org` 或配置可用的 `GOPROXY`。

常用检查命令：

```powershell
go test ./...
```

如果本地没有 `gcc`，Fyne/OpenGL 相关依赖会编译失败，例如：

```text
cgo: C compiler "gcc" not found
```

这种情况不是业务代码错误，可以优先使用 GitHub Actions 打包。

### GitHub Actions 打包

仓库内置工作流：

```text
.github/workflows/build.yml
```

触发方式：

- 推送到 `main` 或 `master` 分支：会构建并上传 Actions Artifact。
- 手动运行 `workflow_dispatch`：会构建并上传 Actions Artifact。
- 创建 GitHub Release：会构建，并把 `helium_updater-windows-amd64.zip` 自动挂到该 Release。

工作流只打包 Windows x64：

```text
fyne-cross windows -arch=amd64
```

### 通过 Release 自动发布安装包

推荐发布方式：

1. 修改 `FyneApp.toml` 中的版本号。
2. 提交并推送代码。
3. 创建 Release，例如：

```powershell
gh release create v0.1.16 --repo cwxsss/helium_plus --target <commit> --title "helium_updater v0.1.16" --notes "Release notes"
```

创建 Release 后，Actions 会自动打包，并将产物上传到 Release：

```text
helium_updater-windows-amd64.zip
```

如果只想打包不发布 Release，可以推送分支或手动运行 Actions，然后到 Actions 页面下载 Artifact。

### 代理问题记录：127.0.0.1:9

本地构建时曾遇到依赖下载被错误代理拦截：

```text
proxyconnect tcp: dial tcp 127.0.0.1:9: connectex: No connection could be made because the target machine actively refused it.
```

原因通常是当前环境变量里存在无效代理，例如：

```text
HTTP_PROXY=127.0.0.1:9
HTTPS_PROXY=127.0.0.1:9
ALL_PROXY=127.0.0.1:9
```

临时规避方式是在本次 PowerShell 会话中清空代理变量后再构建：

```powershell
$env:HTTP_PROXY=""
$env:HTTPS_PROXY=""
$env:ALL_PROXY=""
$env:http_proxy=""
$env:https_proxy=""
$env:all_proxy=""
go test ./...
```

如果需要使用代理，应改成真实可用的代理地址，例如 `127.0.0.1:7890`。不要保留 `127.0.0.1:9`，否则 Go 下载依赖、GitHub API 请求和本地打包都可能失败。

## 来源

- 更新器参考：[libsgh/chrome_updater](https://github.com/libsgh/chrome_updater)
- 增强组件：[Bush2021/chrome_plus](https://github.com/Bush2021/chrome_plus)
- Helium Windows：[imputnet/helium-windows](https://github.com/imputnet/helium-windows)
