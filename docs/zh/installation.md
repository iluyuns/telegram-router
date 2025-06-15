# 安装说明

## 系统要求

- Go 1.16 或更高版本
- Telegram Bot Token（从 [@BotFather](https://t.me/BotFather) 获取）

## 安装步骤

### 1. 创建项目

首先创建一个新的 Go 项目：

```bash
mkdir my-telegram-bot
cd my-telegram-bot
go mod init my-telegram-bot
```

### 2. 安装依赖

安装 Telegram Router 和必要的依赖：

```bash
go get github.com/yourusername/telegram-router
go get github.com/go-telegram-bot-api/telegram-bot-api/v5
```

### 3. 创建机器人

1. 在 Telegram 中打开 [@BotFather](https://t.me/BotFather)
2. 发送 `/newbot` 命令
3. 按照提示设置机器人名称和用户名
4. 保存 BotFather 提供的 API Token

### 4. 基本配置

创建一个 `main.go` 文件：

```go
package main

import (
    "log"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/yourusername/telegram-router"
)

func main() {
    // 初始化机器人
    bot, err := tgbotapi.NewBotAPI("你的机器人令牌")
    if err != nil {
        log.Fatal(err)
    }

    // 设置调试模式（可选）
    bot.Debug = true

    // 创建路由器
    router := telegramrouter.NewTelegramRouter(bot)

    // 注册基本处理器
    router.Command("start", func(c *telegramrouter.Context) {
        c.Reply("欢迎使用机器人！")
    })

    // 启动机器人
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates := bot.GetUpdatesChan(u)

    // 处理更新
    for update := range updates {
        router.HandleUpdate(&update)
    }
}
```

### 5. 运行机器人

```bash
go run main.go
```

## 配置说明

### 环境变量

建议使用环境变量来存储敏感信息：

```go
// 从环境变量获取令牌
token := os.Getenv("TELEGRAM_BOT_TOKEN")
if token == "" {
    log.Fatal("未设置 TELEGRAM_BOT_TOKEN 环境变量")
}
```

### 代理设置（可选）

如果需要使用代理，可以这样配置：

```go
// 设置代理
proxyURL, err := url.Parse("http://proxy-server:port")
if err != nil {
    log.Fatal(err)
}

transport := &http.Transport{
    Proxy: http.ProxyURL(proxyURL),
}

client := &http.Client{Transport: transport}
bot.Client = client
```

### 错误处理

建议添加基本的错误处理：

```go
// 错误处理中间件
func ErrorHandler() telegramrouter.MiddlewareFunc {
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("发生错误: %v", err)
                c.Reply("抱歉，处理您的请求时发生错误")
            }
        }()
        next(c)
    }
}

// 使用错误处理中间件
router.Use(ErrorHandler())
```

## 常见问题

### 1. 无法连接到 Telegram 服务器

- 检查网络连接
- 确认 Bot Token 是否正确
- 如果在中国大陆使用，可能需要配置代理

### 2. 机器人没有响应

- 确认机器人是否在线
- 检查日志输出
- 验证处理器是否正确注册

### 3. 更新处理超时

- 调整 `Timeout` 参数
- 检查处理器是否阻塞
- 考虑使用 goroutine 处理耗时操作

## 下一步

- 查看[基本使用](basic-usage.md)了解如何注册不同类型的消息处理器
- 学习[中间件](middleware.md)的使用方法
- 探索[高级特性](advanced-features.md)实现更复杂的功能 