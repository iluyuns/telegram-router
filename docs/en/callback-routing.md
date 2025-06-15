# Callback Routing

Telegram Router provides powerful callback query routing capabilities, similar to web routing systems. This document explains how to handle callback queries effectively.

## Basic Callback Handling

```go
// Handle simple callback
r.Callback("menu/main", func(c *router.Context) {
    c.AnswerCallback("Main menu selected")
    c.Reply("Main menu")
})

// Handle callback with answer
r.Callback("action/confirm", func(c *router.Context) {
    c.AnswerCallback("Action confirmed", true) // Show alert
    c.Reply("Action has been confirmed")
})
```

## Path Parameters

Support for dynamic path parameters in callback data:

```go
// Handle callback with path parameter
r.Callback("user/:id/profile", func(c *router.Context) {
    userID := c.Param("id")
    c.AnswerCallback(fmt.Sprintf("Viewing profile of user %s", userID))
    c.Reply(fmt.Sprintf("Profile of user %s", userID))
})

// Handle multiple parameters
r.Callback("product/:category/:id", func(c *router.Context) {
    category := c.Param("category")
    productID := c.Param("id")
    c.AnswerCallback(fmt.Sprintf("Viewing %s product %s", category, productID))
    c.Reply(fmt.Sprintf("Product details: %s/%s", category, productID))
})
```

## Query Parameters

Support for URL-style query parameters:

```go
// Handle callback with query parameters
r.Callback("products/list", func(c *router.Context) {
    page := c.QueryInt("page", 1)      // Integer parameter with default
    sort := c.Query("sort", "id")      // String parameter with default
    active := c.QueryBool("active")    // Boolean parameter
    
    c.AnswerCallback(fmt.Sprintf("Page %d, sorted by %s", page, sort))
    c.Reply(fmt.Sprintf("Products page %d, sorted by %s", page, sort))
})

// Handle multiple query parameters
r.Callback("search", func(c *router.Context) {
    query := c.Query("q", "")
    category := c.Query("category", "all")
    minPrice := c.QueryFloat("min_price", 0)
    maxPrice := c.QueryFloat("max_price", 1000)
    
    c.AnswerCallback(fmt.Sprintf("Searching: %s in %s", query, category))
    c.Reply(fmt.Sprintf("Search results for %s in %s, price range: $%.2f - $%.2f",
        query, category, minPrice, maxPrice))
})
```

## Pattern Matching

Support for pattern matching in callback data:

```go
// Handle callback with pattern
r.CallbackMatch("^menu/", func(c *router.Context) {
    c.AnswerCallback("Menu option selected")
    c.Reply("Menu option: " + c.CallbackQuery.Data)
})

// Handle callback with regex
r.CallbackRegex(regexp.MustCompile(`^user/\d+/profile$`), func(c *router.Context) {
    c.AnswerCallback("User profile selected")
    c.Reply("User profile: " + c.CallbackQuery.Data)
})
```

## Inline Keyboard Integration

Example of creating and handling inline keyboard callbacks:

```go
// Create menu with inline keyboard
r.Command("menu", func(c *router.Context) {
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("Profile", "menu/profile"),
            tgbotapi.NewInlineKeyboardButtonData("Settings", "menu/settings"),
        ),
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("Products", "products/list?page=1&sort=name"),
            tgbotapi.NewInlineKeyboardButtonData("Cart", "cart/view"),
        ),
    )

    builder := c.Reply("Please select an option:")
    builder.WithReplyMarkup(keyboard)
    if _, err := builder.Send(); err != nil {
        log.Printf("Error sending message: %v", err)
    }
})

// Handle menu callbacks
r.Callback("menu/profile", func(c *router.Context) {
    c.AnswerCallback("Profile selected")
    c.Reply("Profile menu")
})

r.Callback("menu/settings", func(c *router.Context) {
    c.AnswerCallback("Settings selected")
    c.Reply("Settings menu")
})

r.Callback("products/list", func(c *router.Context) {
    page := c.QueryInt("page", 1)
    sort := c.Query("sort", "name")
    c.AnswerCallback(fmt.Sprintf("Viewing products page %d", page))
    c.Reply(fmt.Sprintf("Products page %d, sorted by %s", page, sort))
})

r.Callback("cart/view", func(c *router.Context) {
    c.AnswerCallback("Viewing cart")
    c.Reply("Your shopping cart")
})
```

## Error Handling

```go
// Handle callback errors
r.Callback("action/delete", func(c *router.Context) {
    if err := performDelete(); err != nil {
        c.AnswerCallback("Error: " + err.Error(), true) // Show error in alert
        return
    }
    c.AnswerCallback("Item deleted")
    c.Reply("Item has been deleted")
})

// Handle invalid callbacks
r.Callback("invalid", func(c *router.Context) {
    c.AnswerCallback("Invalid action", true) // Show alert
    c.Reply("This action is not available")
})
```

## Best Practices

1. Use meaningful callback data:
   ```go
   // Good
   "user/123/profile"
   "products/list?page=1&sort=name"
   
   // Bad
   "a1"
   "b2"
   ```

2. Keep callback data short (Telegram limit is 64 bytes):
   ```go
   // Good
   "user/123/profile"
   
   // Bad
   "user/123456789/profile/view/details/settings"
   ```

3. Use appropriate answer types:
   ```go
   // For simple confirmations
   c.AnswerCallback("Action completed")
   
   // For errors or important messages
   c.AnswerCallback("Error: Invalid action", true)
   ```

4. Handle callback timeouts:
   ```go
   r.Callback("action/long", func(c *router.Context) {
       // Answer callback immediately
       c.AnswerCallback("Processing...")
       
       // Perform long operation
       go func() {
           // Long operation here
           c.Reply("Operation completed")
       }()
   })
   ```

5. Use path parameters for dynamic data:
   ```go
   // Good
   r.Callback("user/:id/profile", func(c *router.Context) {
       userID := c.Param("id")
       // ...
   })
   
   // Bad
   r.Callback("user_profile", func(c *router.Context) {
       userID := extractUserID(c.CallbackQuery.Data)
       // ...
   })
   ```

6. Use query parameters for optional data:
   ```go
   // Good
   r.Callback("products/list", func(c *router.Context) {
       page := c.QueryInt("page", 1)
       sort := c.Query("sort", "name")
       // ...
   })
   
   // Bad
   r.Callback("products_list_1_name", func(c *router.Context) {
       // Parse parameters from string
       // ...
   })
   ``` 