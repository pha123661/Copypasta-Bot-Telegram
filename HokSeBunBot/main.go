package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var FILE_LOCATION string = "../HokSeBun_db"
var CACHE = make(map[string]string)

func delExtension(fileName string) string {
	// utility for removing file extension from filename
	if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
		return fileName[:pos]
	}
	return fileName
}

func build_cache() {
	// updates cache with existing files
	files, err := os.ReadDir(FILE_LOCATION)
	if err != nil {
		log.Println(err)
		return
	}

	for _, file := range files {
		text, _ := os.ReadFile(path.Join(FILE_LOCATION, file.Name()))
		CACHE[delExtension(file.Name())] = string(text) // text is []byte
	}
}

func handleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message.IsCommand() {
		// handle commands
		switch update.Message.Command() {
		case "echo":
			replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			replyMsg.ReplyToMessageID = update.Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
		case "new", "add": // new hok tse bun
			// find file name
			split_tmp := strings.Split(update.Message.Text, " ")
			if len(split_tmp) <= 2 {
				replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "錯誤：新增格式爲 “/new {關鍵字} {內容}”")
				replyMsg.ReplyToMessageID = update.Message.MessageID
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println(err)
				}
				return
			}

			// check file existence
			var filename string = split_tmp[1] + ".txt"
			var content string = update.Message.Text[len(update.Message.Command())+len(filename)-1:]
			content = strings.TrimSpace(content)
			if v, is_exist := CACHE[delExtension(filename)]; is_exist {
				if len(v) >= 100 {
					v = v[:100] + "……"
				}
				replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("「%s」複製文已存在：「%s」，確認是否覆蓋？", split_tmp[1], v))
				replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("是", fmt.Sprintf("%s %s", filename, content)),
						tgbotapi.NewInlineKeyboardButtonData("否", "NIL"),
					),
				)
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println(err)
				}
				return
			}
			// write file
			file, err := os.Create(path.Join(FILE_LOCATION, filename))
			if err != nil {
				log.Println(err)
			}
			file.WriteString(content)
			file.Close()

			// update cache
			CACHE[delExtension(filename)] = content

			// send response to user
			replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("新增複製文「%s」成功", delExtension(filename)))
			replyMsg.ReplyToMessageID = update.Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
		case "random":
			k := rand.Intn(len(CACHE))
			var context string
			for _, v := range CACHE {
				if k == 0 {
					context = v
				}
				k--
			}
			replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, context)
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
		}
	} else {
		// search hok tse bun
		for k := range CACHE {
			if strings.Contains(update.Message.Text, k) {
				// hit
				replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, CACHE[k])
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func main() {
	// keep alive
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprint(res, "Hello World!")
	})
	go http.ListenAndServe(":9000", nil)

	// initialize
	file, _ := os.OpenFile("log.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer file.Close()
	log.SetOutput(file)

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
		if update.Message != nil {
			handleUpdate(bot, update)
		} else if update.CallbackQuery != nil {
			if update.CallbackQuery.Data == "NIL" {
				replyMsg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "其實不按否也沒差啦 哈哈")
				replyMsg.ReplyToMessageID = update.CallbackQuery.Message.MessageID
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println(err)
				}
			} else {
				split_tmp := strings.Split(update.CallbackQuery.Data, " ")
				var filename string = split_tmp[0]
				var content string = strings.TrimSpace(split_tmp[1])
				// write file
				file, err := os.Create(path.Join(FILE_LOCATION, filename))
				if err != nil {
					panic(err)
				}
				file.WriteString(content)
				file.Close()

				// update cache
				CACHE[delExtension(filename)] = content

				// send response to user
				replyMsg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("更新複製文「%s」成功", delExtension(filename)))
				replyMsg.ReplyToMessageID = update.CallbackQuery.Message.MessageID
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println(err)
				}
			}
			editedMsg := tgbotapi.NewEditMessageReplyMarkup(
				update.CallbackQuery.Message.Chat.ID,
				update.CallbackQuery.Message.MessageID,
				tgbotapi.InlineKeyboardMarkup{
					InlineKeyboard: make([][]tgbotapi.InlineKeyboardButton, 0),
				},
			)
			bot.Send(editedMsg)
		}

	}

}
