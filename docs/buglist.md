# Kinpage Bug List

记录当前已确认并待排查修复的问题清单。

## 总览

| 编号 | 优先级 | 状态 | 摘要 |
| --- | --- | --- | --- |
| BUG-001 | P1 | 已修复 | 慢速网关（如 `eth.limo` / `eth.sucks`）订阅网站头像抓取超时回退为默认地球图标 |
| BUG-002 | P1 | 已修复 | 慢速网关下订阅文章配图触发 20 秒总超时导致页面图片不显示 |
| BUG-003 | P2 | 已修复 | 自己文章阅读页与编辑器预览页无法渲染相对路径本地附件图 |
| BUG-004 | P1 | 已修复，待实机复核 | 新关注和手动刷新在界面线程等待网络，窗口卡住 |
| BUG-005 | P1 | 已修复，已验证跨盘复制 | 迁移数据仓库遗漏文章和草稿附件 |
| BUG-006 | P1 | 已升级修复 4 项，余 1 项 DHT 通告 | 随包分发的 Kubo 命中已知安全通告 |
| BUG-007 | P1 | 已恢复，代码 PR 检查通过 | GitHub 的 Windows 质量检查和 main 合并门禁失效 |
| BUG-008 | P1 | 已标注，待签名发行 | 正式发布的 v0.1.0 安装包未签名 |
| BUG-009 | P2 | 已修复 | .planet 备份重新导入后可能无法再次导出 |
| BUG-010 | P2 | 已修复 | 编辑器显示已发布，站点仍待手动发布 |
| BUG-011 | P2 | 已修复，待外部发布验收 | 手动发布站点时窗口阻塞 |
| BUG-012 | P2 | 已修复 | 托盘退出会丢弃未保存编辑 |
| BUG-013 | P2 | 已修复 | 订阅缺少自动刷新，“今天”漏掉常见 RSS 日期 |
| BUG-014 | P2 | 已修复 | 文档中的检查更新缺少桌面入口 |
| BUG-015 | P3 | 已修复，原生卸载验收通过 | NSIS 卸载留下应用创建的桌面快捷方式 |

---

## 详情

### BUG-001 · 慢速网关订阅网站头像抓取超时回退为默认地球图标

- **优先级**：P1
- **状态** 已修复。
- **现象**：订阅的网站原本有正常头像，但在侧边栏中回退显示为默认地球图标。
- **复现方式**：订阅 `https://planetable.eth.limo`。
- **实测数据**：`https://planetable.eth.limo/avatar.png` 返回 HTTP `200`（776,520 字节），但在网关限速下 15 秒仅收到 196,608 字节，完整下载约需 35–50 秒。
- **根因**：
  1. `windows/internal/feeds/icon.go` 中 `iconFetchTimeout = 6 * time.Second`，且 `fetchSiteIcon` 的所有候选图标共用同一个 `ctx`；首个候选 `avatar.png` 体积较大时会在 6 秒内耗尽总预算并中断循环，导致体积更小的 `favicon.ico` 无法继续尝试。
  2. 头像请求使用默认 `http.Client`（未强制 HTTP/1.1）与 `User-Agent: Kinpage/0.1`；此外 `windows/internal/siteimage/icon.go` 的 `NormalizeIcon` 目前仅支持 PNG、JPEG、GIF 和 ICO，不支持 WebP。
- **涉及文件**：
  - `windows/internal/feeds/icon.go`
  - `windows/internal/siteimage/icon.go`
  - `windows/internal/nativeui/theme.go`
- **建议修复方向**：
  - 为单个候选图标设置独立超时预算，避免首个大图饿死后续 `favicon.ico` 回退。
  - 放宽头像下载读取超时，补充 HTTP/1.1 传输配置，并为 `NormalizeIcon` 增加 WebP 解码支持。
