// Package telegramrouter 提供了一个类似 Gin 风格的 Telegram 机器人路由系统。
// 支持命令、文本、媒体等多种消息类型的路由处理，并提供中间件支持。
package telegramrouter

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandlerFunc 定义处理函数的类型。
// 每个处理函数接收一个 Context 参数，包含当前更新的上下文信息。
type HandlerFunc func(*Context)

// MiddlewareFunc 定义中间件函数类型。
// 中间件函数接收 Context 和下一个处理函数作为参数，
// 可以在处理请求前后执行自定义逻辑，并控制是否继续执行处理链。
type MiddlewareFunc func(*Context, HandlerFunc)

// Context 封装了 Telegram 更新的上下文信息。
// 包含原始更新数据、机器人实例、处理函数链等信息。
type Context struct {
	*tgbotapi.Update
	Bot      *tgbotapi.BotAPI
	index    int               // 当前执行的处理函数索引
	handlers []HandlerFunc     // 处理函数链
	aborted  bool              // 是否已中断执行
	params   map[string]string // 路由参数
	query    map[string]string // URL 查询参数
}

// Next 执行处理函数链中的下一个处理函数。
// 如果处理链已中断或已到达末尾，则不会执行任何操作。
func (c *Context) Next() {
	if c.aborted {
		return
	}
	c.index++
	if c.index < len(c.handlers) {
		c.handlers[c.index](c)
	}
}

// Abort 中断处理函数链的执行。
// 调用此方法后，Next() 将不会执行后续的处理函数。
func (c *Context) Abort() {
	c.aborted = true
}

// IsAborted 检查处理函数链是否已被中断。
// 返回 true 表示处理链已被中断，false 表示可以继续执行。
func (c *Context) IsAborted() bool {
	return c.aborted
}

// Param 获取路由参数的值。
// 如果参数不存在，返回空字符串。
func (c *Context) Param(key string) string {
	if c.params == nil {
		return ""
	}
	return c.params[key]
}

// Query 获取 URL 查询参数
// 支持默认值，如果参数不存在返回默认值
func (c *Context) Query(key string, defaultValue ...string) string {
	if c.query == nil {
		return getDefaultValue(defaultValue)
	}
	if value, ok := c.query[key]; ok {
		return value
	}
	return getDefaultValue(defaultValue)
}

// QueryInt 获取整数类型的 URL 查询参数
// 如果参数不存在或解析失败，返回默认值
func (c *Context) QueryInt(key string, defaultValue ...int) int {
	value := c.Query(key)
	if value == "" {
		return getDefaultInt(defaultValue)
	}
	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}
	return getDefaultInt(defaultValue)
}

// QueryBool 获取布尔类型的 URL 查询参数
// 支持 "true", "1", "yes", "y" 等值
func (c *Context) QueryBool(key string, defaultValue ...bool) bool {
	value := c.Query(key)
	if value == "" {
		return getDefaultBool(defaultValue)
	}
	value = strings.ToLower(value)
	return value == "true" || value == "1" || value == "yes" || value == "y"
}

// getDefaultValue 获取字符串默认值
func getDefaultValue(defaults []string) string {
	if len(defaults) > 0 {
		return defaults[0]
	}
	return ""
}

// getDefaultInt 获取整数默认值
func getDefaultInt(defaults []int) int {
	if len(defaults) > 0 {
		return defaults[0]
	}
	return 0
}

// getDefaultBool 获取布尔默认值
func getDefaultBool(defaults []bool) bool {
	if len(defaults) > 0 {
		return defaults[0]
	}
	return false
}

// parseQuery 解析 URL 查询字符串
func parseQuery(query string) map[string]string {
	params := make(map[string]string)
	if query == "" {
		return params
	}

	// 分割查询字符串
	pairs := strings.Split(query, "&")
	for _, pair := range pairs {
		// 分割键值对
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			// URL 解码
			key, _ := url.QueryUnescape(kv[0])
			value, _ := url.QueryUnescape(kv[1])
			params[key] = value
		}
	}
	return params
}

// LocationRange 位置范围匹配器
type LocationRange struct {
	MinLat float64
	MaxLat float64
	MinLon float64
	MaxLon float64
}

// FileType 文件类型匹配器
type FileType struct {
	MimeType string
	MaxSize  int // 单位：字节，改为 int 类型以匹配 tgbotapi.Document.FileSize
}

// PollType 轮询类型匹配器
type PollType struct {
	Type          string // "quiz" 或 "regular"
	MinVotes      int    // 最小投票数
	IsAnonymous   bool   // 是否匿名投票
	AllowMultiple bool   // 是否允许多选（仅 regular 类型有效）
}

