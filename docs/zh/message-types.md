# 消息类型

Telegram Router 支持处理各种类型的 Telegram 消息。本文档描述了如何处理不同类型的消息，并提供了示例。

## 文本消息

### 基本文本处理

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

### 命令消息

```go
// 处理 /start 命令
r.Command("start", func(c *router.Context) {
    c.Reply("欢迎使用机器人！")
})

// 处理带参数的命令
r.Command("echo", func(c *router.Context) {
    // 获取命令参数
    args := c.Args()
    if len(args) > 0 {
        c.Reply(strings.Join(args, " "))
    } else {
        c.Reply("请提供要回显的文本")
    }
})

// 处理带模式匹配的命令
r.CommandMatch("^user_", func(c *router.Context) {
    // 从命令中提取用户 ID
    userID := strings.TrimPrefix(c.Message.Command(), "user_")
    c.Reply("用户 ID：" + userID)
})
```

## 媒体消息

### 图片

```go
// 处理所有图片
r.Photo(func(c *router.Context) {
    // 获取最大尺寸的图片
    photo := c.Message.Photo[len(c.Message.Photo)-1]
    c.Reply(fmt.Sprintf("收到图片：%s", photo.FileID))
})

// 处理带说明的图片
r.PhotoWithCaption(func(c *router.Context) {
    c.Reply("图片说明：" + c.Message.Caption)
})

// 发送图片
builder := c.ReplyWithPhotoFilePath("image.jpg")
builder.WithCaption("图片说明")
builder.WithParseMode("Markdown")
builder.Send()
```

### 文档

```go
// 处理所有文档
r.Document(func(c *router.Context) {
    c.Reply(fmt.Sprintf("收到文档：%s", c.Message.Document.FileName))
})

// 处理特定类型的文档
r.DocumentWithType("application/pdf", 10*1024*1024, func(c *router.Context) {
    c.Reply("收到 PDF 文件，大小不超过 10MB")
})

// 发送文档
builder := c.ReplyWithDocumentFileURL("https://example.com/doc.pdf", "文档说明")
builder.Send()
```

### 音频

```go
// 处理所有音频
r.Audio(func(c *router.Context) {
    c.Reply(fmt.Sprintf("收到音频：%s", c.Message.Audio.Title))
})

// 处理特定时长的音频
r.AudioWithDuration(180, func(c *router.Context) {
    c.Reply("收到不超过 3 分钟的音频")
})

// 发送音频
builder := c.ReplyWithAudioFilePath("audio.mp3", "音频说明")
builder.WithTitle("歌曲标题")
builder.WithPerformer("艺术家")
builder.WithDuration(180)
builder.Send()
```

### 视频

```go
// 处理所有视频
r.Video(func(c *router.Context) {
    c.Reply(fmt.Sprintf("收到视频：%s", c.Message.Video.FileName))
})

// 处理特定时长的视频
r.VideoWithDuration(60, func(c *router.Context) {
    c.Reply("收到不超过 1 分钟的视频")
})

// 发送视频
builder := c.ReplyWithVideoFilePath("video.mp4", "视频说明")
builder.WithDuration(60)
builder.WithSupportsStreaming(true)
builder.Send()
```

### 语音消息

```go
// 处理所有语音消息
r.Voice(func(c *router.Context) {
    c.Reply("收到语音消息")
})

// 处理特定时长的语音
r.VoiceWithDuration(30, func(c *router.Context) {
    c.Reply("收到不超过 30 秒的语音消息")
})

// 发送语音
builder := c.ReplyWithVoiceFilePath("voice.ogg")
builder.Send()
```

## 特殊消息

### 位置

```go
// 处理所有位置
r.Location(func(c *router.Context) {
    c.Reply(fmt.Sprintf("位置：%f, %f", 
        c.Message.Location.Latitude,
        c.Message.Location.Longitude))
})

// 处理特定范围内的位置
r.LocationInRange(40.7, 40.8, -74.0, -73.9, func(c *router.Context) {
    c.Reply("您位于纽约曼哈顿区域")
})

// 发送位置
builder := c.ReplyWithLocation(40.7128, -74.0060)
builder.Send()
```

### 地点

```go
// 处理所有地点
r.Venue(func(c *router.Context) {
    c.Reply(fmt.Sprintf("地点：%s, %s", 
        c.Message.Venue.Title,
        c.Message.Venue.Address))
})

// 发送地点
builder := c.ReplyWithVenue(40.7128, -74.0060, 
    "帝国大厦", 
    "纽约第五大道 350 号")
builder.Send()
```

### 联系人

```go
// 处理所有联系人
r.Contact(func(c *router.Context) {
    c.Reply(fmt.Sprintf("联系人：%s %s", 
        c.Message.Contact.FirstName,
        c.Message.Contact.LastName))
})

// 发送联系人
builder := c.ReplyWithContact("+1234567890", "张三", "李")
builder.Send()
```

### 投票和测验

