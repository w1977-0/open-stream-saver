# Open Stream Saver 中文说明

[English README](README.md) · [常见问题](docs/FAQ.md) · [贡献指南](CONTRIBUTING.md)

Open Stream Saver 是一个本地优先的学习项目：Chrome 插件以只读方式发现当前标签页中公开加载的 `.m3u8` / `.mp4` URL；Go CLI 在你明确确认拥有保存权利后，保存单个公开直链或已结束、未加密的 HLS 媒体清单。

> 本项目不处理 DRM、Cookies、账号登录、付费墙、订阅、地区限制、代理绕过、加密 HLS、直播流、主播放列表或批量下载。

## 使用方式

1. 在 Chrome 打开 `chrome://extensions`，开启开发者模式，选择“加载已解压的扩展程序”，并选中 `extension/` 目录。
2. 打开插件弹窗，主动点击 **Enable discovery** 并阅读 Chrome 的可选站点访问权限提示。
3. 在你拥有保存权利的页面中打开插件。它会显示当前标签页已观察到的公开 `.m3u8` / `.mp4` 请求，并可复制 CLI 命令。
4. 复核命令中的 URL、来源和你的授权情况，再运行：

```bash
open-stream-saver download \
  --url 'https://example.org/your-authorized-video.mp4' \
  --acknowledge-rights \
  --output ./downloads/my-video.mp4
```

对 HLS 媒体清单，需在本机安装 FFmpeg。直链下载不需要 FFmpeg。

## 隐私与权限

插件把每个标签页最多 40 条发现记录保存在浏览器会话存储中，标签关闭后自动删除。插件不发送数据到远端，也不读取 Cookies、请求头、页面内容、账号信息或媒体密钥。CLI 拒绝私有网络、嵌入凭据、重定向、主播放列表、直播和加密片段。

请阅读完整的英文 [FAQ](docs/FAQ.md) 和 [安全政策](SECURITY.md)。如果你希望贡献翻译、测试、可访问性改进或更清晰的错误信息，欢迎阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。
