// Package tgr 提供了一个类似 Gin 风格的 Telegram 机器人路由系统。
// 支持命令、文本、媒体等多种消息类型的路由处理，并提供中间件支持。
package tgr

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// NewTelegramRouter 创建一个新的 Telegram 路由器实例。
// 参数 bot 是已初始化的 Telegram Bot API 实例。
func NewTelegramRouter(bot *tgbotapi.BotAPI) *TelegramRouter {
	return &TelegramRouter{
		Bot:                   bot,
		Logger:                log.New(os.Stdout, "tgr ", log.LstdFlags|log.Lshortfile),
		errorReporter:         nil,
		commandHandlers:       make(map[string][]HandlerFunc),
		locationRangeHandlers: make(map[LocationRange][]HandlerFunc),
		documentTypeHandlers:  make(map[FileType][]HandlerFunc),
		pollTypeHandlers:      make(map[PollType][]HandlerFunc),
	}
}

func NewTelegramRouterWithDefaultRecover(bot *tgbotapi.BotAPI) *TelegramRouter {
	tr := NewTelegramRouter(bot)
	tr.Use(Recover)
	return tr
}

// Recover 恢复处理函数
func Recover(ctx *Context) {
	defer func() {
		if r := recover(); r != nil {
			if ctx != nil && ctx.Logger != nil {
				ctx.Logger.Printf("Telegram router panic, restarting... %v\n%s", r, debug.Stack())
			} else {
				log.Printf("Telegram router panic, restarting... %v\n%s", r, debug.Stack())
			}
			ctx.Abort()
		}
	}()
	ctx.Next()
}

// Context 封装了 Telegram 更新的上下文信息。
// 包含原始更新数据、机器人实例、处理函数链等信息。
type Context struct {
	context.Context
	*tgbotapi.Update
	Bot      *tgbotapi.BotAPI
	Logger   *log.Logger
	index    int               // 当前执行的处理函数索引
	handlers []HandlerFunc     // 处理函数链
	aborted  bool              // 是否已中断执行
	params   map[string]string // 路由参数
	query    map[string]string // URL 查询参数
}

// AnswerCallbackOptions 回答回调的可选参数
type AnswerCallbackOptions struct {
	Text      string
	ShowAlert bool
	URL       string
	CacheTime int
}

