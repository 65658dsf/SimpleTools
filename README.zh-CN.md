<div align="center">
  <img src="build/appicon.png" width="112" alt="SimpleTools 应用图标">
  <h1>SimpleTools</h1>
  <p>快速、注重隐私的图片、二维码与 PDF 桌面工具箱。</p>
  <p>
    <a href="README.md">English</a> | <strong>简体中文</strong>
  </p>
  <p>
    <a href="https://github.com/65658dsf/SimpleTools/actions/workflows/ci.yml"><img src="https://github.com/65658dsf/SimpleTools/actions/workflows/ci.yml/badge.svg" alt="CI 状态"></a>
    <a href="https://github.com/65658dsf/SimpleTools/releases/latest"><img src="https://img.shields.io/github/v/release/65658dsf/SimpleTools?display_name=tag&amp;sort=semver" alt="最新版本"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0--or--later-16a34a" alt="AGPL-3.0-or-later 许可证"></a>
    <img src="https://img.shields.io/badge/platform-Windows%2010%2F11%20%7C%20macOS%2012%2B-4b5563" alt="Windows 10 和 11、macOS 12 或更高版本">
  </p>
</div>

SimpleTools 是一款本地优先的桌面应用，用于处理日常图片和文档任务。它不需要账号，
不包含遥测、云存储或远程处理。除可选的签名更新检查外，所有处理均在本机完成。