- **本轮修复与验证** HTTP 头像在文章列表保存后读取，使用 HTTP/1.1、单候选 55 秒和全流程 90 秒上限。回归用例覆盖 TLS 的 HTTP/1.1 协商、头像停滞后回退到 `favicon.ico` 和 WebP 转 PNG。使用 `https://planetable.eth.limo` 实测，35 篇文章约 1.83 秒就绪，随后约 4.63 秒得到 128 × 128 的头像。 [头像回归](../windows/internal/feeds/icon_test.go) · [WebP 回归](../windows/internal/siteimage/icon_test.go)

---

### BUG-002 · 慢速网关下订阅文章配图触发 20 秒总超时导致页面图片不显示

- **优先级**：P1
- **状态** 已修复。
- **现象**：订阅文章阅读页中的远程截图或配图无法正常显示。
- **复现方式**：打开订阅源 `https://planetable.eth.limo` 下的文章（其 `rss.xml` 正文图片指向 `https://planetable.eth.sucks/...`）。
- **实测数据**：`cf-pages.png` 在 `eth.sucks` 上下载 288,792 字节耗时 12.64 秒；在 `eth.limo` 上下载 437,668 字节时 15 秒仅传完 248,694 字节即超时。
- **根因**：`windows/internal/nativeui/images.go` 中 `http.Client.Timeout` 设置为 20 秒整请求硬超时（包含响应体读取全过程）。在约 13–14 KB/s 网速下，超过 ~280 KB 的文章配图会在下载中途被 20 秒超时掐断，并写入 30 秒失败缓存（`imageErrorCacheTTL`）。
- **涉及文件**：
  - `windows/internal/nativeui/images.go`
  - `windows/internal/nativeui/subscriptions_view.go`
- **建议修复方向**：
  - 移除 20 秒整请求硬超时，改用 `ResponseHeaderTimeout`（如 15 秒）配合读取停滞超时（stall timeout，只要持续收到数据就不中断），并将单图总下载上限放宽至 90 秒。
- **本轮修复与验证** 等待响应头和连续无数据各限 15 秒，单图下载总上限为 90 秒。保留原有体积校验和 `Referer` 回退。本地分块响应测试验证慢速传输可解码，停止传输时请求会被取消。原生阅读器使用的图片加载器也成功解码了 `eth.sucks` 和 `eth.limo` 上的 `cf-pages.png`，当次耗时分别约 2.04 秒和 2.46 秒。 [图片回归](../windows/internal/nativeui/webview_images_test.go)

---

### BUG-003 · 自己文章阅读页与编辑器预览页无法渲染相对路径本地附件图

- **优先级**：P2
- **状态** 已修复。
- **现象**：自己文章或草稿的 Markdown 正文中引用相对路径本地图片（非 `http(s)://` 或 `data:`）时，阅读页与编辑器预览页均不显示图片。
- **复现方式**：在文章或草稿正文中使用相对路径引用本地图片，随后打开文章阅读页或编辑器预览视图。
- **根因**：`windows/internal/nativeui/detail.go`（L134）和 `windows/internal/nativeui/editor.go`（L124）调用 `a.markdownImageLoader("", "")` 时 `localDir` 参数均传为空字符串，导致相对路径直接落入 `unsupported image source` 错误分支。
- **涉及文件**：
  - `windows/internal/nativeui/detail.go`
  - `windows/internal/nativeui/editor.go`
  - `windows/internal/nativeui/images.go`
- **建议修复方向**：
  - 为文章阅读页与编辑器预览页的 `markdownImageLoader` 传入对应文章或草稿的本地资源目录作为 `localDir`（保留防路径穿越校验）。
- **本轮修复与验证** 阅读页及编辑器预览通过既有附件校验读取当前文章或草稿的已登记图片。草稿可预览暂存附件和原文章中仍保留的附件。离屏测试覆盖带空格的中文文件名，库端用例覆盖未登记和已移除附件。 [视图回归](../windows/internal/nativeui/webview_images_test.go) · [注册附件回归](../windows/main_test.go)

