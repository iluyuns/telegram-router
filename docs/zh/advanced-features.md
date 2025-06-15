# 高级特性

Telegram Router 提供了许多高级特性，可以帮助你构建更强大和灵活的机器人应用。

## 路径参数

### 基本用法

```go
// 处理带参数的回调
router.Callback("user/:id/profile", func(c *telegramrouter.Context) {
    userID := c.Param("id")
    c.Reply(fmt.Sprintf("用户 %s 的个人资料", userID))
})

// 处理多级路径
router.Callback("category/:categoryId/product/:productId", func(c *telegramrouter.Context) {
    categoryID := c.Param("categoryId")
    productID := c.Param("productId")
    c.Reply(fmt.Sprintf("类别 %s 的产品 %s", categoryID, productID))
})
```

### 参数验证

```go
// 带验证的回调处理器
func validateUserID(c *telegramrouter.Context) bool {
    userID := c.Param("id")
    // 验证用户ID格式
    if _, err := strconv.ParseInt(userID, 10, 64); err != nil {
        c.Reply("无效的用户ID")
        return false
    }
    return true
}

router.Callback("user/:id/profile", func(c *telegramrouter.Context) {
    if !validateUserID(c) {
        return
    }
    // 处理有效请求
})
```

## 查询参数

### 基本查询

```go
// 处理带查询参数的回调
router.Callback("products/list", func(c *telegramrouter.Context) {
    // 获取带默认值的参数
    page := c.QueryInt("page", 1)
    size := c.QueryInt("size", 10)
    sort := c.Query("sort", "id")
    
    c.Reply(fmt.Sprintf("第 %d 页，每页 %d 条，按 %s 排序", page, size, sort))
})
```

### 高级查询

```go
// 处理复杂的查询参数
router.Callback("search", func(c *telegramrouter.Context) {
    // 获取多个参数
    query := c.Query("q", "")
    category := c.Query("category", "all")
    minPrice := c.QueryInt("min_price", 0)
    maxPrice := c.QueryInt("max_price", 0)
    tags := strings.Split(c.Query("tags", ""), ",")
    
    // 构建查询条件
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
    
    // 处理查询
    c.Reply(fmt.Sprintf("搜索条件：%+v", conditions))
})
```

## 位置处理

### 位置范围匹配

```go
// 处理特定范围内的位置
router.LocationInRange(39.9, 40.1, 116.3, 116.5, func(c *telegramrouter.Context) {
    c.Reply("您在北京天安门附近！")
})

// 处理多个位置范围
func registerLocationHandlers(router *telegramrouter.TelegramRouter) {
    // 北京天安门
    router.LocationInRange(39.9, 40.1, 116.3, 116.5, func(c *telegramrouter.Context) {
        c.Reply("您在北京天安门附近！")
    })
    
    // 上海外滩
    router.LocationInRange(31.2, 31.3, 121.4, 121.5, func(c *telegramrouter.Context) {
        c.Reply("您在上海外滩附近！")
    })
    
    // 广州塔
    router.LocationInRange(23.1, 23.2, 113.3, 113.4, func(c *telegramrouter.Context) {
        c.Reply("您在广州塔附近！")
    })
}
```

### 位置共享

```go
// 处理实时位置共享
router.LiveLocation(func(c *telegramrouter.Context) {
    loc := c.Message.Location
    c.Reply(fmt.Sprintf("收到实时位置：\n纬度：%f\n经度：%f\n精确度：%d米",
        loc.Latitude,
        loc.Longitude,
        loc.HorizontalAccuracy))
})
```

## 文件处理

### 文件类型过滤

```go
// 处理特定类型的文件
router.DocumentWithType("application/pdf", 10*1024*1024, func(c *telegramrouter.Context) {
    c.Reply("收到 PDF 文件，大小不超过 10MB")
})

// 处理多个文件类型
func registerFileHandlers(router *telegramrouter.TelegramRouter) {
    // PDF 文件
    router.DocumentWithType("application/pdf", 10*1024*1024, func(c *telegramrouter.Context) {
        c.Reply("收到 PDF 文件")
    })
    
    // 图片文件
    router.DocumentWithType("image/jpeg", 5*1024*1024, func(c *telegramrouter.Context) {
        c.Reply("收到 JPEG 图片")
    })
    
    // 视频文件
    router.DocumentWithType("video/mp4", 50*1024*1024, func(c *telegramrouter.Context) {
        c.Reply("收到 MP4 视频")
    })
}
```

