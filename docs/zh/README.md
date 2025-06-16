# Telegram Router

一个类似 Gin 风格的 Telegram 机器人路由系统，提供中间件支持和灵活的消息处理。

## 特性

- 🚀 Gin 风格的路由系统
- 🔌 支持链式调用的中间件
- 📝 多种消息类型处理器
- 🔄 基于上下文（Context）的请求处理
- ⛓️ 中间件链式执行
- 🛡️ 支持请求中断
- 🎯 支持路径参数
- 🔍 支持查询参数
- 📊 投票和测验处理
- 📍 基于位置的路由
- 📁 文件类型过滤
- 🌐 支持 Webhook 和长轮询
- 🔗 支持多种 HTTP 框架集成

## 依赖

本项目基于以下开源库构建：

- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) - Go 语言的 Telegram Bot API 封装
  - 许可证：MIT License
  - 版本：v5.x

## 快速链接

- [安装指南](installation.md)
- [基础用法](basic-usage.md)
- [中间件](middleware.md)
- [消息类型](message-types.md)
- [回调路由](callback-routing.md)
- [高级特性](advanced-features.md)
- [示例代码](examples.md)

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](../LICENSE) 文件。

### 第三方许可证

- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api/blob/master/LICENSE) - MIT License
- [Gin](https://github.com/gin-gonic/gin/blob/master/LICENSE) - MIT License (用于设计模式参考)

## 支持项目

如果您觉得这个项目有帮助，可以通过以下方式支持：

**比特币 (BTC) 捐赠：**

<img src="../btc.jpeg" alt="BTC 捐赠二维码" width="220" />

*请仅发送比特币到此地址。其他资产可能会永久丢失。*

*欢迎通过比特币赞赏扶贫，感谢您的支持！*
*Support via Bitcoin donation (your contribution also helps support charitable causes).*

**微信支付 / WeChat Pay：**

<img src="../wechat.jpg" alt="微信支付二维码 WeChat Pay QR" width="220" />

*欢迎通过微信扫码赞赏扶贫，感谢您的支持！*
*Support via WeChat Pay QR code (for mainland China users; your donation also helps support charitable causes).*

## 消息类型

### 通用更新处理器

`OnUpdate` 处理器是一个强大的功能，允许你在一个地方处理所有类型的 Telegram 更新。它在任何特定类型处理器之前执行，并且可以控制消息处理的流程。

> ⚠️ 重要提示：对于全局性的处理逻辑（如日志记录、权限检查等），建议优先使用中间件（Middleware）而不是 `OnUpdate` 处理器。中间件更适合处理横切关注点，而 `OnUpdate` 更适合处理需要直接控制消息流程的场景。

```go
// 推荐：使用中间件处理全局逻辑
router.Use(func(c *Context) {
    // 记录请求日志
    startTime := time.Now()
    logger.Info("收到请求: %s", c.Update.Type())
    
    c.Next()  // 继续处理
    
    // 记录处理时间
    elapsed := time.Since(startTime)
    logger.Info("处理完成，耗时: %s", elapsed)
})

// 使用 OnUpdate 处理需要控制流程的场景
router.OnUpdate(func(c *Context) {
    // 处理需要直接控制流程的场景
    if someSpecialCondition {
        c.Abort()  // 中断处理
        return
    }
})
```

通用更新处理器的主要特点：
- 在任何特定类型处理器之前执行
- 可以处理所有类型的更新（消息、事件、回调查询等）
- 可以使用 `Abort()` 中断后续处理
- 支持中间件链
- 适用于：
  - 全局日志记录
  - 权限检查
  - 消息过滤
  - 全局预处理

权限检查示例：
```go
router.OnUpdate(func(c *Context) {
    // 检查用户权限
    if !hasPermission(c.Message.From.ID) {
        c.Reply("您没有权限使用此功能")
        c.Abort()  // 停止后续处理
        return
    }
    // 继续处理
})

// 只有在权限检查通过后，其他处理器才会执行
router.Text(func(c *Context) {
    c.Reply("处理文本消息")
})
```

执行顺序：
1. 通用更新处理器（OnUpdate）
2. 特定类型处理器（如果未被中断）
   - 命令处理器
   - 文本处理器
   - 图片处理器
   - 等等 