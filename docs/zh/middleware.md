# 中间件

Telegram Router 提供了强大的中间件系统，可以轻松实现请求预处理、后处理和错误处理等功能。

## 基本概念

中间件是一个函数，它接收一个处理函数作为参数，并返回一个新的处理函数。中间件可以：

1. 在执行处理函数之前执行代码
2. 在执行处理函数之后执行代码
3. 修改处理函数的执行流程
4. 处理错误和异常

## 创建中间件

### 基本中间件

```go
// 日志中间件
func Logger() router.MiddlewareFunc {
    return func(next router.HandlerFunc) router.HandlerFunc {
        return func(c *router.Context) {
            // 处理前
            start := time.Now()
            log.Printf("开始处理消息：%s", c.Message.Text)

            // 执行下一个处理器
            next(c)

            // 处理后
            log.Printf("处理完成，耗时：%v", time.Since(start))
        }
    }
}

// 使用中间件
r.Use(Logger())
```

### 带参数的中间件

```go
// 认证中间件
func Auth(allowedUsers []int64) router.MiddlewareFunc {
    return func(next router.HandlerFunc) router.HandlerFunc {
        return func(c *router.Context) {
            userID := c.Message.From.ID
            for _, id := range allowedUsers {
                if id == userID {
                    next(c)
                    return
                }
            }
            c.Reply("未授权访问")
        }
    }
}

// 使用带参数的中间件
r.Use(Auth([]int64{123456789}))
```

### 错误处理中间件

```go
// 错误恢复中间件
func Recovery() router.MiddlewareFunc {
    return func(next router.HandlerFunc) router.HandlerFunc {
        return func(c *router.Context) {
            defer func() {
                if err := recover(); err != nil {
                    log.Printf("发生错误：%v", err)
                    c.Reply("抱歉，处理您的请求时发生错误")
                }
            }()
            next(c)
        }
    }
}

// 使用错误处理中间件
r.Use(Recovery())
```

## 中间件链

可以组合多个中间件：

```go
// 创建中间件链
r.Use(
    Logger(),
    Recovery(),
    Auth([]int64{123456789}),
)

// 或者使用 Group 创建中间件组
r.Group(func(r *router.Router) {
    r.Use(Auth([]int64{123456789}))
    r.Command("admin", func(c *router.Context) {
        c.Reply("管理员命令")
    })
})
```

## 常用中间件示例

### 限流中间件

```go
// 限流中间件
func RateLimit(limit int, window time.Duration) router.MiddlewareFunc {
    // 使用令牌桶算法
    limiter := rate.NewLimiter(rate.Every(window/time.Duration(limit)), limit)
    
    return func(next router.HandlerFunc) router.HandlerFunc {
        return func(c *router.Context) {
            if !limiter.Allow() {
                c.Reply("请求过于频繁，请稍后再试")
                return
            }
            next(c)
        }
    }
}

// 使用限流中间件
r.Use(RateLimit(10, time.Minute)) // 每分钟最多 10 条消息
```

### 黑名单中间件

```go
// 黑名单中间件
func Blacklist(blockedUsers []int64) router.MiddlewareFunc {
    return func(next router.HandlerFunc) router.HandlerFunc {
        return func(c *router.Context) {
            userID := c.Message.From.ID
            for _, id := range blockedUsers {
                if id == userID {
                    c.Reply("您已被禁止使用此机器人")
                    return
                }
            }
            next(c)
        }
    }
}

// 使用黑名单中间件
r.Use(Blacklist([]int64{987654321}))
```

### 消息过滤中间件

```go
// 消息过滤中间件
func MessageFilter(filters ...func(*tgbotapi.Message) bool) router.MiddlewareFunc {
    return func(next router.HandlerFunc) router.HandlerFunc {
        return func(c *router.Context) {
            for _, filter := range filters {
                if !filter(c.Message) {
                    c.Reply("消息不符合要求")
                    return
                }
            }
            next(c)
        }
    }
}

// 使用消息过滤中间件
r.Use(MessageFilter(
    func(m *tgbotapi.Message) bool { return len(m.Text) > 0 },
    func(m *tgbotapi.Message) bool { return !strings.Contains(m.Text, "广告") },
))
```

### 统计中间件

```go
// 统计中间件
func Stats() router.MiddlewareFunc {
    var (
        totalMessages int64
        startTime     = time.Now()
    )

    return func(next router.HandlerFunc) router.HandlerFunc {
        return func(c *router.Context) {
            atomic.AddInt64(&totalMessages, 1)
            next(c)

            // 每小时输出统计信息
            if time.Since(startTime) > time.Hour {
                log.Printf("统计信息：总消息数 %d，运行时间 %v",
                    atomic.LoadInt64(&totalMessages),
                    time.Since(startTime))
            }
        }
    }
}

// 使用统计中间件
r.Use(Stats())
```

## 中间件最佳实践

1. 保持中间件简单
   ```go
   // 好的做法：每个中间件只做一件事
   r.Use(Logger())
   r.Use(Auth([]int64{123456789}))
   
   // 避免：一个中间件做多件事
   r.Use(LoggerAndAuth([]int64{123456789}))
   ```

2. 正确处理错误
   ```go
   // 好的做法：使用 defer 和 recover
   defer func() {
       if err := recover(); err != nil {
           log.Printf("错误：%v", err)
           c.Reply("发生错误")
       }
   }()
   
   // 避免：忽略错误
   next(c) // 可能 panic
   ```

3. 注意中间件顺序
   ```go
   // 好的做法：先处理错误，再处理业务
   r.Use(Recovery())  // 错误处理
   r.Use(Logger())    // 日志记录
   r.Use(Auth(...))   // 认证
   
   // 避免：错误处理在最后
   r.Use(Logger())
   r.Use(Auth(...))
   r.Use(Recovery())  // 可能来不及处理错误
   ```

4. 使用中间件组
   ```go
   // 好的做法：使用中间件组管理相关路由
   r.Group(func(r *router.Router) {
       r.Use(AdminAuth())
       r.Command("admin", ...)
       r.Command("ban", ...)
   })
   
   // 避免：重复中间件
   r.Command("admin", ...).Use(AdminAuth())
   r.Command("ban", ...).Use(AdminAuth())
   ```

5. 性能考虑
   ```go
   // 好的做法：使用缓存
   var userCache = make(map[int64]bool)
   
   func CachedAuth() router.MiddlewareFunc {
       return func(next router.HandlerFunc) router.HandlerFunc {
           return func(c *router.Context) {
               if userCache[c.Message.From.ID] {
                   next(c)
                   return
               }
               // 检查权限并更新缓存
           }
       }
   }
   
   // 避免：每次都检查
   func Auth() router.MiddlewareFunc {
       return func(next router.HandlerFunc) router.HandlerFunc {
           return func(c *router.Context) {
               // 每次都检查权限
           }
       }
   }
   ```

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