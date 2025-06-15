# 回调路由

Telegram Router 提供了强大的回调路由功能，支持类似 Web 框架的路由模式，包括路径参数和查询参数。

## 基本用法

### 静态路径
```go
router.Callback("menu/main", func(c *Context) {
    c.Reply("主菜单")
})
```

### 路径参数
使用 `:参数名` 语法定义路径参数：

```go
router.Callback("user/:id/profile", func(c *Context) {
    userID := c.Param("id")
    c.Reply(fmt.Sprintf("用户 %s 的个人资料", userID))
})
```

### 通配符
使用 `*` 作为通配符：

```go
router.Callback("action/*", func(c *Context) {
    c.Reply("通用操作处理")
})
```

## 查询参数

回调数据支持 URL 风格的查询参数：

```go
// 处理 "products/list?page=2&sort=price" 这样的回调
router.Callback("products/list", func(c *Context) {
    page := c.QueryInt("page", 1)  // 默认值为 1
    sort := c.Query("sort", "id")  // 默认值为 "id"
    c.Reply(fmt.Sprintf("第 %d 页，按 %s 排序", page, sort))
})
```

## 参数获取方法

### 路径参数
```go
// 获取字符串类型的路径参数
id := c.Param("id")

// 示例：处理 "user/123/profile" 回调
router.Callback("user/:id/profile", func(c *Context) {
    userID := c.Param("id")  // 返回 "123"
    c.Reply(fmt.Sprintf("用户ID：%s", userID))
})
```

### 查询参数
```go
// 获取字符串类型的查询参数，支持默认值
sort := c.Query("sort", "id")  // 如果 sort 参数不存在，返回 "id"

// 获取整数类型的查询参数，支持默认值
page := c.QueryInt("page", 1)  // 如果 page 参数不存在或解析失败，返回 1

// 获取布尔类型的查询参数
// 支持 "true", "1", "yes", "y" 等值
isActive := c.QueryBool("active", false)  // 如果 active 参数不存在，返回 false
```

## 实际应用示例

### 分页列表
```go
router.Callback("products/list", func(c *Context) {
    page := c.QueryInt("page", 1)
    perPage := c.QueryInt("per_page", 10)
    sort := c.Query("sort", "id")
    
    // 构建下一页的回调数据
    nextPageData := fmt.Sprintf("products/list?page=%d&per_page=%d&sort=%s", 
        page+1, perPage, sort)
    
    // 构建按钮
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("下一页", nextPageData),
        ),
    )
    
    c.ReplyWithKeyboard("产品列表", keyboard)
})
```

### 用户操作
```go
router.Callback("user/:id/:action", func(c *Context) {
    userID := c.Param("id")
    action := c.Param("action")
    
    switch action {
    case "block":
        // 处理封禁用户
        c.Reply(fmt.Sprintf("已封禁用户 %s", userID))
    case "unblock":
        // 处理解封用户
        c.Reply(fmt.Sprintf("已解封用户 %s", userID))
    case "delete":
        // 处理删除用户
        c.Reply(fmt.Sprintf("已删除用户 %s", userID))
    default:
        c.Reply("未知操作")
    }
})
```

### 设置管理
```go
router.Callback("settings/:module", func(c *Context) {
    module := c.Param("module")
    value := c.QueryBool("value", false)
    
    switch module {
    case "notifications":
        // 处理通知设置
        c.Reply(fmt.Sprintf("通知设置已%s", map[bool]string{true: "开启", false: "关闭"}[value]))
    case "privacy":
        // 处理隐私设置
        c.Reply(fmt.Sprintf("隐私设置已%s", map[bool]string{true: "开启", false: "关闭"}[value]))
    default:
        c.Reply("未知设置模块")
    }
})
```

## 注意事项

1. 回调数据长度限制：Telegram 限制回调数据最大长度为 64 字节
2. 参数命名：路径参数和查询参数的名称应该具有描述性
3. 默认值：始终为查询参数提供合理的默认值
4. 错误处理：注意处理参数解析失败的情况
5. 安全性：不要信任用户提供的参数，始终进行验证

# 回调查询路由

Telegram Router 提供了强大的回调查询路由功能，可以轻松处理内联键盘的回调查询。

## 基本回调查询处理

```go
// 处理简单的回调查询
r.Callback("button1", func(c *router.Context) {
    c.Answer("你点击了按钮 1")
    c.Edit("你选择了按钮 1")
})

// 处理带答案的回调查询
r.Callback("button2", func(c *router.Context) {
    c.Answer("处理中...", true) // 显示加载状态
    // 执行一些操作
    time.Sleep(time.Second)
    c.Edit("操作完成！")
})
```

## 路径参数

支持在回调查询数据中使用动态路径参数：

