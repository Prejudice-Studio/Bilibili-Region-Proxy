# 🌏 Bilibili Region Proxy 哔哩哔哩番剧解析服务器

一个面向 [哔哩哔哩主站](https://www.bilibili.com) 的番剧解析服务器。

## 🚀 快速运行

编译二进制文件：

```powershell
go build -o bilibili-region-proxy .
chmod +x bilibili-region-proxy
```

运行二进制文件：

```powershell
./bilibili-region-proxy --listen-address 127.0.0.1 --port 10001 --config ./Bilibili-Region-Proxy.json
```

配置项:
| 参数 | 说明 | 默认值 | 注释 |
| --- | --- | --- | --- |
| --listen-address | 监听地址 | 127.0.0.1 | 如果需要对外请自行配置为0.0.0.0或其他方式 |
| --port | 监听端口 | 10001 | - |
| --config | 配置文件路径 | ./Bilibili-Region-Proxy.json | 参数指定优先，默认查找二进制运行目录下的``Bilibili-Region-Proxy.json`` 与 ``config.json``  |


健康检查：

```text
GET http://127.0.0.1:10001/healthz
```

成功返回：

```json
{"status":"ok"}
```

## ⚙️ 配置文件说明

配置文件示例：

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

## 📚 文档

| 文档 | 内容 |
| --- | --- |
| [配置说明](CONFIGURATION.md) | 注入字段、数据类型、默认值和兼容规则 |
| [完整配置示例](config.example.json) | 可直接复制的配置及所有注入字段 |
| [配置 Schema](config.schema.json) | JSON 编辑器的字段补全与校验 |
| [贡献指南](CONTRIBUTING.md) | 开发、提交和 Pull Request 流程 |
| [安全说明](SECURITY.md) | 凭据保护与安全问题报告 |

## 📄 许可

MIT License，见 [`LICENSE`](LICENSE)。