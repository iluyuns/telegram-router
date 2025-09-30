# tgr 使用说明

本文档介绍 `github.com/iluyuns/tgr` 包中提供的 Telegram 路由器（`TelegramRouter`）的使用方法，包括快速开始、常用 API、Webhook 与长轮询示例、上下文 (`Context`) 常用方法以及中间件使用方式。

## 简介

`tgr` 提供了一个类 Gin 风格的 Telegram 机器人路由系统，支持命令、文本、媒体、回调查询等多种消息类型的路由处理，并带有中间件支持与多种便捷的消息构建器。

主要类型：

- `TelegramRouter`：路由器本体，用于注册各种消息处理器与中间件。
- `Context`：每次更新分发时构造的上下文，包含原始更新、Bot 实例、参数与实用方法。

## 安装

在你的模块中引入（假设包路径已在模块内）：

```bash
go get github.com/iluyuns/tgr
```

（根据实际模块路径替换上面的包路径）

## 快速开始（长轮询）

```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/iluyuns/tgr"
)

func main() {
    bot, err := tgbotapi.NewBotAPI("TELEGRAM_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    router := tgr.NewTelegramRouterWithDefaultRecover(bot)

    router.Command("start", func(c *tgr.Context) {
        c.Reply("欢迎使用机器人！").Send()
    })

    // 注册文本处理
    router.Text(func(c *tgr.Context) {
        if c.Message != nil {
            c.Reply("收到文本: " + c.Message.Text).Send()
        }
    })

    // 启动带取消上下文的长轮询
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()
    router.ListenWithContext(ctx, 8, 1024)
}
```

## 使用 Webhook（示例）

```go
// 设置 webhook（通过 Bot API）
webhookCfg := tgr.WebhookConfig{
    WebhookURL: "https://example.com:8443/bot",
    CertFile:   "/path/to/cert.pem", // 可选
}
router.SetWebhook(webhookCfg)

// 或者将处理函数挂载到已有的 http.Server
srv := &http.Server{Addr: ":8443"}
router.AttachToServer(srv, "/bot")
// 由外部启动 srv.ListenAndServeTLS
```

另外 `TelegramRouter` 提供 `NewWebhookServer(listenAddr, path)` 来构造不启动的 `*http.Server`，便于自定义启动逻辑。

## 中间件

使用 `Use` 注册全局中间件：

```go
router.Use(func(c *tgr.Context) {
    // 预处理
    c.Next() // 调用下一个处理器
})
```

常用的中间件惯用法：记录日志、鉴权、恢复（recover）等。

`NewTelegramRouterWithDefaultRecover` 会自动添加一个恢复中间件以捕获 panic 并中断当前处理链。

## 路由注册概览

- `router.Command(name, handlers...)`：注册命令处理器（例如 `/start`）。
- `router.Text(handlers...)`：注册文本消息处理器。
- `router.Document(handlers...)`、`router.Photo(handlers...)`、`router.Audio` 等：注册对应媒体类型处理器。
- `router.Callback(pattern, handlers...)`：注册回调查询路由，支持路径参数（如 `action/:id`）与通配符 `*`。
- `router.CommandRegex(regex, handlers...)`：基于正则的命令匹配。
- `router.TextMatch(pattern, handler)` / `router.TextRegex(regex, handler)`：更灵活的文本匹配。

示例：回调路由与参数

```go
router.Callback("order/:orderId/status", func(c *tgr.Context) {
    orderId := c.Param("orderId")
    // 回调数据里也可以带查询参数，框架会解析到 c.query
    status := c.Query("status", "unknown")
    // 处理逻辑 ...
})
```

## Context 常用方法

- `c.Reply(text)`：构造文本回复，返回 `TextMessageBuilder`，可链式调用 `.WithParseMode(...)` / `.WithInlineKeyboard(...)` / `.Send()`。
- `c.ReplyWithPhotoFileID(fileID)` / `c.ReplyWithPhotoFileURL(url)` / `c.ReplyWithPhotoFileBytes(bytes)`：图片回复构建器。
- `c.ReplyWithDocumentFilePath(path)`：直接发送文档（同步返回错误）。
- `c.AnswerCallback(opts)`：在回调查询上下文中回复 CallbackQuery。
- `c.EditMessageText(text, opts)`：编辑回调消息文本（支持 inline message）。
- `c.Param(key)`：获取回调路由或路径参数。
- `c.Query(key, default...)`、`c.QueryInt(...)`、`c.QueryBool(...)`：获取回调数据或 URL 查询参数（当回调数据形如 `path?a=1&b=2` 时解析）。
- `c.Abort()`、`c.Next()`：控制中间件/处理链流程。

示例：回答回调并编辑消息

```go
router.Callback("item/:id", func(c *tgr.Context) {
    id := c.Param("id")
    _ = c.AnswerCallback(tgr.AnswerCallbackOptions{Text: "已收到"})
    _ = c.EditMessageText("已处理: " + id, nil)
})
```

## 进阶功能

- `ListenWithContext(ctx, workers, queueSize)`：带取消上下文的并发长轮询实现，内部使用有界缓冲队列和 worker 池，优雅关闭时会尝试 drain 剩余更新，推荐用于生产环境。
- `SetErrorReporter(r ErrorReporter)`：设置自定义错误上报器（例如 Sentry），路由器在处理失败或 webhook 解析失败时会调用。
- `SetLogger(logger)`：替换默认日志器。

## 常见场景与建议

- 在需要高可靠性接收更新的场合，优先使用 `ListenWithContext` 并设置合理的 `workers` 与 `queueSize`。
- 若使用 Webhook，确保 Bot 的 Webhook URL 与证书配置正确，且 HTTP 服务对外可达。
- 中间件中尽量使用 `c.Next()` 与 `c.Abort()` 明确控制流程，避免无意继续执行下游处理器。

## 示例：组合中间件、命令与回调

```go
router.Use(LoggerMiddleware, AuthMiddleware)

router.Command("start", func(c *tgr.Context) {
    c.Reply("欢迎！").Send()
})

router.Callback("task/:taskId/do", func(c *tgr.Context) {
    taskId := c.Param("taskId")
    // 处理任务
    c.Reply("任务已触发: " + taskId).Send()
})
```

## 链接与参考

- 机器人库参考：`github.com/go-telegram-bot-api/telegram-bot-api/v5`