```go
// 处理带路径参数的回调查询
r.Callback("user/:id", func(c *router.Context) {
    userID := c.Param("id")
    c.Answer(fmt.Sprintf("用户 ID：%s", userID))
})

// 处理多级路径
r.Callback("menu/:category/:item", func(c *router.Context) {
    category := c.Param("category")
    item := c.Param("item")
    c.Answer(fmt.Sprintf("选择了 %s 类别的 %s", category, item))
})
```

## 查询参数

支持在回调查询数据中使用 URL 风格的查询参数：

```go
// 处理带查询参数的回调查询
r.Callback("action?type=edit&id=123", func(c *router.Context) {
    actionType := c.Query("type")
    id := c.Query("id")
    c.Answer(fmt.Sprintf("执行 %s 操作，ID：%s", actionType, id))
})

// 处理多个查询参数
r.Callback("filter?category=books&sort=price&order=desc", func(c *router.Context) {
    category := c.Query("category")
    sort := c.Query("sort")
    order := c.Query("order")
    c.Answer(fmt.Sprintf("按 %s 排序 %s，顺序：%s", sort, category, order))
})
```

## 模式匹配

支持在回调查询数据中使用模式匹配：

```go
// 使用正则表达式匹配回调查询
r.CallbackRegex(regexp.MustCompile(`^user_\d+$`), func(c *router.Context) {
    userID := strings.TrimPrefix(c.CallbackQuery.Data, "user_")
    c.Answer(fmt.Sprintf("用户 ID：%s", userID))
})

// 使用前缀匹配
r.CallbackPrefix("menu_", func(c *router.Context) {
    item := strings.TrimPrefix(c.CallbackQuery.Data, "menu_")
    c.Answer(fmt.Sprintf("选择了菜单项：%s", item))
})
```

## 内联键盘集成

```go
// 创建内联键盘
keyboard := tgbotapi.NewInlineKeyboardMarkup(
    tgbotapi.NewInlineKeyboardRow(
        tgbotapi.NewInlineKeyboardButtonData("选项 1", "choice:1"),
        tgbotapi.NewInlineKeyboardButtonData("选项 2", "choice:2"),
    ),
    tgbotapi.NewInlineKeyboardRow(
        tgbotapi.NewInlineKeyboardButtonData("取消", "choice:cancel"),
    ),
)

// 发送带内联键盘的消息
r.Command("menu", func(c *router.Context) {
    builder := c.Reply("请选择一个选项：")
    builder.WithInlineKeyboard(keyboard)
    builder.Send()
})

// 处理内联键盘回调查询
r.Callback("choice", func(c *router.Context) {
    choice := c.Args()[0]
    switch choice {
    case "1":
        c.Answer("你选择了选项 1")
        c.Edit("你选择了选项 1")
    case "2":
        c.Answer("你选择了选项 2")
        c.Edit("你选择了选项 2")
    case "cancel":
        c.Answer("已取消")
        c.Edit("操作已取消")
    }
})
```

## 错误处理

```go
// 处理回调查询错误
r.Callback("action", func(c *router.Context) {
    if err := c.Answer("处理中..."); err != nil {
        log.Printf("回调查询错误：%v", err)
        return
    }

    // 处理消息编辑错误
    if err := c.Edit("新消息"); err != nil {
        if strings.Contains(err.Error(), "message is not modified") {
            // 忽略消息未修改的错误
            return
        }
        log.Printf("编辑消息错误：%v", err)
    }
})

// 处理无效的回调查询
r.Callback("invalid", func(c *router.Context) {
    c.Answer("无效的操作", true)
    c.Edit("此操作已失效，请重新开始")
})
```

## 最佳实践

1. 使用有意义的回调查询数据
   ```go
   // 好的做法
   "user:123:profile"
   "menu:settings:notifications"
   
   // 避免使用
   "btn1"
   "action1"
   ```

2. 保持回调查询数据简短
   ```go
   // 使用查询参数而不是长路径
   "action?type=edit&id=123"  // 好
   "action/edit/123"         // 不太好
   ```

3. 使用适当的回调查询答案类型
   ```go
   // 显示加载状态
   c.Answer("处理中...", true)
   
   // 显示错误
   c.Answer("操作失败", true)
   
   // 显示成功
   c.Answer("操作成功")
   ```

4. 处理回调查询超时
   ```go
   r.Callback("action", func(c *router.Context) {
       // 检查回调查询是否过期
       if time.Since(c.CallbackQuery.Message.Date) > 48*time.Hour {
           c.Answer("此操作已过期，请重新开始")
           return
       }
       // 处理回调查询
   })
   ```

5. 使用路径和查询参数
   ```go
   // 使用路径参数表示资源
   r.Callback("user/:id", func(c *router.Context) {
       userID := c.Param("id")
       // 处理用户相关操作
   })
   
   // 使用查询参数表示操作和选项
   r.Callback("action?type=edit&id=123", func(c *router.Context) {
       actionType := c.Query("type")
       id := c.Query("id")
       // 处理编辑操作
   })
   ``` 