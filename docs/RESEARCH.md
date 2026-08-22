# 架构与合规研究记录

## Chrome Manifest V3

Chrome 官方 `webRequest` 文档表明，Manifest V3 仍可使用 `webRequest` 观察和分析网络请求；但大多数扩展不能使用 `webRequestBlocking` 来阻塞或修改请求。因此插件将只读地记录页面请求中的公开 `.m3u8` 和 `.mp4` 链接，不拦截、重写、附加认证信息或修改流量。[1]

Chrome 的权限文档指出，`webRequest` 需要相应的 API 权限和 host permissions，并建议在可行时使用可选权限以减少用户警告、给予使用者更明确的控制权。[2] 本项目会采用 `webRequest`、`storage` 与 `activeTab`，并将广泛的 HTTP/HTTPS host access 设为可选权限；只有在使用者主动开启“发现公开流媒体链接”时才请求。

## 下载器边界

CLI 只接受由使用者提供的公开 HTTP(S) 直链或不加密 HLS 清单。它不会接收浏览器 Cookies、Authorization 头、账号信息、DRM 密钥或播放页面脚本；不会尝试解密、登录、规避付费墙或地区限制。M3U8 工作流仅处理媒体分片清单，不支持 `EXT-X-KEY` 加密片段或 `EXT-X-STREAM-INF` 主播放列表的自动选择。

## 架构选择

| 层级 | 技术 | 职责 |
| --- | --- | --- |
| Chrome 插件 | Manifest V3、原生 JavaScript | 只读发现公开 `.m3u8`／`.mp4` 请求、去重、在本地扩展存储中保留最近记录、生成 CLI 命令。 |
| CLI | Go 标准库、Cobra、mpb、grafana/m3u8、errgroup | 下载用户已获授权的直链或未加密媒体清单，显示进度与安全失败提示。 |
| 媒体合并 | 用户本机 FFmpeg | 在没有加密且使用者已确认授权时，将完成的本地 HLS 文件合并／封装为 MP4。 |

## 参考资料

[1]: https://developer.chrome.com/docs/extensions/reference/api/webRequest "Chrome for Developers — webRequest"
[2]: https://developer.chrome.com/docs/extensions/develop/concepts/declare-permissions "Chrome for Developers — Declare permissions"

## Go 依赖选择

| 依赖 | 用途 | 选择理由 |
| --- | --- | --- |
| `github.com/spf13/cobra` | CLI 命令、参数与帮助文本。 | 提供子命令、POSIX 风格 flags、自动帮助与 shell 补全，适合将授权确认、并发数和输出路径暴露为清晰参数。[3] |
| `github.com/vbauerster/mpb/v8` | 下载进度显示。 | 为终端中的并发任务提供动态进度条、已完成字节数与耗时反馈。 |
| `github.com/Eyevinn/hls-m3u8` | HLS 清单解码。 | 该项目仍有近期维护，且明确定位为 Go 的 HLS m3u8 解析与生成库；避免依赖已归档的 `grafov/m3u8`。[4] |
| `golang.org/x/sync/errgroup` | Goroutine 协调与失败取消。 | 在并发下载分片时将错误传播与取消逻辑保持一致。 |

CLI 不引入会隐藏网络行为的“万能下载器”依赖；直链下载、Range 校验、重试和临时目录都用 Go 标准库实现，便于审查和维护。

[3]: https://github.com/spf13/cobra "Cobra — modern Go CLI"
[4]: https://github.com/Eyevinn/hls-m3u8 "Eyevinn — hls-m3u8"
