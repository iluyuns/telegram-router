# 基本使用

## 消息处理器

### 命令处理器

处理 Telegram 命令消息：

```go
// 处理 /start 命令
router.Command("start", func(c *telegramrouter.Context) {
    c.Reply("欢迎使用机器人！")
})

// 处理带参数的命令
router.Command("echo", func(c *telegramrouter.Context) {
    // 获取命令参数
    args := c.Message.CommandArguments()
    if args == "" {
        c.Reply("请提供要回显的文本")
        return
    }
    c.Reply(args)
})
```

### 文本消息处理器

处理普通文本消息：

```go
// 处理所有文本消息
router.Text(func(c *telegramrouter.Context) {
    c.Reply("收到消息：" + c.Message.Text)
})

// 处理特定文本
router.Text(func(c *telegramrouter.Context) {
    if c.Message.Text == "你好" {
        c.Reply("你好！很高兴见到你！")
    }
})

// 处理多个文本处理器
router.Text(func(c *telegramrouter.Context) {
    // 第一个处理器
    if strings.HasPrefix(c.Message.Text, "天气") {
        c.Reply("正在查询天气...")
    }
}).Text(func(c *telegramrouter.Context) {
    // 第二个处理器
    if strings.HasPrefix(c.Message.Text, "新闻") {
        c.Reply("正在获取新闻...")
    }
})
```

### 媒体消息处理器

#### 文档处理器

```go
// 处理文档消息
router.Document(func(c *telegramrouter.Context) {
    doc := c.Message.Document
    c.Reply(fmt.Sprintf("收到文件：%s\n大小：%d 字节\n类型：%s",
        doc.FileName,
        doc.FileSize,
        doc.MimeType))
})

// 处理特定类型的文档
router.DocumentWithType("application/pdf", 10*1024*1024, func(c *telegramrouter.Context) {
    c.Reply("收到 PDF 文件，大小不超过 10MB")
})
```

#### 图片处理器

```go
// 处理图片消息
router.Photo(func(c *telegramrouter.Context) {
    // 获取最大尺寸的图片
    photo := c.Message.Photo[len(c.Message.Photo)-1]
    c.Reply(fmt.Sprintf("收到图片：%dx%d", photo.Width, photo.Height))
})
```

#### 音频处理器

```go
// 处理音频消息
router.Audio(func(c *telegramrouter.Context) {
    audio := c.Message.Audio
    c.Reply(fmt.Sprintf("收到音频：%s\n时长：%d 秒\n大小：%d 字节",
        audio.Title,
        audio.Duration,
        audio.FileSize))
})
```

#### 视频处理器

```go
// 处理视频消息
router.Video(func(c *telegramrouter.Context) {
    video := c.Message.Video
    c.Reply(fmt.Sprintf("收到视频：%dx%d\n时长：%d 秒\n大小：%d 字节",
        video.Width,
        video.Height,
        video.Duration,
        video.FileSize))
})
```

### 位置消息处理器

```go
// 处理位置消息
router.Location(func(c *telegramrouter.Context) {
    loc := c.Message.Location
    c.Reply(fmt.Sprintf("收到位置：\n纬度：%f\n经度：%f",
        loc.Latitude,
        loc.Longitude))
})

// 处理特定范围内的位置
router.LocationInRange(39.9, 40.1, 116.3, 116.5, func(c *telegramrouter.Context) {
    c.Reply("您在北京天安门附近！")
})
```

### 回调查询处理器

```go
// 处理简单回调
router.Callback("menu/main", func(c *telegramrouter.Context) {
    c.Reply("主菜单")
})

// 处理带参数的回调
router.Callback("user/:id/profile", func(c *telegramrouter.Context) {
    userID := c.Param("id")
    c.Reply(fmt.Sprintf("用户 %s 的个人资料", userID))
})

// 处理带查询参数的回调
router.Callback("products/list", func(c *telegramrouter.Context) {
    page := c.QueryInt("page", 1)
    sort := c.Query("sort", "id")
    c.Reply(fmt.Sprintf("第 %d 页，按 %s 排序", page, sort))
})
```

## 上下文对象

`Context` 对象提供了丰富的辅助方法：

### 消息发送

```go
// 发送文本消息
c.Reply("这是一条回复消息")

// 发送带格式的消息
c.Reply("这是*粗体*和_斜体_文本", "Markdown")

// 发送带按钮的消息
keyboard := tgbotapi.NewInlineKeyboardMarkup(
    tgbotapi.NewInlineKeyboardRow(
        tgbotapi.NewInlineKeyboardButton("选项1", "option1"),
        tgbotapi.NewInlineKeyboardButton("选项2", "option2"),
    ),
)
c.ReplyWithKeyboard("请选择一个选项", keyboard)
```

### 参数获取

```go
// 获取路径参数
userID := c.Param("id")

// 获取查询参数
page := c.QueryInt("page", 1)      // 带默认值的整数参数
sort := c.Query("sort", "id")      // 带默认值的字符串参数
active := c.QueryBool("active")    // 布尔参数

// 获取命令参数
args := c.Message.CommandArguments()
```

### 请求控制

```go
// 中断处理链
c.Abort()

// 检查是否已中断
if c.IsAborted() {
    return
}

// 继续处理链
c.Next()
```

## 最佳实践

### 1. 处理器组织

```go
// 按功能分组处理器
func registerUserHandlers(router *telegramrouter.TelegramRouter) {
    router.Command("start", handleStart)
    router.Command("help", handleHelp)
    router.Text(handleUserMessage)
}

func registerAdminHandlers(router *telegramrouter.TelegramRouter) {
    router.Command("admin", handleAdmin)
    router.Callback("admin/*", handleAdminCallback)
}

// 主函数中注册
func main() {
    router := telegramrouter.NewTelegramRouter(bot)
    registerUserHandlers(router)
    registerAdminHandlers(router)
}
```

### 2. 错误处理

```go
// 统一的错误处理
func handleError(c *telegramrouter.Context, err error) {
    log.Printf("错误：%v", err)
    c.Reply("抱歉，处理您的请求时发生错误")
}

// 在处理器中使用
router.Command("process", func(c *telegramrouter.Context) {
    if err := processSomething(); err != nil {
        handleError(c, err)
        return
    }
    c.Reply("处理成功")
})
```

### 3. 异步处理

```go
// 使用 goroutine 处理耗时操作
router.Command("longtask", func(c *telegramrouter.Context) {
    c.Reply("开始处理...")
    
    go func() {
        // 执行耗时操作
        result := doLongTask()
        
        // 发送结果
        c.Reply("处理完成：" + result)
    }()
})
```

## 下一步

- 学习[中间件](middleware.md)的使用方法
- 了解[高级特性](advanced-features.md)
- 查看[示例代码](examples.md)获取更多灵感 