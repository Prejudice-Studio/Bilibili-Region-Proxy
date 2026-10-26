# 🤝 贡献指南

## 🧰 开发环境

- Go 1.23 或更高版本
- Git

## 🔄 修改流程

1. 从 `main` 创建分支，推荐使用 `feat/功能名`、`fix/问题名` 或 `docs/主题`。
2. 保持变更范围清晰；路由、解析、配置和注入逻辑的改动应附带对应测试。
3. 修改 Go 文件后运行 `gofmt`。
4. 提交前运行：

   ```sh
   go test ./...
   go vet ./...
   go build ./...
   ```

5. 更新中文 README 或配置示例中与改动相关的说明。

## ✍️ 提交信息

采用 Conventional Commits：

```text
<type>(<scope>): <简短说明>
```

常用类型：`feat`、`fix`、`refactor`、`docs`、`test`、`build`、`chore`。

## 🔍 Pull Request

- 说明改动目的、主要行为变化和验证结果。
- 配置项需说明默认值、适用请求和兼容行为。注入字段变更时，同步维护 `config.example.json`、`CONFIGURATION.md` 与 `config.schema.json`。
- 不要附带真实账号凭据、Cookie、访问令牌、本地配置、流量抓取文件或构建产物。
- 涉及请求转发时，说明哪些客户端参数会被保留、修改或移除。
