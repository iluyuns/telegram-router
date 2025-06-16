# Advanced Features

Telegram Router provides many advanced features to help you build more powerful and flexible bot applications.

## Path Parameters

### Basic Usage

```go
// Handle callback with parameters
router.Callback("user/:id/profile", func(c *telegramrouter.Context) {
    userID := c.Param("id")
    c.Reply(fmt.Sprintf("Profile of user %s", userID))
})

// Handle multi-level paths
router.Callback("category/:categoryId/product/:productId", func(c *telegramrouter.Context) {
    categoryID := c.Param("categoryId")
    productID := c.Param("productId")
    c.Reply(fmt.Sprintf("Product %s in category %s", productID, categoryID))
})
```

### Parameter Validation

```go
// Callback handler with validation
func validateUserID(c *telegramrouter.Context) bool {
    userID := c.Param("id")
    // Validate user ID format
    if _, err := strconv.ParseInt(userID, 10, 64); err != nil {
        c.Reply("Invalid user ID")
        return false
    }
    return true
}

router.Callback("user/:id/profile", func(c *telegramrouter.Context) {
    if !validateUserID(c) {
        return
    }
    // Handle valid request
})
```

## Query Parameters

### Basic Queries

```go
// Handle callback with query parameters
router.Callback("products/list", func(c *telegramrouter.Context) {
    // Get parameters with default values
    page := c.QueryInt("page", 1)
    size := c.QueryInt("size", 10)
    sort := c.Query("sort", "id")
    
    c.Reply(fmt.Sprintf("Page %d, %d items per page, sorted by %s", page, size, sort))
})
```

### Advanced Queries

```go
// Handle complex query parameters
router.Callback("search", func(c *telegramrouter.Context) {
    // Get multiple parameters
    query := c.Query("q", "")
    category := c.Query("category", "all")
    minPrice := c.QueryInt("min_price", 0)
    maxPrice := c.QueryInt("max_price", 0)
    tags := strings.Split(c.Query("tags", ""), ",")
    
    // Build query conditions
    conditions := make(map[string]interface{})
    if query != "" {
        conditions["query"] = query
    }
    if category != "all" {
        conditions["category"] = category
    }
    if minPrice > 0 {
        conditions["min_price"] = minPrice
    }
    if maxPrice > 0 {
        conditions["max_price"] = maxPrice
    }
    if len(tags) > 0 && tags[0] != "" {
        conditions["tags"] = tags
    }
    
    // Process query
    c.Reply(fmt.Sprintf("Search conditions: %+v", conditions))
})
```

## Location Handling

### Location Range Matching

```go
// Handle location within specific range
router.LocationInRange(39.9, 40.1, 116.3, 116.5, func(c *telegramrouter.Context) {
    c.Reply("You are near Tiananmen Square in Beijing!")
})

// Handle multiple location ranges
func registerLocationHandlers(router *telegramrouter.TelegramRouter) {
    // Beijing Tiananmen
    router.LocationInRange(39.9, 40.1, 116.3, 116.5, func(c *telegramrouter.Context) {
        c.Reply("You are near Tiananmen Square!")
    })
    
    // Shanghai Bund
    router.LocationInRange(31.2, 31.3, 121.4, 121.5, func(c *telegramrouter.Context) {
        c.Reply("You are near the Bund!")
    })
    
    // Guangzhou Tower
    router.LocationInRange(23.1, 23.2, 113.3, 113.4, func(c *telegramrouter.Context) {
        c.Reply("You are near the Canton Tower!")
    })
}
```

### Live Location

```go
// Handle live location sharing
router.LiveLocation(func(c *telegramrouter.Context) {
    loc := c.Message.Location
    c.Reply(fmt.Sprintf("Live location received:\nLatitude: %f\nLongitude: %f\nAccuracy: %dm",
        loc.Latitude,
        loc.Longitude,
        loc.HorizontalAccuracy))
})
```

## File Handling

### File Type Filtering

```go
// Handle specific file types
router.DocumentWithType("application/pdf", 10*1024*1024, func(c *telegramrouter.Context) {
    c.Reply("Received PDF file, size under 10MB")
})

// Handle multiple file types
func registerFileHandlers(router *telegramrouter.TelegramRouter) {
    // PDF files
    router.DocumentWithType("application/pdf", 10*1024*1024, func(c *telegramrouter.Context) {
        c.Reply("Received PDF file")
    })
    
    // Image files
    router.DocumentWithType("image/jpeg", 5*1024*1024, func(c *telegramrouter.Context) {
        c.Reply("Received JPEG image")
    })
    
    // Video files
    router.DocumentWithType("video/mp4", 50*1024*1024, func(c *telegramrouter.Context) {
        c.Reply("Received MP4 video")
    })
}
```

### File Download

