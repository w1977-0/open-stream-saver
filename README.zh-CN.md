# Media Archiver 中文说明

[English README](README.md) · [常见问题](docs/FAQ.md) · [贡献指南](CONTRIBUTING.md) · [本地主机安装](native-host/README.md)

Media Archiver 是一个**本地优先、授权优先**的开源项目。Chrome 插件以只读方式发现当前标签页中公开加载的 `.mp4`、`.m3u8`、`.mpd` URL；Go CLI 只会在你明确确认拥有保存权利后，保存单个公开直链、已结束且未加密的 HLS 媒体，或静态且未加密的 DASH 媒体。

> 本项目不会读取、捕获或转发 Cookie、请求头、Token、账号凭据、页面内容、加密密钥或 DRM 数据；不会绕过登录、订阅、付费墙、地区限制、代理限制、加密或 DRM。

## 与 Media Saver 的关系

**Media Archiver 是正式的跨平台核心**：以 Go CLI 和 Chrome Manifest V3 插件提供透明、可审阅的本地媒体归档能力。[Media Saver](https://github.com/w1977-0/media-saver) 则是更早期的本地 GUI 研究工具，以 Python、Flask 和 Streamlit 提供较简洁的操作流程。两者不是重复实现：本仓库是持续维护的命令行与浏览器扩展基础。

## 一句话安装

如本机已安装 Go 1.25+，可执行：

```bash
go install github.com/w1977-0/media-archiver/cli/cmd/open-stream-saver@v0.3.1
```

> **Go module 结构。** Media Archiver 以仓库根 module `github.com/w1977-0/media-archiver` 发布，CLI 源码有意保留在 `cli/` 目录。安装时应使用 CLI 的 package path，并固定到根版本 tag，例如 `@v0.3.1`；不要在 `go install` 的版本位置使用 `cli/vX.Y.Z` 形式。

也可以从 [Releases](../../releases) 下载相应系统的压缩包。HLS / DASH 的本地合并需要自行安装 **FFmpeg** 并放入 `PATH`；直链下载不需要 FFmpeg。发布包不会捆绑 FFmpeg，便于用户从操作系统软件源或可信上游自行获取。请按[安装验证清单](docs/INSTALLATION_VERIFICATION.md)逐项核对新 module 安装、源码构建、FFmpeg 和已解压扩展的预期结果。

## 使用方式

### 命令行保存公开直链

```bash
open-stream-saver download \
  --url 'https://example.org/your-authorized-video.mp4' \
  --acknowledge-rights \
  --output ./downloads/my-video.mp4
```

`--acknowledge-rights` 是刻意设计的必填参数。只应在你拥有保存权利时使用，例如自己的上传内容、开放许可作品、已获明确授权的资料或网站明确允许保存的媒体。工具拒绝重定向、私有网络地址和覆盖已有文件；完成后会输出本地文件的 SHA-256 摘要。

### 选择公开 HLS 主清单的画质

```bash
open-stream-saver inspect-hls --url 'https://example.org/your-authorized-master.m3u8'
open-stream-saver download \
  --url 'https://example.org/your-authorized-master.m3u8' \
  --variant 1 \
  --acknowledge-rights \
  --output ./downloads/my-video.mp4
```

`--variant -1` 为默认值，表示选择已声明带宽最高的、包含音视频的安全变体。带独立音轨或字幕组的主清单不会被猜测性处理，而是会明确拒绝。

### Chrome 插件与本地主机

1. 在 Chrome 打开 `chrome://extensions`，开启开发者模式，选择“加载已解压的扩展程序”，并选中 `extension/` 目录。
2. 打开插件弹窗，主动点击 **Enable discovery**。Chrome 会请求可选的 HTTP/HTTPS 站点访问权限。
3. 在你拥有保存权利的页面中，复核当前标签页发现的公开链接；你可以复制 CLI 命令，或勾选“我有权保存”后点击 **Save locally**。
4. “一键本地保存”需要安装并注册可选的 `open-stream-saver-host` 本地主机。请严格按 [Native Messaging 主机说明](native-host/README.md) 把你的**准确扩展 ID**写入允许来源列表。

本地主机只接收固定的最小请求：用户复核过的公开 URL、并发数、可选 HLS 变体编号及授权确认。它拒绝未知字段和任意输出路径，仅向你的 `Downloads` 目录写入安全命名的新文件。

## 支持范围

| 输入或条件 | 状态 | 说明 |
| --- | --- | --- |
| 公开 HTTP(S) 直链文件 | 支持 | 有条件使用 HTTP Range，受控并发、临时文件清理和 SHA-256 输出。 |
| 已结束、未加密 HLS 媒体清单 | 支持 | 受控 Worker Pool 下载分片，再调用本机 FFmpeg 合并。 |
| 含音视频的公开 HLS 主清单 | 支持 | 可先 `inspect-hls` 查看，再用 `--variant` 选择一个变体。 |
| 静态、未加密的 DASH `SegmentTemplate` | 支持 | 分别下载公开视频和音频表示，再由本机 FFmpeg 无重编码混流。 |
| DRM / `ContentProtection`、AES-128、SAMPLE-AES | 拒绝 | 不获取密钥、不解密、不包含 DRM 规避能力。 |
| 直播、重定向、Cookie/Token 鉴权、登录/付费墙/地区限制 | 拒绝 | 本项目不携带身份材料，也不实现访问控制绕过。 |
| HLS 字节范围/初始化映射/低延迟分片；DASH `SegmentTimeline` / `SegmentBase` | 拒绝 | 为避免对复杂或访问敏感的传输方式进行猜测性处理。 |

## 隐私与权限

插件把每个标签页最多 40 条发现记录保存到浏览器**会话存储**，标签关闭后自动删除。它不向远端服务发送数据，不修改流量，也不注入 Hook 改写页面的 `fetch` 或 `XMLHttpRequest`。本地主机使用 Chrome Native Messaging 的严格来源白名单；只有配置中指定的扩展 ID 可以发起请求。

本项目用于个人归档你有权保存的资料，并非法律意见。请阅读完整的 [英文 FAQ](docs/FAQ.md)、[架构说明](ARCHITECTURE.md)、[安全政策](SECURITY.md) 和 [贡献指南](CONTRIBUTING.md)。请勿在 Issue、讨论或 PR 中公开私有 URL、账号信息、凭据、Token、密钥，或你无权分享的内容。