// CallbackRoute 回调路由节点
type CallbackRoute struct {
	pattern string         // 路由模式，如 "user/:id/profile"
	handler HandlerFunc    // 处理函数
	params  []string       // 参数名列表，如 ["id"]
	regex   *regexp.Regexp // 编译后的正则表达式
}

// TelegramRouter 是 Telegram 机器人的路由器。
// 负责注册和管理各种消息类型的处理函数，以及中间件。
type TelegramRouter struct {
	Bot *tgbotapi.BotAPI
	// 全局中间件，按注册顺序执行
	middlewares []MiddlewareFunc
	// 文本消息处理器
	textHandlers []HandlerFunc
	// 命令处理器
	commandHandlers map[string]HandlerFunc
	// 文档消息处理器
	documentHandlers []HandlerFunc
	// 音频消息处理器
	audioHandlers []HandlerFunc
	// 视频消息处理器
	videoHandlers []HandlerFunc
	// 照片消息处理器
	photoHandlers []HandlerFunc
	// 贴纸消息处理器
	stickerHandlers []HandlerFunc
	// 回调查询处理器
	callbackHandlers []HandlerFunc
	// 位置消息处理器
	locationHandlers []HandlerFunc
	// 联系信息处理器
	contactHandlers []HandlerFunc
	// 轮询处理器
	pollHandlers []HandlerFunc
	// 轮询处理器（按类型匹配）
	pollTypeHandlers map[PollType][]HandlerFunc
	// 测验处理器（quiz 类型的轮询）
	quizHandlers []HandlerFunc
	// 普通投票处理器（regular 类型的轮询）
	regularPollHandlers []HandlerFunc
	// 游戏处理器
	gameHandlers []HandlerFunc
	// 语音消息处理器
	voiceHandlers []HandlerFunc
	// 视频笔记处理器
	videoNoteHandlers []HandlerFunc
	// 动画处理器
	animationHandlers []HandlerFunc
	// 位置共享处理器
	liveLocationHandlers []HandlerFunc
	// 群组/频道消息处理器
	channelPostHandlers []HandlerFunc
	// 位置消息处理器（带范围匹配）
	locationRangeHandlers map[LocationRange][]HandlerFunc
	// 文档处理器（带类型匹配）
	documentTypeHandlers map[FileType][]HandlerFunc
	// 回调路由处理器
	callbackRoutes []*CallbackRoute
}

// NewTelegramRouter 创建一个新的 Telegram 路由器实例。
// 参数 bot 是已初始化的 Telegram Bot API 实例。
func NewTelegramRouter(bot *tgbotapi.BotAPI) *TelegramRouter {
	return &TelegramRouter{
		Bot:                   bot,
		commandHandlers:       make(map[string]HandlerFunc),
		locationRangeHandlers: make(map[LocationRange][]HandlerFunc),
		documentTypeHandlers:  make(map[FileType][]HandlerFunc),
		pollTypeHandlers:      make(map[PollType][]HandlerFunc),
	}
}

// Use 添加全局中间件，支持链式调用。
// 可以一次添加多个中间件，也可以多次调用添加。
// 返回路由器实例以支持链式调用。
//
// 示例:
//
//	router.Use(Logger()).
//	    Use(Auth([]int64{123456789})).
//	    Use(Recovery())
//
// 或者:
//
//	router.Use(Logger(), Auth([]int64{123456789}), Recovery())
func (t *TelegramRouter) Use(middlewares ...MiddlewareFunc) *TelegramRouter {
	t.middlewares = append(t.middlewares, middlewares...)
	return t
}

// Command 注册命令处理函数。
// 参数 command 是命令名称（不含 / 前缀），handler 是处理函数。
// 当收到对应的命令消息时，将执行注册的处理函数。
//
// 示例:
//
//	router.Command("start", func(c *Context) {
//	    c.Reply("欢迎使用机器人！")
//	})
func (t *TelegramRouter) Command(command string, handler HandlerFunc) {
	t.commandHandlers[command] = handler
}

// Text 注册文本消息处理函数。
// 当收到文本消息时，将执行所有注册的处理函数，直到被中断。
//
// 示例:
//
//	router.Text(func(c *Context) {
//	    c.Reply("收到文本消息：" + c.Message.Text)
//	})
func (t *TelegramRouter) Text(handler HandlerFunc) {
	t.textHandlers = append(t.textHandlers, handler)
}

