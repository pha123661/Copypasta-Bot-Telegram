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
var Queued_Overrides = make(map[string]*Queued_Override_Entity)

type Queued_Override_Entity struct {
	content string
	done    bool
}

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
			var content string = strings.TrimSpace(Message.Text[len(Message.Command())+len(filename)-1:])
			if v, is_exist := CACHE[delExtension(filename)]; is_exist {
				old_content := trimString(v.content, 100)
				Queued_Overrides[filename] = &Queued_Override_Entity{content, false}
				replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("「%s」複製文已存在：「%s」，確認是否覆蓋？", Command_Args[0], old_content))
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
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, "運算中，請稍後……")
			replyMsg.ReplyToMessageID = Message.MessageID
			to_be_delete_message, err := bot.Send(replyMsg)
			if err != nil {
				log.Println(err)
			}
			to_be_delete_message_id := to_be_delete_message.MessageID

			// write file
			file, err := os.Create(path.Join(CONFIG.FILE_LOCATION, filename))
			if err != nil {
				log.Println(err)
			}
			file.WriteString(content)
			file.Close()

			// update cache
			CACHE[delExtension(filename)] = HokSeBun{content: content, summarization: getSingleSummarization(filename, content, true)}

			// delete tmp message
			bot.Request(tgbotapi.NewDeleteMessage(Message.Chat.ID, to_be_delete_message_id))

			// send response to user
			replyMsg = tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("新增複製文「%s」成功，\n自動生成的摘要如下：「%s」", delExtension(filename), CACHE[delExtension(filename)].summarization))
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
					context = v.content
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
			if utf8.RuneCountInString(Keyword) < 2 {
				if _, err := bot.Send(tgbotapi.NewMessage(Message.Chat.ID, "搜尋關鍵字至少要兩個字！")); err != nil {
					log.Println(err)
				}
				return
			}
			if _, err := bot.Send(tgbotapi.NewMessage(Message.Chat.ID, "正在搜尋中…… 請稍後")); err != nil {
				log.Println(err)
			}
			if Message.Chat.ID != Message.From.ID {
				if _, err := bot.Send(tgbotapi.NewMessage(Message.From.ID, fmt.Sprintf("「%s」的搜尋結果如下：", Keyword))); err != nil {
					log.Println(err)
				}
			}
			for k, v := range CACHE {
				if fuzzy.Match(Keyword, k) || fuzzy.Match(k, Keyword) || fuzzy.Match(Keyword, v.summarization) || fuzzy.Match(Keyword, v.content) {
					ResultCount++
					content := trimString(v.content, 100)
					if _, err := bot.Send(tgbotapi.NewMessage(Message.From.ID, fmt.Sprintf("名稱：「%s」\n摘要：「%s」\n內容：「%s」", k, v.summarization, content))); err != nil {
						log.Println(err)
					}
				}
			}
			if _, err := bot.Send(tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("搜尋完成，共 %d 筆吻合\n(結果在與bot的私訊中)", ResultCount))); err != nil {
				log.Println(err)
			}
			if Message.Chat.ID != Message.From.ID {
				if _, err := bot.Send(tgbotapi.NewMessage(Message.From.ID, fmt.Sprintf("搜尋完成，共 %d 筆吻合\n", ResultCount))); err != nil {
					log.Println(err)
				}
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
		if Message.Text == "" || Message.Text == " " {
			return
		}

		send := func(ChatID int64, Content string) {
			replyMsg := tgbotapi.NewMessage(ChatID, Content)
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
		}
		// fuzzy.Match("abc", "a1b2c3") = true
		// strings.Contains("AAABBBCCC", "AB") = true
		var runeLengthLimit int = Min(500, 100*utf8.RuneCountInString(Message.Text))
		for k, v := range CACHE {
			switch {
			case utf8.RuneCountInString(Message.Text) >= 3:
				if fuzzy.Match(k, Message.Text) || (fuzzy.Match(Message.Text, k) && Abs(len(Message.Text)-len(k)) <= 3) || fuzzy.Match(Message.Text, v.summarization) {
					send(Message.Chat.ID, CACHE[k].content)
					runeLengthLimit -= utf8.RuneCountInString(CACHE[k].content)
				}
			case utf8.RuneCountInString(Message.Text) >= 2:
				if strings.Contains(Message.Text, k) || strings.Contains(k, Message.Text) {
					send(Message.Chat.ID, CACHE[k].content)
					runeLengthLimit -= utf8.RuneCountInString(CACHE[k].content)
				}
			case utf8.RuneCountInString(Message.Text) == 1:
				if utf8.RuneCountInString(k) == 1 && Message.Text == k {
					send(Message.Chat.ID, CACHE[k].content)
					runeLengthLimit -= utf8.RuneCountInString(CACHE[k].content)
				}
			}
			if runeLengthLimit <= 0 {
				break
			}
		}
	}
}

