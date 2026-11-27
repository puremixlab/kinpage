# KinPage Privacy Policy / 邻页（KinPage）隐私政策

Last Updated / 最后更新日期: 2026-11-26

---

## English

### 1. Overview (Local-First, No Telemetry)

KinPage, published by PureMixLab, is a local-first desktop application for writing, feed reading, and decentralized publishing. KinPage does **not** require an account, does **not** collect, store, or sell personal information, and does **not** send analytics, telemetry, crash reports, or tracking data to PureMixLab.

### 2. Local Data Storage

Your sites, articles, private drafts, subscriptions, reading state, preferences, and IPNS private keys are stored locally on your device. Private drafts remain local-only and are never transmitted over the network unless you explicitly publish them.

### 3. Network Activity & Decentralized Publishing

KinPage connects to the network only to perform actions you initiate:

- **IPFS / IPNS Publishing**: When you run the bundled local Kubo node or publish a site, your published site files, attachments, and IPNS records are shared over the public peer-to-peer IPFS network, where published content is publicly accessible by design.
- **Feed, ENS & Web Reading**: When you follow or refresh RSS, Atom, JSON Feed, website homepages, IPNS, or ENS (`.eth`) sites, or open an original webpage view, KinPage connects directly to those third-party websites, your configured public IPFS gateways, or ENS resolvers.
- **Optional Third-Party Integrations**: Only when explicitly configured by you (such as Cloudflare Pages, SSH/rsync, Pinnable, Filebase, `dWebServices.xyz`, Plausible, Telegram, X, or a local/remote AI endpoint) does KinPage send your requested content or prompts directly to those services, which are governed by their own privacy policies.

### 4. Backups & Key Security

Exported `.planet` backup archives contain unencrypted IPNS private keys. You are solely responsible for keeping your backups private and secure.

### 5. Changes & Contact

Updates to this policy will be published in this repository. For privacy questions, please open an issue at:

https://github.com/puremixlab/kinpage/issues

---

## 简体中文

### 1. 总体原则（本地优先，无遥测）

邻页（KinPage，由 PureMixLab 开发）是一款本地优先的桌面写作、订阅阅读与去中心化发布工具。本应用**无需注册账号**，**不收集、不存储、不出售**任何个人身份信息，也不向 PureMixLab 发送任何使用统计、崩溃日志或广告追踪数据。

### 2. 本地数据存储

您的站点内容、文章、本地私有草稿、订阅列表、阅读记录、界面偏好以及本地 IPNS 私钥均仅保存在您设备的本地数据目录中。私有草稿默认仅保留在本地，除非您主动点击发布，否则不会上传至网络。

### 3. 网络连接与去中心化发布

本应用仅在您主动使用相关功能时发起网络连接：

- **IPFS / IPNS 发布**：当您启用内置 Kubo 节点或点击「发布」时，您所发布站点的公开内容、媒体附件及 IPNS 记录将同步至公开的点对点 IPFS 网络；公开网络上的内容可被其他节点访问与缓存。
- **订阅与网页阅读**：当您订阅或刷新 RSS、Atom、JSON Feed、网站首页、IPNS 或 ENS（`.eth`）站点，或切换查看原网页时，应用会直接向目标网站、您选择的公共 IPFS 网关或 ENS 解析服务发起请求。
- **可选第三方集成**：仅当您手动配置并启用第三方服务（如 Cloudflare Pages、SSH/rsync、Pinnable、Filebase、`dWebServices.xyz`、Plausible、Telegram、X 或本地/远程 AI 接口）时，应用才会将您指定的内容或提问直接发送至对应服务，相关数据处理受该第三方自身的隐私政策约束。

### 4. 备份与私钥安全

导出的 `.planet` 备份文件包含未加密的 IPNS 私钥，请您妥善保管，切勿上传至公开网盘或代码仓库。

### 5. 隐私政策更新与联系方式

本隐私政策如随功能更新而调整，将在本仓库中同步更新。如有隐私相关问题，请通过 GitHub Issues 联系我们：

https://github.com/puremixlab/kinpage/issues
