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