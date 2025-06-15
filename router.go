// Package telegramrouter 提供了一个类似 Gin 风格的 Telegram 机器人路由系统。
// 支持命令、文本、媒体等多种消息类型的路由处理，并提供中间件支持。
package telegramrouter

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
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

// Reply 创建文本消息构建器
func (c *Context) Reply(text string) *TextMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewMessage(c.Message.Chat.ID, text)
	msg.ReplyToMessageID = c.Message.MessageID
	return &TextMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithPhotoFileID 创建图片消息构建器（文件ID）
func (c *Context) ReplyWithPhotoFileID(fileID string) *PhotoMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewPhoto(c.Message.Chat.ID, tgbotapi.FileID(fileID))
	msg.ReplyToMessageID = c.Message.MessageID
	return &PhotoMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithPhotoFileURL 创建图片消息构建器（URL）
func (c *Context) ReplyWithPhotoFileURL(url string) *PhotoMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewPhoto(c.Message.Chat.ID, tgbotapi.FileURL(url))
	msg.ReplyToMessageID = c.Message.MessageID
	return &PhotoMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithPhotoFileBytes 创建图片消息构建器（字节数据）
func (c *Context) ReplyWithPhotoFileBytes(data []byte) *PhotoMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewPhoto(c.Message.Chat.ID, tgbotapi.FileBytes{
		Name:  "photo.jpg",
		Bytes: data,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	return &PhotoMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithPhotoFilePath 创建图片消息构建器（文件路径）
func (c *Context) ReplyWithPhotoFilePath(path string) *PhotoMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewPhoto(c.Message.Chat.ID, tgbotapi.FilePath(path))
	msg.ReplyToMessageID = c.Message.MessageID
	return &PhotoMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithPhotoFileReader 创建图片消息构建器（io.Reader）
func (c *Context) ReplyWithPhotoFileReader(reader io.Reader) *PhotoMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewPhoto(c.Message.Chat.ID, tgbotapi.FileReader{
		Name:   "photo.jpg",
		Reader: reader,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	return &PhotoMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithDocumentFileID 通过文件ID发送文档
func (c *Context) ReplyWithDocumentFileID(fileID string, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewDocument(c.Message.Chat.ID, tgbotapi.FileID(fileID))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithDocumentFileURL 通过URL发送文档
func (c *Context) ReplyWithDocumentFileURL(url string, caption string) *DocumentMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewDocument(c.Message.Chat.ID, tgbotapi.FileURL(url))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &DocumentMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithDocumentFileBytes 通过字节数据发送文档
func (c *Context) ReplyWithDocumentFileBytes(data []byte, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewDocument(c.Message.Chat.ID, tgbotapi.FileBytes{
		Name:  "document",
		Bytes: data,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithDocumentFilePath 通过文件路径发送文档
func (c *Context) ReplyWithDocumentFilePath(path string, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewDocument(c.Message.Chat.ID, tgbotapi.FilePath(path))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithDocumentFileReader 通过io.Reader发送文档
func (c *Context) ReplyWithDocumentFileReader(reader io.Reader, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewDocument(c.Message.Chat.ID, tgbotapi.FileReader{
		Name:   "document",
		Reader: reader,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithAudioFileID 通过文件ID发送音频
func (c *Context) ReplyWithAudioFileID(fileID string, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewAudio(c.Message.Chat.ID, tgbotapi.FileID(fileID))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithAudioFileURL 通过URL发送音频
func (c *Context) ReplyWithAudioFileURL(url string, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewAudio(c.Message.Chat.ID, tgbotapi.FileURL(url))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithAudioFileBytes 通过字节数据发送音频
func (c *Context) ReplyWithAudioFileBytes(data []byte, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewAudio(c.Message.Chat.ID, tgbotapi.FileBytes{
		Name:  "audio.mp3",
		Bytes: data,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithAudioFilePath 通过文件路径发送音频
func (c *Context) ReplyWithAudioFilePath(path string, caption string) *AudioMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewAudio(c.Message.Chat.ID, tgbotapi.FilePath(path))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &AudioMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithAudioFileReader 通过io.Reader发送音频
func (c *Context) ReplyWithAudioFileReader(reader io.Reader, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewAudio(c.Message.Chat.ID, tgbotapi.FileReader{
		Name:   "audio.mp3",
		Reader: reader,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVideoFileID 通过文件ID发送视频
func (c *Context) ReplyWithVideoFileID(fileID string, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVideo(c.Message.Chat.ID, tgbotapi.FileID(fileID))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVideoFileURL 通过URL发送视频
func (c *Context) ReplyWithVideoFileURL(url string, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVideo(c.Message.Chat.ID, tgbotapi.FileURL(url))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVideoFileBytes 通过字节数据发送视频
func (c *Context) ReplyWithVideoFileBytes(data []byte, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVideo(c.Message.Chat.ID, tgbotapi.FileBytes{
		Name:  "video.mp4",
		Bytes: data,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVideoFilePath 通过文件路径发送视频
func (c *Context) ReplyWithVideoFilePath(path string, caption string) *VideoMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVideo(c.Message.Chat.ID, tgbotapi.FilePath(path))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &VideoMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithVideoFileReader 通过io.Reader发送视频
func (c *Context) ReplyWithVideoFileReader(reader io.Reader, caption string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVideo(c.Message.Chat.ID, tgbotapi.FileReader{
		Name:   "video.mp4",
		Reader: reader,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVoiceFileID 通过文件ID发送语音
func (c *Context) ReplyWithVoiceFileID(fileID string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVoice(c.Message.Chat.ID, tgbotapi.FileID(fileID))
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVoiceFileURL 通过URL发送语音
func (c *Context) ReplyWithVoiceFileURL(url string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVoice(c.Message.Chat.ID, tgbotapi.FileURL(url))
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVoiceFileBytes 通过字节数据发送语音
func (c *Context) ReplyWithVoiceFileBytes(data []byte) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVoice(c.Message.Chat.ID, tgbotapi.FileBytes{
		Name:  "voice.ogg",
		Bytes: data,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVoiceFilePath 通过文件路径发送语音
func (c *Context) ReplyWithVoiceFilePath(path string) *VoiceMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVoice(c.Message.Chat.ID, tgbotapi.FilePath(path))
	msg.ReplyToMessageID = c.Message.MessageID
	return &VoiceMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithVoiceFileReader 通过io.Reader发送语音
func (c *Context) ReplyWithVoiceFileReader(reader io.Reader) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVoice(c.Message.Chat.ID, tgbotapi.FileReader{
		Name:   "voice.ogg",
		Reader: reader,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVideoNoteFileID 通过文件ID发送视频笔记
func (c *Context) ReplyWithVideoNoteFileID(fileID string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVideoNote(c.Message.Chat.ID, 0, tgbotapi.FileID(fileID))
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVideoNoteFileURL 通过URL发送视频笔记
func (c *Context) ReplyWithVideoNoteFileURL(url string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVideoNote(c.Message.Chat.ID, 0, tgbotapi.FileURL(url))
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVideoNoteFileBytes 通过字节数据发送视频笔记
func (c *Context) ReplyWithVideoNoteFileBytes(data []byte) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVideoNote(c.Message.Chat.ID, 0, tgbotapi.FileBytes{
		Name:  "video_note.mp4",
		Bytes: data,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVideoNoteFilePath 通过文件路径发送视频笔记
func (c *Context) ReplyWithVideoNoteFilePath(path string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVideoNote(c.Message.Chat.ID, 0, tgbotapi.FilePath(path))
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithVideoNoteFileReader 通过io.Reader发送视频笔记
func (c *Context) ReplyWithVideoNoteFileReader(reader io.Reader) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewVideoNote(c.Message.Chat.ID, 0, tgbotapi.FileReader{
		Name:   "video_note.mp4",
		Reader: reader,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithStickerFileID 通过文件ID发送贴纸
func (c *Context) ReplyWithStickerFileID(fileID string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewSticker(c.Message.Chat.ID, tgbotapi.FileID(fileID))
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithStickerFileURL 通过URL发送贴纸
func (c *Context) ReplyWithStickerFileURL(url string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewSticker(c.Message.Chat.ID, tgbotapi.FileURL(url))
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithStickerFileBytes 通过字节数据发送贴纸
func (c *Context) ReplyWithStickerFileBytes(data []byte) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewSticker(c.Message.Chat.ID, tgbotapi.FileBytes{
		Name:  "sticker.webp",
		Bytes: data,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithStickerFilePath 通过文件路径发送贴纸
func (c *Context) ReplyWithStickerFilePath(path string) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewSticker(c.Message.Chat.ID, tgbotapi.FilePath(path))
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
}

// ReplyWithStickerFileReader 通过io.Reader发送贴纸
func (c *Context) ReplyWithStickerFileReader(reader io.Reader) error {
	if c.Message == nil {
		return fmt.Errorf("no message to reply to")
	}
	msg := tgbotapi.NewSticker(c.Message.Chat.ID, tgbotapi.FileReader{
		Name:   "sticker.webp",
		Reader: reader,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	_, err := c.Bot.Send(msg)
	return err
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

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	ListenAddr string // 监听地址，如 ":8443"
	CertFile   string // SSL 证书文件路径
	KeyFile    string // SSL 私钥文件路径
	WebhookURL string // Webhook URL，如 "https://example.com:8443/bot"
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
	t.Poll(func(c *Context) {
		poll := c.Message.Poll
		if poll == nil {
			return
		}

		// 检查类型匹配
		typeMatch := pollType == "" || poll.Type == pollType
		// 检查投票数
		votesMatch := minVotes == 0 || poll.TotalVoterCount >= minVotes
		// 检查匿名设置
		anonymousMatch := poll.IsAnonymous == isAnonymous
		// 检查多选设置（仅对 regular 类型有效）
		multipleMatch := poll.Type != "regular" || poll.AllowsMultipleAnswers == allowMultiple

		if typeMatch && votesMatch && anonymousMatch && multipleMatch {
			handler(c)
		}
	})
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
	t.Location(func(c *Context) {
		loc := c.Message.Location
		if loc.Latitude >= minLat && loc.Latitude <= maxLat &&
			loc.Longitude >= minLon && loc.Longitude <= maxLon {
			handler(c)
		}
	})
}

// DocumentWithType 注册文档类型处理器
// 当文档类型和大小符合要求时触发
func (t *TelegramRouter) DocumentWithType(mimeType string, maxSize int, handler HandlerFunc) {
	t.Document(func(c *Context) {
		doc := c.Message.Document
		if (mimeType == "" || doc.MimeType == mimeType) &&
			(maxSize == 0 || doc.FileSize <= maxSize) {
			handler(c)
		}
	})
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

// MessageBuilder 基础消息构建器接口
type MessageBuilder interface {
	Send() (tgbotapi.Message, error)
	WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder
	WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder
}

// TextMessageBuilder 文本消息构建器
type TextMessageBuilder struct {
	Msg tgbotapi.MessageConfig
	bot *tgbotapi.BotAPI
}

func (b *TextMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *TextMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *TextMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *TextMessageBuilder) WithParseMode(mode string) *TextMessageBuilder {
	b.Msg.ParseMode = mode
	return b
}

// PhotoMessageBuilder 图片消息构建器
type PhotoMessageBuilder struct {
	Msg tgbotapi.PhotoConfig
	bot *tgbotapi.BotAPI
}

func (b *PhotoMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *PhotoMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *PhotoMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *PhotoMessageBuilder) WithCaption(caption string) *PhotoMessageBuilder {
	b.Msg.Caption = caption
	return b
}

func (b *PhotoMessageBuilder) WithParseMode(mode string) *PhotoMessageBuilder {
	b.Msg.ParseMode = mode
	return b
}

// PollMessageBuilder 投票消息构建器
type PollMessageBuilder struct {
	Msg tgbotapi.SendPollConfig
	bot *tgbotapi.BotAPI
}

func (b *PollMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *PollMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *PollMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

// LocationMessageBuilder 位置消息构建器
type LocationMessageBuilder struct {
	Msg tgbotapi.LocationConfig
	bot *tgbotapi.BotAPI
}

func (b *LocationMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *LocationMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *LocationMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

// VenueMessageBuilder 地点消息构建器
type VenueMessageBuilder struct {
	Msg tgbotapi.VenueConfig
	bot *tgbotapi.BotAPI
}

func (b *VenueMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *VenueMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *VenueMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

// ContactMessageBuilder 联系人消息构建器
type ContactMessageBuilder struct {
	Msg tgbotapi.ContactConfig
	bot *tgbotapi.BotAPI
}

func (b *ContactMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *ContactMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *ContactMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

// ReplyWithLocation 创建位置消息构建器
func (c *Context) ReplyWithLocation(latitude, longitude float64) *LocationMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewLocation(c.Message.Chat.ID, latitude, longitude)
	msg.ReplyToMessageID = c.Message.MessageID
	return &LocationMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithVenue 创建地点消息构建器
func (c *Context) ReplyWithVenue(latitude, longitude float64, title, address string) *VenueMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVenue(c.Message.Chat.ID, title, address, latitude, longitude)
	msg.ReplyToMessageID = c.Message.MessageID
	return &VenueMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithContact 创建联系人消息构建器
func (c *Context) ReplyWithContact(phoneNumber, firstName string, lastName ...string) *ContactMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewContact(c.Message.Chat.ID, phoneNumber, firstName)
	if len(lastName) > 0 {
		msg.LastName = lastName[0]
	}
	msg.ReplyToMessageID = c.Message.MessageID
	return &ContactMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithPoll 创建投票消息构建器
func (c *Context) ReplyWithPoll(question string, options []string, isAnonymous bool, pollType string) *PollMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewPoll(c.Message.Chat.ID, question, options...)
	msg.ReplyToMessageID = c.Message.MessageID
	msg.IsAnonymous = isAnonymous
	msg.Type = pollType
	return &PollMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// ReplyWithQuiz 创建测验消息构建器
func (c *Context) ReplyWithQuiz(question string, options []string, correctOptionID int64) *PollMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewPoll(c.Message.Chat.ID, question, options...)
	msg.ReplyToMessageID = c.Message.MessageID
	msg.Type = "quiz"
	msg.CorrectOptionID = correctOptionID
	return &PollMessageBuilder{
		Msg: msg,
		bot: c.Bot,
	}
}

// DocumentMessageBuilder 文档消息构建器
type DocumentMessageBuilder struct {
	Msg tgbotapi.DocumentConfig
	bot *tgbotapi.BotAPI
}

func (b *DocumentMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *DocumentMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *DocumentMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *DocumentMessageBuilder) WithCaption(caption string) *DocumentMessageBuilder {
	b.Msg.Caption = caption
	return b
}

func (b *DocumentMessageBuilder) WithParseMode(mode string) *DocumentMessageBuilder {
	b.Msg.ParseMode = mode
	return b
}

// AudioMessageBuilder 音频消息构建器
type AudioMessageBuilder struct {
	Msg tgbotapi.AudioConfig
	bot *tgbotapi.BotAPI
}

func (b *AudioMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *AudioMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *AudioMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *AudioMessageBuilder) WithCaption(caption string) *AudioMessageBuilder {
	b.Msg.Caption = caption
	return b
}

func (b *AudioMessageBuilder) WithParseMode(mode string) *AudioMessageBuilder {
	b.Msg.ParseMode = mode
	return b
}

func (b *AudioMessageBuilder) WithTitle(title string) *AudioMessageBuilder {
	b.Msg.Title = title
	return b
}

func (b *AudioMessageBuilder) WithDuration(duration int) *AudioMessageBuilder {
	b.Msg.Duration = duration
	return b
}

func (b *AudioMessageBuilder) WithPerformer(performer string) *AudioMessageBuilder {
	b.Msg.Performer = performer
	return b
}

// VideoMessageBuilder 视频消息构建器
type VideoMessageBuilder struct {
	Msg tgbotapi.VideoConfig
	bot *tgbotapi.BotAPI
}

func (b *VideoMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *VideoMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *VideoMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *VideoMessageBuilder) WithCaption(caption string) *VideoMessageBuilder {
	b.Msg.Caption = caption
	return b
}

func (b *VideoMessageBuilder) WithParseMode(mode string) *VideoMessageBuilder {
	b.Msg.ParseMode = mode
	return b
}

func (b *VideoMessageBuilder) WithDuration(duration int) *VideoMessageBuilder {
	b.Msg.Duration = duration
	return b
}

func (b *VideoMessageBuilder) WithSupportsStreaming(supportsStreaming bool) *VideoMessageBuilder {
	b.Msg.SupportsStreaming = supportsStreaming
	return b
}

// VoiceMessageBuilder 语音消息构建器
type VoiceMessageBuilder struct {
	Msg tgbotapi.VoiceConfig
	bot *tgbotapi.BotAPI
}

func (b *VoiceMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *VoiceMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *VoiceMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *VoiceMessageBuilder) WithCaption(caption string) *VoiceMessageBuilder {
	b.Msg.Caption = caption
	return b
}

func (b *VoiceMessageBuilder) WithParseMode(mode string) *VoiceMessageBuilder {
	b.Msg.ParseMode = mode
	return b
}

func (b *VoiceMessageBuilder) WithDuration(duration int) *VoiceMessageBuilder {
	b.Msg.Duration = duration
	return b
}

// VideoNoteMessageBuilder 视频笔记消息构建器
type VideoNoteMessageBuilder struct {
	Msg tgbotapi.VideoNoteConfig
	bot *tgbotapi.BotAPI
}

func (b *VideoNoteMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *VideoNoteMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *VideoNoteMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *VideoNoteMessageBuilder) WithDuration(duration int) *VideoNoteMessageBuilder {
	b.Msg.Duration = duration
	return b
}

func (b *VideoNoteMessageBuilder) WithLength(length int) *VideoNoteMessageBuilder {
	b.Msg.Length = length
	return b
}

// StickerMessageBuilder 贴纸消息构建器
type StickerMessageBuilder struct {
	Msg tgbotapi.StickerConfig
	bot *tgbotapi.BotAPI
}

func (b *StickerMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *StickerMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *StickerMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

// AnimationMessageBuilder 动画消息构建器
type AnimationMessageBuilder struct {
	Msg tgbotapi.AnimationConfig
	bot *tgbotapi.BotAPI
}

func (b *AnimationMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(b.Msg)
}

func (b *AnimationMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *AnimationMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *AnimationMessageBuilder) WithCaption(caption string) *AnimationMessageBuilder {
	b.Msg.Caption = caption
	return b
}

func (b *AnimationMessageBuilder) WithParseMode(mode string) *AnimationMessageBuilder {
	b.Msg.ParseMode = mode
	return b
}

func (b *AnimationMessageBuilder) WithDuration(duration int) *AnimationMessageBuilder {
	b.Msg.Duration = duration
	return b
}

// SetWebhook 设置 Webhook
func (r *TelegramRouter) SetWebhook(config WebhookConfig) error {
	webhookURL, err := url.Parse(config.WebhookURL)
	if err != nil {
		return fmt.Errorf("解析 Webhook URL 失败: %v", err)
	}

	webhookConfig := tgbotapi.WebhookConfig{
		URL: webhookURL,
	}

	// 如果有证书，设置证书
	if config.CertFile != "" {
		certData, err := os.ReadFile(config.CertFile)
		if err != nil {
			return fmt.Errorf("读取证书文件失败: %v", err)
		}
		webhookConfig.Certificate = tgbotapi.FileBytes{
			Name:  "cert.pem",
			Bytes: certData,
		}
	}

	_, err = r.Bot.Request(webhookConfig)
	return err
}

// RemoveWebhook 移除 Webhook
func (r *TelegramRouter) RemoveWebhook() error {
	_, err := r.Bot.Request(tgbotapi.DeleteWebhookConfig{
		DropPendingUpdates: true,
	})
	return err
}

// HandleWebhookRequest 直接处理 Webhook HTTP 请求
// 可以在任何 HTTP 框架中使用，如 Gin、Echo 等
func (r *TelegramRouter) HandleWebhookRequest(w http.ResponseWriter, req *http.Request) {
	update, err := r.Bot.HandleUpdate(req)
	if err != nil {
		log.Printf("处理更新失败: %v", err)
		http.Error(w, "处理更新失败", http.StatusBadRequest)
		return
	}
	r.HandleUpdate(update)
	w.WriteHeader(http.StatusOK)
}

// StartWebhook 启动 Webhook 服务器
func (r *TelegramRouter) StartWebhook(config WebhookConfig) error {
	// 设置 Webhook
	if err := r.SetWebhook(config); err != nil {
		return fmt.Errorf("设置 Webhook 失败: %v", err)
	}

	// 创建 HTTP 服务器
	http.HandleFunc("/bot", r.HandleWebhookRequest)

	// 启动 HTTPS 服务器
	if config.CertFile != "" && config.KeyFile != "" {
		return http.ListenAndServeTLS(config.ListenAddr, config.CertFile, config.KeyFile, nil)
	}

	// 如果没有证书，使用 HTTP（不推荐用于生产环境）
	return http.ListenAndServe(config.ListenAddr, nil)
}

// Listen 使用长轮询方式启动机器人
func (r *TelegramRouter) Listen() {
	updates := r.Bot.GetUpdatesChan(tgbotapi.UpdateConfig{
		Offset:  0,
		Timeout: 60,
	})
	for update := range updates {
		r.HandleUpdate(&update)
	}
}

// TextMatch 注册文本匹配处理器
// 当文本消息匹配指定模式时触发
func (t *TelegramRouter) TextMatch(pattern string, handler HandlerFunc) {
	t.Text(func(c *Context) {
		if strings.HasPrefix(c.Message.Text, pattern) {
			handler(c)
		}
	})
}

// TextRegex 注册正则表达式文本处理器
// 当文本消息匹配正则表达式时触发
func (t *TelegramRouter) TextRegex(regex *regexp.Regexp, handler HandlerFunc) {
	t.Text(func(c *Context) {
		if regex.MatchString(c.Message.Text) {
			handler(c)
		}
	})
}

// CommandRegex 注册正则表达式命令处理器
// 当命令匹配正则表达式时触发
func (t *TelegramRouter) CommandRegex(regex *regexp.Regexp, handler HandlerFunc) {
	t.Command("", func(c *Context) {
		if regex.MatchString(c.Message.Command()) {
			handler(c)
		}
	})
}
