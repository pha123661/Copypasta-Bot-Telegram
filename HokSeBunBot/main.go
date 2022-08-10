package main

import (
	"fmt"
	"os"
	"path"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var FILE_LOCATION string = "../HokSeBun_db"

func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	if update.Message.IsCommand() {
		switch update.Message.Command() {
		case "echo":
			replyMsg.Text = update.Message.Text
			replyMsg.ReplyToMessageID = update.Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				fmt.Println(err)
			}
		default: // 複製文
			command := update.Message.Command()
			fmt.Println(path.Join(FILE_LOCATION, command+".txt"))
			file, err := os.Create(path.Join(FILE_LOCATION, command+".txt"))
			if err != nil {
				panic(err)
			}
			file.WriteString(update.Message.Text[len(command)+2:])
			file.Close()
		}

	}
}

func main() {
	if _, err := os.Stat(FILE_LOCATION); os.IsNotExist(err) {
		os.Mkdir(FILE_LOCATION, 0755)
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_API_TOKEN"))
	if err != nil {
		panic(err)
	}
	bot.Debug = true
	fmt.Println("***", "Sucessful logged in as", bot.Self.UserName, "***")

	// update config
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	// get messages
	updates := bot.GetUpdatesChan(updateConfig)
	for update := range updates {
		// ignore nil
		if update.Message == nil {
			continue
		}
		handleUpdate(bot, update)
	}

}
