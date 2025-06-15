# Examples

This document provides various examples of using Telegram Router in different scenarios.

## Basic Bot

A simple bot that responds to commands and text messages:

```go
package main

import (
    "log"
    "os"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    router "github.com/iluyuns/telegram-router"
)

func main() {
    // Create bot instance
    bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
    if err != nil {
        log.Fatal(err)
    }

    // Create router
    r := router.NewTelegramRouter(bot)

    // Register command handlers
    r.Command("start", func(c *router.Context) {
        // Send welcome message
        builder := c.Reply("Welcome to the bot!\n\n" +
            "Available commands:\n" +
            "/photo - Send photo\n" +
            "/document - Send document\n" +
            "/audio - Send audio\n" +
            "/video - Send video\n" +
            "/voice - Send voice\n" +
            "/location - Send location\n" +
            "/contact - Send contact\n" +
            "/poll - Send poll")

        // Add keyboard
        keyboard := tgbotapi.NewReplyKeyboard(
            tgbotapi.NewKeyboardButtonRow(
                tgbotapi.NewKeyboardButton("Send Photo"),
                tgbotapi.NewKeyboardButton("Send Location"),
            ),
            tgbotapi.NewKeyboardButtonRow(
                tgbotapi.NewKeyboardButton("Send Contact"),
                tgbotapi.NewKeyboardButton("Send Poll"),
            ),
        )
        builder.WithReplyMarkup(keyboard)

        // Send message
        if _, err := builder.Send(); err != nil {
            log.Printf("Error sending message: %v", err)
        }
    })

    // Register text message handlers
    r.Text(func(c *router.Context) {
        switch c.Message.Text {
        case "Send Photo":
            // Send photo using file path
            builder := c.ReplyWithPhotoFilePath("examples/photo.jpg")
            if _, err := builder.
                WithCaption("Photo caption").
                WithParseMode("Markdown").
                Send(); err != nil {
                log.Printf("Error sending photo: %v", err)
            }

        case "Send Location":
            // Send location
            builder := c.ReplyWithLocation(40.7128, -74.0060)
            if _, err := builder.Send(); err != nil {
                log.Printf("Error sending location: %v", err)
            }

        case "Send Contact":
            // Send contact
            builder := c.ReplyWithContact("+1234567890", "John", "Doe")
            if _, err := builder.Send(); err != nil {
                log.Printf("Error sending contact: %v", err)
            }

        case "Send Poll":
            // Send poll
            options := []string{"Option 1", "Option 2", "Option 3"}
            builder := c.ReplyWithPoll("What's your favorite color?", options, true, "regular")
            if _, err := builder.Send(); err != nil {
                log.Printf("Error sending poll: %v", err)
            }
        }
    })

    // Start bot
    log.Printf("Bot started: %s", bot.Self.UserName)
    r.Listen()
}
```

## Webhook Bot

A bot using webhook mode with HTTP framework integration:

```go
package main

import (
    "log"
    "os"

    "github.com/gin-gonic/gin"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    router "github.com/iluyuns/telegram-router"
)

func main() {
    // Create bot instance
    bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
    if err != nil {
        log.Fatal(err)
    }

    // Create router
    r := router.NewTelegramRouter(bot)

    // Register handlers
    r.Command("start", func(c *router.Context) {
        c.Reply("Welcome to the webhook bot!")
    })

    // Create Gin router
    ginRouter := gin.Default()

    // Handle webhook requests
    ginRouter.POST("/bot", func(c *gin.Context) {
        r.HandleWebhookRequest(c.Writer, c.Request)
    })

    // Configure webhook
    config := router.WebhookConfig{
        ListenAddr: ":8443",
        CertFile:   "cert.pem",
        KeyFile:    "key.pem",
        WebhookURL: "https://example.com:8443/bot",
    }

    // Set webhook
    if err := r.SetWebhook(config); err != nil {
        log.Fatal(err)
    }

    // Start server
    log.Printf("Bot started: %s", bot.Self.UserName)
    if err := ginRouter.RunTLS(":8443", "cert.pem", "key.pem"); err != nil {
        log.Fatal(err)
    }
}
```

## Middleware Example

A bot with middleware for logging and authentication:

```go
package main

import (
    "log"
    "os"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    router "github.com/iluyuns/telegram-router"
)

// Logger middleware
func Logger() router.MiddlewareFunc {
    return func(c *router.Context, next router.HandlerFunc) {
        start := time.Now()
        next(c)
        log.Printf("Request processed in %v", time.Since(start))
    }
}

// Auth middleware
func Auth(allowedUsers []int64) router.MiddlewareFunc {
    return func(c *router.Context, next router.HandlerFunc) {
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

func main() {
    // Create bot instance
    bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
    if err != nil {
        log.Fatal(err)
    }

    // Create router
    r := router.NewTelegramRouter(bot)

    // Add middleware
    r.Use(Logger(), Auth([]int64{123456789}))

    // Register handlers
    r.Command("start", func(c *router.Context) {
        c.Reply("Welcome to the authenticated bot!")
    })

    // Start bot
    log.Printf("Bot started: %s", bot.Self.UserName)
    r.Listen()
}
```

