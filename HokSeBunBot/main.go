package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	c "github.com/ostafen/clover/v2"
)

func init() {
	// setup log file
	file, _ := os.OpenFile(CONFIG.LOCATION.LOG_FILE, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	log.SetOutput(file)
	log.Println("*** Starting Server ***")
	InitConfig("./config.toml")
	InitDB()
	InitNLP()
}

func handleTextMessage(bot *tgbotapi.BotAPI, Message *tgbotapi.Message) {
	if Message.IsCommand() {
		// handle commands
		switch Message.Command() {
		case "start":
			// Startup
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, "歡迎使用，使用方式可以參考我的github: https://github.com/pha123661/Hok_tse_bun_tgbot")
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println("[start]", err)
			}
		case "echo":
			// Echo
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, Message.CommandArguments())
			replyMsg.ReplyToMessageID = Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println("[echo]", err)
			}

		case "random":
			docs, err := DB.FindAll(c.NewQuery(CONFIG.DB.COLLECTION))
			if err != nil {
				log.Println("[random]", err)
				return
			}
			RandomIndex := rand.Intn(len(docs))

			var Keyword, Content string
			var Type int
			for idx, doc := range docs {
				if idx == RandomIndex {
					Keyword = doc.Get("Keyword").(string)
					Content = doc.Get("Content").(string)
					Type = int(doc.Get("Type").(int64))
					break
				}
			}
			switch Type {
			case 1:
				Content = fmt.Sprintf("幫你從 %d 篇文章中精心選擇了「%s」：\n%s", len(docs), Keyword, Content)
				replyMsg := tgbotapi.NewMessage(Message.Chat.ID, Content)
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println("[random]", err)
				}
			case 2:
				PhotoConfig := tgbotapi.NewPhoto(Message.Chat.ID, tgbotapi.FileID(Content))
				PhotoConfig.Caption = fmt.Sprintf("幫你從 %d 張圖片中精心選擇了「%s」", len(docs), Keyword)
				if _, err := bot.Request(PhotoConfig); err != nil {
					log.Println("[random]", err)
				}
			}

		case "new", "add": // new hok tse bun
			// Parse command
			Command_Args := strings.Fields(Message.CommandArguments())
			if len(Command_Args) <= 1 {
				replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("錯誤：新增格式爲 “/%s {關鍵字} {內容}”", Message.Command()))
				replyMsg.ReplyToMessageID = Message.MessageID
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println("[new]", err)
				}
				return
			}
			var Keyword string = Command_Args[0]
			var Content string = strings.TrimSpace(Message.Text[strings.Index(Message.Text, Command_Args[1]):])

			if utf8.RuneCountInString(Keyword) >= 30 {
				replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("關鍵字長度不可大於 30, 目前爲 %d 字”", utf8.RuneCountInString(Keyword)))
				replyMsg.ReplyToMessageID = Message.MessageID
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println("[new]", err)
				}
				return
			}

			docs, err := DB.FindAll(c.NewQuery(CONFIG.DB.COLLECTION).Where(
				c.Field("Keyword").Eq(Keyword)))
			if err != nil {
				log.Println("[new]", err)
				return
			}
			if len(docs) > 0 {
				Reply_Content := fmt.Sprintf("相同關鍵字的複製文已有 %d 篇（內容如下），是否繼續添加？", len(docs))
				for idx, doc := range docs {
					// same keyword & content
					if doc.Get("Content").(string) == Content {
						replyMsg := tgbotapi.NewMessage(Message.Chat.ID, "傳過了啦 腦霧?")
						replyMsg.ReplyToMessageID = Message.MessageID
						if _, err := bot.Send(replyMsg); err != nil {
							log.Println(err)
						}
						return
					}
					Reply_Content += fmt.Sprintf("\n%d.「%s」", idx+1, TruncateString(doc.Get("Content").(string), 30))
				}

				// TODO: staged changes

				replyMsg := tgbotapi.NewMessage(Message.Chat.ID, Reply_Content)
				replyMsg.ReplyToMessageID = Message.MessageID
				replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("是", Keyword),
						tgbotapi.NewInlineKeyboardButtonData("否", "NIL"),
					),
				)
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println("[new]", err)
				}
				return
			}

			// Create tmp message
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, "運算中，請稍後……")
			replyMsg.ReplyToMessageID = Message.MessageID
			to_be_delete_message, err := bot.Send(replyMsg)
			if err != nil {
				log.Println("[new]", err)
			}
			to_be_delete_message_id := to_be_delete_message.MessageID
			// Insert CP
			Sum, err := InsertCP(Message.From.ID, Keyword, Content, 1)
			if err != nil {
				log.Println("[new]", err)
				return
			}
			// Delete tmp message
			bot.Request(tgbotapi.NewDeleteMessage(Message.Chat.ID, to_be_delete_message_id))
			// send response to user
			replyMsg = tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("新增複製文「%s」成功，\n自動生成的摘要如下：「%s」", Keyword, Sum))
			replyMsg.ReplyToMessageID = Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println("[new]", err)
			}
		}
	}
}

func handleImageMessage(bot *tgbotapi.BotAPI, Message *tgbotapi.Message) {
	if Message.Caption == "" {
		return
	}
	var (
		Keyword  string = strings.TrimSpace(Message.Caption)
		Content  string
		max_area int = 0
	)

	for _, image := range Message.Photo {
		if image.Width*image.Height >= max_area {
			max_area = image.Width * image.Height
			Content = image.FileID
		}
	}

	// find existing images
	doc, err := DB.FindFirst(c.NewQuery(CONFIG.DB.COLLECTION).Where(
		c.Field("Keyword").Eq(Keyword).And(c.Field("Content").Eq(Content))))
	if err != nil {
		log.Println("[newImage]", err)
		return
	}
	if doc != nil {
		replyMsg := tgbotapi.NewMessage(Message.Chat.ID, "傳過了啦 腦霧?")
		replyMsg.ReplyToMessageID = Message.MessageID
		if _, err := bot.Send(replyMsg); err != nil {
			log.Println(err)
		}
		return
	}

	InsertCP(Message.From.ID, Keyword, Content, 2)
	replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("新增圖片「%s」成功", Keyword))
	replyMsg.ReplyToMessageID = Message.MessageID
	if _, err := bot.Send(replyMsg); err != nil {
		log.Println(err)
	}

}
func main() {
	// keep alive
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprint(res, "Hello World!")
	})
	go http.ListenAndServe(":9000", nil)

	// start bot
	bot, err := tgbotapi.NewBotAPI(CONFIG.API.TG.TOKEN)
	if err != nil {
		log.Panicln(err)
	}
	bot.Debug = true
	fmt.Println("***", "Sucessful logged in as", bot.Self.UserName, "***")

	// update config
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	// get messages
	updates := bot.GetUpdatesChan(updateConfig)
	for update := range updates {
		switch {
		case update.Message != nil:
			if update.Message.Photo != nil {
				// handle image updates
				go handleImageMessage(bot, update.Message)
			} else {
				// handle text updates
				go handleTextMessage(bot, update.Message)
			}
		case update.CallbackQuery != nil:
			// handle callback query

		}
	}
}