[下载最新版本](https://github.com/65658dsf/SimpleTools/releases/latest) |
[反馈问题](https://github.com/65658dsf/SimpleTools/issues) |
[架构说明](docs/ai/architecture.md)

## 功能

| 工具 | 能力 |
| --- | --- |
| 图片转换 | 批量转换 PNG、JPEG、WebP、AVIF、ICO，以及内嵌光栅快照的自包含 SVG。 |
| 图片压缩 | 保持源格式，通过质量、无损编码或尽力满足的目标文件大小设置减小体积。 |
| 文字水印 | 支持简体中文、样式预设、九宫格定位、平铺、旋转、阴影，以及实时前后对比。 |
| 二维码 | 使用 L/M/Q/H 纠错等级生成自定义 PNG 二维码，或在本地解析二维码图片并复制结果。 |
| PDF 转 PNG | 将全部页面或从 1 开始的指定页码范围按 72、150、300 或 600 DPI 渲染为 PNG。 |

所有批处理工具还共享以下能力：

- 支持拖放文件和文件夹、递归导入文件夹与批量队列。
- 显示文件级实时进度，支持取消、重试，并在部分文件失败时保留成功结果。
- 通过临时文件原子写入，使用确定的重名后缀，绝不覆盖已有文件。
- 任务成功或部分成功后，自动在系统文件管理器中显示输出文件。
- 提供简体中文和英文界面，以及浅色、深色和跟随系统主题。

## 下载

请从 [GitHub Releases](https://github.com/65658dsf/SimpleTools/releases/latest) 下载对应安装包：

| 平台 | 支持目标 | 安装包 |
| --- | --- | --- |
| Windows 10/11 | x64 | NSIS 安装程序、便携版 ZIP |
| macOS 12+ | Intel、Apple Silicon | 通用 DMG、ZIP |

Windows 版本使用 WebView2。安装程序包含 WebView2 引导程序，便携版则要求系统中已有兼容的
WebView2 Runtime。

当前发布流程未使用 Windows 或 Apple 平台代码签名证书，因此首次启动时 SmartScreen 或
Gatekeeper 可能显示警告。发布的更新文件仍使用 Ed25519 签名和 SHA-256 校验保护。在确认并
继续之前，请先核对发布文件。

## 基本使用流程

1. 从侧边栏选择工具。
2. 将文件或文件夹拖入工作区，也可以使用系统文件选择器；生成二维码时直接输入文本。
3. 调整工具选项，并按需选择输出目录。
4. 开始处理。运行中的批量任务可以取消，失败项可以重试，已成功的结果不会丢失。

未选择输出目录时，SimpleTools 会在当前可执行文件旁创建 `output` 文件夹。如果该位置只读，
请选择其他可写目录。任务完成后，应用会在平台支持时通过系统文件管理器显示生成的文件。

## 格式说明

- 输出 JPEG 时，透明区域会铺为白色。PNG 始终无损；WebP 和 AVIF 支持有损与无损编码。
- ICO 输出包含一张以 PNG 存储的图像，最长边不超过 256 像素。
- SVG 输出是内嵌 PNG 快照的自包含 SVG。输入 SVG 必须包含受支持的内嵌光栅图，不会对任意
  矢量标记进行栅格化。
- 目标文件大小压缩是尽力而为。如果无法达到指定大小，任务会保留处理结果并给出警告，不会静默
  更换格式。
- 当前编码器会移除元数据。选择保留元数据时，应用会给出未保留的警告，不会错误宣称已保留。
  JPEG 的 EXIF 方向会在编码前应用到像素。
- PDF 逐页渲染，并拒绝受密码保护的文档。原生构建内置简体中文 Noto Sans SC 回退字体子集；
  日文、韩文、繁体中文和子集之外的字符需要 PDF 自带字体。

## 本地开发

### 环境要求

| 依赖 | 版本或说明 |
| --- | --- |
| Go | 1.26.x，准确版本见 [.go-version](.go-version) |
| Node.js 与 npm | Node.js 24.x、npm 12.x；Node.js 准确版本见 [.nvmrc](.nvmrc) |
| Wails CLI | v2.15.0 |
| 原生工具链 | MuPDF 需要支持 CGO 的 C 工具链；macOS 需要 Xcode Command Line Tools |
| Windows 打包 | NSIS 和 WebView2 引导程序 |

安装 Wails CLI 和前端依赖，然后启动开发版应用：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
cd frontend
npm ci
cd ..
wails dev
```

默认开发构建不会编译原生 MuPDF 渲染器，但图片、水印和二维码工具仍可使用。若要启用 PDF
渲染，请在具备 CGO 和所需 C 工具链的原生目标机器上运行：

```powershell
# Windows PowerShell
$env:CGO_ENABLED = "1"
wails dev -tags "mupdf,nodynamic"
```

```bash
# macOS
CGO_ENABLED=1 wails dev -tags "mupdf,nodynamic"
```

正式发布构建会在原生 Windows 和 macOS 构建机上使用相同的 `mupdf,nodynamic` 标签。

## 验证

提交变更前，请对所有受影响的层执行检查：

| 范围 | 工作目录 | 命令 |
| --- | --- | --- |
| Go | 仓库根目录 | `gofmt -l .`、`go vet ./...`、`go test ./...` |
| Wails | 仓库根目录 | `wails doctor`、`wails generate module` |
| 前端 | `frontend/` | `npm run typecheck`、`npm run lint`、`npm run test`、`npm run build` |

生成绑定后，请检查 `frontend/wailsjs`，并确认再次生成不会产生额外改动。完整的本地、原生平台与
打包验证矩阵见 [docs/ai/verification.md](docs/ai/verification.md)。

## 项目结构

依赖方向为 `frontend -> Wails bindings -> internal/app -> internal/tools` 与
`internal/platform`。完整媒体文件不会跨越原生桥接层；前端只交换路径、选项、任务 ID、进度事件
和受限尺寸的预览。

| 路径 | 职责 |
| --- | --- |
| `main.go` | Wails 入口与应用初始化 |
| `internal/app` | 请求校验、任务生命周期、绑定和事件发送 |
| `internal/tools` | 图片编解码与处理、二维码处理、PDF 渲染和校验 |
| `internal/platform` | 文件系统安全、原生对话框、输出显示和签名更新 |
| `frontend/src` | Vue 页面、Pinia 状态、Wails 适配器、翻译和样式 |
| `docs/ai` | 架构与验证契约 |
| `.github/workflows` | CI 与发布自动化 |

完整的处理流程和平台契约见 [docs/ai/architecture.md](docs/ai/architecture.md)。

## 发布

手动触发的[发布工作流](.github/workflows/release.yml)会构建 Windows x64 版本和原生 macOS
Intel/Apple Silicon 版本，将 macOS 二进制合并为通用包，生成校验值，并将 Ed25519 资产签名
写入更新清单。可再分发产物包含应用许可证和必要的第三方声明。

维护者通过 [.env.example](.env.example) 中说明的变量配置更新仓库和 Ed25519 密钥。私钥只能
保存在 CI Secret 中，绝不能随应用发布。

## 许可证

SimpleTools 使用 [GNU Affero General Public License v3 或更高版本](LICENSE)发布。MuPDF 和
其他内置组件还有各自的许可义务，完整依赖与再分发声明见
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
