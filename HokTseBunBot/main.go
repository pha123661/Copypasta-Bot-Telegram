package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"unicode/utf8"

	_ "net/http/pprof"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	xurls "mvdan.cc/xurls/v2"
)

/*
Doc format:

	Dict{
			"Type":          Type,
			"Keyword":       Keyword,
			"Summarization": Summarization,
			"Content":       Content,
			"From":          FromID,
			"CreateTime":    time.Now(),
		}
*/

var bot *tgbotapi.BotAPI

func init() {
	InitConfig("./config.toml")
	// setup log file
	file, _ := os.OpenFile(CONFIG.SETTING.LOG_FILE, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("*** Starting Server ***")
}

func main() {
	// keep alive
	// http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
	// 	fmt.Fprint(res, "Hello World!")
	// })
	go http.ListenAndServe(":6060", nil)

	var err error
	// start bot
	bot, err = tgbotapi.NewBotAPI(CONFIG.API.TG.TOKEN)
	if err != nil {
		log.Panicln(err)
	}
	bot.Debug = true
	fmt.Println("***", "Sucessful logged in as", bot.Self.UserName, "***")

	InitVLP()
	InitDB()

	// update config
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	// get messages
	updates := bot.GetUpdatesChan(updateConfig)
	for update := range updates {
		switch {
		case update.Message != nil:
			switch {
			case update.Message.Animation != nil || update.Message.Video != nil:
				go handleAnimatedMessage(update.Message)
			case update.Message.Photo != nil:
				go handleImageMessage(update.Message)
			case update.Message.IsCommand():
				go handleCommand(update.Message)
			case update.Message.Text != "":
				if xurls.Relaxed().FindString(update.Message.Text) != "" {
					// messages contain url are ignored
					break
				}
				if utf8.RuneCountInString(update.Message.Text) >= 200 {
					break
				}
				go handleTextMessage(update.Message)
			default:
				// PrintStructAsTOML(update)
			}
		case update.CallbackQuery != nil:
			// handle callback query
			go handleCallbackQuery(update.CallbackQuery)
		case update.MyChatMember != nil:
			if update.MyChatMember.NewChatMember.Status == "restricted" || update.MyChatMember.NewChatMember.Status == "kicked" || update.MyChatMember.NewChatMember.Status == "left" {
				fmt.Println("[Kicked] Get Kicked by", update.MyChatMember.Chat.ID, update.MyChatMember.Chat.UserName, update.MyChatMember.Chat.Title)
				log.Println("[Kicked] Get Kicked by", update.MyChatMember.Chat.ID, update.MyChatMember.Chat.UserName, update.MyChatMember.Chat.Title)
			} else {
				fmt.Println("[Joining] Joining", update.MyChatMember.Chat.ID, update.MyChatMember.Chat.UserName, update.MyChatMember.Chat.Title)
				log.Println("[Joining] Joining", update.MyChatMember.Chat.ID, update.MyChatMember.Chat.UserName, update.MyChatMember.Chat.Title)
			}
			if update.MyChatMember.Chat.Type == "group" || update.MyChatMember.Chat.Type == "supergroup" {
				// get invited in a group
				SendText(update.MyChatMember.Chat.ID, "歡迎使用，使用方式可以參考我的github: https://github.com/pha123661/Hok_tse_bun_tgbot", 0)
			}
		}
	}
}

func NewChat(ChatID int64) {
	if err := DB.CreateCollection(CONFIG.GetColbyChatID(ChatID)); err != nil {
		log.Println("[NewChat]", err)
	} else {
		log.Printf("[NewChat] new db %s created!\n", CONFIG.GetColbyChatID(ChatID))
	}
	SendText(ChatID, "歡迎使用，請輸入或點擊 /example 以查看使用方式\n我的github: https://github.com/pha123661/Hok_tse_bun_tgbot", 0)
}
