# 基本使用

本文档介绍了 Telegram Router 的基本使用方法，包括创建机器人、处理消息和发送回复等。

## 创建机器人

### 初始化

```go
package main

import (
    "log"
    "github.com/yourusername/telegram-router"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
    // 创建机器人实例
    bot, err := tgbotapi.NewBotAPI("YOUR_BOT_TOKEN")
    if err != nil {
        log.Fatal(err)
    }

    // 创建路由器
    r := router.New(bot)

    // 启动机器人
    r.Start()
}
```

### 配置选项

```go
// 使用自定义配置
config := &router.PollingConfig{
    Timeout: 60,           // 超时时间（秒）
    DropPendingUpdates: true, // 启动时丢弃待处理更新
}
r.StartWithConfig(config)

// 使用 webhook 模式
webhook := &router.WebhookConfig{
    ListenAddr: ":8443",
    CertFile:   "cert.pem",
    KeyFile:    "key.pem",
    WebhookURL: "https://your-domain.com/bot",
}
r.StartWebhook(webhook)
```

## 处理消息

### 命令处理

```go
// 处理 /start 命令
r.Command("start", func(c *router.Context) {
    c.Reply("欢迎使用机器人！")
})

// 处理带参数的命令
r.Command("echo", func(c *router.Context) {
    args := c.Args()
    if len(args) > 0 {
        c.Reply(strings.Join(args, " "))
    } else {
        c.Reply("请提供要回显的文本")
    }
})

// 使用正则表达式匹配命令
r.CommandRegex(regexp.MustCompile(`^user_\d+$`), func(c *router.Context) {
    userID := strings.TrimPrefix(c.Message.Command(), "user_")
    c.Reply("用户 ID：" + userID)
})
```

### 文本消息处理

```go
// 处理所有文本消息
r.Text(func(c *router.Context) {
    c.Reply("收到消息：" + c.Message.Text)
})

// 处理特定文本模式
r.TextMatch("^你好", func(c *router.Context) {
    c.Reply("你好！")
})

// 使用正则表达式处理文本
r.TextRegex(regexp.MustCompile(`(?i)帮助`), func(c *router.Context) {
    c.Reply("需要什么帮助？")
})
```

### 媒体消息处理

```go
// 处理图片
r.Photo(func(c *router.Context) {
    photo := c.Message.Photo[len(c.Message.Photo)-1]
    c.Reply(fmt.Sprintf("收到图片：%s", photo.FileID))
})

// 处理文档
r.Document(func(c *router.Context) {
    c.Reply(fmt.Sprintf("收到文档：%s", c.Message.Document.FileName))
})

// 处理音频
r.Audio(func(c *router.Context) {
    c.Reply(fmt.Sprintf("收到音频：%s", c.Message.Audio.Title))
})

// 处理视频
r.Video(func(c *router.Context) {
    c.Reply(fmt.Sprintf("收到视频：%s", c.Message.Video.FileName))
})
```

### 特殊消息处理

```go
// 处理位置
r.Location(func(c *router.Context) {
    c.Reply(fmt.Sprintf("位置：%f, %f",
        c.Message.Location.Latitude,
        c.Message.Location.Longitude))
})

// 处理地点
r.Venue(func(c *router.Context) {
    c.Reply(fmt.Sprintf("地点：%s, %s",
        c.Message.Venue.Title,
        c.Message.Venue.Address))
})

// 处理联系人
r.Contact(func(c *router.Context) {
    c.Reply(fmt.Sprintf("联系人：%s %s",
        c.Message.Contact.FirstName,
        c.Message.Contact.LastName))
})

// 处理投票
r.Poll(func(c *router.Context) {
    c.Reply("收到投票：" + c.Message.Poll.Question)
})
```

## 发送消息

### 基本回复

```go
// 发送文本消息
c.Reply("你好！")

// 使用解析模式
c.Reply("**粗体** _斜体_", "Markdown")

// 回复特定消息
c.ReplyTo(c.Message.MessageID, "回复消息")
```

### 媒体消息

```go
// 发送图片
builder := c.ReplyWithPhotoFilePath("image.jpg")
builder.WithCaption("图片说明")
builder.Send()

// 发送文档
builder := c.ReplyWithDocumentFileURL("https://example.com/doc.pdf")
builder.WithCaption("文档说明")
builder.Send()

// 发送音频
builder := c.ReplyWithAudioFilePath("audio.mp3")
builder.WithTitle("歌曲标题")
builder.WithPerformer("艺术家")
builder.Send()

// 发送视频
builder := c.ReplyWithVideoFilePath("video.mp4")
builder.WithCaption("视频说明")
builder.WithDuration(60)
builder.Send()
```

