package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	fuzzy "github.com/lithammer/fuzzysearch/fuzzy"
)

// for override confirm
// "existed_filename.txt": "new content"
var Queued_Overrides = make(map[string]string)

func handleUpdateMessage(bot *tgbotapi.BotAPI, Message *tgbotapi.Message) {
	if Message.IsCommand() {
		// handle commands
		switch Message.Command() {
		case "echo": // echo
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, Message.CommandArguments())
			replyMsg.ReplyToMessageID = Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
		case "new", "add": // new hok tse bun
			// find file name
			Command_Args := strings.Split(Message.CommandArguments(), " ")
			if len(Command_Args) <= 1 {
				replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("錯誤：新增格式爲 “/%s {關鍵字} {內容}”", Message.Command()))
				replyMsg.ReplyToMessageID = Message.MessageID
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println(err)
				}
				return
			}
			if Command_Args[0] == " " || Command_Args[0] == "" {
				replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("好好打字啦 /%s後面一個空格就夠了", Message.Command()))
				replyMsg.ReplyToMessageID = Message.MessageID
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println(err)
				}
				return
			}

			// check file existence
			var filename string = Command_Args[0] + ".txt"
			var content string = strings.TrimSpace(Command_Args[1])
			if v, is_exist := CACHE[delExtension(filename)]; is_exist {
				if utf8.RuneCountInString(v) >= 100 {
					r := []rune(v)[:100]
					v = string(r) + "……"
				}
				Queued_Overrides[filename] = content
				replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("「%s」複製文已存在：「%s」，確認是否覆蓋？", Command_Args[0], v))
				replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("是", filename),
						tgbotapi.NewInlineKeyboardButtonData("否", "NIL"),
					),
				)
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println(err)
				}
				return
			}
			// write file
			file, err := os.Create(path.Join(CONFIG.FILE_LOCATION, filename))
			if err != nil {
				log.Println(err)
			}
			file.WriteString(content)
			file.Close()

			// update cache
			CACHE[delExtension(filename)] = content

			// send response to user
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("新增複製文「%s」成功", delExtension(filename)))
			replyMsg.ReplyToMessageID = Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
		case "random": // random post
			k := rand.Intn(len(CACHE))
			var keyword string
			var context string
			for key, v := range CACHE {
				if k == 0 {
					keyword = key
					context = v
				}
				k--
			}
			context = fmt.Sprintf("幫你從 %d 篇文章中精心選擇了「%s」：\n%s", len(CACHE), keyword, context)
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, context)
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
		case "search": // fuzzy search both filename & content
			var Keyword string = Message.CommandArguments()
			var ResultCount int
			if _, err := bot.Send(tgbotapi.NewMessage(Message.Chat.ID, "正在搜尋中…… 請稍後")); err != nil {
				log.Println(err)
			}
			for k, v := range CACHE {
				if fuzzy.Match(Keyword, k) || fuzzy.Match(Keyword, v) {
					ResultCount++
					if _, err := bot.Send(tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("「%s」：「%s」", k, v))); err != nil {
						log.Println(err)
					}
				}
			}
			if _, err := bot.Send(tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("搜尋完成，共 %d 筆吻合", ResultCount))); err != nil {
				log.Println(err)
			}

		default:
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("錯誤：我不會 “/%s” 啦", Message.Command()))
			replyMsg.ReplyToMessageID = Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
			return
		}
	} else {
		// search hok tse bun
		for k := range CACHE {
			if fuzzy.Match(k, Message.Text) {
				// hit
				replyMsg := tgbotapi.NewMessage(Message.Chat.ID, CACHE[k])
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func handleUpdateCallbackQuery(bot *tgbotapi.BotAPI, CallbackQuery *tgbotapi.CallbackQuery) {
	if CallbackQuery.Data == "NIL" {
		replyMsg := tgbotapi.NewMessage(CallbackQuery.Message.Chat.ID, "其實不按否也沒差啦 哈哈")
		replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
		if _, err := bot.Send(replyMsg); err != nil {
			log.Println(err)
		}
	} else {
		var filename string = CallbackQuery.Data
		var content string = Queued_Overrides[filename]
		// write file
		file, err := os.Create(path.Join(CONFIG.FILE_LOCATION, filename))
		if err != nil {
			panic(err)
		}
		file.WriteString(content)
		file.Close()

		// update cache
		CACHE[delExtension(filename)] = content

		// send response to user
		replyMsg := tgbotapi.NewMessage(CallbackQuery.Message.Chat.ID, fmt.Sprintf("更新複製文「%s」成功", delExtension(filename)))
		replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
		if _, err := bot.Send(replyMsg); err != nil {
			log.Println(err)
		}
	}
	editedMsg := tgbotapi.NewEditMessageReplyMarkup(
		CallbackQuery.Message.Chat.ID,
		CallbackQuery.Message.MessageID,
		tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: make([][]tgbotapi.InlineKeyboardButton, 0),
		},
	)
	bot.Send(editedMsg)
}

func main() {
	// keep alive
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprint(res, "Hello World!")
	})
	go http.ListenAndServe(":9000", nil)

	// initialize
	// setup logging
	file, _ := os.OpenFile("log.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer file.Close()
	log.SetOutput(file)
	// read config
	CONFIG = initConfig("../config.toml")
	// build cache
	if _, err := os.Stat(CONFIG.FILE_LOCATION); os.IsNotExist(err) {
		os.Mkdir(CONFIG.FILE_LOCATION, 0755)
	}
	build_cache()

	// start bot
	bot, err := tgbotapi.NewBotAPI(CONFIG.TELEGRAM_API_TOKEN)
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
			go handleUpdateMessage(bot, update.Message)
		} else if update.CallbackQuery != nil {
			go handleUpdateCallbackQuery(bot, update.CallbackQuery)
		}

	}

}