---

### BUG-004 · 新关注和刷新时窗口卡住

- **优先级** P1
- **状态** 已修复，待 Windows 实机慢网关复核。原始源码和离屏 UI 复现基线为 `0fac18c`。
- **现象** 从「设置 → 关注站点」提交一个响应慢的站点后，窗口在读取站点信息和文章时停顿。「刷新订阅」和工作台的「关注精选站点」也会等待网络。
- **调用链** MyGo 的 `App.View` 渲染关注弹窗。提交按钮和回车都在当前界面帧调用 `submitFollow()`，它同步等待 `Library.Follow()` 完成抓取、解析和保存。成功后再调用 `Reload()`、打开订阅和显示结果。在 `Follow()` 返回前，当前帧无法结束，窗口也无法重绘或处理新的输入。 [弹窗与提交处理](../windows/internal/nativeui/subscriptions_view.go) · [库端订阅入口](../windows/library_subscriptions.go)
- **HTTP(S) 订阅** 打开首页、按顺序尝试发现的订阅源、解析 Feed 后才寻找头像。首页和订阅源的网络请求共用 15 秒预算，头像发现另有独立的 6 秒预算，两个阶段都在同一次 `Follow()` 内。普通 RSS、Atom 和 JSON Feed 由订阅源文档提供文章清单和已附带的正文。 [Feed 读取](../windows/internal/feeds/feeds.go) · [头像候选](../windows/internal/feeds/icon.go)
- **Planet、IPNS 和 ENS 订阅** 必要时先启动 Kubo，再依次解析站点名、读取 `planet.json`、尝试 `avatar.png` 和 `favicon.ico`、遍历同一份元数据中的文章。每次 Kubo 内容读取有各自的 90 秒超时。 [Planet 订阅流程](../windows/internal/follow/follow.go) · [Kubo 内容读取](../windows/internal/ipfs/content.go)
- **额外等待** HTTP 抓取持有 `feedMu`，Planet 抓取持有与发布共享的 `publicMu`。工作台逐个关注精选站点。关注成功后的 `Reload()` 还会同步读取写作台等全局数据并查询 Kubo 状态。如果自动选中的第一篇未读，还会写入已读状态并再次全量刷新。 [锁与刷新入口](../windows/library_subscriptions.go) · [批量关注](../windows/internal/nativeui/dashboard_view.go) · [全量刷新](../windows/internal/nativeui/app.go)
- **复现证据** 在隔离副本用 MyGo 离屏测试点击「关注」，让模拟后端等待 300 毫秒。点击和当前帧一直未返回。释放后端后，界面完成提交并显示订阅。现有首页发现订阅源、保存头像及刷新保留已读状态的本地测试通过。真实慢站点的卡顿时长尚未测量。
- **建议修复方向** 用可取消的后台任务完成关注、刷新和更改订阅地址，通过 `Window.Update` 回到界面线程提交状态。立即显示进度，Feed 就绪即显示列表，头像稍后补齐。缩短持锁范围，提交时按操作复核目标和请求是否仍有效，防止取消关注后迟到的刷新把它创建回来。优先刷新订阅相关视图，避免连续全量重载。
- **验收** 用本地延迟 HTTP 响应和模拟慢 Kubo 验证等待期间窗口可重绘、输入和取消。头像超时时文章列表仍可用。重复点击仅产生一次有效提交，取消或切换后过期结果不会覆盖当前界面，刷新失败保留已有文章和阅读状态。
- **本轮修复与验证** 新关注、手动刷新、编辑订阅和关注精选在后台执行。关闭关注或编辑弹窗会取消请求。保存文章后界面只刷新订阅，头像独立加载并核对内容版本。离屏测试覆盖待处理期间重绘、重复提交和取消，本地 HTTP 测试验证取消后不会写入或恢复已移除的订阅。既有 Planet 订阅及固定内容用例也已用 Kubo 跑通。订阅提交沿用既有并发保护。 [后台任务](../windows/internal/nativeui/subscription_tasks.go) · [取消与延迟头像回归](../windows/subscription_discovery_test.go)

