package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
	log.Println("*** Starting Server ***")
	InitDB()
	InitNLP()
}

func main() {
	// keep alive
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprint(res, "Hello World!")
	})
	go http.ListenAndServe(":9000", nil)

	var err error
	// start bot
	bot, err = tgbotapi.NewBotAPI(CONFIG.API.TG.TOKEN)
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
				go handleImageMessage(update.Message)
			} else {
				if update.Message.IsCommand() {
					// handle commands
					go handleCommand(update.Message)
				} else {
					// handle text updates
					go handleTextMessage(update.Message)
				}
			}
		case update.CallbackQuery != nil:
			// handle callback query
			go handleCallbackQuery(update.CallbackQuery)
		}
	}
}
