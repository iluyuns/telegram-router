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
func Logger(c *router.Context) {
    start := time.Now()
    c.Next()
    log.Printf("处理消息用时：%v", time.Since(start))
}

// 认证中间件
func Auth(allowedUsers []int64) router.HandlerFunc {
    return func(c *router.Context) {
        userID := c.Message.From.ID
        for _, id := range allowedUsers {
            if id == userID {
                c.Next()
                return
            }
        }
        c.Reply("未授权访问")
    }
}

func main() {
    // 创建机器人实例
    bot, err := tgbotapi.NewBotAPI("YOUR_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    // 创建路由器
    r := router.New(bot)

    // 添加中间件
    r.Use(Logger, Auth([]int64{123456789}))

    // 注册命令处理器
    r.Command("start", func(c *router.Context) {
        c.Reply("欢迎使用机器人！")
    })

    // 启动机器人
    log.Printf("机器人已启动：%s", bot.Self.UserName)
    r.Listen()
}
```

## 命令处理示例

这个示例展示了如何使用多个处理器处理命令：

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

    // 注册命令处理器，使用多个处理函数
    r.Command("start",
        // 发送欢迎消息
        func(c *router.Context) {
            c.Reply("欢迎使用机器人！")
        },
        // 记录用户信息
        func(c *router.Context) {
            log.Printf("用户 %d 使用了 start 命令", c.Message.From.ID)
        },
        // 发送帮助信息
        func(c *router.Context) {
            c.Reply("使用 /help 查看帮助信息")
        },
    )

    // 注册文本消息处理器，使用多个处理函数
    r.Text(
        // 回复消息
        func(c *router.Context) {
            c.Reply("收到您的消息：" + c.Message.Text)
        },
        // 记录消息
        func(c *router.Context) {
            log.Printf("用户 %d 发送消息：%s", 
                c.Message.From.ID, 
                c.Message.Text)
        },
    )

    // 启动机器人
    log.Printf("机器人已启动：%s", bot.Self.UserName)
    r.Listen()
}
```

## 回调查询示例

这个示例展示了如何处理带参数的回调查询：

```go
package main

import (
    "fmt"
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
    r.Command("menu", func(c *router.Context) {
        // 创建内联键盘
        keyboard := tgbotapi.NewInlineKeyboardMarkup(
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButtonData("选项 1", "menu/option1"),
                tgbotapi.NewInlineKeyboardButtonData("选项 2", "menu/option2"),
            ),
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButtonData("用户资料", "user/123/profile"),
            ),
        )

        // 发送消息和键盘
        c.Reply("请选择一个选项：").WithReplyMarkup(keyboard)
    })

    // 注册回调查询处理器，使用多个处理函数
    r.Callback("menu/option1",
        // 回复回调查询
        func(c *router.Context) {
            c.AnswerCallback("您选择了选项 1")
        },
        // 发送消息
        func(c *router.Context) {
            c.Reply("选项 1 已选择")
        },
        // 记录操作
        func(c *router.Context) {
            log.Printf("用户 %d 选择了选项 1", c.Message.From.ID)
        },
    )

    // 注册带参数的回调查询处理器
    r.Callback("user/:id/profile",
        // 验证用户权限
        func(c *router.Context) {
            userID := c.Param("id")
            if userID != fmt.Sprintf("%d", c.Message.From.ID) {
                c.AnswerCallback("无权访问其他用户的资料")
                return
            }
            c.Next()
        },
        // 发送用户资料
        func(c *router.Context) {
            c.AnswerCallback("正在加载资料...")
            c.Reply(fmt.Sprintf("用户 %s 的资料", c.Param("id")))
        },
    )

    // 启动机器人
    log.Printf("机器人已启动：%s", bot.Self.UserName)
    r.Listen()
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