func handleUpdateCallbackQuery(bot *tgbotapi.BotAPI, CallbackQuery *tgbotapi.CallbackQuery) {
	// close the inline keyboard
	editedMsg := tgbotapi.NewEditMessageReplyMarkup(
		CallbackQuery.Message.Chat.ID,
		CallbackQuery.Message.MessageID,
		tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: make([][]tgbotapi.InlineKeyboardButton, 0),
		},
	)
	bot.Send(editedMsg)
	if CallbackQuery.Data == "NIL" {
		// 否
		// show respond
		callback := tgbotapi.NewCallback(CallbackQuery.ID, "不覆蓋")
		if _, err := bot.Request(callback); err != nil {
			log.Println(err)
		}

		replyMsg := tgbotapi.NewMessage(CallbackQuery.Message.Chat.ID, "其實不按否也沒差啦 哈哈")
		replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
		if _, err := bot.Send(replyMsg); err != nil {
			log.Println(err)
		}
	} else {
		// 是
		// show respond
		callback := tgbotapi.NewCallback(CallbackQuery.ID, "正在覆蓋中……")
		if _, err := bot.Request(callback); err != nil {
			log.Println(err)
		}
		// over write existing files
		var filename string = CallbackQuery.Data
		var content string = Queued_Overrides[filename].content
		if Queued_Overrides[filename].done {
			return
		}
		Queued_Overrides[filename].done = true
		fmt.Println(filename, content)
		if utf8.RuneCountInString(content) >= 100 {
			replyMsg := tgbotapi.NewMessage(CallbackQuery.Message.Chat.ID, "運算中，請稍後……")
			replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
		}

		// write file
		file, err := os.Create(path.Join(CONFIG.FILE_LOCATION, filename))
		if err != nil {
			log.Panicln(err)
		}
		file.WriteString(content)
		file.Close()

		// update cache
		CACHE[delExtension(filename)] = HokSeBun{content: content, summarization: getSingleSummarization(filename, content, true)}

		// send response to user
		replyMsg := tgbotapi.NewMessage(CallbackQuery.Message.Chat.ID, fmt.Sprintf("更新複製文「%s」成功，自動生成的摘要如下：「%s」", delExtension(filename), CACHE[delExtension(filename)].summarization))
		replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
		if _, err := bot.Send(replyMsg); err != nil {
			log.Println(err)
		}
	}
}

func init() {
	// initialize
	init_utils()
	// setup logging
	file, _ := os.OpenFile(CONFIG.LOG_FILE, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer file.Close()
	log.SetOutput(file)
	log.Println("Starting Server")

	init_nlp()

	// build cache
	if _, err := os.Stat(CONFIG.FILE_LOCATION); os.IsNotExist(err) {
		os.Mkdir(CONFIG.FILE_LOCATION, 0755)
	}
	if _, err := os.Stat(CONFIG.SUMMARIZATION_LOCATION); os.IsNotExist(err) {
		os.Mkdir(CONFIG.SUMMARIZATION_LOCATION, 0755)
	}
	buildCache()

}

func main() {
	// keep alive
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprint(res, "Hello World!")
	})
	go http.ListenAndServe(":9000", nil)

	// start bot
	bot, err := tgbotapi.NewBotAPI(CONFIG.TELEGRAM_API_TOKEN)
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
		// ignore nil
		if update.Message != nil {
			go handleUpdateMessage(bot, update.Message)
		} else if update.CallbackQuery != nil {
			go handleUpdateCallbackQuery(bot, update.CallbackQuery)
		}
	}
}
