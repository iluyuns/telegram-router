# 示例代码

本文档提供了使用 Telegram Router 的各种示例。每个示例都展示了不同的功能和使用场景。

## 基础机器人

这是一个简单的机器人，可以响应命令和文本消息：

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

    r.Command("help", func(c *router.Context) {
        c.Reply("可用命令：\n/start - 开始使用\n/help - 显示帮助")
    })

    // 注册文本消息处理器
    r.Text(func(c *router.Context) {
        c.Reply("收到消息：" + c.Message.Text)
    })

    // 启动机器人
    r.Start()
}
```

## Webhook 机器人

这是一个使用 webhook 模式的机器人，展示了如何与 HTTP 框架集成：

```go
package main

import (
    "log"
    "github.com/gin-gonic/gin"
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

    // 注册处理器
    r.Command("start", func(c *router.Context) {
        c.Reply("欢迎使用 Webhook 机器人！")
    })

    // 创建 Gin 路由
    g := gin.Default()

    // 注册 webhook 处理器
    g.POST("/bot", func(c *gin.Context) {
        r.HandleWebhookRequest(c.Writer, c.Request)
    })

    // 配置 webhook
    webhook := &router.WebhookConfig{
        ListenAddr: ":8443",
        CertFile:   "cert.pem",
        KeyFile:    "key.pem",
        WebhookURL: "https://your-domain.com/bot",
    }

    // 启动 webhook
    if err := r.StartWebhook(webhook); err != nil {
        log.Fatal(err)
    }

    // 启动 HTTP 服务器
    g.Run(":8080")
}
```

## 中间件示例

这个示例展示了如何使用中间件进行日志记录和认证：

```go
package main

import (
    "log"
    "time"
    "github.com/yourusername/telegram-router"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// 日志中间件
func Logger() router.MiddlewareFunc {
    return func(next router.HandlerFunc) router.HandlerFunc {
        return func(c *router.Context) {
            start := time.Now()
            next(c)
            log.Printf("处理消息用时：%v", time.Since(start))
        }
    }
}

// 认证中间件
func Auth(allowedUsers []int64) router.MiddlewareFunc {
    return func(next router.HandlerFunc) router.HandlerFunc {
        return func(c *router.Context) {
            userID := c.Message.From.ID
            for _, id := range allowedUsers {
                if id == userID {
                    next(c)
                    return
                }
            }
            c.Reply("未授权访问")
        }
    }
}

func main() {
    bot, err := tgbotapi.NewBotAPI("YOUR_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    r := router.New(bot)

    // 使用中间件
    r.Use(Logger())
    r.Use(Auth([]int64{123456789}))

    // 注册处理器
    r.Command("start", func(c *router.Context) {
        c.Reply("欢迎使用带中间件的机器人！")
    })

    r.Start()
}
```

## 回调查询示例

这个示例展示了如何处理内联键盘和回调查询：

```go
package main

import (
    "log"
    "github.com/yourusername/telegram-router"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
    bot, err := tgbotapi.NewBotAPI("YOUR_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    r := router.New(bot)

    // 创建内联键盘
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("选项 1", "choice:1"),
            tgbotapi.NewInlineKeyboardButtonData("选项 2", "choice:2"),
        ),
    )

    // 处理 /menu 命令
    r.Command("menu", func(c *router.Context) {
        builder := c.Reply("请选择一个选项：")
        builder.WithInlineKeyboard(keyboard)
        builder.Send()
    })

    // 处理回调查询
    r.Callback("choice", func(c *router.Context) {
        choice := c.Args()[0]
        c.Answer("你选择了选项 " + choice)
        c.Edit("你选择了选项 " + choice)
    })

    r.Start()
}
```

## 文件处理示例

这个示例展示了如何处理不同类型的文件：

```go
package main

import (
    "log"
    "github.com/yourusername/telegram-router"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
    bot, err := tgbotapi.NewBotAPI("YOUR_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    r := router.New(bot)

    // 处理图片
    r.Photo(func(c *router.Context) {
        photo := c.Message.Photo[len(c.Message.Photo)-1]
        c.Reply("收到图片：" + photo.FileID)
    })

    // 处理文档
    r.Document(func(c *router.Context) {
        c.Reply("收到文档：" + c.Message.Document.FileName)
    })

    // 处理音频
    r.Audio(func(c *router.Context) {
        c.Reply("收到音频：" + c.Message.Audio.Title)
    })

    // 处理视频
    r.Video(func(c *router.Context) {
        c.Reply("收到视频：" + c.Message.Video.FileName)
    })

    r.Start()
}
```

## 位置机器人

这个示例展示了如何处理位置消息：

```go
package main

import (
    "log"
    "github.com/yourusername/telegram-router"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
    bot, err := tgbotapi.NewBotAPI("YOUR_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    r := router.New(bot)

    // 处理位置
    r.Location(func(c *router.Context) {
        c.Reply(fmt.Sprintf("你的位置：%f, %f",
            c.Message.Location.Latitude,
            c.Message.Location.Longitude))
    })

    // 处理特定范围内的位置
    r.LocationInRange(40.7, 40.8, -74.0, -73.9, func(c *router.Context) {
        c.Reply("你位于纽约曼哈顿区域")
    })

    // 处理地点
    r.Venue(func(c *router.Context) {
        c.Reply(fmt.Sprintf("地点：%s\n地址：%s",
            c.Message.Venue.Title,
            c.Message.Venue.Address))
    })

    r.Start()
}
```

## 投票机器人

这个示例展示了如何处理投票和测验：

```go
package main

import (
    "log"
    "github.com/yourusername/telegram-router"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
    bot, err := tgbotapi.NewBotAPI("YOUR_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    r := router.New(bot)

    // 处理 /poll 命令
    r.Command("poll", func(c *router.Context) {
        options := []string{"选项 1", "选项 2", "选项 3"}
        builder := c.ReplyWithPoll("你最喜欢什么颜色？", options, true, "regular")
        builder.Send()
    })

    // 处理 /quiz 命令
    r.Command("quiz", func(c *router.Context) {
        options := []string{"红色", "绿色", "蓝色"}
        builder := c.ReplyWithQuiz("天空是什么颜色？", options, 1)
        builder.Send()
    })

    // 处理投票
    r.Poll(func(c *router.Context) {
        c.Reply("收到投票：" + c.Message.Poll.Question)
    })

    // 处理测验
    r.Quiz(func(c *router.Context) {
        c.Reply("收到测验答案")
    })

    r.Start()
} 