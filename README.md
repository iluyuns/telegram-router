# tgr - Telegram Router

一个类似 Gin 风格的 Telegram 机器人路由系统，支持命令、文本、媒体、回调查询等多种消息类型的路由处理，并提供中间件支持。

## 为什么选择 tgr？

### 🎯 相比原生 tgbotapi 的优势
- **更简洁的 API**：告别繁琐的 if-else 判断，使用声明式路由注册
- **中间件支持**：统一的日志、鉴权、错误处理等横切关注点
- **类型安全**：编译时检查路由参数，减少运行时错误
- **更好的可维护性**：清晰的路由结构，易于扩展和调试

### 🚀 相比其他框架的优势
- **零依赖**：只依赖官方 tgbotapi，无额外第三方库
- **高性能**：基于 worker 池和缓冲队列的并发处理
- **生产就绪**：内置优雅关闭、错误恢复、Webhook 支持
- **灵活部署**：支持长轮询和 Webhook 两种模式

## 核心特性

- 🚀 **类 Gin 风格的 API 设计** - 熟悉 Web 开发的开发者零学习成本
- 📱 **全消息类型支持** - 文本、图片、视频、音频、文档、位置、联系人等
- 🔄 **强大的中间件系统** - 支持全局和局部中间件，链式调用
- 🎯 **灵活的路由匹配** - 命令、正则、回调路由、参数提取
- 🌐 **双模式支持** - Webhook 和长轮询，适应不同部署场景
- 🛡️ **企业级特性** - 错误恢复、优雅关闭、并发控制
- 📝 **丰富的构建器 API** - 链式调用，类型安全的消息构建
- ⚡ **高性能架构** - Worker 池 + 缓冲队列，支持高并发

## 快速开始

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/iluyuns/tgr"
)

func main() {
    bot, err := tgbotapi.NewBotAPI("YOUR_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    router := tgr.NewTelegramRouterWithDefaultRecover(bot)

    router.Command("start", func(c *tgr.Context) {
        c.Reply("Hello!").Send()
    })

    router.Text(func(c *tgr.Context) {
        c.Reply("Received: " + c.Message.Text).Send()
    })

    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()
    router.ListenWithContext(ctx, 8, 1024)
}
```

## 文档

- 📖 [中文文档](doc.md)
- 📖 [English Documentation](doc_en.md)

## 安装

```bash
go get github.com/iluyuns/tgr
```

## 许可证

MIT License

## 支持

如果这个项目对你有帮助，欢迎支持：

<div align="center">
  <img src="docs/btc.jpeg" alt="BTC Support" width="200" />
  <img src="docs/wechat.jpg" alt="WeChat Support" width="200" />
</div>