// Document 注册文档消息处理函数。
// 当收到文档消息时，将执行所有注册的处理函数，直到被中断。
//
// 示例:
//
//	router.Document(func(c *Context) {
//	    doc := c.Message.Document
//	    c.Reply(fmt.Sprintf("收到文件：%s", doc.FileName))
//	})
func (t *TelegramRouter) Document(handler HandlerFunc) {
	t.documentHandlers = append(t.documentHandlers, handler)
}

// Audio 注册音频消息处理函数。
// 当收到音频消息时，将执行所有注册的处理函数，直到被中断。
//
// 示例:
//
//	router.Audio(func(c *Context) {
//	    audio := c.Message.Audio
//	    c.Reply(fmt.Sprintf("收到音频：%s", audio.Title))
//	})
func (t *TelegramRouter) Audio(handler HandlerFunc) {
	t.audioHandlers = append(t.audioHandlers, handler)
}

// Video 注册视频消息处理函数。
// 当收到视频消息时，将执行所有注册的处理函数，直到被中断。
//
// 示例:
//
//	router.Video(func(c *Context) {
//	    video := c.Message.Video
//	    c.Reply(fmt.Sprintf("收到视频：%dx%d", video.Width, video.Height))
//	})
func (t *TelegramRouter) Video(handler HandlerFunc) {
	t.videoHandlers = append(t.videoHandlers, handler)
}

// Photo 注册照片消息处理函数。
// 当收到照片消息时，将执行所有注册的处理函数，直到被中断。
//
// 示例:
//
//	router.Photo(func(c *Context) {
//	    photo := c.Message.Photo[len(c.Message.Photo)-1]
//	    c.Reply(fmt.Sprintf("收到照片：%dx%d", photo.Width, photo.Height))
//	})
func (t *TelegramRouter) Photo(handler HandlerFunc) {
	t.photoHandlers = append(t.photoHandlers, handler)
}

// Sticker 注册贴纸消息处理函数。
// 当收到贴纸消息时，将执行所有注册的处理函数，直到被中断。
//
// 示例:
//
//	router.Sticker(func(c *Context) {
//	    sticker := c.Message.Sticker
//	    c.Reply(fmt.Sprintf("收到贴纸：%s", sticker.Emoji))
//	})
func (t *TelegramRouter) Sticker(handler HandlerFunc) {
	t.stickerHandlers = append(t.stickerHandlers, handler)
}

// Callback 注册回调查询处理器
// pattern 支持以下格式：
// - 静态路径：如 "menu/main"
// - 参数路径：如 "user/:id/profile"
// - 通配符：如 "action/*"
func (t *TelegramRouter) Callback(pattern string, handler HandlerFunc) {
	paramNames := make([]string, 0)

	// 处理参数和通配符
	parts := strings.Split(pattern, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			// 参数匹配，如 :id
			paramName := part[1:]
			paramNames = append(paramNames, paramName)
			parts[i] = "([^/]+)"
		} else if part == "*" {
			// 通配符匹配
			parts[i] = ".*"
		}
	}

	regexPattern := "^" + strings.Join(parts, "/") + "$"
	regex := regexp.MustCompile(regexPattern)

	route := &CallbackRoute{
		pattern: pattern,
		handler: handler,
		params:  paramNames,
		regex:   regex,
	}

	t.callbackRoutes = append(t.callbackRoutes, route)
}

// Location 注册位置消息处理器
func (t *TelegramRouter) Location(handler HandlerFunc) {
	t.locationHandlers = append(t.locationHandlers, handler)
}

// Contact 注册联系信息处理器
func (t *TelegramRouter) Contact(handler HandlerFunc) {
	t.contactHandlers = append(t.contactHandlers, handler)
}

// Poll 注册轮询处理器
func (t *TelegramRouter) Poll(handler HandlerFunc) {
	t.pollHandlers = append(t.pollHandlers, handler)
}

// PollWithType 注册轮询类型处理器
// 当轮询类型和条件符合要求时触发
func (t *TelegramRouter) PollWithType(pollType string, minVotes int, isAnonymous, allowMultiple bool, handler HandlerFunc) {
	pType := PollType{
		Type:          pollType,
		MinVotes:      minVotes,
		IsAnonymous:   isAnonymous,
		AllowMultiple: allowMultiple,
	}
	t.pollTypeHandlers[pType] = append(t.pollTypeHandlers[pType], handler)
}

// Quiz 注册测验处理器
// 处理 quiz 类型的轮询（测验）
func (t *TelegramRouter) Quiz(handler HandlerFunc) {
	t.quizHandlers = append(t.quizHandlers, handler)
}

// RegularPoll 注册普通投票处理器
// 处理 regular 类型的轮询（普通投票）
func (t *TelegramRouter) RegularPoll(handler HandlerFunc) {
	t.regularPollHandlers = append(t.regularPollHandlers, handler)
}