### 特殊消息

```go
// 发送位置
builder := c.ReplyWithLocation(40.7128, -74.0060)
builder.Send()

// 发送地点
builder := c.ReplyWithVenue(40.7128, -74.0060,
    "帝国大厦",
    "纽约第五大道 350 号")
builder.Send()

// 发送联系人
builder := c.ReplyWithContact("+1234567890", "张三", "李")
builder.Send()

// 发送投票
options := []string{"选项 1", "选项 2", "选项 3"}
builder := c.ReplyWithPoll("你最喜欢什么颜色？", options, true, "regular")
builder.Send()

// 发送测验
options := []string{"红色", "绿色", "蓝色"}
builder := c.ReplyWithQuiz("天空是什么颜色？", options, 1)
builder.Send()
```

## 键盘和按钮

### 回复键盘

```go
// 创建回复键盘
keyboard := tgbotapi.NewReplyKeyboard(
    tgbotapi.NewKeyboardButtonRow(
        tgbotapi.NewKeyboardButton("按钮 1"),
        tgbotapi.NewKeyboardButton("按钮 2"),
    ),
    tgbotapi.NewKeyboardButtonRow(
        tgbotapi.NewKeyboardButtonLocation("发送位置"),
        tgbotapi.NewKeyboardButtonContact("分享联系人"),
    ),
)

// 发送带键盘的消息
builder := c.Reply("请选择一个选项：")
builder.WithReplyMarkup(keyboard)
builder.Send()
```

### 内联键盘

```go
// 创建内联键盘
keyboard := tgbotapi.NewInlineKeyboardMarkup(
    tgbotapi.NewInlineKeyboardRow(
        tgbotapi.NewInlineKeyboardButtonData("选项 1", "choice:1"),
        tgbotapi.NewInlineKeyboardButtonData("选项 2", "choice:2"),
    ),
    tgbotapi.NewInlineKeyboardRow(
        tgbotapi.NewInlineKeyboardButtonURL("访问网站", "https://example.com"),
    ),
)

// 发送带内联键盘的消息
builder := c.Reply("请选择一个选项：")
builder.WithInlineKeyboard(keyboard)
builder.Send()
```

## 错误处理

```go
// 处理消息发送错误
builder := c.Reply("消息")
if _, err := builder.Send(); err != nil {
    log.Printf("发送消息时出错：%v", err)
    // 适当处理错误
}

// 处理特定类型的错误
if err != nil {
    switch {
    case strings.Contains(err.Error(), "bot was blocked"):
        // 处理机器人被屏蔽
    case strings.Contains(err.Error(), "chat not found"):
        // 处理聊天未找到
    default:
        // 处理其他错误
    }
}
```

## 上下文功能

### 参数获取

```go
// 获取命令参数
args := c.Args()

// 获取路径参数
userID := c.Param("id")

// 获取查询参数
actionType := c.Query("type")
```

### 请求控制

```go
// 中断处理
c.Abort()

// 检查是否已中断
if c.IsAborted() {
    return
}

// 设置状态
c.Set("user", user)

// 获取状态
if user, ok := c.Get("user"); ok {
    // 使用用户信息
}
```

### 消息回复

```go
// 回复消息
c.Reply("回复消息")

// 编辑消息
c.Edit("编辑后的消息")

// 删除消息
c.Delete()

// 回复回调查询
c.Answer("处理中...", true)
```

## 最佳实践

1. 使用中间件处理通用逻辑
   ```go
   r.Use(Logger())
   r.Use(Recovery())
   r.Use(Auth([]int64{123456789}))
   ```

2. 使用路由组组织代码
   ```go
   r.Group(func(r *router.Router) {
       r.Use(AdminAuth())
       r.Command("admin", ...)
       r.Command("ban", ...)
   })
   ```

3. 正确处理错误
   ```go
   defer func() {
       if err := recover(); err != nil {
           log.Printf("错误：%v", err)
           c.Reply("发生错误")
       }
   }()
   ```

4. 使用环境变量配置
   ```go
   token := os.Getenv("TELEGRAM_BOT_TOKEN")
   if token == "" {
       log.Fatal("未设置 TELEGRAM_BOT_TOKEN 环境变量")
   }
   ```

5. 添加适当的日志记录
   ```go
   log.Printf("处理消息：%s", c.Message.Text)
   log.Printf("用户：%d", c.Message.From.ID)
   ```

## 下一步

- 学习[中间件](middleware.md)的使用方法
- 了解[高级特性](advanced-features.md)
- 查看[示例代码](examples.md)获取更多灵感 