---

### BUG-005 · 数据仓库迁移漏掉附件

- **优先级** P1
- **状态** 已修复，已验证跨盘复制。
- **现象与根因** 迁移白名单遗漏 `attachments`、`draft-attachments` 和本地 API 配置所在的 `integrations`。迁移后的 `library.json` 仍有附件引用，重启后新库找不到对应文件。迁移完成到重启之间的保存仍写入旧库。 [仓库迁移](../windows/internal/store/repo_location.go) · [文章附件](../windows/internal/attachments/attachments.go) · [草稿附件](../windows/internal/attachments/draft_lifecycle.go)
- **建议修复方向** 在下次启动、业务模块打开仓库前执行迁移。用统一清单复制持久数据，在目标卷暂存和校验，最后切换仓库指针。
- **验收** 跨盘迁移后附件可预览、导出和发布，本地 API 配置可读取，重启前的保存进入新库。磁盘写满或复制中断后仍可打开原库并重试。
- **本轮修复与验证** 选择目标后在下次启动复制数据，目标先暂存文件并校验大小，成功后更新指针。测试覆盖文章和草稿附件、本地 API 配置、重启前最后一次保存、目标目录发生变化后的重试，以及原盘不可访问后重新连接。用隔离仓库从 `C:` 复制到 `D:` 并重新打开，附件和配置均可读。跨盘后的发布、磁盘写满和突然中断仍待实机验证。 [迁移回归](../windows/internal/store/repo_location_integration_test.go)
- **Windows 路径复核** CI 的临时目录采用 8.3 路径，而迁移逻辑保存规范长路径。断言已改为核对目录身份，三个相关用例都通过了 8.3 临时目录复测。 [CI 检查](https://github.com/puremixlab/kinpage/actions/runs/46100214525)

---

### BUG-006 · 随包 Kubo 命中已知安全通告

- **优先级** P1
- **状态** 已升级修复 4 项。默认联网模式的 DHT 客户端仍受 `GO-2024-3218` 影响，上游暂无已知修复版本。
- **历史对照** 包内符号级匹配在 Kubo `0.38.2` 时为 68 项，官方 Windows x64 `0.43.2` 时为 27 项，使用 Go `1.27.4` 重编译时为 5 项。本轮依赖升级后的两个包只剩 1 项。 [Kubo 0.43.2](https://github.com/ipfs/kubo/releases/tag/v0.43.2) · [安装包审计](../windows/docs/release-audit.md)
- **默认配置取证** 在隔离目录调用 Kinpage 的真实 Kubo 初始化入口，确认 `Routing.Type=autoclient`，RPC 和网关仅监听 IPv4 回环地址，libp2p 网关和流挂载关闭，存储采用 FlatFS/LevelDB。根据固定源码和 Windows x64 的包依赖图判断各分支的可达性。 [Kinpage 节点配置](../windows/internal/ipfs/daemon.go) · [Kubo 默认存储](https://github.com/ipfs/kubo/blob/v0.43.2/config/init.go#L122-L132)

| 通告 | 升级前在默认配置下的影响 | 本轮处置 |
| --- | --- | --- |
| `GO-2024-3218` | **联网时受影响。** `autoclient` 仍启用 DHT 客户端，DHT 路由受干扰时可能影响内容发现。 | `go-libp2p-kad-dht v0.42.2` 仍有匹配，上游暂无已知修复版本。 [Go 通告](https://pkg.go.dev/vuln/GO-2024-3218) · [Kubo 路由](https://github.com/ipfs/kubo/blob/v0.43.2/cmd/ipfs/kubo/daemon.go#L489-L501) |
| `GO-2026-6443` | 默认配置不运行 gRPC xDS 服务端。实际编译图也未链接 xDS 服务端包，gRPC 主要由 OTLP 导出客户端及其依赖引入。 | gRPC `1.83.1 → 1.83.2`，复扫无匹配。 [Go 通告](https://pkg.go.dev/vuln/GO-2026-6443) |
| `GO-2026-6865` | 默认 FlatFS/LevelDB 不走该 zstd 路径。实际编译图中的 zstd 由可选的 Pebble 数据存储引入。 | `klauspost/compress 1.20.1 → 1.20.2`，复扫无匹配。 [Go 通告](https://pkg.go.dev/vuln/GO-2026-6865) |
| `GO-2026-6974` | 默认未配置追踪导出器，Kubo 使用空操作 TracerProvider。相关 SDK 路径与启用追踪及属性长度限制的配置有关。 | OpenTelemetry 的 SDK 和 Trace 升到 `1.47.0`，复扫无匹配。 [Go 通告](https://pkg.go.dev/vuln/GO-2026-6974) · [Kubo 追踪设置](https://github.com/ipfs/kubo/blob/v0.43.2/tracing/tracing.go#L33-L61) |
| `GO-2026-6975` | **本机 HTTP 请求可达。** 默认传播器注册了 baggage，网关和 RPC 的 HTTP 中间件会提取。回环绑定限制了默认监听范围。 | OpenTelemetry 主模块升到 `1.47.0`，复扫无匹配。 [Go 通告](https://pkg.go.dev/vuln/GO-2026-6975) · [Kubo HTTP 网关](https://github.com/ipfs/kubo/blob/v0.43.2/core/corehttp/gateway.go) |

- **新包复扫** 2026-11-28 用 Go `1.27.4` 和固定的依赖补丁构建 Kubo `0.43.2`。NSIS 与 MSIX 的十项共有文件通过摘要核对，两份包中提取的 `ipfs.exe` SHA-256 均为 `ba75778d566239e1b0f068d6fdd8fd5490b5bba7306c507e33079881e5099e22`。针对这份二进制的 `govulncheck v1.9.0` 仅匹配 `GO-2024-3218`。242 个依赖模块的许可材料已绑定同一摘要，缺项为零。 [构建脚本](../windows/scripts/prepare-kubo.ps1) · [依赖补丁](../windows/scripts/release-audit/kubo-dependencies.patch)
  - NSIS SHA-256 `771fdee21d9163c67b0df86ecc12ba3a280c7833dddc9e5bc8d237fda1e0915b`
  - MSIX SHA-256 `9bdb986732e5c4afc9b51593feae8f20d39ba25d68359dbafdc62a37812ca9df`
- **兼容性复核** 默认配置、离线网关读取、节点启停、固定内容、Planet 关注及两项备份往返均通过。带补丁的 Kubo 从 `0.38.2` 隔离仓库副本读取到了相同的 IPNS 密钥与递归固定记录，也能读取前一构建创建的 Pebble 仓库及其固定内容。 [网关状态用例](../windows/internal/ipfs/status_test.go)

---

### BUG-007 · GitHub 的 Windows 质量检查门禁失效

- **优先级** P1
- **状态** 工作流和必需检查已恢复，代码 PR 检查通过。
- **证据** 2026-11-27 通过 GitHub API 核对，`Windows checks` 为 `disabled_manually`，当前 `main` 生效规则中没有必需状态检查。仓库里的规则文件仍记录了 `Windows quality checks`。 [规则记录](../.github/rulesets/main.json) · [Windows 工作流](https://github.com/puremixlab/kinpage/actions/workflows/windows-ci.yml)
- **建议修复方向** 恢复工作流，并在远端启用该任务作为 `main` 的必需检查。保留纯文档与代码改动的检查分流。
- **验收** 两类 PR 都产生同名检查结果。使用无绕过权限的账号验证检查失败时无法合并。
- **本轮修复与验证** 恢复工作流后的 `main` 检查发现 Windows 快捷方式测试受 8.3 路径别名影响而失败。已修复路径判断，独立分支和代码 PR 的 `Windows quality checks` 均通过。GitHub 当前将该检查强制用于 `main`，要求分支同步，并保留既有管理规则。纯文档 PR 和无绕过账号的实际合并行为待验收。 [首次运行](https://github.com/puremixlab/kinpage/actions/runs/46091924822) · [代码 PR 的检查](https://github.com/puremixlab/kinpage/actions/runs/46093386328) · [路径修复 PR](https://github.com/puremixlab/kinpage/pull/106)
- **复核** 修复 PR 的 `Windows quality checks` 已通过，GitHub API 显示该检查仍是 `main` 的合并门禁。 [修复 PR 的检查](https://github.com/puremixlab/kinpage/actions/runs/46100214525)

---

### BUG-008 · 正式发布的 v0.1.0 安装包未签名

- **优先级** P1
- **状态** 下载说明已更正，签名发行待完成。
- **证据** GitHub 将 `v0.1.0` 标记为正式发布。已下载的 NSIS 和 MSIX 均与公布的 SHA-256 一致，Windows 对两份文件均返回 `NotSigned`。 [发行页](https://github.com/puremixlab/kinpage/releases/tag/v0.1.0) · [稳定版签名规范](releasing.md)
- **建议修复方向** 在下载入口标注 `v0.1.0` 的历史未签名状态。下一个稳定版使用新版本号并核验第一方程序及分发包的签名和发布者。
- **验收** 最终发布的字节有与之对应的 SHA-256、源码清单和 Windows 10/11 原生安装、升级、卸载验收记录。
- **本轮修复与限制** 中英文下载入口已明确标出 `v0.1.0` 的 NSIS、MSIX 未签名，并区分文件摘要与发布者签名。本机证书库未找到代码签名证书。签名稳定版仍需要受信任的发布者签名和最终安装包的原生验收。 [中文下载入口](../README_zh.md#下载安装) · [英文下载入口](../README.md#download)

---

### BUG-009 · .planet 备份再次导出失败

- **优先级** P2
- **状态** 已修复。
- **证据** 使用真实 Kubo 离线复跑，`TestTaggedBackupRoundTripAfterArticleDeletion` 在二次导出时拒绝此前生成的 `heroImage` 字段。`TestSepiaTimestampedTagPagesKuboBackupRoundTrip` 在重导入并删除文章后无法确认旧标签页归属。 [往返测试](../windows/internal/migration/tags_test.go) · [公开元数据校验](../windows/internal/backup/plain.go)
- **建议修复方向** 统一导出和重建对 `heroImage` 的规则，接受与已登记图片附件匹配的安全值。保留对未知归档文件的保护，记录原标签页重建失败的具体原因。
- **验收** 两条离线往返测试通过，修改后的标签页正确更新，无法归属的自定义归档页受到保护。通过后将这些用例纳入相关 PR 和发行候选检查。
- **本轮修复与验证** 可重新导出由已登记图片生成的封面数据；对归档中手动选择的封面按附件是否还在决定是否保留。无法确认标签页归属时会保留原错误和重建失败原因。使用现有随包 Kubo 离线复跑，两条往返用例均通过。 [备份往返回归](../windows/internal/migration/tags_test.go)

---

### BUG-010 · 编辑器显示已发布，站点仍待发布

- **优先级** P2
- **状态** 已修复。
- **现象与根因** 编辑器把草稿保存为文章后立即显示「已发布」。现有后台维护会跳过 `PublishPending`，实际发布仍需用户手动触发。 [编辑器发布](../windows/internal/nativeui/app.go) · [后台维护](../windows/library_publish.go)
- **建议修复方向** 先将编辑器反馈改为「已保存，待发布」，再将两处明确的发布操作接入同一后台队列。既有待发布标记作为提醒，公开发布由用户明确发起。
- **验收** 以实际发布的内容版本为准显示成功。旧任务完成后，更新的内容仍正确显示为待发布。
- **本轮修复与验证** 编辑器把草稿存为文章时显示「已保存，待发布」。站点发布按本次快照的版本显示结果，已有更新的版本继续提示待发布。离屏回归验证本地保存不会触发站点发布。 [发布回归](../windows/internal/nativeui/publish_async_test.go)

---

### BUG-011 · 手动发布站点时窗口阻塞

- **优先级** P2
- **状态** 已修复，真实第三方发布待验收。
- **根因** `triggerPublishSite` 直接在界面线程等待 Kubo 和已启用的外部服务，调用返回后才显示结果。 [界面入口](../windows/internal/nativeui/dialogs_site.go) · [外部发布](../windows/deployment.go)
- **建议修复方向** 与 BUG-010 共用后台队列。按站点串行发布不可变快照，界面显示准备、发布中、成功及失败状态。保留外部服务已有的防重复规则和结果不确定状态。
- **验收** 用模拟慢网络验证窗口可继续操作。离线任务重新启动后可重试，并准确报告各已配置目标的结果。
- **本轮修复与验证** 工具栏与站点菜单共用后台发布队列，重复提交同一个站点时保留原任务。站点文件和附件先生成独立快照，再提交给 Kubo 和已配置的外部目标。发布详情可查看目标结果。离屏回归验证模拟慢发布时当前帧仍可返回。未成功的站点版本继续标记为待发布，可在恢复连接后发起重试。 [后台队列](../windows/internal/nativeui/publish_tasks.go) · [发布快照](../windows/library_publish.go)

---

### BUG-012 · 托盘退出丢弃未保存编辑

- **优先级** P2
- **状态** 已修复。
- **根因** 托盘真正退出时只针对正在发布的操作确认，未调用已有的编辑器未保存检测。 [托盘退出](../windows/main.go) · [未保存检查](../windows/internal/nativeui/dialogs_article.go)
- **建议修复方向** 托盘退出及迁移重启复用「保存并继续／放弃／取消」。
- **验收** 保存失败可继续编辑，取消后稿件仍在，明确放弃可以结束进程。
- **本轮修复与验证** 托盘退出与无托盘时关闭窗口共用编辑器的保存确认。迁移计划仍在下次启动应用，当前编辑由同一个退出流程保护。离屏回归覆盖取消、保存后退出、保存失败留在编辑器及明确放弃。 [退出回归](../windows/internal/nativeui/exit_guard_test.go)

---

### BUG-013 · 订阅自动刷新缺失且“今天”漏项

- **优先级** P2
- **状态** 已修复。
- **现象与根因** 启动流程没有接入指南所述的每 15 分钟刷新任务。「今天」列表和计数按原始日期字符串前缀筛选，会漏掉常见 RFC 1123 日期。 [启动流程](../windows/main.go) · [日期筛选](../windows/internal/nativeui/subscriptions_view.go) · [用户指南](../windows/README.md)
- **建议修复方向** 在 BUG-004 的后台订阅流程上接入自动刷新，失败保留缓存并按退避策略重试。列表和计数共用本地日期判断，兼容现有订阅数据的日期格式。
- **验收** 覆盖定时刷新、暂时离线、RFC 1123、RFC 3339 以及跨时区午夜。
- **本轮修复与验证** 启动时以及之后每 15 分钟检查过期订阅。失败后保留缓存和阅读状态，自动重试按 15 分钟起步并逐次延长至最多 2 小时。离线回归覆盖恢复、有无请求和阅读状态。「今天」列表与计数共用系统时区判断，测试覆盖历史日期和跨时区午夜。 [自动刷新回归](../windows/subscription_refresh_test.go) · [日期回归](../windows/internal/nativeui/subscription_today_test.go)

---

### BUG-014 · 检查更新缺少桌面入口

- **优先级** P2
- **状态** 已修复。
- **根因** 应用保留了检查、下载和自动检查配置的后端逻辑，当前设置界面与启动流程未接入指南描述的入口。 [更新后端](../windows/app_updates.go) · [用户指南](../windows/README.md)
- **建议修复方向** 接入手动检查。用户配置可信源后，按开关执行启动检查，沿用现有签名与 SHA-256 校验。
- **验收** 覆盖未配置、离线、清单签名无效、下载文件哈希不符和合法更新。
- **本轮修复与验证** 通用设置提供「检查更新」、可信源配置、启动检查开关及校验后的下载入口。手动检查和下载在后台执行。离屏回归覆盖未配置入口及启动检查开关，既有后端用例覆盖签名、回滚和摘要校验。 [界面回归](../windows/internal/nativeui/update_entry_test.go) · [更新协议回归](../windows/internal/updates/updates_test.go)
- **安装包下载复核** 使用回环 TLS 服务、临时 Ed25519 签名清单和真实构建的 NSIS 安装包，完成下载及磁盘摘要复核。 [安装包下载用例](../windows/internal/updates/updates_test.go)

---

### BUG-015 · NSIS 卸载留下桌面快捷方式

- **优先级** P3
- **状态** 已修复，隔离 Windows 原生卸载验收通过。
- **根因** 应用可创建桌面 `Kinpage.lnk`，NSIS 卸载只处理开始菜单快捷方式。 [桌面快捷方式](../windows/internal/desktopshortcut/shortcut_windows.go) · [卸载逻辑](../windows/scripts/release-audit/safe-installer.nsi)
- **建议修复方向** 卸载时核对链接归属和目标，清理应用创建并指向本次安装的桌面链接。
- **验收** 由本次安装创建的桌面链接被移除，同名外部链接和用户数据得到保留。
- **本轮修复与验证** NSIS 验证安装归属后，检查桌面快捷方式的目标程序和应用管理的图标，并核对原链接字节后移除。真实 Windows COM 回归覆盖归属当前安装的快捷方式、指向其他程序的同名链接，以及用户自行配置图标的同目标链接。隔离 Windows 安装验收脚本也覆盖了拥有和外部链接两种情况。 [快捷方式回归](../windows/internal/desktopshortcut/uninstall_windows_test.go) · [原生验收脚本](../windows/scripts/test-installer-native.ps1)
- **原生验收** 2026-11-28 的隔离 Windows 工作流通过七项场景。实际运行 NSIS 卸载后，本次安装创建的桌面快捷方式已删除，同名外部链接和用户数据的摘要一致。工作流保存了受测安装包、静态审计和 Kubo 二进制扫描。 [原生验收记录](https://github.com/puremixlab/kinpage/actions/runs/46100211329)

## 建议实施顺序

1. 恢复 CI 门禁、修复数据仓库迁移和备份往返，分别提交独立 PR。同步处理 BUG-004 的订阅阻塞，并在隔离环境评估 Kubo 升级。
2. 统一发布反馈和后台发布队列，修复退出时的草稿保护。下一稳定版在最终安装包完成签名、安全扫描、源码对应校验和原生验收后放行。
3. 基于后台订阅流程补齐自动刷新和日期筛选，再处理更新界面与卸载清理。

## 全局审查验证范围

2026-11-27 的全局审查基线为 `5b94eab`。`bun run check` 及应用依赖扫描通过，包含 99 项 Bun 测试、Go 静态分析和 Go 测试。NSIS 构建与包内校验、MSIX 打包、源码归档校验及 vendor 离线编译均通过，另完成了桌面快捷方式测试和 17 张离屏界面快照。额外的 Kubo 二进制扫描和两条备份失败记录分别见 BUG-006、BUG-009。原生安装、升级和卸载实机验收、公网发布、真实第三方账号和真实 macOS 备份往返未执行。