// Game 注册游戏处理器
func (t *TelegramRouter) Game(handler HandlerFunc) {
	t.gameHandlers = append(t.gameHandlers, handler)
}

// Voice 注册语音消息处理器
func (t *TelegramRouter) Voice(handler HandlerFunc) {
	t.voiceHandlers = append(t.voiceHandlers, handler)
}

// VideoNote 注册视频笔记处理器
func (t *TelegramRouter) VideoNote(handler HandlerFunc) {
	t.videoNoteHandlers = append(t.videoNoteHandlers, handler)
}

// Animation 注册动画处理器
func (t *TelegramRouter) Animation(handler HandlerFunc) {
	t.animationHandlers = append(t.animationHandlers, handler)
}

// LiveLocation 注册位置共享处理器
func (t *TelegramRouter) LiveLocation(handler HandlerFunc) {
	t.liveLocationHandlers = append(t.liveLocationHandlers, handler)
}

// ChannelPost 注册群组/频道消息处理器
func (t *TelegramRouter) ChannelPost(handler HandlerFunc) {
	t.channelPostHandlers = append(t.channelPostHandlers, handler)
}

// LocationInRange 注册位置范围处理器
// 当位置在指定范围内时触发
func (t *TelegramRouter) LocationInRange(minLat, maxLat, minLon, maxLon float64, handler HandlerFunc) {
	range_ := LocationRange{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLon: minLon,
		MaxLon: maxLon,
	}
	t.locationRangeHandlers[range_] = append(t.locationRangeHandlers[range_], handler)
}

// DocumentWithType 注册文档类型处理器
// 当文档类型和大小符合要求时触发
func (t *TelegramRouter) DocumentWithType(mimeType string, maxSize int, handler HandlerFunc) {
	fileType := FileType{
		MimeType: mimeType,
		MaxSize:  maxSize,
	}
	t.documentTypeHandlers[fileType] = append(t.documentTypeHandlers[fileType], handler)
}

// applyMiddlewares 应用中间件到处理函数。
// 按照注册顺序从后向前应用中间件，形成处理链。
func (t *TelegramRouter) applyMiddlewares(handler HandlerFunc) HandlerFunc {
	for i := len(t.middlewares) - 1; i >= 0; i-- {
		middleware := t.middlewares[i]
		next := handler
		handler = func(c *Context) {
			middleware(c, next)
		}
	}
	return handler
}

