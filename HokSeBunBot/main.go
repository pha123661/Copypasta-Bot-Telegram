package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var FILE_LOCATION string = "../HokSeBun_db"
var CACHE = make(map[string]string)

func build_cache() {
	// updates cache with existing files
	files, err := ioutil.ReadDir(FILE_LOCATION)
	if err != nil {
		fmt.Println(err)
		return
	}

	// utility for removing file extension from filename
	delExtension := func(fileName string) string {
		if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
			return fileName[:pos]
		}
		return fileName
	}

	for _, file := range files {
		text, _ := os.ReadFile(path.Join(FILE_LOCATION, file.Name()))
		CACHE[delExtension(file.Name())] = string(text) // text is []byte
	}
	fmt.Println(CACHE)
}

func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message.IsCommand() {
		// handle commands
		switch update.Message.Command() {
		case "echo":
			replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			replyMsg.ReplyToMessageID = update.Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				fmt.Println(err)
			}
		case "new": // new hok tse bun
			// find file name
			split_tmp := strings.Split(update.Message.Text, " ")
			if len(split_tmp) <= 2 {
				return
			}

			// write file
			var filename string = split_tmp[1] + ".txt"
			file, err := os.Create(path.Join(FILE_LOCATION, filename))
			if err != nil {
				panic(err)
			}
			var content string = update.Message.Text[len(update.Message.Command())+len(filename)-1:]
			file.WriteString(content)
			file.Close()

			// update cache
			CACHE[split_tmp[1]] = content

			// send response to user
			replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("新增複製文「%s」成功", filename[:len(filename)-4]))
			replyMsg.ReplyToMessageID = update.Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				fmt.Println(err)
			}
		}
	} else {
		// search hok tse bun
		for k := range CACHE {
			if strings.Contains(update.Message.Text, k) {
				// hit
				replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, CACHE[k])
				if _, err := bot.Send(replyMsg); err != nil {
					fmt.Println(err)
				}
				break
			}
		}
	}
}

func main() {
	// initialize
	if _, err := os.Stat(FILE_LOCATION); os.IsNotExist(err) {
		os.Mkdir(FILE_LOCATION, 0755)
	}
	build_cache()
	// start bot
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
