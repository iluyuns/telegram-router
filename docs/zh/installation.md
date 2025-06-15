# 安装指南

本文档提供了安装和配置 Telegram Router 的详细说明。

## 系统要求

- Go 1.16 或更高版本
- 有效的 Telegram Bot Token（从 [@BotFather](https://t.me/BotFather) 获取）

## 安装

使用 Go 模块安装：

```bash
# 创建新项目
mkdir my-bot
cd my-bot
go mod init my-bot

# 安装 Telegram Router
go get github.com/yourusername/telegram-router
```

## 基本配置

1. 创建 `main.go` 文件：

```go
package main

import (
    "log"
    "github.com/yourusername/telegram-router"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
    // 创建机器人实例
    bot, err := tgbotapi.NewBotAPI("YOUR_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    // 创建路由器
    r := router.New(bot)

    // 注册命令处理器
    r.Command("start", func(c *router.Context) {
        c.Reply("欢迎使用机器人！")
    })

    // 启动机器人
    r.Start()
}
```

2. 运行机器人：

```bash
go run main.go
```

## 配置选项

### 长轮询模式

```go
// 使用默认配置
r.Start()

// 使用自定义配置
config := &router.PollingConfig{
    Timeout: 60,           // 超时时间（秒）
    DropPendingUpdates: true, // 启动时丢弃待处理更新
}
r.StartWithConfig(config)
```

### Webhook 模式

```go
// 配置 webhook
webhook := &router.WebhookConfig{
    ListenAddr: ":8443",           // 监听地址
    CertFile:   "cert.pem",        // SSL 证书文件
    KeyFile:    "key.pem",         // SSL 密钥文件
    WebhookURL: "https://your-domain.com/bot", // Webhook URL
}

// 启动 webhook
if err := r.StartWebhook(webhook); err != nil {
    log.Fatal(err)
}
```

### 中间件配置

```go
// 添加全局中间件
r.Use(Logger())
r.Use(Auth([]int64{123456789}))

// 添加路由特定中间件
r.Group(func(r *router.Router) {
    r.Use(Auth([]int64{123456789}))
    r.Command("admin", func(c *router.Context) {
        c.Reply("管理员命令")
    })
})
```

## 环境变量

可以通过环境变量配置机器人：

```bash
# 设置机器人令牌
export TELEGRAM_BOT_TOKEN="your-bot-token"

# 设置 webhook URL
export TELEGRAM_WEBHOOK_URL="https://your-domain.com/bot"

# 设置监听地址
export TELEGRAM_LISTEN_ADDR=":8443"
```

在代码中使用环境变量：

```go
token := os.Getenv("TELEGRAM_BOT_TOKEN")
if token == "" {
    log.Fatal("未设置 TELEGRAM_BOT_TOKEN 环境变量")
}

bot, err := tgbotapi.NewBotAPI(token)
if err != nil {
    log.Fatal(err)
}
```

## 依赖管理

### 使用 Go Modules

```go
// go.mod
module my-bot

go 1.16

require (
    github.com/yourusername/telegram-router v1.0.0
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.0.0
)
```

### 使用 Dep

```bash
# 安装 dep
go get -u github.com/golang/dep/cmd/dep

# 初始化 dep
dep init

# 添加依赖
dep ensure -add github.com/yourusername/telegram-router
```

## 开发环境设置

1. 安装开发工具：

```bash
# 安装 golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 安装 delve 调试器
go install github.com/go-delve/delve/cmd/dlv@latest
```

2. 配置编辑器：

- VS Code：安装 Go 扩展
- GoLand：使用内置的 Go 工具
- Vim：安装 vim-go 插件

3. 运行测试：

```bash
# 运行所有测试
go test ./...

# 运行特定测试
go test -v ./... -run TestRouter

# 运行基准测试
go test -bench=. ./...
```

## 生产环境部署

### 使用 Docker

1. 创建 Dockerfile：

```dockerfile
FROM golang:1.16-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o bot

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/bot .
COPY cert.pem key.pem ./
CMD ["./bot"]
```

2. 构建和运行：

```bash
# 构建镜像
docker build -t my-bot .

# 运行容器
docker run -d \
    -p 8443:8443 \
    -v $(pwd)/cert.pem:/app/cert.pem \
    -v $(pwd)/key.pem:/app/key.pem \
    -e TELEGRAM_BOT_TOKEN=your-token \
    my-bot
```

### 使用 Systemd

1. 创建服务文件 `/etc/systemd/system/telegram-bot.service`：

```ini
[Unit]
Description=Telegram Bot Service
After=network.target

[Service]
Type=simple
User=bot
WorkingDirectory=/opt/bot
ExecStart=/opt/bot/bot
Environment=TELEGRAM_BOT_TOKEN=your-token
Restart=always

[Install]
WantedBy=multi-user.target
```

2. 启用和启动服务：

```bash
sudo systemctl enable telegram-bot
sudo systemctl start telegram-bot
```

## 故障排除

### 常见问题

1. 机器人无法启动
   - 检查 Bot Token 是否正确
   - 确保网络连接正常
   - 查看日志输出

2. Webhook 模式问题
   - 确保证书有效且未过期
   - 检查防火墙设置
   - 验证域名解析正确

3. 中间件问题
   - 检查中间件顺序
   - 确保中间件正确返回
   - 查看错误日志

### 日志记录

```go
// 配置日志
log.SetFlags(log.LstdFlags | log.Lshortfile)
log.SetOutput(os.Stdout)

// 在代码中使用日志
log.Printf("机器人启动：%s", bot.Self.UserName)
log.Printf("处理消息：%s", c.Message.Text)
```

### 调试

```go
// 启用调试模式
bot.Debug = true

// 使用 delve 调试
dlv debug main.go

// 添加调试日志
if bot.Debug {
    log.Printf("收到更新：%+v", update)
}
```

## 下一步

- 查看[基本使用](basic-usage.md)了解如何注册不同类型的消息处理器
- 学习[中间件](middleware.md)的使用方法
- 探索[高级特性](advanced-features.md)实现更复杂的功能 