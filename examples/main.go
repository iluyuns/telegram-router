package main

import (
	"log"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	router "github.com/iluyuns/telegram-router"
)

func main() {
	// 创建机器人实例
	bot, err := tgbotapi.NewBotAPIWithAPIEndpoint(os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("TELEGRAM_API_PROXY"))
	log.Printf("TELEGRAM_BOT_TOKEN: %s", os.Getenv("TELEGRAM_BOT_TOKEN"))
	log.Printf("TELEGRAM_API_PROXY: %s", os.Getenv("TELEGRAM_API_PROXY"))
	if err != nil {
		log.Panicf("创建机器人实例时出错：%v", err)
	}

	// 创建路由器
	r := router.NewTelegramRouter(bot)
	r.Use(func(c *router.Context) {
		start := time.Now()
		c.Next()
		elapsed := time.Since(start)
		log.Printf("请求结束: %s", elapsed)
	})

	var startIndex int
	start := func(c *router.Context) {
		log.Printf("startIndex: %d", startIndex)
		startIndex++

		// 发送欢迎消息
		builder := c.Reply("欢迎使用机器人！\n\n" +
			"可用命令：\n" +
			"/photo - 发送图片\n" +
			"/document - 发送文档\n" +
			"/audio - 发送音频\n" +
			"/video - 发送视频\n" +
			"/voice - 发送语音\n" +
			"/location - 发送位置\n" +
			"/contact - 发送联系人\n" +
			"/poll - 发送投票")

		// 添加键盘
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("发送图片"),
				tgbotapi.NewKeyboardButton("发送位置"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("发送联系人"),
				tgbotapi.NewKeyboardButton("发送投票"),
			),
		)
		builder.WithReplyMarkup(keyboard)
		msg := builder.Msg
		// 可以继续修改 msg
		msg.Text += "----test"
		// 发送消息
		if _, err := builder.Send(); err != nil {
			log.Printf("发送消息时出错：%v", err)
		}
	}

	// 注册命令处理器
	r.Command("start", start, func(ctx *router.Context) {
		log.Printf("startIndex: %d", startIndex)
		startIndex++
		if _, err := ctx.Reply("startIndex: " + strconv.Itoa(startIndex)).Send(); err != nil {
			log.Printf("发送消息时出错：%v", err)
		}
	})
	r.OnUpdate(func(ctx *router.Context) {
		_, err := ctx.Reply("this is update, type: all router").Send()
		if err != nil {
			log.Printf("发送消息时出错：%v", err)
		}
	})
	// 群组消息处理器
	r.OnSupergroupChatCreated(func(c *router.Context) {
		log.Printf("群组消息: %s", c.Message.Text)
	})

	// 注册文本消息处理器
	r.Text(func(c *router.Context) {
		switch c.Message.Text {
		case "发送图片":
			// 使用文件路径发送图片
			builder := c.ReplyWithPhotoFilePath("./../docs/btc.jpeg")
			if _, err := builder.
				WithCaption("微信支付二维码").
				WithParseMode("Markdown").
				Send(); err != nil {
				log.Printf("发送图片时出错：%v", err)
			}

		case "发送位置":
			// 发送位置
			builder := c.ReplyWithLocation(40.7128, -74.0060)
			if _, err := builder.Send(); err != nil {
				log.Printf("发送位置时出错：%v", err)
			}

		case "发送联系人":
			// 发送联系人
			builder := c.ReplyWithContact("+1234567890", "张三", "李")
			if _, err := builder.Send(); err != nil {
				log.Printf("发送联系人时出错：%v", err)
			}

		case "发送投票":
			// 发送投票
			options := []string{"选项 1", "选项 2", "选项 3"}
			builder := c.ReplyWithPoll("你最喜欢什么颜色？", options, true, "regular")
			if _, err := builder.Send(); err != nil {
				log.Printf("发送投票时出错：%v", err)
			}
		}
	})

	// 注册命令处理器
	r.Command("photo", func(c *router.Context) {
		// 使用文件 ID 发送图片
		builder := c.ReplyWithPhotoFileID("your_file_id")
		if _, err := builder.
			WithCaption("*图片说明*").
			WithParseMode("Markdown").
			Send(); err != nil {
			log.Printf("发送图片时出错：%v", err)
		}
	})

	r.Command("document", func(c *router.Context) {
		// 使用 URL 发送文档
		builder := c.ReplyWithDocumentFileURL("https://example.com/doc.pdf", "文档说明")
		if _, err := builder.Send(); err != nil {
			log.Printf("发送文档时出错：%v", err)
		}
	})

	r.Command("audio", func(c *router.Context) {
		// 使用文件路径发送音频
		builder := c.ReplyWithAudioFilePath("examples/audio.mp3", "音频说明")
		if _, err := builder.Send(); err != nil {
			log.Printf("发送音频时出错：%v", err)
		}
	})

	r.Command("video", func(c *router.Context) {
		// 使用文件路径发送视频
		builder := c.ReplyWithVideoFilePath("examples/video.mp4", "视频说明")
		if _, err := builder.Send(); err != nil {
			log.Printf("发送视频时出错：%v", err)
		}
	})

	r.Command("voice", func(c *router.Context) {
		// 使用文件路径发送语音
		builder := c.ReplyWithVoiceFilePath("examples/voice.ogg")
		if _, err := builder.Send(); err != nil {
			log.Printf("发送语音时出错：%v", err)
		}
	})

	r.Command("location", func(c *router.Context) {
		// 发送地点
		builder := c.ReplyWithVenue(40.7128, -74.0060, "帝国大厦", "纽约第五大道 350 号")

		if _, err := builder.Send(); err != nil {
			log.Printf("发送地点时出错：%v", err)
		}
	})

	r.Command("contact", func(c *router.Context) {
		// 发送联系人
		builder := c.ReplyWithContact("+1234567890", "张三")
		if _, err := builder.Send(); err != nil {
			log.Printf("发送联系人时出错：%v", err)
		}
	})

	r.Command("poll", func(c *router.Context) {
		// 发送测验
		options := []string{"红色", "绿色", "蓝色"}
		builder := c.ReplyWithQuiz("天空是什么颜色？", options, 1) // 蓝色是正确答案
		if _, err := builder.Send(); err != nil {
			log.Printf("发送测验时出错：%v", err)
		}
	})

	// 启动机器人
	log.Printf("机器人已启动：%s", bot.Self.UserName)
	r.Listen()
}