// HandleUpdate 处理 Telegram 更新消息。
// 根据消息类型分发到对应的处理函数，并应用中间件。
// 支持命令、文本、文档、音频、视频、照片、贴纸和回调查询等消息类型。
func (t *TelegramRouter) HandleUpdate(update *tgbotapi.Update) {
	ctx := &Context{
		Update: update,
		Bot:    t.Bot,
		params: make(map[string]string),
		query:  make(map[string]string),
	}

	// 处理命令消息
	if update.Message != nil && update.Message.IsCommand() {
		if handler, ok := t.commandHandlers[update.Message.Command()]; ok {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			return
		}
	}

	// 处理文本消息
	if update.Message != nil && update.Message.Text != "" {
		for _, handler := range t.textHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理文档消息
	if update.Message != nil && update.Message.Document != nil {
		for _, handler := range t.documentHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理音频消息
	if update.Message != nil && update.Message.Audio != nil {
		for _, handler := range t.audioHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理视频消息
	if update.Message != nil && update.Message.Video != nil {
		for _, handler := range t.videoHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理照片消息
	if update.Message != nil && len(update.Message.Photo) > 0 {
		for _, handler := range t.photoHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理贴纸消息
	if update.Message != nil && update.Message.Sticker != nil {
		for _, handler := range t.stickerHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理回调查询
	if update.CallbackQuery != nil {
		callback := update.CallbackQuery

		// 解析回调数据中的查询参数
		if idx := strings.Index(callback.Data, "?"); idx != -1 {
			// 分离路径和查询参数
			path := callback.Data[:idx]
			queryStr := callback.Data[idx+1:]
			ctx.query = parseQuery(queryStr)

			// 尝试匹配路由（使用路径部分）
			for _, route := range t.callbackRoutes {
				matches := route.regex.FindStringSubmatch(path)
				if matches != nil {
					// 提取参数
					params := make(map[string]string)
					for i, name := range route.params {
						if i+1 < len(matches) {
							params[name] = matches[i+1]
						}
					}

					// 设置参数到上下文
					ctx.params = params

					// 执行处理函数
					handler := t.applyMiddlewares(route.handler)
					handler(ctx)
					if ctx.IsAborted() {
						return
					}
				}
			}
		} else {
			// 没有查询参数，直接匹配整个回调数据
			for _, route := range t.callbackRoutes {
				matches := route.regex.FindStringSubmatch(callback.Data)
				if matches != nil {
					// 提取参数
					params := make(map[string]string)
					for i, name := range route.params {
						if i+1 < len(matches) {
							params[name] = matches[i+1]
						}
					}

					// 设置参数到上下文
					ctx.params = params

					// 执行处理函数
					handler := t.applyMiddlewares(route.handler)
					handler(ctx)
					if ctx.IsAborted() {
						return
					}
				}
			}
		}

		// 处理未匹配的回调（通用处理器）
		for _, handler := range t.callbackHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理位置消息
	if update.Message != nil && update.Message.Location != nil {
		loc := update.Message.Location

		// 检查是否在某个范围内
		for range_, handlers := range t.locationRangeHandlers {
			if loc.Latitude >= range_.MinLat && loc.Latitude <= range_.MaxLat &&
				loc.Longitude >= range_.MinLon && loc.Longitude <= range_.MaxLon {
				for _, handler := range handlers {
					handler = t.applyMiddlewares(handler)
					handler(ctx)
					if ctx.IsAborted() {
						return
					}
				}
			}
		}

		// 处理普通位置消息
		for _, handler := range t.locationHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理联系信息
	if update.Message != nil && update.Message.Contact != nil {
		for _, handler := range t.contactHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理轮询消息
	if update.Poll != nil {
		poll := update.Poll

		// 检查轮询类型和条件
		for pollType, handlers := range t.pollTypeHandlers {
			// 检查类型匹配
			typeMatch := pollType.Type == "" || poll.Type == pollType.Type
			// 检查投票数
			votesMatch := pollType.MinVotes == 0 || poll.TotalVoterCount >= pollType.MinVotes
			// 检查匿名设置
			anonymousMatch := pollType.IsAnonymous == poll.IsAnonymous
			// 检查多选设置（仅对 regular 类型有效）
			multipleMatch := poll.Type != "regular" || pollType.AllowMultiple == poll.AllowsMultipleAnswers

			if typeMatch && votesMatch && anonymousMatch && multipleMatch {
				for _, handler := range handlers {
					handler = t.applyMiddlewares(handler)
					handler(ctx)
					if ctx.IsAborted() {
						return
					}
				}
			}
		}

		// 根据轮询类型分发到对应的处理器
		if poll.Type == "quiz" {
			// 处理测验
			for _, handler := range t.quizHandlers {
				handler = t.applyMiddlewares(handler)
				handler(ctx)
				if ctx.IsAborted() {
					return
				}
			}
		} else {
			// 处理普通投票
			for _, handler := range t.regularPollHandlers {
				handler = t.applyMiddlewares(handler)
				handler(ctx)
				if ctx.IsAborted() {
					return
				}
			}
		}

		// 处理所有轮询（通用处理器）
		for _, handler := range t.pollHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理投票
	if update.Message != nil && update.Message.Poll != nil && update.Message.Poll.Type == "quiz" {
		for _, handler := range t.quizHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理游戏
	if update.Message != nil && update.Message.Game != nil {
		for _, handler := range t.gameHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理语音消息
	if update.Message != nil && update.Message.Voice != nil {
		for _, handler := range t.voiceHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理视频笔记
	if update.Message != nil && update.Message.VideoNote != nil {
		for _, handler := range t.videoNoteHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理动画
	if update.Message != nil && update.Message.Animation != nil {
		for _, handler := range t.animationHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理位置共享
	if update.Message != nil && update.Message.Location != nil && update.Message.Location.LivePeriod > 0 {
		for _, handler := range t.liveLocationHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理群组/频道消息
	if update.ChannelPost != nil {
		for _, handler := range t.channelPostHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}

	// 处理文档类型消息
	if update.Message != nil && update.Message.Document != nil {
		doc := update.Message.Document

		// 检查文档类型和大小
		for fileType, handlers := range t.documentTypeHandlers {
			if (fileType.MimeType == "" || doc.MimeType == fileType.MimeType) &&
				(fileType.MaxSize == 0 || doc.FileSize <= fileType.MaxSize) {
				for _, handler := range handlers {
					handler = t.applyMiddlewares(handler)
					handler(ctx)
					if ctx.IsAborted() {
						return
					}
				}
			}
		}

		// 处理普通文档消息
		for _, handler := range t.documentHandlers {
			handler = t.applyMiddlewares(handler)
			handler(ctx)
			if ctx.IsAborted() {
				return
			}
		}
		return
	}
}
