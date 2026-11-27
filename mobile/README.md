# 邻页移动版

邻页在 iOS 和 Android 上提供离线写作、订阅阅读和桌面协同发布。应用随系统切换浅色与深色外观，支持简体中文和英文。

- 在写作台创建 Markdown 草稿，随输入保存，切换预览或从系统分享菜单导出正文。
- 订阅 RSS、Atom、JSON Feed 及 Planet 站点。可输入 HTTPS 网站、`.eth` 域名或 `planet://`、`ipns://`、`ipfs://` 地址。正文缓存在本机，已读状态和收藏在刷新后仍保留。订阅里的图片和外部媒体可从原文查看。
- 连接桌面版邻页后，选择已启用 IPNS 的站点，将文章发布到 IPFS。发布记录提供该文章的链接。

## 运行和检查

使用 Node.js 22.18 或更高版本，从本目录运行以下命令。

```powershell
npm ci
npm run check
npm run bundle
```

`npm run bundle` 将 iOS 和 Android 的生产 JS bundle 输出到 `dist/`。原生工程由 Expo 根据 `app.json` 生成。

| 平台 | 运行命令 | 原生工具 |
| --- | --- | --- |
| Android | `npm run android` | JDK 17、Android SDK 37.0、模拟器或 USB 调试设备 |
| iOS | `npm run ios` | macOS、Xcode 26.4 或更新版本、CocoaPods、iOS 模拟器 |

运行命令会生成平台工程并安装调试构建。Android 7 及以上和 iOS 16.4 及以上均受 Expo SDK 58 支持。设备分发需要各平台的签名配置。[Expo SDK 58](https://docs.expo.dev/versions/v58.0.0/)，[Expo 原生运行与构建](https://docs.expo.dev/workflow/overview/)。

在 Windows 的深目录中构建 Android 时，Ninja 可能遇到路径长度限制。可将同一版本源码检出到较短的实际磁盘路径，例如 `D:\kp`，再从其中的 `mobile` 目录运行构建。

## 连接桌面后发布

1. 从支持移动发布的邻页桌面源码运行 Windows 版。在本地 API 设置中启动服务，开启 Basic 认证并设置口令。默认监听 `127.0.0.1:8086`。
2. 将手机可访问的可信 HTTPS 地址配置为转发 `/v0/mobile/*` 到桌面本地 API，保留请求路径和 `Authorization` 头。桌面端保持运行。下例中的域名、证书和网络访问由使用者配置。
3. 在手机的「设置 → 桌面连接」输入 HTTPS 地址、用户名和口令，点「验证并保存」。写好文章后，在写作台选「发布到 IPFS」，选择站点并确认。

```caddy
kinpage.example.org {
    handle /v0/mobile/* {
        reverse_proxy 127.0.0.1:8086
    }
}
```

手机仅在确认发布后提交内容。发送前会记录草稿和所选站点。若桌面保存成功后连接中断，重试会继续使用同一篇文章和站点。若桌面修改了对应文章，手机会显示冲突。请先在桌面核对内容。

草稿、订阅和阅读状态使用本机 SQLite。桌面口令通过 iOS Keychain 或 Android Keystore 保护。草稿尚未提交时可离线编辑，发布需要手机连到桌面，且桌面的 IPFS 节点可以发布。
