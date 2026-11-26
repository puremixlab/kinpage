# Microsoft Store MSIX 打包说明

本仓库提供面向 Microsoft Store 与本地侧载测试的 MSIX 打包配置，位于 `windows/msix/` 与 `windows/scripts/build-msix.ps1`。

- [windows/msix/msix.config.json](../msix/msix.config.json)：包标识、发布者、显示名称与支持语言配置。
- [windows/msix/AppxManifest.template.xml](../msix/AppxManifest.template.xml)：`AppxManifest.xml` 模板，声明 `internetClient`、`privateNetworkClientServer` 与 `runFullTrust` 权限。
- [windows/scripts/build-msix.ps1](../scripts/build-msix.ps1)：校验版本与构建产物、从 `windows/resources/icon.png` 生成商店视觉素材，并调用 Windows SDK 的 `makeappx.exe` 生成 `.msix`。

## 前置条件

1. 已安装 Windows 10/11 SDK（包含 `makeappx.exe`，脚本会自动在 `C:\Program Files (x86)\Windows Kits\10\bin\` 下查找最新 x64 版本，也可用 `-MakeAppx` 手动指定）。
2. 已在 `windows/` 下运行 `bun run build`，生成最新的 `build/windows-amd64/` 产物。

## 本地打包与测试

在 `windows/` 下执行：

```powershell
bun run build:msix
```

脚本会校验 `Kinpage.exe --planet-version` 与 [windows/release.json](../release.json) 一致，自动生成全部 MSIX PNG 图标（若在 `windows/msix/Assets/` 放置同名文件则优先使用自定义素材），并在 `.mygo/msix/<构建ID>/` 下输出 `kinpage-<版本>-x64.msix` 及 SHA-256 摘要。

如需指定独立发行目录或输出路径：

```powershell
powershell -NoProfile -File scripts/build-msix.ps1 `
  -PayloadDir '.mygo/releases/<构建ID>/windows-amd64' `
  -OutputFile '.mygo/msix/kinpage-x64.msix'
```

## 提交到 Microsoft Store

1. 在 Microsoft Partner Center 保留应用名称后，从 **产品管理 → 产品标识** 获取微软分配的 `Package/Identity/Name`、`Package/Identity/Publisher` 与 `Package/Properties/PublisherDisplayName`。
2. 更新 `windows/msix/msix.config.json`，或在打包时通过参数传入，并开启 `-Store` 校验模式：

```powershell
powershell -NoProfile -File scripts/build-msix.ps1 `
  -Store `
  -PackageName '<PartnerCenterPackageName>' `
  -Publisher 'CN=<PartnerCenterPublisherId>' `
  -PublisherDisplayName '<YourPublisherDisplayName>'
```

`-Store` 模式会强制检查：

- `Publisher` 不能为默认的本地占位符 `CN=KinpageLocalDev`。
- MSIX 版本号为四段纯数字且末位为 `0`（由 `windows/release.json` 自动派生为 `Major.Minor.Patch.0`）。
- 禁止直接将预发布版本打包提交商店（若仅用于商店内部测试轨道，可显式附加 `-AllowPrerelease`）。

上传到 Partner Center 后，微软审核流程会自动对 `.msix` 重新签名，无需预先购买商业代码签名证书。

## 本地侧载与 WACK 预检

- **本地侧载安装**：Windows 安装未过审的本地 `.msix` 需要先使用自签名证书签名（`signtool.exe sign /fd SHA256 ...`）并将证书导入本机“受信任的人”存储区；请勿将任何 `.pfx` 私钥提交到仓库。
- **WACK 预检**：提交商店前建议使用 Windows SDK 自带的 **Windows App Certification Kit (WACK)** 对 `.msix` 执行本地认证检查。

## 已知运行注意事项

- **版本号规则**：MSIX 仅支持四段 uint16 数字（`Major.Minor.Build.Revision`），不支持 SemVer 预发布后缀；`windows/release.json` 仍是唯一版本来源。
- **容器虚拟化与子进程**：MSIX 会虚拟化 `%APPDATA%` / `%LOCALAPPDATA%` 写入。打包后须在干净 Windows 10/11 环境下实际验证内置 `ipfs.exe`（Kubo）启动/退出、本地网关端口、WebView2 渲染与备份导入导出。