```go
// 处理所有投票
r.Poll(func(c *router.Context) {
    c.Reply("收到投票：" + c.Message.Poll.Question)
})

// 处理测验
r.Quiz(func(c *router.Context) {
    c.Reply("收到测验答案")
})

// 处理普通投票
r.RegularPoll(func(c *router.Context) {
    c.Reply("收到普通投票")
})

// 处理特定类型的投票
r.PollWithType("regular", 10, true, false, func(c *router.Context) {
    c.Reply("收到匿名投票，最少 10 票，单选")
})

// 发送投票
options := []string{"选项 1", "选项 2", "选项 3"}
builder := c.ReplyWithPoll("你最喜欢什么颜色？", options, true, "regular")
builder.Send()

// 发送测验
options := []string{"红色", "绿色", "蓝色"}
builder := c.ReplyWithQuiz("天空是什么颜色？", options, 1) // 蓝色是正确答案
builder.Send()
```

## 频道消息

```go
// 处理所有频道消息
r.ChannelPost(func(c *router.Context) {
    c.Reply("收到频道消息")
})

// 处理特定类型的频道消息
r.ChannelPostWithType("text", func(c *router.Context) {
    c.Reply("收到文本频道消息")
})
```

## 消息构建器

所有消息类型都支持消息构建器模式：

```go
// 通用构建器方法
builder := c.Reply("消息")
builder.WithParseMode("Markdown")           // 设置解析模式
builder.WithReplyMarkup(keyboard)          // 设置键盘
builder.WithReplyToMessageID(123)          // 回复特定消息
builder.WithAllowSendingWithoutReply(true) // 允许无回复发送
builder.WithProtectContent(true)           // 保护内容
builder.Send()                             // 发送消息
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

## 最佳实践

1. 始终检查上下文是否有可回复的消息：
   ```go
   if ctx.Message == nil {
       return
   }
   ```

2. 使用适当的文件类型和大小：
   - 图片：JPG、PNG
   - 文档：任意文件类型
   - 音频：MP3、M4A
   - 视频：MP4
   - 语音：OGG
   - 视频笔记：MP4（正方形）
   - 贴纸：WEBP
   - 动画：GIF、MP4

3. 设置适当的说明文字和解析模式：
   ```go
   builder.WithCaption("带*粗体*文本的说明").WithParseMode("Markdown")
   ```

4. 对大视频使用流式传输：
   ```go
   builder.WithSupportsStreaming(true)
   ```

5. 为媒体设置适当的时长：
   ```go
   builder.WithDuration(30) // 30 秒
   ```

## 基本消息类型

### 照片消息
```go
router.Photo(func(c *Context) {
    photo := c.Message.Photo[len(c.Message.Photo)-1]
    c.Reply(fmt.Sprintf("收到照片：%dx%d", photo.Width, photo.Height))
})
```

### 贴纸消息
```go
router.Sticker(func(c *Context) {
    sticker := c.Message.Sticker
    c.Reply(fmt.Sprintf("收到贴纸：%s", sticker.Emoji))
})
```

## 特殊消息类型

### 位置范围消息
```go
router.LocationInRange(40.0, 41.0, -74.0, -73.0, func(c *Context) {
    c.Reply("您位于纽约市范围内")
})
```

### 轮询消息
```go
// 处理所有轮询
router.Poll(func(c *Context) {
    poll := c.Poll
    c.Reply(fmt.Sprintf("轮询：%s", poll.Question))
})

// 处理测验类型轮询
router.Quiz(func(c *Context) {
    c.Reply("收到测验")
})

// 处理普通投票
router.RegularPoll(func(c *Context) {
    c.Reply("收到投票")
})

// 处理特定类型的轮询
router.PollWithType("quiz", 5, true, false, func(c *Context) {
    c.Reply("收到匿名测验，最少5票")
})
```

### 游戏消息
```go
router.Game(func(c *Context) {
    game := c.Message.Game
    c.Reply(fmt.Sprintf("游戏：%s", game.Title))
})
```

### 视频笔记
```go
router.VideoNote(func(c *Context) {
    videoNote := c.Message.VideoNote
    c.Reply(fmt.Sprintf("视频笔记：%d秒", videoNote.Duration))
})
```

### 动画消息
```go
router.Animation(func(c *Context) {
    animation := c.Message.Animation
    c.Reply(fmt.Sprintf("动画：%s", animation.FileName))
})
```

### 实时位置
```go
router.LiveLocation(func(c *Context) {
    loc := c.Message.Location
    c.Reply(fmt.Sprintf("实时位置：%f, %f", loc.Latitude, loc.Longitude))
})
```

### 频道消息
```go
router.ChannelPost(func(c *Context) {
    c.Reply("收到频道消息")
})
```

## 文档类型过滤

可以根据文档的 MIME 类型和大小进行过滤：

```go
router.DocumentWithType("application/pdf", 10*1024*1024, func(c *Context) {
    c.Reply("收到PDF文件，大小不超过10MB")
})
```

## 注意事项

1. 每种消息类型可以注册多个处理函数，它们会按注册顺序依次执行
2. 使用 `c.Abort()` 可以中断处理链
3. 所有处理函数都支持中间件
4. 处理函数可以访问完整的消息上下文 