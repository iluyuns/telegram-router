# Telegram Router Documentation

## Table of Contents

1. [Installation](installation.md)
2. [Basic Usage](basic-usage.md)
3. [Middleware](middleware.md)
4. [Message Types](message-types.md)
5. [Callback Routing](callback-routing.md)
6. [Advanced Features](advanced-features.md)
7. [Examples](examples.md)

## Overview

Telegram Router is a powerful routing system for Telegram bots, inspired by the Gin web framework. It provides a flexible and intuitive way to handle various types of Telegram messages and updates.

### Key Features

- **Gin-style Routing**: Familiar routing patterns for Telegram bot developers
- **Middleware Support**: Chain multiple middleware functions
- **Message Type Handlers**: Handle different types of messages (text, command, media, etc.)
- **Context-based Processing**: Rich context object with helper methods
- **Path Parameters**: Support for dynamic route parameters
- **Query Parameters**: URL-style query parameter support
- **Poll & Quiz Handling**: Specialized handlers for polls and quizzes
- **Location-based Routing**: Route based on geographic location
- **File Type Filtering**: Filter and handle specific file types

### Basic Example

```go
package main

import (
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/yourusername/telegram-router"
)

func main() {
    // Initialize bot
    bot, err := tgbotapi.NewBotAPI("YOUR_BOT_TOKEN")
    if err != nil {
        panic(err)
    }

    // Create router
    router := telegramrouter.NewTelegramRouter(bot)

    // Add middleware
    router.Use(Logger(), Auth([]int64{123456789}))

    // Register handlers
    router.Command("start", func(c *Context) {
        c.Reply("Welcome to the bot!")
    })

    router.Text(func(c *Context) {
        c.Reply("Received: " + c.Message.Text)
    })

    // Start bot
    u := tgbotapi.NewUpdate(0)
    updates := bot.GetUpdatesChan(u)

    for update := range updates {
        router.HandleUpdate(&update)
    }
}
```

### Message Type Support

The router supports various message types:

- Text messages
- Commands
- Documents
- Audio
- Video
- Photos
- Stickers
- Location
- Contact
- Polls and Quizzes
- Voice messages
- Video notes
- Animations
- Channel posts

### Callback Routing

Support for advanced callback routing with path parameters:

```go
// Basic callback
router.Callback("menu/main", func(c *Context) {
    c.Reply("Main menu")
})

// With path parameters
router.Callback("user/:id/profile", func(c *Context) {
    userID := c.Param("id")
    c.Reply(fmt.Sprintf("User %s's profile", userID))
})

// With query parameters
router.Callback("products/list", func(c *Context) {
    page := c.QueryInt("page", 1)
    sort := c.Query("sort", "id")
    c.Reply(fmt.Sprintf("Page %d, sorted by %s", page, sort))
})
```

### Middleware

Chain multiple middleware functions:

```go
// Logger middleware
func Logger() telegramrouter.MiddlewareFunc {
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        start := time.Now()
        next(c)
        log.Printf("Request processed in %v", time.Since(start))
    }
}

// Auth middleware
func Auth(allowedUsers []int64) telegramrouter.MiddlewareFunc {
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        userID := c.Message.From.ID
        for _, id := range allowedUsers {
            if id == userID {
                next(c)
                return
            }
        }
        c.Reply("Unauthorized access")
        c.Abort()
    }
}
```

### Advanced Features

- **Location-based Routing**: Route messages based on geographic location
- **File Type Filtering**: Handle specific file types and sizes
- **Poll Type Handling**: Specialized handlers for different poll types
- **Query Parameter Support**: URL-style query parameters in callbacks
- **Request Abortion**: Stop processing the request chain
- **Context Methods**: Rich set of helper methods in the context

For more detailed information, please refer to the specific documentation sections. 