## Callback Query Example

A bot with inline keyboard and callback query handling:

```go
package main

import (
    "fmt"
    "log"
    "os"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    router "github.com/iluyuns/telegram-router"
)

func main() {
    // Create bot instance
    bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
    if err != nil {
        log.Fatal(err)
    }

    // Create router
    r := router.NewTelegramRouter(bot)

    // Register command handler
    r.Command("menu", func(c *router.Context) {
        // Create inline keyboard
        keyboard := tgbotapi.NewInlineKeyboardMarkup(
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButtonData("Option 1", "menu/option1"),
                tgbotapi.NewInlineKeyboardButtonData("Option 2", "menu/option2"),
            ),
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButtonData("User Profile", "user/123/profile"),
            ),
        )

        // Send message with keyboard
        builder := c.Reply("Please select an option:")
        builder.WithReplyMarkup(keyboard)
        if _, err := builder.Send(); err != nil {
            log.Printf("Error sending message: %v", err)
        }
    })

    // Register callback handlers
    r.Callback("menu/option1", func(c *router.Context) {
        c.AnswerCallback("You selected Option 1")
        c.Reply("Option 1 selected")
    })

    r.Callback("menu/option2", func(c *router.Context) {
        c.AnswerCallback("You selected Option 2")
        c.Reply("Option 2 selected")
    })

    // Handle callback with path parameters
    r.Callback("user/:id/profile", func(c *router.Context) {
        userID := c.Param("id")
        c.AnswerCallback(fmt.Sprintf("Viewing profile of user %s", userID))
        c.Reply(fmt.Sprintf("Profile of user %s", userID))
    })

    // Start bot
    log.Printf("Bot started: %s", bot.Self.UserName)
    r.Listen()
}
```

## File Handling Example

A bot that handles different types of files:

```go
package main

import (
    "fmt"
    "log"
    "os"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    router "github.com/iluyuns/telegram-router"
)

func main() {
    // Create bot instance
    bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
    if err != nil {
        log.Fatal(err)
    }

    // Create router
    r := router.NewTelegramRouter(bot)

    // Handle photos
    r.Photo(func(c *router.Context) {
        photo := c.Message.Photo[len(c.Message.Photo)-1]
        c.Reply(fmt.Sprintf("Received photo: %dx%d", photo.Width, photo.Height))
    })

    // Handle documents
    r.DocumentWithType("application/pdf", 10*1024*1024, func(c *router.Context) {
        c.Reply(fmt.Sprintf("Received PDF file: %s", c.Message.Document.FileName))
    })

    // Handle audio
    r.AudioWithDuration(180, func(c *router.Context) {
        c.Reply(fmt.Sprintf("Received audio: %s", c.Message.Audio.Title))
    })

    // Handle video
    r.VideoWithDuration(60, func(c *router.Context) {
        c.Reply(fmt.Sprintf("Received video: %s", c.Message.Video.FileName))
    })

    // Start bot
    log.Printf("Bot started: %s", bot.Self.UserName)
    r.Listen()
}
```

## Location-based Bot

A bot that responds to location messages:

```go
package main

import (
    "log"
    "os"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    router "github.com/iluyuns/telegram-router"
)

func main() {
    // Create bot instance
    bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
    if err != nil {
        log.Fatal(err)
    }

    // Create router
    r := router.NewTelegramRouter(bot)

    // Handle all locations
    r.Location(func(c *router.Context) {
        c.Reply(fmt.Sprintf("Your location: %f, %f",
            c.Message.Location.Latitude,
            c.Message.Location.Longitude))
    })

    // Handle locations in Manhattan
    r.LocationInRange(40.7, 40.8, -74.0, -73.9, func(c *router.Context) {
        c.Reply("You are in Manhattan, New York")
    })

    // Handle venues
    r.Venue(func(c *router.Context) {
        c.Reply(fmt.Sprintf("Venue: %s, %s",
            c.Message.Venue.Title,
            c.Message.Venue.Address))
    })

    // Start bot
    log.Printf("Bot started: %s", bot.Self.UserName)
    r.Listen()
}
```

## Poll Bot

A bot that handles polls and quizzes:

```go
package main

import (
    "log"
    "os"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    router "github.com/iluyuns/telegram-router"
)

func main() {
    // Create bot instance
    bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
    if err != nil {
        log.Fatal(err)
    }

    // Create router
    r := router.NewTelegramRouter(bot)

    // Handle all polls
    r.Poll(func(c *router.Context) {
        c.Reply("Received poll: " + c.Message.Poll.Question)
    })

    // Handle quizzes
    r.Quiz(func(c *router.Context) {
        c.Reply("Received quiz answer")
    })

    // Handle regular polls
    r.RegularPoll(func(c *router.Context) {
        c.Reply("Received regular poll")
    })

    // Handle specific poll type
    r.PollWithType("regular", 10, true, false, func(c *router.Context) {
        c.Reply("Received anonymous poll, minimum 10 votes, single choice")
    })

    // Start bot
    log.Printf("Bot started: %s", bot.Self.UserName)
    r.Listen()
} 