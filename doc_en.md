<!--
Package tgr usage documentation (English)
This file provides examples and API reference for the Telegram router implemented in pkg/tgr.
-->

# tgr Usage Guide (English)

This file has a Chinese version: `doc.md`. You can use that to switch back to the Chinese documentation.

This document describes the `github.com/iluyuns/tgr` package which provides a Gin-like Telegram router. It includes quick start examples, common APIs, Webhook and long polling examples, context (`Context`) utilities and middleware usage.

## Overview

`tgr` offers a Telegram bot router with support for commands, text, media, callback queries and middleware. The main types are:

- `TelegramRouter`: the router for registering handlers and middleware.
- `Context`: per-update context that contains the raw update, Bot instance, params and helper methods.

## Installation

```bash
go get github.com/iluyuns/tgr
```

## Quick Start (Long Polling)

```go
package main

import (
    "context"
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
        c.Reply("Welcome!").Send()
    })

    router.Text(func(c *tgr.Context) {
        if c.Message != nil {
            c.Reply("Received: " + c.Message.Text).Send()
        }
    })

    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()
    router.ListenWithContext(ctx, 8, 1024)
}
```

## Webhook Example

```go
webhookCfg := tgr.WebhookConfig{
    WebhookURL: "https://example.com:8443/bot",
    CertFile:   "/path/to/cert.pem",
}
router.SetWebhook(webhookCfg)

srv := &http.Server{Addr: ":8443"}
router.AttachToServer(srv, "/bot")
```

## Middleware

Register global middleware using `Use`:

```go
router.Use(func(c *tgr.Context) {
    // pre-processing
    c.Next()
})
```

`NewTelegramRouterWithDefaultRecover` registers a default recover middleware.

## Handlers Overview

- `Command`, `Text`, `Document`, `Photo`, `Audio`, `Callback`, `CommandRegex`, `TextMatch`, `TextRegex` etc.

Callback route example:

```go
router.Callback("order/:orderId/status", func(c *tgr.Context) {
    orderId := c.Param("orderId")
    status := c.Query("status", "unknown")
})
```

## Context Helpers

- `c.Reply(text)` returns a `TextMessageBuilder` with `.Send()`.
- `c.AnswerCallback(opts)` answers a callback query.
- `c.EditMessageText(text, opts)` edits messages in callback context.
- `c.Param`, `c.Query`, `c.QueryInt`, `c.QueryBool` for params and query parsing.

## Advanced

- `ListenWithContext` provides graceful shutdown with worker pool and bounded queue.
- `SetErrorReporter` and `SetLogger` for integrations and custom logging.

## Support the Project

If this project helps you, please consider supporting it:

![BTC Support](docs/btc.jpeg)
![WeChat Support](docs/wechat.jpg)



