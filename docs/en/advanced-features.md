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