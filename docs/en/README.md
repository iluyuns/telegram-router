# Telegram Router

A Gin-style router for Telegram Bot API, providing middleware support and flexible message handling.

## Features

- 🚀 Gin-style routing system
- 🔌 Middleware support with chain calls
- 📝 Multiple message type handlers
- 🔄 Context-based request handling
- ⛓️ Middleware chain execution
- 🛡️ Request abortion support
- 🎯 Path parameter support
- 🔍 Query parameter support
- 📊 Poll and quiz handling
- 📍 Location-based routing
- 📁 File type filtering
- 🌐 Webhook support with HTTP framework integration

## Dependencies

This project is built on top of the following open-source libraries:

- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) - Telegram Bot API wrapper for Go
  - License: MIT License
  - Version: v5.x

## Quick Links

- [Installation](installation.md)
- [Basic Usage](basic-usage.md)
- [Middleware](middleware.md)
- [Message Types](message-types.md)
- [Callback Routing](callback-routing.md)
- [Advanced Features](advanced-features.md)
- [Examples](examples.md)

## Features in Detail

### Routing System
- Command routing with pattern matching
- Text message routing
- Callback query routing
- Location-based routing
- File type filtering
- Poll and quiz handling

### Middleware Support
- Chain-style middleware execution
- Request abortion support
- Context-based middleware
- Custom middleware creation
- Built-in middleware (logging, recovery, etc.)

### Message Handling
- Rich message builder API
- Support for all Telegram message types
- File upload handling
- Media message support
- Keyboard and inline keyboard support

### Webhook Support
- Both Long Polling and Webhook modes
- HTTP framework integration (Gin, Echo, net/http)
- SSL/TLS support
- Webhook management
- Production-ready configuration

### Context Features
- Path parameter support
- Query parameter support
- Message builder methods
- Request control methods
- Error handling

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.

### Third-party Licenses

- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api/blob/master/LICENSE) - MIT License
- [Gin](https://github.com/gin-gonic/gin/blob/master/LICENSE) - MIT License (for inspiration and design patterns)

## Support the Project

If you find this project helpful, you can support it via the following ways:

**Bitcoin (BTC) Donation:**

<img src="../btc.jpeg" alt="BTC Donation QR" width="220" />

*Only send Bitcoin to this address. Other assets may be lost forever.*

*Support via Bitcoin donation (your contribution also helps support charitable causes).*

**WeChat Pay:**

<img src="../wechat.jpg" alt="WeChat Pay QR" width="220" />

*Support via WeChat Pay QR code (for mainland China users; your donation also helps support charitable causes).* 