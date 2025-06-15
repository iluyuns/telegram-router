# Telegram Router 文档

## 目录

1. [安装说明](installation.md)
2. [基本使用](basic-usage.md)
3. [中间件](middleware.md)
4. [消息类型](message-types.md)
5. [回调路由](callback-routing.md)
6. [高级特性](advanced-features.md)
7. [示例代码](examples.md)

## 概述

Telegram Router 是一个强大的 Telegram 机器人路由系统，灵感来自 Gin Web 框架。它提供了一种灵活直观的方式来处理各种类型的 Telegram 消息和更新。

### 主要特性

- **Gin 风格路由**：为 Telegram 机器人开发者提供熟悉的路由模式
- **中间件支持**：支持链式调用多个中间件函数
- **消息类型处理器**：处理不同类型的消息（文本、命令、媒体等）
- **基于上下文的处理**：提供丰富的上下文对象和辅助方法
- **路径参数**：支持动态路由参数
- **查询参数**：支持 URL 风格的查询参数
- **投票和测验处理**：专门的投票和测验处理器
- **基于位置的路由**：支持基于地理位置的路由
- **文件类型过滤**：过滤和处理特定类型的文件

### 基本示例

```go
package main

import (
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/yourusername/telegram-router"
)

func main() {
    // 初始化机器人
    bot, err := tgbotapi.NewBotAPI("你的机器人令牌")
    if err != nil {
        panic(err)
    }

    // 创建路由器
    router := telegramrouter.NewTelegramRouter(bot)

    // 添加中间件
    router.Use(Logger(), Auth([]int64{123456789}))

    // 注册处理器
    router.Command("start", func(c *Context) {
        c.Reply("欢迎使用机器人！")
    })

    router.Text(func(c *Context) {
        c.Reply("收到消息：" + c.Message.Text)
    })

    // 启动机器人
    u := tgbotapi.NewUpdate(0)
    updates := bot.GetUpdatesChan(u)

    for update := range updates {
        router.HandleUpdate(&update)
    }
}
```

### 支持的消息类型

路由器支持多种消息类型：

- 文本消息
- 命令
- 文档
- 音频
- 视频
- 照片
- 贴纸
- 位置
- 联系信息
- 投票和测验
- 语音消息
- 视频笔记
- 动画
- 频道消息

### 回调路由

支持带路径参数的高级回调路由：

```go
// 基本回调
router.Callback("menu/main", func(c *Context) {
    c.Reply("主菜单")
})

// 带路径参数
router.Callback("user/:id/profile", func(c *Context) {
    userID := c.Param("id")
    c.Reply(fmt.Sprintf("用户 %s 的个人资料", userID))
})

// 带查询参数
router.Callback("products/list", func(c *Context) {
    page := c.QueryInt("page", 1)
    sort := c.Query("sort", "id")
    c.Reply(fmt.Sprintf("第 %d 页，按 %s 排序", page, sort))
})
```

### 中间件

链式调用多个中间件函数：

```go
// 日志中间件
func Logger() telegramrouter.MiddlewareFunc {
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        start := time.Now()
        next(c)
        log.Printf("请求处理耗时：%v", time.Since(start))
    }
}

// 认证中间件
func Auth(allowedUsers []int64) telegramrouter.MiddlewareFunc {
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        userID := c.Message.From.ID
        for _, id := range allowedUsers {
            if id == userID {
                next(c)
                return
            }
        }
        c.Reply("未授权的访问")
        c.Abort()
    }
}
```

### 高级特性

- **基于位置的路由**：根据地理位置路由消息
- **文件类型过滤**：处理特定类型和大小的文件
- **投票类型处理**：专门的投票类型处理器
- **查询参数支持**：回调中的 URL 风格查询参数
- **请求中断**：停止处理请求链
- **上下文方法**：丰富的上下文辅助方法

更多详细信息，请参考具体的文档章节。 