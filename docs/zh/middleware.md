# 中间件

中间件是 Telegram Router 的核心特性之一，它允许你在处理请求前后执行自定义逻辑。中间件可以用于日志记录、认证、错误处理等场景。

## 基本概念

中间件是一个函数，它接收 `Context` 和下一个处理函数作为参数：

```go
type MiddlewareFunc func(*Context, HandlerFunc)
```

中间件可以：
1. 在执行处理函数前执行代码
2. 在执行处理函数后执行代码
3. 决定是否继续执行处理链
4. 修改请求上下文

## 创建中间件

### 1. 日志中间件

```go
func Logger() telegramrouter.MiddlewareFunc {
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        // 记录请求开始时间
        start := time.Now()
        
        // 记录请求信息
        log.Printf("收到请求：%s", c.Message.Text)
        
        // 执行下一个处理函数
        next(c)
        
        // 记录处理时间
        log.Printf("请求处理完成，耗时：%v", time.Since(start))
    }
}
```

### 2. 认证中间件

```go
func Auth(allowedUsers []int64) telegramrouter.MiddlewareFunc {
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        // 获取用户 ID
        userID := c.Message.From.ID
        
        // 检查用户是否在允许列表中
        for _, id := range allowedUsers {
            if id == userID {
                next(c)
                return
            }
        }
        
        // 未授权用户
        c.Reply("抱歉，您没有权限执行此操作")
        c.Abort()
    }
}
```

### 3. 错误处理中间件

```go
func ErrorHandler() telegramrouter.MiddlewareFunc {
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        defer func() {
            if err := recover(); err != nil {
                // 记录错误
                log.Printf("发生错误：%v", err)
                
                // 发送错误消息
                c.Reply("抱歉，处理您的请求时发生错误")
                
                // 中断处理链
                c.Abort()
            }
        }()
        
        next(c)
    }
}
```

### 4. 限流中间件

```go
func RateLimit(limit int, window time.Duration) telegramrouter.MiddlewareFunc {
    // 使用 map 存储用户请求计数
    counters := make(map[int64][]time.Time)
    var mu sync.Mutex
    
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        userID := c.Message.From.ID
        
        mu.Lock()
        // 清理过期的请求记录
        now := time.Now()
        var valid []time.Time
        for _, t := range counters[userID] {
            if now.Sub(t) < window {
                valid = append(valid, t)
            }
        }
        
        // 检查是否超过限制
        if len(valid) >= limit {
            mu.Unlock()
            c.Reply("请求过于频繁，请稍后再试")
            c.Abort()
            return
        }
        
        // 记录新请求
        counters[userID] = append(valid, now)
        mu.Unlock()
        
        next(c)
    }
}
```

## 使用中间件

### 全局中间件

```go
// 创建路由器
router := telegramrouter.NewTelegramRouter(bot)

// 添加全局中间件
router.Use(Logger())
router.Use(Auth([]int64{123456789}))
router.Use(ErrorHandler())

// 链式调用
router.Use(Logger()).
    Use(Auth([]int64{123456789})).
    Use(ErrorHandler())

// 一次添加多个中间件
router.Use(Logger(), Auth([]int64{123456789}), ErrorHandler())
```

### 中间件执行顺序

中间件按照注册的顺序执行：

```go
router.Use(Middleware1())  // 最先执行
router.Use(Middleware2())  // 第二个执行
router.Use(Middleware3())  // 最后执行

// 处理函数执行顺序：
// Middleware1 前 -> Middleware2 前 -> Middleware3 前 -> 
// 处理函数 -> 
// Middleware3 后 -> Middleware2 后 -> Middleware1 后
```

## 高级用法

### 1. 条件中间件

```go
func ConditionalMiddleware(condition func(*telegramrouter.Context) bool) telegramrouter.MiddlewareFunc {
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        if condition(c) {
            // 满足条件时执行
            log.Printf("条件满足，用户：%d", c.Message.From.ID)
        }
        next(c)
    }
}

// 使用示例
router.Use(ConditionalMiddleware(func(c *telegramrouter.Context) bool {
    return c.Message.From.ID == 123456789
}))
```

### 2. 组合中间件

```go
func CombineMiddlewares(middlewares ...telegramrouter.MiddlewareFunc) telegramrouter.MiddlewareFunc {
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        // 从后向前组合中间件
        handler := next
        for i := len(middlewares) - 1; i >= 0; i-- {
            middleware := middlewares[i]
            next := handler
            handler = func(c *telegramrouter.Context) {
                middleware(c, next)
            }
        }
        handler(c)
    }
}

// 使用示例
combined := CombineMiddlewares(
    Logger(),
    Auth([]int64{123456789}),
    ErrorHandler(),
)
router.Use(combined)
```

### 3. 带配置的中间件

```go
type MiddlewareConfig struct {
    LogLevel    string
    AllowedUsers []int64
    RateLimit   int
    TimeWindow  time.Duration
}

func ConfigurableMiddleware(config MiddlewareConfig) telegramrouter.MiddlewareFunc {
    return func(c *telegramrouter.Context, next telegramrouter.HandlerFunc) {
        // 使用配置参数
        if config.LogLevel == "debug" {
            log.Printf("调试信息：%+v", c.Message)
        }
        
        // 检查用户权限
        userID := c.Message.From.ID
        for _, id := range config.AllowedUsers {
            if id == userID {
                next(c)
                return
            }
        }
        
        c.Reply("未授权访问")
        c.Abort()
    }
}

// 使用示例
config := MiddlewareConfig{
    LogLevel:     "debug",
    AllowedUsers: []int64{123456789},
    RateLimit:    10,
    TimeWindow:   time.Minute,
}
router.Use(ConfigurableMiddleware(config))
```

## 最佳实践

1. **保持中间件简单**：每个中间件只负责一个功能
2. **注意性能影响**：避免在中间件中执行耗时操作
3. **合理使用 Abort**：只在必要时中断处理链
4. **错误处理**：确保中间件中的错误被正确处理
5. **资源清理**：使用 defer 确保资源被正确释放

## 常见问题

### 1. 中间件不执行

- 检查中间件注册顺序
- 确认中间件函数正确返回
- 验证处理链是否被中断

### 2. 中间件执行顺序问题

- 中间件按照注册顺序执行
- 使用链式调用可以更清晰地表达执行顺序
- 考虑使用 `CombineMiddlewares` 组合多个中间件

### 3. 性能问题

- 避免在中间件中执行耗时操作
- 使用缓存减少重复计算
- 考虑使用 goroutine 处理异步任务

## 下一步

- 查看[高级特性](advanced-features.md)了解更多功能
- 参考[示例代码](examples.md)获取更多灵感
- 探索如何[自定义中间件](custom-middleware.md) 