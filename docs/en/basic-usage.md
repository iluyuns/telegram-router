# Basic Usage

This document introduces the basic usage of Telegram Router, including bot creation, message handling, and sending replies.

## Creating a Bot

### Initialization

```go
package main

import (
    "log"
    "github.com/yourusername/telegram-router"
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

    // Start the bot
    r.Start()
}
```

### Configuration Options

```go
// Use custom configuration
config := &router.PollingConfig{
    Timeout: 60,           // Timeout in seconds
    DropPendingUpdates: true, // Drop pending updates on startup
}
r.StartWithConfig(config)

// Use webhook mode
webhook := &router.WebhookConfig{
    ListenAddr: ":8443",
    CertFile:   "cert.pem",
    KeyFile:    "key.pem",
    WebhookURL: "https://your-domain.com/bot",
}
r.StartWebhook(webhook)
```

## Message Handling

### Command Handling

```go
// Handle /start command
r.Command("start", func(c *router.Context) {
    c.Reply("Welcome to the bot!")
})

// Handle commands with arguments
r.Command("echo", func(c *router.Context) {
    args := c.Args()
    if len(args) > 0 {
        c.Reply(strings.Join(args, " "))
    } else {
        c.Reply("Please provide text to echo")
    }
})

// Use regex to match commands
r.CommandRegex(regexp.MustCompile(`^user_\d+$`), func(c *router.Context) {
    userID := strings.TrimPrefix(c.Message.Command(), "user_")
    c.Reply("User ID: " + userID)
})
```

### Text Message Handling

```go
// Handle all text messages
r.Text(func(c *router.Context) {
    c.Reply("Received message: " + c.Message.Text)
})

// Handle specific text patterns
r.TextMatch("^hello", func(c *router.Context) {
    c.Reply("Hello!")
})

// Use regex to handle text
r.TextRegex(regexp.MustCompile(`(?i)help`), func(c *router.Context) {
    c.Reply("What help do you need?")
})
```

### Media Message Handling

```go
// Handle photos
r.Photo(func(c *router.Context) {
    photo := c.Message.Photo[len(c.Message.Photo)-1]
    c.Reply(fmt.Sprintf("Received photo: %s", photo.FileID))
})

// Handle documents
r.Document(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Received document: %s", c.Message.Document.FileName))
})

// Handle audio
r.Audio(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Received audio: %s", c.Message.Audio.Title))
})

// Handle video
r.Video(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Received video: %s", c.Message.Video.FileName))
})
```

### Special Message Handling

```go
// Handle location
r.Location(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Location: %f, %f",
        c.Message.Location.Latitude,
        c.Message.Location.Longitude))
})

// Handle venue
r.Venue(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Venue: %s, %s",
        c.Message.Venue.Title,
        c.Message.Venue.Address))
})

// Handle contact
r.Contact(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Contact: %s %s",
        c.Message.Contact.FirstName,
        c.Message.Contact.LastName))
})

// Handle poll
r.Poll(func(c *router.Context) {
    c.Reply("Received poll: " + c.Message.Poll.Question)
})
```

## Sending Messages

### Basic Replies

```go
// Send text message
c.Reply("Hello!")

// Use parse mode
c.Reply("**Bold** _Italic_", "Markdown")

// Reply to specific message
c.ReplyTo(c.Message.MessageID, "Reply message")
```

### Media Messages

```go
// Send photo
builder := c.ReplyWithPhotoFilePath("image.jpg")
builder.WithCaption("Photo caption")
builder.Send()

// Send document
builder := c.ReplyWithDocumentFileURL("https://example.com/doc.pdf")
builder.WithCaption("Document caption")
builder.Send()

// Send audio
builder := c.ReplyWithAudioFilePath("audio.mp3")
builder.WithTitle("Song title")
builder.WithPerformer("Artist")
builder.Send()

// Send video
builder := c.ReplyWithVideoFilePath("video.mp4")
builder.WithCaption("Video caption")
builder.WithDuration(60)
builder.Send()
```

### Special Messages

```go
// Send location
builder := c.ReplyWithLocation(40.7128, -74.0060)
builder.Send()

// Send venue
builder := c.ReplyWithVenue(40.7128, -74.0060,
    "Empire State Building",
    "350 5th Ave, New York")
builder.Send()

// Send contact
builder := c.ReplyWithContact("+1234567890", "John", "Doe")
builder.Send()

// Send poll
options := []string{"Option 1", "Option 2", "Option 3"}
builder := c.ReplyWithPoll("What's your favorite color?", options, true, "regular")
builder.Send()

// Send quiz
options := []string{"Red", "Green", "Blue"}
builder := c.ReplyWithQuiz("What color is the sky?", options, 1)
builder.Send()
```

## Keyboards and Buttons

### Reply Keyboard

```go
// Create reply keyboard
keyboard := tgbotapi.NewReplyKeyboard(
    tgbotapi.NewKeyboardButtonRow(
        tgbotapi.NewKeyboardButton("Button 1"),
        tgbotapi.NewKeyboardButton("Button 2"),
    ),
    tgbotapi.NewKeyboardButtonRow(
        tgbotapi.NewKeyboardButtonLocation("Send Location"),
        tgbotapi.NewKeyboardButtonContact("Share Contact"),
    ),
)

// Send message with keyboard
builder := c.Reply("Please select an option:")
builder.WithReplyMarkup(keyboard)
builder.Send()
```

### Inline Keyboard

```go
// Create inline keyboard
keyboard := tgbotapi.NewInlineKeyboardMarkup(
    tgbotapi.NewInlineKeyboardRow(
        tgbotapi.NewInlineKeyboardButtonData("Button 1", "data1"),
        tgbotapi.NewInlineKeyboardButtonURL("Website", "https://example.com"),
    ),
)

// Send message with inline keyboard
builder := c.Reply("Select an option:")
builder.WithReplyMarkup(keyboard)
builder.Send()
``` 