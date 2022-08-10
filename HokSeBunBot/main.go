package main

import (
	"fmt"
	"os"
	"path"
	"strings"

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
		case "new": // 複製文
			// write new file
			split_tmp := strings.Split(update.Message.Text, " ")
			if len(split_tmp) <= 2 {
				return
			}
			fmt.Println(split_tmp)
			var filename string = split_tmp[1] + ".txt"
			fmt.Println(path.Join(FILE_LOCATION, filename))
			file, err := os.Create(path.Join(FILE_LOCATION, filename))
			if err != nil {
				panic(err)
			}
			file.WriteString(update.Message.Text[len(update.Message.Command())+len(filename)-1:])
			file.Close()

			// send responce to user
			replyMsg.Text = fmt.Sprintf("新增複製文「%s」成功", filename[:len(filename)-4])
			replyMsg.ReplyToMessageID = update.Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				fmt.Println(err)
			}
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
