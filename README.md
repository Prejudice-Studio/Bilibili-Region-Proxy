# 🌏 Bilibili Region Proxy - Go

面向 **Bilibili 主站** 的 Go HTTP 代理。普通 API 按原路径转发；东南亚番剧搜索可使用主站 `/zh-Hans/bangumi` 网页搜索，并转换为客户端可识别的番剧搜索 JSON。

> 📍 搜索内容区域由代理服务器的出口 IP 决定。

## 🧭 目录

- [🚀 快速运行](#quick-start)
- [🔎 番剧搜索](#bangumi-search)
- [⚙️ 配置](#configuration)
- [🧪 测试与构建](#testing-build)
- [📚 文档](#documentation)
- [🔐 安全与隐私](#security)

<a id="quick-start"></a>
## 🚀 快速运行

需要 Go 1.23 或更新版本。

```powershell
Copy-Item .\config.example.json .\Bilibili-Region-Proxy.json
go run . --listen-address 127.0.0.1 --port 10001 --config .\Bilibili-Region-Proxy.json
```

本地使用推荐监听 `127.0.0.1`。需要让其他设备访问时，可改用 `0.0.0.0` 并配置防火墙规则。

```text
GET http://127.0.0.1:10001/healthz
```

健康检查成功时返回 `{"status":"ok"}`。可用启动参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--listen-address` | `0.0.0.0` | HTTP 监听地址 |
| `--port` | `10001` | HTTP 监听端口 |
| `--config` | 自动查找 | JSON 配置文件路径 |

配置按可执行文件目录、当前工作目录的顺序查找 `Bilibili-Region-Proxy.json`；未找到时兼容使用 `config.json`。也可用 `--config` 指定路径。保存配置后，后续请求会自动读取。

<a id="bangumi-search"></a>
## 🔎 番剧搜索

客户端继续调用普通主站搜索接口，并携带 `resolver_mode`：

```text
GET /x/web-interface/wbi/search/type?search_type=media_bangumi&keyword=葬送的芙莉莲&page=1&resolver_mode=INTL
```

`resolver_mode` 可放在 Query，也可放在 `resolver_mode`、`x-resolver-mode` 或 `x-resolver_mode` 请求头中；请求头优先。

| 模式 | 处理方式 |
| --- | --- |
| `CN`、`HK`、`MO`、`TW` | 保持普通 HTTP 搜索转发 |
| `INTL`、`SEA`、`TH`、`SG`、`MY`、`ID`、`PH`、`VN` 等 | 使用 `/zh-Hans/bangumi` 并转换搜索结果 |
| 其他非空值 | 兼容为国际网页番剧搜索 |
| 未提供 | 普通 HTTP 搜索；可由配置强制切换 |

仅匹配路径上的 `search_type=media_bangumi` 请求进入番剧适配。处理链路：

```text
客户端搜索请求 → 区域路由 → Bilibili 网页搜索 → 提取番剧结果 → 常规搜索 JSON
```

新搜索会保留客户端传入的其他参数、Cookie、Authorization、账号及设备请求头。代理移除内部 resolver 标记、逐跳头和转发链路头，并按搜索请求设置关键词、分类和页码。

<a id="configuration"></a>
## ⚙️ 配置

从 [`config.example.json`](config.example.json) 复制一份为 `Bilibili-Region-Proxy.json` 后编辑。配置使用 UTF-8 JSON。

```json
{
  "force_intl_bangumi_search": false,
  "injection": {
    "enabled": false,
    "position": "head",
    "items": []
  }
}
```

| 选项 | 用途 |
| --- | --- |
| `force_intl_bangumi_search` | `true` 时，匹配的番剧搜索不需要 `resolver_mode`，直接使用国际网页搜索；普通视频搜索和其他 API 不变。 |
| `injection.enabled` | 开启搜索结果注入。 |
| `injection.position` | `head` 放在真实结果前；`tail` 放在真实结果后。 |
| `injection.items` | 注入条目列表；不计入上游 `numResults`。 |

注入条目最小示例：

```json
{
  "title": "示例番剧",
  "link": "https://example.com/watch/123",
  "image": "https://example.com/poster.webp",
  "desc": "简介",
  "media_score": { "score": 9.2, "user_count": 12000 }
}
```

所有字段的类型、默认值、别名优先级、链接规则和长度限制见[配置说明](CONFIGURATION.md)。

<a id="testing-build"></a>
## 🧪 测试与构建

```powershell
gofmt -w .\main.go .\internal\proxy\*.go
go test ./...
go vet ./...
go build ./...
```

构建 Linux amd64：

```powershell
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -trimpath -o .\output\bilibili-region-proxy-linux-amd64 .
```

<a id="documentation"></a>
## 📚 文档

| 文档 | 内容 |
| --- | --- |
| [配置说明](CONFIGURATION.md) | 注入字段、数据类型、默认值和兼容规则 |
| [完整配置示例](config.example.json) | 可直接复制的配置及所有注入字段 |
| [配置 Schema](config.schema.json) | JSON 编辑器的字段补全与校验 |
| [贡献指南](CONTRIBUTING.md) | 开发、提交和 Pull Request 流程 |
| [安全说明](SECURITY.md) | 凭据保护与安全问题报告 |

<a id="security"></a>
## 🔐 安全与隐私

- 搜索认证信息会转发给 Bilibili；请限制代理访问范围，避免开放为不受控的公共转发服务。
- 不要提交真实 Cookie、访问令牌、个人配置或抓包文件。
- 日志不记录 Query、Cookie、Authorization 或设备凭据。

## 📄 许可

MIT License，见 [`LICENSE`](LICENSE)。
