# Message Types

Telegram Router supports handling various types of Telegram messages. This document describes how to handle different message types and provides examples.

## Text Messages

### Basic Text Handling

```go
// Handle all text messages
r.Text(func(c *router.Context) {
    c.Reply("Received: " + c.Message.Text)
})

// Handle specific text pattern
r.TextMatch("^hello", func(c *router.Context) {
    c.Reply("Hello there!")
})

// Handle text with regex
r.TextRegex(regexp.MustCompile(`(?i)help`), func(c *router.Context) {
    c.Reply("How can I help you?")
})
```

### Command Messages

```go
// Handle /start command
r.Command("start", func(c *router.Context) {
    c.Reply("Welcome to the bot!")
})

// Handle command with parameters
r.Command("echo", func(c *router.Context) {
    // Get command arguments
    args := c.Args()
    if len(args) > 0 {
        c.Reply(strings.Join(args, " "))
    } else {
        c.Reply("Please provide some text to echo")
    }
})

// Handle command with pattern
r.CommandMatch("^user_", func(c *router.Context) {
    // Extract user ID from command
    userID := strings.TrimPrefix(c.Message.Command(), "user_")
    c.Reply("User ID: " + userID)
})
```

## Media Messages

### Photos

```go
// Handle all photos
r.Photo(func(c *router.Context) {
    // Get the largest photo
    photo := c.Message.Photo[len(c.Message.Photo)-1]
    c.Reply(fmt.Sprintf("Received photo: %s", photo.FileID))
})

// Handle photos with caption
r.PhotoWithCaption(func(c *router.Context) {
    c.Reply("Photo caption: " + c.Message.Caption)
})

// Send photo
builder := c.ReplyWithPhotoFilePath("image.jpg")
builder.WithCaption("Image description")
builder.WithParseMode("Markdown")
builder.Send()
```

### Documents

```go
// Handle all documents
r.Document(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Received document: %s", c.Message.Document.FileName))
})

// Handle specific document type
r.DocumentWithType("application/pdf", 10*1024*1024, func(c *router.Context) {
    c.Reply("Received PDF file under 10MB")
})

// Send document
builder := c.ReplyWithDocumentFileURL("https://example.com/doc.pdf", "Document description")
builder.Send()
```

### Audio

```go
// Handle all audio
r.Audio(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Received audio: %s", c.Message.Audio.Title))
})

// Handle audio with duration
r.AudioWithDuration(180, func(c *router.Context) {
    c.Reply("Received audio under 3 minutes")
})

// Send audio
builder := c.ReplyWithAudioFilePath("audio.mp3", "Audio description")
builder.WithTitle("Song Title")
builder.WithPerformer("Artist")
builder.WithDuration(180)
builder.Send()
```

### Video

```go
// Handle all videos
r.Video(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Received video: %s", c.Message.Video.FileName))
})

// Handle video with duration
r.VideoWithDuration(60, func(c *router.Context) {
    c.Reply("Received video under 1 minute")
})

// Send video
builder := c.ReplyWithVideoFilePath("video.mp4", "Video description")
builder.WithDuration(60)
builder.WithSupportsStreaming(true)
builder.Send()
```

### Voice Messages

```go
// Handle all voice messages
r.Voice(func(c *router.Context) {
    c.Reply("Received voice message")
})

// Handle voice with duration
r.VoiceWithDuration(30, func(c *router.Context) {
    c.Reply("Received voice message under 30 seconds")
})

// Send voice
builder := c.ReplyWithVoiceFilePath("voice.ogg")
builder.Send()
```

## Special Messages

### Location

```go
// Handle all locations
r.Location(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Location: %f, %f", 
        c.Message.Location.Latitude,
        c.Message.Location.Longitude))
})

// Handle location in range
r.LocationInRange(40.7, 40.8, -74.0, -73.9, func(c *router.Context) {
    c.Reply("You are in Manhattan, New York")
})

// Send location
builder := c.ReplyWithLocation(40.7128, -74.0060)
builder.Send()
```

### Venue

```go
// Handle all venues
r.Venue(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Venue: %s, %s", 
        c.Message.Venue.Title,
        c.Message.Venue.Address))
})

// Send venue
builder := c.ReplyWithVenue(40.7128, -74.0060, 
    "Empire State Building", 
    "350 5th Ave, New York")
builder.Send()
```

### Contact

```go
// Handle all contacts
r.Contact(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Contact: %s %s", 
        c.Message.Contact.FirstName,
        c.Message.Contact.LastName))
})

// Send contact
builder := c.ReplyWithContact("+1234567890", "John", "Doe")
builder.Send()
```

### Polls and Quizzes

```go
// Handle all polls
r.Poll(func(c *router.Context) {
    c.Reply("Received poll: " + c.Message.Poll.Question)
})

// Handle quiz
r.Quiz(func(c *router.Context) {
    c.Reply("Received quiz answer")
})

// Handle regular poll
r.RegularPoll(func(c *router.Context) {
    c.Reply("Received regular poll")
})

// Handle specific poll type
r.PollWithType("regular", 10, true, false, func(c *router.Context) {
    c.Reply("Received anonymous poll, minimum 10 votes, single choice")
})

// Send poll
options := []string{"Option 1", "Option 2", "Option 3"}
builder := c.ReplyWithPoll("What's your favorite color?", options, true, "regular")
builder.Send()

// Send quiz
options := []string{"Red", "Green", "Blue"}
builder := c.ReplyWithQuiz("What color is the sky?", options, 1) // Blue is correct
builder.Send()
```

## Channel Posts

```go
// Handle all channel posts
r.ChannelPost(func(c *router.Context) {
    c.Reply("Received channel post")
})

// Handle channel post with specific type
r.ChannelPostWithType("text", func(c *router.Context) {
    c.Reply("Received text channel post")
})
```

## Message Builder

All message types support the message builder pattern for sending messages:

```go
// Common builder methods
builder := c.Reply("Message")
builder.WithParseMode("Markdown")           // Set parse mode
builder.WithReplyMarkup(keyboard)          // Set keyboard
builder.WithReplyToMessageID(123)          // Reply to specific message
builder.WithAllowSendingWithoutReply(true) // Allow sending without reply
builder.WithProtectContent(true)           // Protect content
builder.Send()                             // Send message
```

## Error Handling

```go
// Handle errors in message sending
builder := c.Reply("Message")
if _, err := builder.Send(); err != nil {
    log.Printf("Error sending message: %v", err)
    // Handle error appropriately
}

// Handle specific error types
if err != nil {
    switch {
    case strings.Contains(err.Error(), "bot was blocked"):
        // Handle blocked bot
    case strings.Contains(err.Error(), "chat not found"):
        // Handle chat not found
    default:
        // Handle other errors
    }
}