```go
// Download and process files
router.Document(func(c *telegramrouter.Context) {
    doc := c.Message.Document
    
    // Get file info
    file, err := c.Bot.GetFile(tgbotapi.FileConfig{
        FileID: doc.FileID,
    })
    if err != nil {
        c.Reply("Failed to get file info")
        return
    }
    
    // Download file
    resp, err := http.Get(file.Link(c.Bot.Token))
    if err != nil {
        c.Reply("Failed to download file")
        return
    }
    defer resp.Body.Close()
    
    // Process file
    // ...
})
```

## Polls and Quizzes

### Poll Handling

```go
// Handle regular polls
router.RegularPoll(func(c *telegramrouter.Context) {
    poll := c.Message.Poll
    c.Reply(fmt.Sprintf("Poll received: %s\nOptions: %d\nTotal votes: %d",
        poll.Question,
        len(poll.Options),
        poll.TotalVoterCount))
})

// Handle quizzes
router.Quiz(func(c *telegramrouter.Context) {
    poll := c.Message.Poll
    c.Reply(fmt.Sprintf("Quiz received: %s\nCorrect answer: %d\nExplanation: %s",
        poll.Question,
        poll.CorrectOptionID,
        poll.Explanation))
})

// Handle specific poll types
router.PollWithType("regular", 5, true, true, func(c *telegramrouter.Context) {
    c.Reply("Received anonymous multiple-choice poll, minimum 5 votes required")
})
```

## Advanced Callback Handling

### Dynamic Callbacks

```go
// Generate dynamic callback data
func generateCallbackData(action string, params map[string]string) string {
    // Build base path
    data := action
    
    // Add query parameters
    if len(params) > 0 {
        values := url.Values{}
        for k, v := range params {
            values.Set(k, v)
        }
        data += "?" + values.Encode()
    }
    
    return data
}

// Use dynamic callbacks
router.Command("menu", func(c *telegramrouter.Context) {
    // Create menu buttons
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButton(
                "Product List",
                generateCallbackData("products/list", map[string]string{
                    "page": "1",
                    "sort": "name",
                }),
            ),
        ),
    )
    
    c.ReplyWithKeyboard("Please select an action:", keyboard)
})
```

### Callback Chains

```go
// Handle callback chain
func handleCallbackChain(c *telegramrouter.Context) {
    // Get current state
    state := c.Query("state", "initial")
    
    // Handle based on state
    switch state {
    case "initial":
        // Show first step
        keyboard := tgbotapi.NewInlineKeyboardMarkup(
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButton(
                    "Next",
                    generateCallbackData("wizard/step", map[string]string{
                        "state": "step1",
                    }),
                ),
            ),
        )
        c.ReplyWithKeyboard("Please start:", keyboard)
        
    case "step1":
        // Handle first step
        keyboard := tgbotapi.NewInlineKeyboardMarkup(
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButton(
                    "Complete",
                    generateCallbackData("wizard/step", map[string]string{
                        "state": "complete",
                    }),
                ),
            ),
        )
        c.ReplyWithKeyboard("Step 1 completed, please continue:", keyboard)
        
    case "complete":
        // Complete processing
        c.Reply("Operation completed!")
    }
}

// Register callback handler
router.Callback("wizard/step", handleCallbackChain)
```

## Best Practices

1. **Parameter Validation**: Always validate user input and parameters
2. **Error Handling**: Use middleware for unified error handling
3. **State Management**: Use callback chains for complex state management
4. **Resource Cleanup**: Clean up temporary files and resources promptly
5. **Performance Optimization**: Use caching and async processing for better performance

## Common Issues

### 1. Callback Data Too Long

- Use short parameter names
- Only pass necessary data
- Consider using state storage

### 2. Location Accuracy Issues

- Check location accuracy parameters
- Appropriately expand matching ranges
- Consider using geocoding services

### 3. File Processing Timeout

- Use async processing
- Set reasonable timeout values
- Implement resumable downloads

## Next Steps

- Check [examples](examples.md) for more inspiration
- Explore [custom middleware](custom-middleware.md)
- Learn about [performance optimization](performance.md) techniques

## Webhook Support

Telegram Router supports both Long Polling and Webhook modes. Webhook mode is recommended for production environments as it provides better performance and reliability.

### Webhook Configuration

```go
config := router.WebhookConfig{
    ListenAddr: ":8443",                    // Listening address
    CertFile:   "cert.pem",                 // SSL certificate file path
    KeyFile:    "key.pem",                  // SSL private key file path
    WebhookURL: "https://example.com:8443/bot", // Webhook URL
}

// Start Webhook server
if err := r.StartWebhook(config); err != nil {
    log.Fatal(err)
}
```

### Using with HTTP Frameworks

Telegram Router provides the `HandleWebhookRequest` method that can be used with any HTTP framework:

#### Gin Framework

```go
router := gin.Default()
router.POST("/bot", func(c *gin.Context) {
    r.HandleWebhookRequest(c.Writer, c.Request)
})
```

#### Echo Framework

```go
e := echo.New()
e.POST("/bot", func(c echo.Context) error {
    r.HandleWebhookRequest(c.Response().Writer, c.Request())
    return nil
})
```

#### Standard Library net/http

```go
http.HandleFunc("/bot", r.HandleWebhookRequest)
```

