# Middleware

Telegram Router provides a simple middleware system that makes it easy to implement request preprocessing, post-processing, and error handling.

## Basic Concepts

Middleware is just a regular handler function that can control the execution flow using `c.Next()`. Middleware can:

1. Execute code before `c.Next()` (pre-processing)
2. Execute code after `c.Next()` (post-processing)
3. Control whether to continue execution by calling `c.Next()`
4. Handle errors and exceptions

## Creating Middleware

### Basic Middleware

```go
// Logger middleware
func Logger(c *router.Context) {
    // Before processing
    start := time.Now()
    log.Printf("Processing message: %s", c.Message.Text)

    // Execute next handler
    c.Next()

    // After processing
    log.Printf("Processing completed, time taken: %v", time.Since(start))
}

// Using middleware
r.Use(Logger)
```

### Middleware with Parameters

```go
// Authentication middleware
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

// Using middleware with parameters
r.Use(Auth([]int64{123456789}))
```

### Error Handling Middleware

```go
// Recovery middleware
func Recovery(c *router.Context) {
    defer func() {
        if err := recover(); err != nil {
            log.Printf("Error occurred: %v", err)
            c.Reply("Sorry, an error occurred while processing your request")
        }
    }()
    c.Next()
}

// Using error handling middleware
r.Use(Recovery)
```

## Middleware Chain

You can combine multiple middleware:

```go
// Create middleware chain
r.Use(
    Logger,
    Recovery,
    Auth([]int64{123456789}),
)

// Or use Group to create middleware groups
r.Group(func(r *router.Router) {
    r.Use(Auth([]int64{123456789}))
    r.Command("admin", func(c *router.Context) {
        c.Reply("Admin command")
    })
})
```

## Common Middleware Examples

### Rate Limiting Middleware

```go
// Rate limiting middleware
func RateLimit(limit int, window time.Duration) router.HandlerFunc {
    // Using token bucket algorithm
    limiter := rate.NewLimiter(rate.Every(window/time.Duration(limit)), limit)
    
    return func(c *router.Context) {
        if !limiter.Allow() {
            c.Reply("Too many requests, please try again later")
            return
        }
        c.Next()
    }
}

// Using rate limiting middleware
r.Use(RateLimit(10, time.Minute)) // Maximum 10 messages per minute
```

### Blacklist Middleware

```go
// Blacklist middleware
func Blacklist(blockedUsers []int64) router.HandlerFunc {
    return func(c *router.Context) {
        userID := c.Message.From.ID
        for _, id := range blockedUsers {
            if id == userID {
                c.Reply("You are banned from using this bot")
                return
            }
        }
        c.Next()
    }
}

// Using blacklist middleware
r.Use(Blacklist([]int64{987654321}))
```

### Message Filter Middleware

```go
// Message filter middleware
func MessageFilter(filters ...func(*tgbotapi.Message) bool) router.HandlerFunc {
    return func(c *router.Context) {
        for _, filter := range filters {
            if !filter(c.Message) {
                c.Reply("Message does not meet requirements")
                return
            }
        }
        c.Next()
    }
}

// Using message filter middleware
r.Use(MessageFilter(
    func(m *tgbotapi.Message) bool { return len(m.Text) > 0 },
    func(m *tgbotapi.Message) bool { return !strings.Contains(m.Text, "advertisement") },
))
```

### Statistics Middleware

```go
// Statistics middleware
func Stats() router.HandlerFunc {
    var (
        totalMessages int64
        startTime     = time.Now()
    )

    return func(c *router.Context) {
        atomic.AddInt64(&totalMessages, 1)
        c.Next()

        // Output statistics every hour
        if time.Since(startTime) > time.Hour {
            log.Printf("Statistics: Total messages %d, Running time %v",
                atomic.LoadInt64(&totalMessages),
                time.Since(startTime))
        }
    }
}

// Using statistics middleware
r.Use(Stats())
```

## Middleware Best Practices

1. Keep middleware simple
   ```go
   // Good practice: Each middleware does one thing
   r.Use(Logger)
   r.Use(Auth([]int64{123456789}))
   
   // Avoid: One middleware doing multiple things
   r.Use(LoggerAndAuth([]int64{123456789}))
   ```

2. Handle errors properly
   ```go
   // Good practice: Use defer and recover
   defer func() {
       if err := recover(); err != nil {
           log.Printf("Error: %v", err)
           c.Reply("An error occurred")
       }
   }()
   c.Next()
   
   // Avoid: Ignoring errors
   c.Next() // May panic
   ```

3. Pay attention to middleware order
   ```go
   // Good practice: Handle errors first, then business logic
   r.Use(Recovery)  // Error handling
   r.Use(Logger)    // Logging
   r.Use(Auth(...)) // Authentication
   
   // Avoid: Error handling at the end
   r.Use(Logger)
   r.Use(Auth(...))
   r.Use(Recovery)  // May be too late to handle errors
   ```

4. Use middleware groups
   ```go
   // Good practice: Use middleware groups to manage related routes
   r.Group(func(r *router.Router) {
       r.Use(Auth([]int64{123456789}))
       r.Command("admin", func(c *router.Context) {
           c.Reply("Admin commands")
       })
   })
   ``` 