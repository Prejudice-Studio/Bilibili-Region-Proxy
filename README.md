# Bilibili Region Proxy - Go

一个面向 Bilibili 主站 API 的 HTTP 代理。它将客户端请求转发到 Bilibili，并可把番剧搜索请求适配到主站的国际化网页搜索，再转换为常规番剧搜索 JSON。

## 功能

- 转发 Bilibili 主站 HTTP API 请求。
- 支持 `resolver_mode` 查询参数及请求头，供客户端选择区域搜索路由。
- 可选强制使用 `/zh-Hans/bangumi` 网页番剧搜索。
- 将网页搜索页面中的番剧结果转换为 `media_bangumi` 响应。
- 支持搜索结果注入，并兼容原有 JSON 注入配置。
- 配置文件在修改后自动重新加载。

搜索结果区域由解析服务器的出口 IP 决定。

## 环境

- Go 1.23 或更高版本
- Windows、Linux、macOS 均可构建；线上部署建议使用 Linux amd64

## 快速开始

复制示例配置并启动服务：

```powershell
Copy-Item .\config.example.json .\Bilibili-Region-Proxy.json
go run . --listen-address 0.0.0.0 --port 10001 --config .\Bilibili-Region-Proxy.json
```

默认监听 `0.0.0.0:10001`。`GET /healthz` 返回服务状态。启动参数：

```text
--listen-address 监听地址
--port           监听端口
--config         JSON 配置文件路径
```

构建 Linux amd64：

```powershell
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -trimpath -o .\output\bilibili-region-proxy-linux-amd64 .
```

## 番剧搜索

客户端沿用普通主站番剧搜索请求，并通过 `resolver_mode` 选择国际网页搜索：

```text
GET /x/web-interface/wbi/search/type?search_type=media_bangumi&keyword=我的英雄学院&page=1&resolver_mode=INTL
```

`resolver_mode` 可以放在查询参数，或放在 `resolver_mode`、`x-resolver-mode`、`x-resolver_mode` 请求头中；请求头优先。

| 模式 | 处理方式 |
| --- | --- |
| `CN`、`HK`、`MO`、`TW` | 按原路径转发普通搜索 |
| `INTL`、`SEA`、`TH`、`SG`、`MY`、`ID`、`PH`、`VN` 等 | 番剧搜索使用 `/zh-Hans/bangumi` |
| 自定义非空值 | 兼容为国际网页番剧搜索 |

只有 `search_type=media_bangumi` 的搜索请求会进入网页适配。代理从搜索页的页面状态提取番剧结果，并转换成常规 `media_bangumi` 响应。网页请求会保留客户端传入的查询参数和请求头，包括 Cookie、Authorization、账号参数和设备参数；代理会移除内部 resolver 控制项、逐跳头和转发链路头，并使用当前请求的关键词、分类与页码。

## 配置

配置文件使用 JSON。程序默认优先读取可执行文件目录或当前目录中的 `Bilibili-Region-Proxy.json`；未找到时回退到 `config.json`。也可以通过 `--config` 指定路径。配置变化会在后续请求中自动生效。

### 强制国际网页番剧搜索

配置顶层的 `force_intl_bangumi_search` 默认为 `false`。设为 `true` 后，所有匹配路径上的 `media_bangumi` 请求都会使用 `/zh-Hans/bangumi`，不再要求请求携带 `resolver_mode`。普通视频搜索和其他 API 不受影响。

### 搜索结果注入

`injection` 配置兼容现有格式，可选择把条目追加到搜索结果头部或尾部。注入条目不计入上游 `numResults`。

完整可复制模板见 [`config.example.json`](config.example.json)。所有注入字段的用途、JSON 类型、默认值、别名优先级和长度限制见 [`CONFIGURATION.md`](CONFIGURATION.md)；[`config.schema.json`](config.schema.json) 为编辑器提供字段类型和补全说明。

## 请求与凭据

搜索适配会把客户端请求中的认证信息转发给 Bilibili。请只通过可信网络提供代理服务，避免将服务开放为不受控的公共转发入口。程序日志不会记录查询字符串或认证头。

## 开发

```powershell
gofmt -w .\main.go .\internal\proxy\*.go
go test ./...
go vet ./...
go build ./...
```

分支、代码风格、提交信息及变更检查要求见 [`CONTRIBUTING.md`](CONTRIBUTING.md) 和 [`AGENTS.md`](AGENTS.md)。GitHub Actions 会在推送和 Pull Request 中执行格式、测试、静态检查与构建。

## 许可

本项目使用 MIT License，见 [`LICENSE`](LICENSE)。
