# Installation Guide

## System Requirements

- Go 1.16 or higher
- Telegram Bot Token (get it from [@BotFather](https://t.me/BotFather))

## Quick Start

1. Install the package:

```bash
go get github.com/iluyuns/telegram-router
```

2. Create `main.go`:

```go
package main

import (
    "log"
    "github.com/iluyuns/telegram-router"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
    // Create bot instance
    bot, err := tgbotapi.NewBotAPI("YOUR_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    // Create router
    r := router.New(bot)

    // Register command handler
    r.Command("start", func(c *router.Context) {
        c.Reply("Welcome to the bot!")
    })

    // Start the bot
    r.Start()
}
```

3. Run the bot:

```bash
go run main.go
```

## Basic Configuration

### Environment Variables (Optional)

```bash
export TELEGRAM_BOT_TOKEN="your-bot-token"
```

## Next Steps

- Check out [Basic Usage](basic-usage.md) to learn how to register different types of message handlers
- Learn about [Middleware](middleware.md) usage 