### 文件下载

```go
// 下载并处理文件
router.Document(func(c *telegramrouter.Context) {
    doc := c.Message.Document
    
    // 获取文件信息
    file, err := c.Bot.GetFile(tgbotapi.FileConfig{
        FileID: doc.FileID,
    })
    if err != nil {
        c.Reply("获取文件信息失败")
        return
    }
    
    // 下载文件
    resp, err := http.Get(file.Link(c.Bot.Token))
    if err != nil {
        c.Reply("下载文件失败")
        return
    }
    defer resp.Body.Close()
    
    // 处理文件
    // ...
})
```

## 投票和测验

### 投票处理

```go
// 处理普通投票
router.RegularPoll(func(c *telegramrouter.Context) {
    poll := c.Message.Poll
    c.Reply(fmt.Sprintf("收到投票：%s\n选项数：%d\n总票数：%d",
        poll.Question,
        len(poll.Options),
        poll.TotalVoterCount))
})

// 处理测验
router.Quiz(func(c *telegramrouter.Context) {
    poll := c.Message.Poll
    c.Reply(fmt.Sprintf("收到测验：%s\n正确答案：%d\n解释：%s",
        poll.Question,
        poll.CorrectOptionID,
        poll.Explanation))
})

// 处理特定类型的投票
router.PollWithType("regular", 5, true, true, func(c *telegramrouter.Context) {
    c.Reply("收到匿名多选投票，最少需要 5 票")
})
```

## 高级回调处理

### 动态回调

```go
// 生成动态回调数据
func generateCallbackData(action string, params map[string]string) string {
    // 构建基础路径
    data := action
    
    // 添加查询参数
    if len(params) > 0 {
        values := url.Values{}
        for k, v := range params {
            values.Set(k, v)
        }
        data += "?" + values.Encode()
    }
    
    return data
}

// 使用动态回调
router.Command("menu", func(c *telegramrouter.Context) {
    // 创建菜单按钮
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButton(
                "产品列表",
                generateCallbackData("products/list", map[string]string{
                    "page": "1",
                    "sort": "name",
                }),
            ),
        ),
    )
    
    c.ReplyWithKeyboard("请选择操作：", keyboard)
})
```

### 回调链

```go
// 处理回调链
func handleCallbackChain(c *telegramrouter.Context) {
    // 获取当前状态
    state := c.Query("state", "initial")
    
    // 根据状态处理
    switch state {
    case "initial":
        // 显示第一步
        keyboard := tgbotapi.NewInlineKeyboardMarkup(
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButton(
                    "下一步",
                    generateCallbackData("wizard/step", map[string]string{
                        "state": "step1",
                    }),
                ),
            ),
        )
        c.ReplyWithKeyboard("请开始操作：", keyboard)
        
    case "step1":
        // 处理第一步
        keyboard := tgbotapi.NewInlineKeyboardMarkup(
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButton(
                    "完成",
                    generateCallbackData("wizard/step", map[string]string{
                        "state": "complete",
                    }),
                ),
            ),
        )
        c.ReplyWithKeyboard("第一步完成，请继续：", keyboard)
        
    case "complete":
        // 完成处理
        c.Reply("操作已完成！")
    }
}

// 注册回调处理器
router.Callback("wizard/step", handleCallbackChain)
```

## 最佳实践

1. **参数验证**：始终验证用户输入和参数
2. **错误处理**：使用中间件统一处理错误
3. **状态管理**：使用回调链管理复杂状态
4. **资源清理**：及时清理临时文件和资源
5. **性能优化**：使用缓存和异步处理提高性能

## 常见问题

### 1. 回调数据过长

- 使用短参数名
- 只传递必要数据
- 考虑使用状态存储

### 2. 位置精度问题

- 检查位置精度参数
- 适当扩大匹配范围
- 考虑使用地理编码服务

### 3. 文件处理超时

- 使用异步处理
- 设置合理的超时时间
- 实现断点续传

## 下一步

- 查看[示例代码](examples.md)获取更多灵感
- 探索[自定义中间件](custom-middleware.md)
- 了解[性能优化](performance.md)技巧 