### Webhook Management

```go
// Set Webhook
err := r.SetWebhook(config)

// Remove Webhook
err = r.RemoveWebhook()
```

### Important Notes

1. Webhook mode requires HTTPS support (Telegram requirement)
2. Use official SSL certificates in production
3. Consider using Let's Encrypt for free SSL certificates
4. Ensure server firewall allows the required port
5. Recommended to use a reverse proxy (like Nginx) for HTTPS

## Message Builder

Telegram Router provides a rich message builder API with chain-style calls:

### Text Messages

```go
builder := c.Reply("Hello, World!")
builder.WithParseMode("Markdown")
builder.WithReplyMarkup(keyboard)
builder.Send()
```

### Media Messages

```go
// Photo
builder := c.ReplyWithPhotoFilePath("image.jpg")
builder.WithCaption("Image caption")
builder.WithParseMode("Markdown")
builder.Send()

// Document
builder := c.ReplyWithDocumentFileURL("https://example.com/doc.pdf", "Document description")
builder.Send()

// Audio
builder := c.ReplyWithAudioFilePath("audio.mp3", "Audio description")
builder.WithTitle("Song Title")
builder.WithPerformer("Artist")
builder.WithDuration(180)
builder.Send()

// Video
builder := c.ReplyWithVideoFilePath("video.mp4", "Video description")
builder.WithDuration(60)
builder.WithSupportsStreaming(true)
builder.Send()
```

### Special Messages

```go
// Location
builder := c.ReplyWithLocation(40.7128, -74.0060)
builder.Send()

// Venue
builder := c.ReplyWithVenue(40.7128, -74.0060, "Empire State Building", "350 5th Ave, New York")
builder.Send()

// Contact
builder := c.ReplyWithContact("+1234567890", "John", "Doe")
builder.Send()

// Poll
options := []string{"Option 1", "Option 2", "Option 3"}
builder := c.ReplyWithPoll("What's your favorite color?", options, true, "regular")
builder.Send()

// Quiz
options := []string{"Red", "Green", "Blue"}
builder := c.ReplyWithQuiz("What color is the sky?", options, 1) // Blue is the correct answer
builder.Send()
```

## Context

Context provides rich helper methods:

### Parameter Retrieval

```go
// Path parameters
userID := c.Param("id")

// Query parameters
page := c.QueryInt("page", 1)      // Integer parameter with default
sort := c.Query("sort", "id")      // String parameter with default
active := c.QueryBool("active")    // Boolean parameter
```

### Request Control

```go
// Abort processing
c.Abort()

// Check if aborted
if c.IsAborted() {
    return
}

// Continue processing
c.Next()
```

### Message Reply

```go
// Text message
c.Reply("Hello, World!")

// Formatted message
c.Reply("*Bold* and _italic_").WithParseMode("Markdown")

// Message with keyboard
keyboard := tgbotapi.NewReplyKeyboard(
    tgbotapi.NewKeyboardButtonRow(
        tgbotapi.NewKeyboardButton("Button 1"),
        tgbotapi.NewKeyboardButton("Button 2"),
    ),
)
c.Reply("Please select:").WithReplyMarkup(keyboard)
```

## Middleware Chain

Middleware can be chained and supports pre/post request processing:

```go
// Logger middleware
func Logger(c *router.Context) {
    start := time.Now()
    c.Next()
    log.Printf("Request processed in %v", time.Since(start))
}

// Auth middleware
func Auth(allowedUsers []int64) router.HandlerFunc {
    return func(c *router.Context) {
        userID := c.Message.From.ID
        for _, id := range allowedUsers {
            if id == userID {
                c.Next()
                return
            }
        }
        c.Reply("Unauthorized access")
    }
}

// Use middleware
r.Use(Logger, Auth([]int64{123456789}))
```

## Location Routing

Support for location-based routing:

```go
// Handle location in specific range
r.LocationInRange(40.7, 40.8, -74.0, -73.9, func(c *router.Context) {
    c.Reply("You are in Manhattan, New York")
})

// Handle all locations
r.Location(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Your location: %f, %f", 
        c.Message.Location.Latitude,
        c.Message.Location.Longitude))
})
```

## File Type Filtering

Support for file type and size filtering:

```go
// Handle specific document type
r.DocumentWithType("application/pdf", 10*1024*1024, func(c *router.Context) {
    c.Reply("Received PDF file, size under 10MB")
})

// Handle all documents
r.Document(func(c *router.Context) {
    c.Reply(fmt.Sprintf("Received file: %s", c.Message.Document.FileName))
})
```

## Poll Handling

Support for different types of polls and quizzes:

```go
// Handle quiz
r.Quiz(func(c *router.Context) {
    c.Reply("Received quiz answer")
})

// Handle regular poll
r.RegularPoll(func(c *router.Context) {
    c.Reply("Received poll")
})

// Handle specific poll type
r.PollWithType("regular", 10, true, false, func(c *router.Context) {
    c.Reply("Received anonymous poll, minimum 10 votes, single choice")
})
``` 