// EditOptions 编辑消息的可选参数
type EditOptions struct {
	ParseMode             string
	DisableWebPagePreview bool
	ReplyMarkup           *tgbotapi.InlineKeyboardMarkup
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

// CommandRegexRoute 正则命令路由
type CommandRegexRoute struct {
	regex    *regexp.Regexp
	handlers []HandlerFunc
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	ListenAddr string // 监听地址，如 ":8443"
	CertFile   string // SSL 证书文件路径
	KeyFile    string // SSL 私钥文件路径
	WebhookURL string // Webhook URL，如 "https://example.com:8443/bot"
	Path       string // 自定义 Path，如 "/bot"，默认 "/bot"
}

// HandlerFunc 定义处理函数的类型。
// 每个处理函数接收一个 Context 参数，包含当前更新的上下文信息。
type HandlerFunc func(*Context)

// executeHandler 执行单个处理函数
func (c *Context) executeHandler(handler HandlerFunc) {
	if handler != nil && !c.IsAborted() {
		handler(c)
	}
}

// Next 执行处理函数链中的下一个处理函数。
// 如果处理链已中断或已到达末尾，则不会执行任何操作。
func (c *Context) Next() {
	c.index++
	for c.index < len(c.handlers) {
		if c.IsAborted() {
			return
		}
		c.executeHandler(c.handlers[c.index])
		c.index++
	}
}

// Reply 创建文本消息构建器
func (c *Context) Reply(text string) *TextMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewMessage(c.Message.Chat.ID, text)
	msg.ReplyToMessageID = c.Message.MessageID
	return &TextMessageBuilder{
		Msg: &msg,
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
		Msg: &msg,
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
		Msg: &msg,
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
		Msg: &msg,
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
		Msg: &msg,
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
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithDocumentFileID 通过文件ID发送文档
func (c *Context) ReplyWithDocumentFileID(fileID string, caption string) *DocumentMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewDocument(c.Message.Chat.ID, tgbotapi.FileID(fileID))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &DocumentMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
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
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithDocumentFileBytes 通过字节数据发送文档
func (c *Context) ReplyWithDocumentFileBytes(data []byte, caption string) *DocumentMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewDocument(c.Message.Chat.ID, tgbotapi.FileBytes{
		Name:  "document",
		Bytes: data,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &DocumentMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
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
func (c *Context) ReplyWithDocumentFileReader(reader io.Reader, caption string) *DocumentMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewDocument(c.Message.Chat.ID, tgbotapi.FileReader{
		Name:   "document",
		Reader: reader,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &DocumentMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithAudioFileID 通过文件ID发送音频
func (c *Context) ReplyWithAudioFileID(fileID string, caption string) *AudioMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewAudio(c.Message.Chat.ID, tgbotapi.FileID(fileID))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &AudioMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithAudioFileURL 通过URL发送音频
func (c *Context) ReplyWithAudioFileURL(url string, caption string) *AudioMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewAudio(c.Message.Chat.ID, tgbotapi.FileURL(url))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &AudioMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithAudioFileBytes 通过字节数据发送音频
func (c *Context) ReplyWithAudioFileBytes(data []byte, caption string) *AudioMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewAudio(c.Message.Chat.ID, tgbotapi.FileBytes{
		Name:  "audio.mp3",
		Bytes: data,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &AudioMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
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
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithAudioFileReader 通过io.Reader发送音频
func (c *Context) ReplyWithAudioFileReader(reader io.Reader, caption string) *AudioMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewAudio(c.Message.Chat.ID, tgbotapi.FileReader{
		Name:   "audio.mp3",
		Reader: reader,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &AudioMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithVideoFileID 通过文件ID发送视频
func (c *Context) ReplyWithVideoFileID(fileID string, caption string) *VideoMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVideo(c.Message.Chat.ID, tgbotapi.FileID(fileID))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &VideoMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithVideoFileURL 通过URL发送视频
func (c *Context) ReplyWithVideoFileURL(url string, caption string) *VideoMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVideo(c.Message.Chat.ID, tgbotapi.FileURL(url))
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &VideoMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithVideoFileBytes 通过字节数据发送视频
func (c *Context) ReplyWithVideoFileBytes(data []byte, caption string) *VideoMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVideo(c.Message.Chat.ID, tgbotapi.FileBytes{
		Name:  "video.mp4",
		Bytes: data,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &VideoMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
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
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithVideoFileReader 通过io.Reader发送视频
func (c *Context) ReplyWithVideoFileReader(reader io.Reader, caption string) *VideoMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVideo(c.Message.Chat.ID, tgbotapi.FileReader{
		Name:   "video.mp4",
		Reader: reader,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	if caption != "" {
		msg.Caption = caption
	}
	return &VideoMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithVoiceFileID 通过文件ID发送语音
func (c *Context) ReplyWithVoiceFileID(fileID string) *VoiceMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVoice(c.Message.Chat.ID, tgbotapi.FileID(fileID))
	msg.ReplyToMessageID = c.Message.MessageID
	return &VoiceMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithVoiceFileURL 通过URL发送语音
func (c *Context) ReplyWithVoiceFileURL(url string) *VoiceMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVoice(c.Message.Chat.ID, tgbotapi.FileURL(url))
	msg.ReplyToMessageID = c.Message.MessageID
	return &VoiceMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithVoiceFileBytes 通过字节数据发送语音
func (c *Context) ReplyWithVoiceFileBytes(data []byte) *VoiceMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVoice(c.Message.Chat.ID, tgbotapi.FileBytes{
		Name:  "voice.ogg",
		Bytes: data,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	return &VoiceMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithVoiceFilePath 通过文件路径发送语音
func (c *Context) ReplyWithVoiceFilePath(path string) *VoiceMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVoice(c.Message.Chat.ID, tgbotapi.FilePath(path))
	msg.ReplyToMessageID = c.Message.MessageID
	return &VoiceMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
}

// ReplyWithVoiceFileReader 通过io.Reader发送语音
func (c *Context) ReplyWithVoiceFileReader(reader io.Reader) *VoiceMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewVoice(c.Message.Chat.ID, tgbotapi.FileReader{
		Name:   "voice.ogg",
		Reader: reader,
	})
	msg.ReplyToMessageID = c.Message.MessageID
	return &VoiceMessageBuilder{
		Msg: &msg,
		bot: c.Bot,
	}
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

// Callback 便捷 API
// AnswerCallback 回答 CallbackQuery
func (c *Context) AnswerCallback(opts AnswerCallbackOptions) error {
	if c.CallbackQuery == nil {
		return fmt.Errorf("no callback query to answer")
	}
	cfg := tgbotapi.NewCallback(c.CallbackQuery.ID, opts.Text)
	cfg.ShowAlert = opts.ShowAlert
	if opts.URL != "" {
		cfg.URL = opts.URL
	}
	if opts.CacheTime > 0 {
		cfg.CacheTime = opts.CacheTime
	}
	_, err := c.Bot.Request(cfg)
	return err
}

// EditMessageText 根据 CallbackQuery 上下文编辑消息文本
func (c *Context) EditMessageText(text string, opts *EditOptions) error {
	if c.CallbackQuery == nil {
		return fmt.Errorf("no callback query context for edit")
	}
	var err error
	if c.CallbackQuery.Message != nil {
		msg := tgbotapi.NewEditMessageText(c.CallbackQuery.Message.Chat.ID, c.CallbackQuery.Message.MessageID, text)
		if opts != nil {
			msg.ParseMode = opts.ParseMode
			msg.DisableWebPagePreview = opts.DisableWebPagePreview
			if opts.ReplyMarkup != nil {
				msg.ReplyMarkup = opts.ReplyMarkup
			}
		}
		_, err = c.Bot.Request(msg)
		return err
	}
	if c.CallbackQuery.InlineMessageID != "" {
		msg := tgbotapi.NewEditMessageText(0, 0, text)
		msg.InlineMessageID = c.CallbackQuery.InlineMessageID
		if opts != nil {
			msg.ParseMode = opts.ParseMode
			msg.DisableWebPagePreview = opts.DisableWebPagePreview
			if opts.ReplyMarkup != nil {
				msg.ReplyMarkup = opts.ReplyMarkup
			}
		}
		_, err = c.Bot.Request(msg)
		return err
	}
	return fmt.Errorf("no message to edit")
}

// EditMessageCaption 根据 CallbackQuery 上下文编辑消息标题
func (c *Context) EditMessageCaption(caption string, opts *EditOptions) error {
	if c.CallbackQuery == nil {
		return fmt.Errorf("no callback query context for edit")
	}
	var err error
	if c.CallbackQuery.Message != nil {
		msg := tgbotapi.NewEditMessageCaption(c.CallbackQuery.Message.Chat.ID, c.CallbackQuery.Message.MessageID, caption)
		if opts != nil {
			msg.ParseMode = opts.ParseMode
			if opts.ReplyMarkup != nil {
				msg.ReplyMarkup = opts.ReplyMarkup
			}
		}
		_, err = c.Bot.Request(msg)
		return err
	}
	if c.CallbackQuery.InlineMessageID != "" {
		msg := tgbotapi.NewEditMessageCaption(0, 0, caption)
		msg.InlineMessageID = c.CallbackQuery.InlineMessageID
		if opts != nil {
			msg.ParseMode = opts.ParseMode
			if opts.ReplyMarkup != nil {
				msg.ReplyMarkup = opts.ReplyMarkup
			}
		}
		_, err = c.Bot.Request(msg)
		return err
	}
	return fmt.Errorf("no message to edit caption")
}

// EditMessageReplyMarkup 根据 CallbackQuery 上下文编辑内联键盘
func (c *Context) EditMessageReplyMarkup(markup *tgbotapi.InlineKeyboardMarkup) error {
	if c.CallbackQuery == nil {
		return fmt.Errorf("no callback query context for edit")
	}
	var err error
	if c.CallbackQuery.Message != nil {
		msg := tgbotapi.NewEditMessageReplyMarkup(c.CallbackQuery.Message.Chat.ID, c.CallbackQuery.Message.MessageID, *markup)
		_, err = c.Bot.Request(msg)
		return err
	}
	if c.CallbackQuery.InlineMessageID != "" {
		msg := tgbotapi.NewEditMessageReplyMarkup(0, 0, *markup)
		msg.InlineMessageID = c.CallbackQuery.InlineMessageID
		_, err = c.Bot.Request(msg)
		return err
	}
	return fmt.Errorf("no message to edit reply markup")
}

// EditMessageMedia 根据 CallbackQuery 上下文编辑媒体（调用方需构造好 Media）
func (c *Context) EditMessageMedia(cfg tgbotapi.EditMessageMediaConfig, opts *EditOptions) error {
	if c.CallbackQuery == nil {
		return fmt.Errorf("no callback query context for edit")
	}
	if c.CallbackQuery.Message != nil {
		cfg.BaseEdit.ChatID = c.CallbackQuery.Message.Chat.ID
		cfg.BaseEdit.MessageID = c.CallbackQuery.Message.MessageID
	} else if c.CallbackQuery.InlineMessageID != "" {
		cfg.InlineMessageID = c.CallbackQuery.InlineMessageID
	} else {
		return fmt.Errorf("no message to edit media")
	}
	if opts != nil && opts.ReplyMarkup != nil {
		cfg.ReplyMarkup = opts.ReplyMarkup
	}
	_, err := c.Bot.Request(cfg)
	return err
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

// TelegramRouter 是 Telegram 机器人的路由器。
// 负责注册和管理各种消息类型的处理函数，以及中间件。
type TelegramRouter struct {
	Bot *tgbotapi.BotAPI
	// 可插拔日志器
	Logger *log.Logger
	// 错误上报器
	errorReporter ErrorReporter
	// 读写锁，保护注册与组合缓存
	mu sync.RWMutex
	// 全局中间件，按注册顺序执行
	middlewares []HandlerFunc
	// 文本消息处理器
	textHandlers []HandlerFunc
	// 命令处理器
	commandHandlers map[string][]HandlerFunc
	// 正则命令处理器
	commandRegexRoutes []*CommandRegexRoute
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
	// Inline 模式
	inlineQueryHandlers        []HandlerFunc
	chosenInlineResultHandlers []HandlerFunc
	// 回调路由处理器
	callbackRoutes []*CallbackRoute
	// 群组相关处理器（支持多注册）
	groupChatCreatedHandlers      []HandlerFunc
	supergroupChatCreatedHandlers []HandlerFunc
	channelChatCreatedHandlers    []HandlerFunc
	newChatMembersHandlers        []HandlerFunc
	leftChatMemberHandlers        []HandlerFunc
	newChatTitleHandlers          []HandlerFunc
	newChatPhotoHandlers          []HandlerFunc
	deleteChatPhotoHandlers       []HandlerFunc
	editedMessageHandlers         []HandlerFunc
	editedChannelPostHandlers     []HandlerFunc
	myChatMemberHandlers          []HandlerFunc
	chatMemberHandlers            []HandlerFunc
	pollAnswerHandlers            []HandlerFunc
	preCheckoutQueryHandlers      []HandlerFunc
	shippingQueryHandlers         []HandlerFunc
	successfulPaymentHandlers     []HandlerFunc
	// 重命名为 updateHandlers
	updateHandlers []HandlerFunc

	// --- 组合后缓存，避免分发时重复包装中间件 ---
	composedDirty                  bool
	textHandlersC                  []HandlerFunc
	documentHandlersC              []HandlerFunc
	audioHandlersC                 []HandlerFunc
	videoHandlersC                 []HandlerFunc
	photoHandlersC                 []HandlerFunc
	stickerHandlersC               []HandlerFunc
	callbackHandlersC              []HandlerFunc
	locationHandlersC              []HandlerFunc
	contactHandlersC               []HandlerFunc
	pollHandlersC                  []HandlerFunc
	pollTypeHandlersC              map[PollType][]HandlerFunc
	quizHandlersC                  []HandlerFunc
	regularPollHandlersC           []HandlerFunc
	gameHandlersC                  []HandlerFunc
	voiceHandlersC                 []HandlerFunc
	videoNoteHandlersC             []HandlerFunc
	animationHandlersC             []HandlerFunc
	liveLocationHandlersC          []HandlerFunc
	channelPostHandlersC           []HandlerFunc
	locationRangeHandlersC         map[LocationRange][]HandlerFunc
	documentTypeHandlersC          map[FileType][]HandlerFunc
	callbackRoutesC                []*CallbackRoute
	commandHandlersC               map[string][]HandlerFunc
	commandRegexRoutesC            []*CommandRegexRoute
	inlineQueryHandlersC           []HandlerFunc
	chosenInlineResultHandlersC    []HandlerFunc
	groupChatCreatedHandlersC      []HandlerFunc
	supergroupChatCreatedHandlersC []HandlerFunc
	channelChatCreatedHandlersC    []HandlerFunc
	newChatMembersHandlersC        []HandlerFunc
	leftChatMemberHandlersC        []HandlerFunc
	newChatTitleHandlersC          []HandlerFunc
	newChatPhotoHandlersC          []HandlerFunc
	deleteChatPhotoHandlersC       []HandlerFunc
	editedMessageHandlersC         []HandlerFunc
	editedChannelPostHandlersC     []HandlerFunc
	myChatMemberHandlersC          []HandlerFunc
	chatMemberHandlersC            []HandlerFunc
	pollAnswerHandlersC            []HandlerFunc
	preCheckoutQueryHandlersC      []HandlerFunc
	shippingQueryHandlersC         []HandlerFunc
	successfulPaymentHandlersC     []HandlerFunc
}

// Use 添加全局中间件，支持链式调用。
// 可以一次添加多个中间件，也可以多次调用添加。
// 返回路由器实例以支持链式调用。
//
// 示例:
//
//	router.Use(Logger).
//	    Use(Auth([]int64{123456789})).
//	    Use(Recovery)
//
// 或者:
//
//	router.Use(Logger, Auth([]int64{123456789}), Recovery)
func (t *TelegramRouter) Use(middlewares ...HandlerFunc) *TelegramRouter {
	t.mu.Lock()
	t.middlewares = append(t.middlewares, middlewares...)
	t.composedDirty = true
	t.mu.Unlock()
	return t
}

// Command registers handlers for command messages.
// These handlers will be called when a user sends a command (e.g., /start).
// Multiple handlers can be registered for the same command and they will be executed in sequence.
//
// Command 注册命令消息的处理函数。
// 当用户发送命令（如 /start）时，这些处理函数会被调用。
// 可以为同一个命令注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.Command("start", func(c *Context) {
//	    c.Reply("欢迎使用机器人！").Send()
//	})
func (t *TelegramRouter) Command(command string, handlers ...HandlerFunc) {
	t.mu.Lock()
	t.commandHandlers[command] = append(t.commandHandlers[command], handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Text registers handlers for text messages.
// These handlers will be called when a user sends a text message.
// Multiple handlers can be registered and they will be executed in sequence.
//
// Text 注册文本消息的处理函数。
// 当用户发送文本消息时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.Text(func(c *Context) {
//	    c.Reply("收到文本消息：" + c.Message.Text).Send()
//	})
func (t *TelegramRouter) Text(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.textHandlers = append(t.textHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Document registers handlers for document messages.
// These handlers will be called when a user sends a document.
// Multiple handlers can be registered and they will be executed in sequence.
//
// Document 注册文档消息的处理函数。
// 当用户发送文档时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.Document(func(c *Context) {
//	    c.Reply("收到文档：" + c.Message.Document.FileName).Send()
//	})
func (t *TelegramRouter) Document(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.documentHandlers = append(t.documentHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Audio registers handlers for audio messages.
// These handlers will be called when a user sends an audio file.
// Multiple handlers can be registered and they will be executed in sequence.
//
// Audio 注册音频消息的处理函数。
// 当用户发送音频文件时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.Audio(func(c *Context) {
//	    c.Reply("收到音频文件").Send()
//	})
func (t *TelegramRouter) Audio(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.audioHandlers = append(t.audioHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Video registers handlers for video messages.
// These handlers will be called when a user sends a video.
// Multiple handlers can be registered and they will be executed in sequence.
//
// Video 注册视频消息的处理函数。
// 当用户发送视频时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.Video(func(c *Context) {
//	    c.Reply("收到视频文件").Send()
//	})
func (t *TelegramRouter) Video(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.videoHandlers = append(t.videoHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Photo registers handlers for photo messages.
// These handlers will be called when a user sends a photo.
// Multiple handlers can be registered and they will be executed in sequence.
//
// Photo 注册图片消息的处理函数。
// 当用户发送图片时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.Photo(func(c *Context) {
//	    c.Reply("收到图片消息").Send()
//	})
func (t *TelegramRouter) Photo(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.photoHandlers = append(t.photoHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Sticker registers handlers for sticker messages.
// These handlers will be called when a user sends a sticker.
// Multiple handlers can be registered and they will be executed in sequence.
//
// Sticker 注册贴纸消息的处理函数。
// 当用户发送贴纸时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.Sticker(func(c *Context) {
//	    c.Reply("收到贴纸").Send()
//	})
func (t *TelegramRouter) Sticker(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.stickerHandlers = append(t.stickerHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Callback 注册回调查询处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) Callback(pattern string, handlers ...HandlerFunc) {
	t.mu.Lock()
	t.callbackRoutes = append(t.callbackRoutes, &CallbackRoute{
		pattern: pattern,
		handler: func(c *Context) {
			c.handlers = handlers
			c.index = -1
			c.Next()
		},
		params: parseRouteParams(pattern),
		regex:  compileRoutePattern(pattern),
	})
	t.composedDirty = true
	t.mu.Unlock()
}

// Location registers handlers for location messages.
// These handlers will be called when a user sends a location.
// Multiple handlers can be registered and they will be executed in sequence.
//
// Location 注册位置消息的处理函数。
// 当用户发送位置时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.Location(func(c *Context) {
//	    loc := c.Message.Location
//	    c.Reply(fmt.Sprintf("收到位置：%.6f, %.6f", loc.Latitude, loc.Longitude)).Send()
//	})
func (t *TelegramRouter) Location(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.locationHandlers = append(t.locationHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Contact registers handlers for contact messages.
// These handlers will be called when a user sends a contact.
// Multiple handlers can be registered and they will be executed in sequence.
//
// Contact 注册联系人消息的处理函数。
// 当用户发送联系人信息时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.Contact(func(c *Context) {
//	    contact := c.Message.Contact
//	    c.Reply("收到联系人：" + contact.FirstName + " " + contact.LastName).Send()
//	})
func (t *TelegramRouter) Contact(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.contactHandlers = append(t.contactHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Poll 注册轮询处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) Poll(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.pollHandlers = append(t.pollHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// PollWithType 根据类型与条件注册轮询处理器（便捷 API）
func (t *TelegramRouter) PollWithType(pt PollType, handlers ...HandlerFunc) {
	t.mu.Lock()
	t.pollTypeHandlers[pt] = append(t.pollTypeHandlers[pt], handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Quiz 注册测验处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) Quiz(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.quizHandlers = append(t.quizHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// RegularPoll registers handlers for regular (non-quiz) polls.
// Multiple handlers can be registered and they will be executed in sequence until interrupted.
//
// RegularPoll 注册普通投票（非测验类型）的处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
// 普通投票允许用户选择多个选项，且不显示正确答案。
//
// Example 示例:
//
//	router.RegularPoll(func(c *Context) {
//	    log.Printf("Received regular poll: %s", c.Message.Poll.Question)
//	})
func (t *TelegramRouter) RegularPoll(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.regularPollHandlers = append(t.regularPollHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Game 注册游戏处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) Game(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.gameHandlers = append(t.gameHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Voice registers handlers for voice messages.
// These handlers will be called when a user sends a voice message.
// Multiple handlers can be registered and they will be executed in sequence.
//
// Voice 注册语音消息的处理函数。
// 当用户发送语音消息时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.Voice(func(c *Context) {
//	    voice := c.Message.Voice
//	    c.Reply("收到语音消息：" + strconv.Itoa(voice.Duration) + " 秒").Send()
//	})
func (t *TelegramRouter) Voice(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.voiceHandlers = append(t.voiceHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// VideoNote registers handlers for video note messages.
// These handlers will be called when a user sends a video note (circular video).
// Multiple handlers can be registered and they will be executed in sequence.
//
// VideoNote 注册视频笔记消息的处理函数。
// 当用户发送视频笔记（圆形视频）时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.VideoNote(func(c *Context) {
//	    videoNote := c.Message.VideoNote
//	    c.Reply("收到视频笔记：" + strconv.Itoa(videoNote.Duration) + " 秒").Send()
//	})
func (t *TelegramRouter) VideoNote(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.videoNoteHandlers = append(t.videoNoteHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// Animation registers handlers for animation messages.
// These handlers will be called when a user sends an animation (GIF-like).
// Multiple handlers can be registered and they will be executed in sequence.
//
// Animation 注册动画消息的处理函数。
// 当用户发送动画（类似 GIF）时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.Animation(func(c *Context) {
//	    anim := c.Message.Animation
//	    c.Reply("收到动画：" + anim.FileName).Send()
//	})
func (t *TelegramRouter) Animation(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.animationHandlers = append(t.animationHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// LiveLocation registers handlers for live location updates.
// These handlers will be called when a user shares their live location.
// Multiple handlers can be registered and they will be executed in sequence.
//
// LiveLocation 注册实时位置更新的处理函数。
// 当用户分享实时位置时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.LiveLocation(func(c *Context) {
//	    loc := c.Message.Location
//	    c.Reply(fmt.Sprintf("实时位置更新：%.6f, %.6f", loc.Latitude, loc.Longitude)).Send()
//	})
func (t *TelegramRouter) LiveLocation(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.liveLocationHandlers = append(t.liveLocationHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// ChannelPost registers handlers for channel post messages.
// These handlers will be called when a message is posted in a channel.
// Multiple handlers can be registered and they will be executed in sequence.
//
// ChannelPost 注册频道消息的处理函数。
// 当频道发布消息时，这些处理函数会被调用。
// 可以注册多个处理函数，它们会按顺序执行。
//
// Example 示例:
//
//	router.ChannelPost(func(c *Context) {
//	    c.Reply("收到频道消息：" + c.ChannelPost.Text).Send()
//	})
func (t *TelegramRouter) ChannelPost(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.channelPostHandlers = append(t.channelPostHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
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
	t.mu.Lock()
	t.composedDirty = true
	t.mu.Unlock()
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
	t.mu.Lock()
	t.composedDirty = true
	t.mu.Unlock()
}

// applyMiddlewares 应用中间件到处理函数。
// 按照注册顺序从后向前应用中间件，形成处理链。
func (t *TelegramRouter) applyMiddlewares(handler HandlerFunc) HandlerFunc {
	return func(c *Context) {
		// 创建一个新的处理链，包含所有中间件和原始处理链
		t.mu.RLock()
		mws := make([]HandlerFunc, len(t.middlewares))
		copy(mws, t.middlewares)
		t.mu.RUnlock()
		chain := make([]HandlerFunc, 0, len(mws)+1)
		chain = append(chain, mws...)
		chain = append(chain, handler)
		c.handlers = chain
		c.index = -1
		c.Next()
	}
}

// composeHandlers 将所有注册的处理器与中间件组合并缓存，避免分发时重复包装
func (t *TelegramRouter) composeHandlers() {
	if !t.composedDirty {
		return
	}
	// 组合期间持有写锁，避免并发注册改动
	t.mu.Lock()
	defer t.mu.Unlock()

	wrapMany := func(src []HandlerFunc) []HandlerFunc {
		if len(src) == 0 {
			return nil
		}
		out := make([]HandlerFunc, 0, len(src))
		for _, h := range src {
			out = append(out, t.applyMiddlewares(h))
		}
		return out
	}

	t.textHandlersC = wrapMany(t.textHandlers)
	t.documentHandlersC = wrapMany(t.documentHandlers)
	t.audioHandlersC = wrapMany(t.audioHandlers)
	t.videoHandlersC = wrapMany(t.videoHandlers)
	t.photoHandlersC = wrapMany(t.photoHandlers)
	t.stickerHandlersC = wrapMany(t.stickerHandlers)
	t.callbackHandlersC = wrapMany(t.callbackHandlers)
	t.locationHandlersC = wrapMany(t.locationHandlers)
	t.contactHandlersC = wrapMany(t.contactHandlers)
	t.pollHandlersC = wrapMany(t.pollHandlers)
	t.quizHandlersC = wrapMany(t.quizHandlers)
	t.regularPollHandlersC = wrapMany(t.regularPollHandlers)
	t.gameHandlersC = wrapMany(t.gameHandlers)
	t.voiceHandlersC = wrapMany(t.voiceHandlers)
	t.videoNoteHandlersC = wrapMany(t.videoNoteHandlers)
	t.animationHandlersC = wrapMany(t.animationHandlers)
	t.liveLocationHandlersC = wrapMany(t.liveLocationHandlers)
	t.channelPostHandlersC = wrapMany(t.channelPostHandlers)
	t.inlineQueryHandlersC = wrapMany(t.inlineQueryHandlers)
	t.chosenInlineResultHandlersC = wrapMany(t.chosenInlineResultHandlers)
	t.groupChatCreatedHandlersC = wrapMany(t.groupChatCreatedHandlers)
	t.supergroupChatCreatedHandlersC = wrapMany(t.supergroupChatCreatedHandlers)
	t.channelChatCreatedHandlersC = wrapMany(t.channelChatCreatedHandlers)
	t.newChatMembersHandlersC = wrapMany(t.newChatMembersHandlers)
	t.leftChatMemberHandlersC = wrapMany(t.leftChatMemberHandlers)
	t.newChatTitleHandlersC = wrapMany(t.newChatTitleHandlers)
	t.newChatPhotoHandlersC = wrapMany(t.newChatPhotoHandlers)
	t.deleteChatPhotoHandlersC = wrapMany(t.deleteChatPhotoHandlers)
	t.editedMessageHandlersC = wrapMany(t.editedMessageHandlers)
	t.editedChannelPostHandlersC = wrapMany(t.editedChannelPostHandlers)
	t.myChatMemberHandlersC = wrapMany(t.myChatMemberHandlers)
	t.chatMemberHandlersC = wrapMany(t.chatMemberHandlers)
	t.pollAnswerHandlersC = wrapMany(t.pollAnswerHandlers)
	t.preCheckoutQueryHandlersC = wrapMany(t.preCheckoutQueryHandlers)
	t.shippingQueryHandlersC = wrapMany(t.shippingQueryHandlers)
	t.successfulPaymentHandlersC = wrapMany(t.successfulPaymentHandlers)

	if len(t.pollTypeHandlers) > 0 {
		t.pollTypeHandlersC = make(map[PollType][]HandlerFunc, len(t.pollTypeHandlers))
		for k, v := range t.pollTypeHandlers {
			t.pollTypeHandlersC[k] = wrapMany(v)
		}
	} else {
		t.pollTypeHandlersC = nil
	}

	if len(t.locationRangeHandlers) > 0 {
		t.locationRangeHandlersC = make(map[LocationRange][]HandlerFunc, len(t.locationRangeHandlers))
		for k, v := range t.locationRangeHandlers {
			t.locationRangeHandlersC[k] = wrapMany(v)
		}
	} else {
		t.locationRangeHandlersC = nil
	}

	if len(t.documentTypeHandlers) > 0 {
		t.documentTypeHandlersC = make(map[FileType][]HandlerFunc, len(t.documentTypeHandlers))
		for k, v := range t.documentTypeHandlers {
			t.documentTypeHandlersC[k] = wrapMany(v)
		}
	} else {
		t.documentTypeHandlersC = nil
	}

	// Callback 路由本身持有 handler，这里也包装一层后缓存
	if len(t.callbackRoutes) > 0 {
		t.callbackRoutesC = make([]*CallbackRoute, 0, len(t.callbackRoutes))
		for _, r := range t.callbackRoutes {
			cr := &CallbackRoute{pattern: r.pattern, params: r.params, regex: r.regex}
			cr.handler = t.applyMiddlewares(r.handler)
			t.callbackRoutesC = append(t.callbackRoutesC, cr)
		}
	} else {
		t.callbackRoutesC = nil
	}

	// 命令
	if len(t.commandHandlers) > 0 {
		t.commandHandlersC = make(map[string][]HandlerFunc, len(t.commandHandlers))
		for k, v := range t.commandHandlers {
			t.commandHandlersC[k] = wrapMany(v)
		}
	} else {
		t.commandHandlersC = nil
	}
	if len(t.commandRegexRoutes) > 0 {
		t.commandRegexRoutesC = make([]*CommandRegexRoute, 0, len(t.commandRegexRoutes))
		for _, r := range t.commandRegexRoutes {
			t.commandRegexRoutesC = append(t.commandRegexRoutesC, &CommandRegexRoute{regex: r.regex, handlers: wrapMany(r.handlers)})
		}
	} else {
		t.commandRegexRoutesC = nil
	}

	t.composedDirty = false
}

// SetLogger 设置自定义日志器
func (t *TelegramRouter) SetLogger(logger *log.Logger) *TelegramRouter {
	if logger != nil {
		t.Logger = logger
	}
	return t
}

// HandleUpdate 处理 Telegram 更新消息。
// 根据消息类型分发到对应的处理函数，并应用中间件。
// 支持命令、文本、文档、音频、视频、照片、贴纸和回调查询等消息类型。
func (t *TelegramRouter) HandleUpdate(update *tgbotapi.Update) {
	if t.composedDirty {
		t.composeHandlers()
	}
	c := &Context{
		Context:  context.Background(),
		Update:   update,
		Bot:      t.Bot,
		Logger:   t.Logger,
		index:    -1,
		handlers: nil,
		aborted:  false,
		params:   make(map[string]string),
		query:    make(map[string]string),
	}

	// 首先执行通用更新处理器
	if len(t.updateHandlers) > 0 {
		for _, handler := range t.updateHandlers {
			if c.IsAborted() {
				return
			}
			handler(c)
		}
	}

	// 如果通用处理器没有中断，继续执行特定类型的处理器
	if !c.IsAborted() {
		// 处理群组相关事件
		if update.Message != nil {
			// 处理群组聊天创建
			if update.Message.GroupChatCreated {
				for _, h := range t.groupChatCreatedHandlersC {
					h(c)
					if c.IsAborted() {
						return
					}
				}
			}

			// 处理超级群组聊天创建
			if update.Message.SuperGroupChatCreated {
				for _, h := range t.supergroupChatCreatedHandlersC {
					h(c)
					if c.IsAborted() {
						return
					}
				}
			}

			// 处理频道聊天创建
			if update.Message.ChannelChatCreated {
				for _, h := range t.channelChatCreatedHandlersC {
					h(c)
					if c.IsAborted() {
						return
					}
				}
			}

			// 处理新聊天成员
			if len(update.Message.NewChatMembers) > 0 {
				for _, h := range t.newChatMembersHandlersC {
					h(c)
					if c.IsAborted() {
						return
					}
				}
			}

			// 处理离开聊天成员
			if update.Message.LeftChatMember != nil {
				for _, h := range t.leftChatMemberHandlersC {
					h(c)
					if c.IsAborted() {
						return
					}
				}
			}

			// 处理新聊天标题
			if update.Message.NewChatTitle != "" {
				for _, h := range t.newChatTitleHandlersC {
					h(c)
					if c.IsAborted() {
						return
					}
				}
			}

			// 处理新聊天照片
			if len(update.Message.NewChatPhoto) > 0 {
				for _, h := range t.newChatPhotoHandlersC {
					h(c)
					if c.IsAborted() {
						return
					}
				}
			}

			// 处理删除聊天照片
			if update.Message.DeleteChatPhoto {
				for _, h := range t.deleteChatPhotoHandlersC {
					h(c)
					if c.IsAborted() {
						return
					}
				}
			}
		}

		// 处理编辑后的消息
		if update.EditedMessage != nil {
			for _, h := range t.editedMessageHandlersC {
				h(c)
				if c.IsAborted() {
					return
				}
			}
		}

		// 处理编辑后的频道消息
		if update.EditedChannelPost != nil {
			for _, h := range t.editedChannelPostHandlersC {
				h(c)
				if c.IsAborted() {
					return
				}
			}
		}

		// 处理我的聊天成员更新
		if update.MyChatMember != nil {
			for _, h := range t.myChatMemberHandlersC {
				h(c)
				if c.IsAborted() {
					return
				}
			}
		}

		// 处理聊天成员更新
		if update.ChatMember != nil {
			for _, h := range t.chatMemberHandlersC {
				h(c)
				if c.IsAborted() {
					return
				}
			}
		}

		// 处理投票答案
		if update.PollAnswer != nil {
			for _, h := range t.pollAnswerHandlersC {
				h(c)
				if c.IsAborted() {
					return
				}
			}
		}

		// 处理预结账查询
		if update.PreCheckoutQuery != nil {
			for _, h := range t.preCheckoutQueryHandlersC {
				h(c)
				if c.IsAborted() {
					return
				}
			}
		}

		// 处理运费查询
		if update.ShippingQuery != nil {
			for _, h := range t.shippingQueryHandlersC {
				h(c)
				if c.IsAborted() {
					return
				}
			}
		}

		// 处理成功支付
		if update.Message != nil && update.Message.SuccessfulPayment != nil {
			for _, h := range t.successfulPaymentHandlersC {
				h(c)
				if c.IsAborted() {
					return
				}
			}
		}

		// 处理命令消息
		if update.Message != nil && update.Message.IsCommand() {
			cmd := update.Message.Command()
			if handlers, ok := t.commandHandlersC[cmd]; ok {
				for _, h := range handlers {
					h(c)
					if c.IsAborted() {
						return
					}
				}
				return
			}
			if len(t.commandRegexRoutesC) > 0 {
				for _, route := range t.commandRegexRoutesC {
					if route.regex.MatchString(cmd) {
						for _, h := range route.handlers {
							h(c)
							if c.IsAborted() {
								return
							}
						}
						return
					}
				}
			}
		}

		// 处理文本消息
		if update.Message != nil && update.Message.Text != "" {
			for _, handler := range t.textHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理 Inline 模式
		if update.InlineQuery != nil {
			for _, handler := range t.inlineQueryHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}
		if update.ChosenInlineResult != nil {
			for _, handler := range t.chosenInlineResultHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理文档消息
		if update.Message != nil && update.Message.Document != nil {
			for _, handler := range t.documentHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理音频消息
		if update.Message != nil && update.Message.Audio != nil {
			for _, handler := range t.audioHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理视频消息
		if update.Message != nil && update.Message.Video != nil {
			for _, handler := range t.videoHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理照片消息
		if update.Message != nil && len(update.Message.Photo) > 0 {
			for _, handler := range t.photoHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理贴纸消息
		if update.Message != nil && update.Message.Sticker != nil {
			for _, handler := range t.stickerHandlersC {
				handler(c)
				if c.IsAborted() {
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
				c.query = parseQuery(queryStr)

				// 尝试匹配路由（使用路径部分）
				for _, route := range t.callbackRoutesC {
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
						c.params = params

						// 执行处理函数
						route.handler(c)
						if c.IsAborted() {
							return
						}
					}
				}
			} else {
				// 没有查询参数，直接匹配整个回调数据
				for _, route := range t.callbackRoutesC {
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
						c.params = params

						// 执行处理函数
						route.handler(c)
						if c.IsAborted() {
							return
						}
					}
				}
			}

			// 处理未匹配的回调（通用处理器）
			for _, handler := range t.callbackHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理位置消息
		if update.Message != nil && update.Message.Location != nil {
			loc := update.Message.Location

			// 检查是否在某个范围内
			for range_, handlers := range t.locationRangeHandlersC {
				if loc.Latitude >= range_.MinLat && loc.Latitude <= range_.MaxLat &&
					loc.Longitude >= range_.MinLon && loc.Longitude <= range_.MaxLon {
					for _, handler := range handlers {
						handler(c)
						if c.IsAborted() {
							return
						}
					}
				}
			}

			// 处理普通位置消息
			for _, handler := range t.locationHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理联系信息
		if update.Message != nil && update.Message.Contact != nil {
			for _, handler := range t.contactHandlers {
				handler = t.applyMiddlewares(handler)
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理轮询消息
		if update.Poll != nil {
			poll := update.Poll

			// 检查轮询类型和条件
			for pollType, handlers := range t.pollTypeHandlersC {
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
						handler(c)
						if c.IsAborted() {
							return
						}
					}
				}
			}

			// 根据轮询类型分发到对应的处理器
			if poll.Type == "quiz" {
				// 处理测验
				for _, handler := range t.quizHandlersC {
					handler(c)
					if c.IsAborted() {
						return
					}
				}
			} else {
				// 处理普通投票
				for _, handler := range t.regularPollHandlersC {
					handler(c)
					if c.IsAborted() {
						return
					}
				}
			}

			// 处理所有轮询（通用处理器）
			for _, handler := range t.pollHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理投票
		if update.Message != nil && update.Message.Poll != nil && update.Message.Poll.Type == "quiz" {
			for _, handler := range t.quizHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理游戏
		if update.Message != nil && update.Message.Game != nil {
			for _, handler := range t.gameHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理语音消息
		if update.Message != nil && update.Message.Voice != nil {
			for _, handler := range t.voiceHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理视频笔记
		if update.Message != nil && update.Message.VideoNote != nil {
			for _, handler := range t.videoNoteHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理动画
		if update.Message != nil && update.Message.Animation != nil {
			for _, handler := range t.animationHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理位置共享
		if update.Message != nil && update.Message.Location != nil && update.Message.Location.LivePeriod > 0 {
			for _, handler := range t.liveLocationHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理群组/频道消息
		if update.ChannelPost != nil {
			for _, handler := range t.channelPostHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}

		// 处理文档类型消息
		if update.Message != nil && update.Message.Document != nil {
			doc := update.Message.Document

			// 检查文档类型和大小
			for fileType, handlers := range t.documentTypeHandlersC {
				if (fileType.MimeType == "" || doc.MimeType == fileType.MimeType) &&
					(fileType.MaxSize == 0 || doc.FileSize <= fileType.MaxSize) {
					for _, handler := range handlers {
						handler(c)
						if c.IsAborted() {
							return
						}
					}
				}
			}

			// 处理普通文档消息
			for _, handler := range t.documentHandlersC {
				handler(c)
				if c.IsAborted() {
					return
				}
			}
			return
		}
	}
}

// MessageBuilder 基础消息构建器接口
type MessageBuilder interface {
	Send() (tgbotapi.Message, error)
	WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder
	WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder
}

// ErrorReporter 错误上报接口
type ErrorReporter interface {
	Report(ctx context.Context, err error, fields ...any)
}

// SetErrorReporter 设置错误上报器
func (t *TelegramRouter) SetErrorReporter(r ErrorReporter) *TelegramRouter {
	t.mu.Lock()
	t.errorReporter = r
	t.mu.Unlock()
	return t
}

// TextMessageBuilder 文本消息构建器
type TextMessageBuilder struct {
	Msg *tgbotapi.MessageConfig
	bot *tgbotapi.BotAPI
}

func (b *TextMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
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
	Msg *tgbotapi.PhotoConfig
	bot *tgbotapi.BotAPI
}

func (b *PhotoMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
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
	Msg *tgbotapi.SendPollConfig
	bot *tgbotapi.BotAPI
}

func (b *PollMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
}

func (b *PollMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *PollMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

// 常用便捷方法
// DeleteMessage 删除当前消息（在 Message 上下文中）
func (c *Context) DeleteMessage() error {
	if c.Message == nil {
		return fmt.Errorf("no message to delete")
	}
	_, err := c.Bot.Request(tgbotapi.DeleteMessageConfig{ChatID: c.Message.Chat.ID, MessageID: c.Message.MessageID})
	return err
}

// ForwardTo 转发当前消息到指定 chat
func (c *Context) ForwardTo(chatID int64) (tgbotapi.Message, error) {
	if c.Message == nil {
		return tgbotapi.Message{}, fmt.Errorf("no message to forward")
	}
	msg := tgbotapi.NewForward(chatID, c.Message.Chat.ID, c.Message.MessageID)
	return c.Bot.Send(msg)
}

// CopyTo 复制当前消息到指定 chat
func (c *Context) CopyTo(chatID int64) (tgbotapi.Message, error) {
	if c.Message == nil {
		return tgbotapi.Message{}, fmt.Errorf("no message to copy")
	}
	msg := tgbotapi.NewCopyMessage(chatID, c.Message.Chat.ID, c.Message.MessageID)
	return c.Bot.Send(msg)
}

// MediaGroupBuilder 相册/媒体组发送
type MediaGroupBuilder struct {
	ChatID int64
	Media  []interface{}
	bot    *tgbotapi.BotAPI
}

// ReplyWithMediaGroup 构建媒体组
func (c *Context) ReplyWithMediaGroup() *MediaGroupBuilder {
	if c.Message == nil {
		return nil
	}
	return &MediaGroupBuilder{ChatID: c.Message.Chat.ID, bot: c.Bot}
}

func (b *MediaGroupBuilder) Add(media interface{}) *MediaGroupBuilder {
	b.Media = append(b.Media, media)
	return b
}

func (b *MediaGroupBuilder) Send() ([]tgbotapi.Message, error) {
	cfg := tgbotapi.MediaGroupConfig{ChatID: b.ChatID}
	cfg.Media = b.Media
	// 直接请求底层，因 SendMediaGroup 的 builder 在 v5 里使用 MediaGroupConfig
	resp, err := b.bot.Request(cfg)
	if err != nil {
		return nil, err
	}
	// 交由调用方解析；此处返回空切片以保持兼容，避免引入 json 解析
	_ = resp
	return []tgbotapi.Message{}, nil
}

// SendChatAction 发送聊天动作（typing 等）
func (c *Context) SendChatAction(action string) error {
	if c.Message == nil {
		return fmt.Errorf("no message context for chat action")
	}
	_, err := c.Bot.Request(tgbotapi.NewChatAction(c.Message.Chat.ID, action))
	return err
}

// InvoiceBuilder 支付发票（简化版）
type InvoiceBuilder struct {
	Msg *tgbotapi.InvoiceConfig
	bot *tgbotapi.BotAPI
}

func (c *Context) SendInvoice() *InvoiceBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.InvoiceConfig{BaseChat: tgbotapi.BaseChat{ChatID: c.Message.Chat.ID}}
	return &InvoiceBuilder{Msg: &msg, bot: c.Bot}
}

func (b *InvoiceBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
}

// LocationMessageBuilder 位置消息构建器
type LocationMessageBuilder struct {
	Msg *tgbotapi.LocationConfig
	bot *tgbotapi.BotAPI
}

func (b *LocationMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
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
	Msg *tgbotapi.VenueConfig
	bot *tgbotapi.BotAPI
}

func (b *VenueMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
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
	Msg *tgbotapi.ContactConfig
	bot *tgbotapi.BotAPI
}

func (b *ContactMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
}

func (b *ContactMessageBuilder) WithReplyMarkup(markup tgbotapi.ReplyKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

func (b *ContactMessageBuilder) WithInlineKeyboard(markup tgbotapi.InlineKeyboardMarkup) MessageBuilder {
	b.Msg.ReplyMarkup = markup
	return b
}

// DocumentMessageBuilder 文档消息构建器
type DocumentMessageBuilder struct {
	Msg *tgbotapi.DocumentConfig
	bot *tgbotapi.BotAPI
}

func (b *DocumentMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
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
	Msg *tgbotapi.AudioConfig
	bot *tgbotapi.BotAPI
}

func (b *AudioMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
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
	Msg *tgbotapi.VideoConfig
	bot *tgbotapi.BotAPI
}

func (b *VideoMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
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
	Msg *tgbotapi.VoiceConfig
	bot *tgbotapi.BotAPI
}

func (b *VoiceMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
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
	Msg *tgbotapi.VideoNoteConfig
	bot *tgbotapi.BotAPI
}

func (b *VideoNoteMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
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
	Msg *tgbotapi.StickerConfig
	bot *tgbotapi.BotAPI
}

func (b *StickerMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
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
	Msg *tgbotapi.AnimationConfig
	bot *tgbotapi.BotAPI
}

func (b *AnimationMessageBuilder) Send() (tgbotapi.Message, error) {
	return b.bot.Send(*b.Msg)
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
		if r.Logger != nil {
			r.Logger.Printf("处理更新失败: %v", err)
		} else {
			log.Printf("处理更新失败: %v", err)
		}
		if r.errorReporter != nil {
			r.errorReporter.Report(req.Context(), err, "path", req.URL.Path)
		}
		http.Error(w, "处理更新失败", http.StatusBadRequest)
		return
	}
	r.HandleUpdate(update)
	w.WriteHeader(http.StatusOK)
}

// NewWebhookServer 基于自定义 mux 构造 *http.Server（不启动）
// 遵循“不强占默认 ServeMux”的注意事项
func (t *TelegramRouter) NewWebhookServer(listenAddr, path string) *http.Server {
	if path == "" {
		path = "/bot"
	}
	mux := http.NewServeMux()
	mux.HandleFunc(path, t.HandleWebhookRequest)
	return &http.Server{Addr: listenAddr, Handler: mux}
}

// AttachToServer 将处理函数挂载到外部 *http.Server（不启动）
func (t *TelegramRouter) AttachToServer(srv *http.Server, path string) {
	if srv == nil {
		return
	}
	if path == "" {
		path = "/bot"
	}
	// 如果已有 mux 则复用，否则创建一个新的 mux
	var mux *http.ServeMux
	if sm, ok := srv.Handler.(*http.ServeMux); ok && sm != nil {
		mux = sm
	} else {
		mux = http.NewServeMux()
		srv.Handler = mux
	}
	mux.HandleFunc(path, t.HandleWebhookRequest)
}

const defaultQueueSize = 1024

// ListenWithContext 长轮询，带取消上下文且使用有界缓冲队列（保证外部取消时尽量不丢消息）
// 默认队列大小为 1024；如果需要自定义可以改此实现或添加参数。
// 默认并发度为 8；如果需要自定义可以改此实现或添加参数。
func (r *TelegramRouter) ListenWithContext(ctx context.Context, workers int, queueSize int) {
	if workers <= 0 {
		workers = 8
	}
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}

	updates := r.Bot.GetUpdatesChan(tgbotapi.UpdateConfig{Offset: 0, Timeout: 60})

	jobs := make(chan tgbotapi.Update, queueSize)
	var wg sync.WaitGroup

	// 启动 worker
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range jobs {
				uu := u
				r.HandleUpdate(&uu)
			}
		}()
	}

	// 辅助函数：尝试将 update 安全入队，支持取消和超时
	enqueue := func(ctx context.Context, u tgbotapi.Update, timeout time.Duration) bool {
		if timeout <= 0 {
			select {
			case jobs <- u:
				return true
			case <-ctx.Done():
				return false
			}
		}
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case jobs <- u:
			return true
		case <-ctx.Done():
			return false
		case <-timer.C:
			return false
		}
	}

	// 生产者：从 updates 读并写入 jobs
	produceDone := make(chan struct{})
	go func() {
		defer close(produceDone)
		for {
			select {
			case <-ctx.Done():
				// 外部发起取消：停止接收新更新并尝试把剩余更新 drain 到队列，防止永久阻塞
				r.Bot.StopReceivingUpdates()
				// 在 drain 阶段对入队做超时保护，避免当 jobs 已满且 worker 无法消费时阻塞
				drainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				for {
					select {
					case u, ok := <-updates:
						if !ok {
							return
						}
						// 尝试入队，超时则记录并放弃该 update
						okEnq := enqueue(drainCtx, u, 2*time.Second)
						if !okEnq {
							if r.Logger != nil {
								r.Logger.Printf("drop update during shutdown: %d", u.UpdateID)
							}
						}
					case <-drainCtx.Done():
						return
					}
				}
			case u, ok := <-updates:
				if !ok {
					return
				}
				// 尝试将更新入队，遇到外部取消或超时则放弃以避免阻塞生产者
				select {
				case jobs <- u:
				case <-ctx.Done():
					if r.Logger != nil {
						r.Logger.Printf("enqueue canceled for update: %d", u.UpdateID)
					}
					return
				}
			}
		}
	}()

	// 等待生产者退出，然后关闭 jobs，等待 worker 处理完队列中所有任务
	<-produceDone
	close(jobs)
	wg.Wait()
}

// Listen 使用长轮询方式启动机器人
func (r *TelegramRouter) Listen() {
	updates := r.Bot.GetUpdatesChan(tgbotapi.UpdateConfig{Offset: 0, Timeout: 60})
	for update := range updates {
		u := update
		go r.HandleUpdate(&u)
	}
}

// Handler 返回 http.Handler，便于集成外部 mux
func (t *TelegramRouter) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.HandleWebhookRequest(w, r)
	})
}

// HandleFunc 返回 http.HandlerFunc
func (t *TelegramRouter) HandleFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.HandleWebhookRequest(w, r)
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
func (t *TelegramRouter) CommandRegex(regex *regexp.Regexp, handlers ...HandlerFunc) {
	if regex == nil {
		return
	}
	t.commandRegexRoutes = append(t.commandRegexRoutes, &CommandRegexRoute{
		regex:    regex,
		handlers: handlers,
	})
	t.composedDirty = true
}

// parseRouteParams 解析路由参数
// 从路由模式中提取参数名，如 "user/:id/profile" 中的 "id"
func parseRouteParams(pattern string) []string {
	params := make([]string, 0)
	parts := strings.Split(pattern, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			params = append(params, part[1:])
		}
	}
	return params
}

// compileRoutePattern 编译路由模式为正则表达式
// 支持以下格式：
// - 静态路径：如 "menu/main"
// - 参数路径：如 "user/:id/profile"
// - 通配符：如 "action/*"
func compileRoutePattern(pattern string) *regexp.Regexp {
	parts := strings.Split(pattern, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			// 参数匹配，如 :id
			parts[i] = "([^/]+)"
		} else if part == "*" {
			// 通配符匹配
			parts[i] = ".*"
		}
	}
	regexPattern := "^" + strings.Join(parts, "/") + "$"
	return regexp.MustCompile(regexPattern)
}

// ReplyWithLocation 创建位置消息构建器
func (c *Context) ReplyWithLocation(latitude, longitude float64) *LocationMessageBuilder {
	if c.Message == nil {
		return nil
	}
	msg := tgbotapi.NewLocation(c.Message.Chat.ID, latitude, longitude)
	msg.ReplyToMessageID = c.Message.MessageID
	return &LocationMessageBuilder{
		Msg: &msg,
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
		Msg: &msg,
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
		Msg: &msg,
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
		Msg: &msg,
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
		Msg: &msg,
		bot: c.Bot,
	}
}

// OnGroupChatCreated 注册群组聊天创建处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnGroupChatCreated(handlers ...HandlerFunc) {
	t.groupChatCreatedHandlers = append(t.groupChatCreatedHandlers, handlers...)
	t.composedDirty = true
}

// OnSupergroupChatCreated 注册超级群组聊天创建处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnSupergroupChatCreated(handlers ...HandlerFunc) {
	t.supergroupChatCreatedHandlers = append(t.supergroupChatCreatedHandlers, handlers...)
	t.composedDirty = true
}

// OnChannelChatCreated 注册频道聊天创建处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnChannelChatCreated(handlers ...HandlerFunc) {
	t.channelChatCreatedHandlers = append(t.channelChatCreatedHandlers, handlers...)
	t.composedDirty = true
}

// OnNewChatMembers 注册新聊天成员处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnNewChatMembers(handlers ...HandlerFunc) {
	t.newChatMembersHandlers = append(t.newChatMembersHandlers, handlers...)
	t.composedDirty = true
}

// OnLeftChatMember 注册离开聊天成员处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnLeftChatMember(handlers ...HandlerFunc) {
	t.leftChatMemberHandlers = append(t.leftChatMemberHandlers, handlers...)
	t.composedDirty = true
}

// OnNewChatTitle 注册新聊天标题处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnNewChatTitle(handlers ...HandlerFunc) {
	t.newChatTitleHandlers = append(t.newChatTitleHandlers, handlers...)
	t.composedDirty = true
}

// OnNewChatPhoto 注册新聊天照片处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnNewChatPhoto(handlers ...HandlerFunc) {
	t.newChatPhotoHandlers = append(t.newChatPhotoHandlers, handlers...)
	t.composedDirty = true
}

// OnDeleteChatPhoto 注册删除聊天照片处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnDeleteChatPhoto(handlers ...HandlerFunc) {
	t.deleteChatPhotoHandlers = append(t.deleteChatPhotoHandlers, handlers...)
	t.composedDirty = true
}

// OnEditedMessage 注册编辑后的消息处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnEditedMessage(handlers ...HandlerFunc) {
	t.editedMessageHandlers = append(t.editedMessageHandlers, handlers...)
	t.composedDirty = true
}

// OnEditedChannelPost 注册编辑后的频道消息处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnEditedChannelPost(handlers ...HandlerFunc) {
	t.editedChannelPostHandlers = append(t.editedChannelPostHandlers, handlers...)
	t.composedDirty = true
}

// OnMyChatMember 注册我的聊天成员更新处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnMyChatMember(handlers ...HandlerFunc) {
	t.myChatMemberHandlers = append(t.myChatMemberHandlers, handlers...)
	t.composedDirty = true
}

// OnChatMember 注册聊天成员更新处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnChatMember(handlers ...HandlerFunc) {
	t.chatMemberHandlers = append(t.chatMemberHandlers, handlers...)
	t.composedDirty = true
}

// OnPollAnswer 注册投票答案处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnPollAnswer(handlers ...HandlerFunc) {
	t.pollAnswerHandlers = append(t.pollAnswerHandlers, handlers...)
	t.composedDirty = true
}

// OnPreCheckoutQuery 注册预结账查询处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnPreCheckoutQuery(handlers ...HandlerFunc) {
	t.preCheckoutQueryHandlers = append(t.preCheckoutQueryHandlers, handlers...)
	t.composedDirty = true
}

// OnShippingQuery 注册运费查询处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnShippingQuery(handlers ...HandlerFunc) {
	t.shippingQueryHandlers = append(t.shippingQueryHandlers, handlers...)
	t.composedDirty = true
}

// OnSuccessfulPayment 注册成功支付处理函数。
// 可以一次注册多个处理函数，它们会按顺序执行，直到被中断。
func (t *TelegramRouter) OnSuccessfulPayment(handlers ...HandlerFunc) {
	t.successfulPaymentHandlers = append(t.successfulPaymentHandlers, handlers...)
	t.composedDirty = true
}

// OnUpdate 注册通用更新处理函数
// 可以处理所有类型的 Telegram 更新，包括：
// - 所有类型的消息（文本、图片、视频等）
// - 所有类型的事件（群组创建、成员更新等）
// - 所有类型的回调查询
// - 所有类型的频道消息
// - 所有类型的支付相关更新
func (t *TelegramRouter) OnUpdate(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.updateHandlers = append(t.updateHandlers, func(c *Context) {
		// 读取中间件快照，避免运行时并发修改
		t.mu.RLock()
		mws := make([]HandlerFunc, len(t.middlewares))
		copy(mws, t.middlewares)
		t.mu.RUnlock()
		chain := make([]HandlerFunc, 0, len(mws)+len(handlers))
		chain = append(chain, mws...)
		chain = append(chain, handlers...)
		c.handlers = chain
		c.index = -1
		c.Next()
	})
	t.mu.Unlock()
}

// Inline 注册与分发
// OnInlineQuery 注册 InlineQuery 处理器
func (t *TelegramRouter) OnInlineQuery(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.inlineQueryHandlers = append(t.inlineQueryHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// OnChosenInlineResult 注册 ChosenInlineResult 处理器
func (t *TelegramRouter) OnChosenInlineResult(handlers ...HandlerFunc) {
	t.mu.Lock()
	t.chosenInlineResultHandlers = append(t.chosenInlineResultHandlers, handlers...)
	t.composedDirty = true
	t.mu.Unlock()
}

// InlineAnswerBuilder 用于回答 inline query
type InlineAnswerBuilder struct {
	QueryID string
	Results []interface{}
	Options struct {
		CacheTime  int
		IsPersonal bool
		NextOffset string
	}
	bot *tgbotapi.BotAPI
}

func (b *InlineAnswerBuilder) Send() error {
	cfg := tgbotapi.InlineConfig{
		InlineQueryID: b.QueryID,
		IsPersonal:    b.Options.IsPersonal,
		CacheTime:     b.Options.CacheTime,
		NextOffset:    b.Options.NextOffset,
	}
	// 将 interface 列表透传；调用方需提供 tgbotapi.InlineQueryResultXxx
	cfg.Results = b.Results
	_, err := b.bot.Request(cfg)
	return err
}

// AnswerInlineQuery 从 Context 构建 InlineAnswerBuilder
func (c *Context) AnswerInlineQuery() *InlineAnswerBuilder {
	if c.InlineQuery == nil {
		return nil
	}
	return &InlineAnswerBuilder{QueryID: c.InlineQuery.ID, bot: c.Bot}
}
