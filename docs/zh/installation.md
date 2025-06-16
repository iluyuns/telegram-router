# 安装指南

## 系统要求

- Go 1.16 或更高版本
- Telegram Bot Token（从 [@BotFather](https://t.me/BotFather) 获取）

## 快速开始

1. 安装包：

```bash
go get github.com/iluyuns/telegram-router
```

2. 创建 `main.go`：

```go
package main

import (
    "log"
    "github.com/iluyuns/telegram-router"
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

3. 运行机器人：

```bash
go run main.go
```

## 基本配置

### 环境变量（可选）

```bash
export TELEGRAM_BOT_TOKEN="your-bot-token"
```

## 下一步

- 查看[基本使用](basic-usage.md)了解如何注册不同类型的消息处理器
- 学习[中间件](middleware.md)的使用方法 