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
var Queued_Overrides = make(map[string]*Queued_Overwrite_Entity)

type Queued_Overwrite_Entity struct {
	// attribute
	IsText  bool
	IsImage bool

	// file content
	TextContent string
	ImageFileID tgbotapi.FileID

	// status
	Done bool
}

func newOverwriteText(TextContent string) *Queued_Overwrite_Entity {
	data := Queued_Overwrite_Entity{
		IsText:      true,
		IsImage:     false,
		TextContent: TextContent,
		Done:        false,
	}
	return &data
}

func newOverwriteImage(ImageFileID tgbotapi.FileID) *Queued_Overwrite_Entity {
	data := Queued_Overwrite_Entity{
		IsText:      false,
		IsImage:     true,
		ImageFileID: ImageFileID,
		Done:        false,
	}
	return &data
}

func handleTextMessage(bot *tgbotapi.BotAPI, Message *tgbotapi.Message) {
	if Message.IsCommand() {
		// handle commands
		switch Message.Command() {
		case "start":
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, "歡迎使用，使用方式可以參考我的github: https://github.com/pha123661/Hok_tse_bun_tgbot")
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
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
			if HSB, is_exist := TEXT_CACHE[delExtension(filename)]; is_exist {
				old_content := trimString(HSB.content, 100)
				Queued_Overrides[filename] = newOverwriteText(content)
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
			TEXT_CACHE[delExtension(filename)] = HokSeBun{content: content, summarization: getSingleSummarization(filename, content, true)}

			// delete tmp message
			bot.Request(tgbotapi.NewDeleteMessage(Message.Chat.ID, to_be_delete_message_id))

			// send response to user
			replyMsg = tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("新增複製文「%s」成功，\n自動生成的摘要如下：「%s」", delExtension(filename), TEXT_CACHE[delExtension(filename)].summarization))
			replyMsg.ReplyToMessageID = Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}

		case "random": // random post
			k := rand.Intn(len(TEXT_CACHE))
			var keyword string
			var context string
			for key, v := range TEXT_CACHE {
				if k == 0 {
					keyword = key
					context = v.content
				}
				k--
			}
			context = fmt.Sprintf("幫你從 %d 篇文章中精心選擇了「%s」：\n%s", len(TEXT_CACHE), keyword, context)
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, context)
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
		case "search": // fuzzy search both filename & content
			var Query string = Message.CommandArguments()
			var ResultCount int
			if utf8.RuneCountInString(Query) < 2 {
				if _, err := bot.Send(tgbotapi.NewMessage(Message.Chat.ID, "搜尋關鍵字至少要兩個字！")); err != nil {
					log.Println(err)
				}
				return
			}
			if _, err := bot.Send(tgbotapi.NewMessage(Message.Chat.ID, "正在搜尋中…… 請稍後")); err != nil {
				log.Println(err)
			}
			if Message.Chat.ID != Message.From.ID {
				if _, err := bot.Send(tgbotapi.NewMessage(Message.From.ID, fmt.Sprintf("「%s」的搜尋結果如下：", Query))); err != nil {
					log.Println(err)
				}
			}

			// search text
			for Key, HSB := range TEXT_CACHE {
				if fuzzy.Match(Query, Key) || fuzzy.Match(Key, Query) || fuzzy.Match(Query, HSB.summarization) || fuzzy.Match(Query, HSB.content) {
					ResultCount++

					content := trimString(HSB.content, 100)
					if _, err := bot.Send(tgbotapi.NewMessage(Message.From.ID, fmt.Sprintf("名稱：「%s」\n摘要：「%s」\n內容：「%s」", Key, HSB.summarization, content))); err != nil {
						log.Println(err)
					}
				}
			}

			// search image
			for Key, HST := range IMAGE_CACHE {
				if fuzzy.Match(Query, Key) || fuzzy.Match(Key, Query) || fuzzy.Match(Query, HST.summarization) {
					ResultCount++

					PhotoConfig := tgbotapi.NewPhoto(Message.Chat.ID, HST.FileID)
					PhotoConfig.Caption = fmt.Sprintf("名稱：「%s」", Key)
					if _, err := bot.Request(PhotoConfig); err != nil {
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

		// search text
		go func() {
			SendTextResult := func(ChatID int64, Content string) {
				replyMsg := tgbotapi.NewMessage(ChatID, Content)
				if _, err := bot.Send(replyMsg); err != nil {
					log.Println(err)
				}
			}
			// fuzzy.Match("abc", "a1b2c3") = true
			// strings.Contains("AAABBBCCC", "AB") = true
			var Query = Message.Text
			var RuneLengthLimit int = Min(500, 100*utf8.RuneCountInString(Query))
			for Key, HSB := range TEXT_CACHE {
				switch {
				case utf8.RuneCountInString(Query) >= 3:
					if fuzzy.Match(Key, Query) || (fuzzy.Match(Query, Key) && Abs(len(Query)-len(Key)) <= 3) || fuzzy.Match(Query, HSB.summarization) {
						SendTextResult(Message.Chat.ID, TEXT_CACHE[Key].content)
						RuneLengthLimit -= utf8.RuneCountInString(TEXT_CACHE[Key].content)
					}
				case utf8.RuneCountInString(Query) >= 2:
					if strings.Contains(Query, Key) || strings.Contains(Key, Query) {
						SendTextResult(Message.Chat.ID, TEXT_CACHE[Key].content)
						RuneLengthLimit -= utf8.RuneCountInString(TEXT_CACHE[Key].content)
					}
				case utf8.RuneCountInString(Query) == 1:
					if utf8.RuneCountInString(Key) == 1 && Query == Key {
						SendTextResult(Message.Chat.ID, TEXT_CACHE[Key].content)
						RuneLengthLimit -= utf8.RuneCountInString(TEXT_CACHE[Key].content)
					}
				}
				if RuneLengthLimit <= 0 {
					break
				}
			}
		}()

		// search image
		go func() {
			SendImageResult := func(ChatID int64, FileID tgbotapi.FileID, Keyword string) {
				PhotoConfig := tgbotapi.NewPhoto(ChatID, FileID)
				PhotoConfig.Caption = Keyword
				if _, err := bot.Request(PhotoConfig); err != nil {
					log.Println(err)
				}
			}

			var Query = Message.Text
			var ImageCountLimit int = 2
			for Key, HST := range IMAGE_CACHE {
				switch {
				case utf8.RuneCountInString(Query) >= 3:
					if fuzzy.Match(Key, Query) || (fuzzy.Match(Query, Key) && Abs(len(Query)-len(Key)) <= 3) || fuzzy.Match(Query, HST.summarization) {
						SendImageResult(Message.Chat.ID, IMAGE_CACHE[Key].FileID, Key)
						ImageCountLimit--
					}
				case utf8.RuneCountInString(Query) >= 2:
					if strings.Contains(Query, Key) || strings.Contains(Key, Query) {
						SendImageResult(Message.Chat.ID, IMAGE_CACHE[Key].FileID, Key)
						ImageCountLimit--
					}
				case utf8.RuneCountInString(Query) == 1:
					if utf8.RuneCountInString(Key) == 1 && Query == Key {
						SendImageResult(Message.Chat.ID, IMAGE_CACHE[Key].FileID, Key)
						ImageCountLimit--
					}
				}
				if ImageCountLimit <= 0 {
					break
				}
			}
		}()
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, CallbackQuery *tgbotapi.CallbackQuery) {
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
		var Keyword string = CallbackQuery.Data
		var Entity = Queued_Overrides[Keyword]
		if Entity.Done {
			return
		}
		Queued_Overrides[Keyword].Done = true

		if Entity.IsText {
			var content = Entity.TextContent
			replyMsg := tgbotapi.NewMessage(CallbackQuery.Message.Chat.ID, "運算中，請稍後……")
			replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
			to_be_delete_message, err := bot.Send(replyMsg)
			if err != nil {
				log.Println(err)
			}

			to_be_delete_message_id := to_be_delete_message.MessageID

			// write file
			file, err := os.Create(path.Join(CONFIG.FILE_LOCATION, Keyword))
			if err != nil {
				log.Panicln(err)
			}
			file.WriteString(content)
			file.Close()

			// update cache
			TEXT_CACHE[delExtension(Keyword)] = HokSeBun{content: content, summarization: getSingleSummarization(Keyword, content, true)}

			// delete tmp message
			bot.Request(tgbotapi.NewDeleteMessage(CallbackQuery.Message.Chat.ID, to_be_delete_message_id))

			// send response to user
			replyMsg = tgbotapi.NewMessage(CallbackQuery.Message.Chat.ID, fmt.Sprintf("更新複製文「%s」成功，自動生成的摘要如下：「%s」", delExtension(Keyword), TEXT_CACHE[delExtension(Keyword)].summarization))
			replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
		} else if Entity.IsImage {
			var FileID = Entity.ImageFileID
			// add to cache
			addToo2Cache(Keyword, newToo(FileID))

			PhotoConfig := tgbotapi.NewPhoto(CallbackQuery.Message.Chat.ID, FileID)
			PhotoConfig.Caption = fmt.Sprintf("成功更新圖片「%s」", Keyword)
			if _, err := bot.Request(PhotoConfig); err != nil {
				log.Println(err)
			}
		}
	}
}

func init() {
	// initialize
	init_utils()
	// setup logging
	file, _ := os.OpenFile(CONFIG.LOG_FILE, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	log.SetOutput(file)
	log.Println("*** Starting Server ***")

	init_nlp()
	init_image()
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
		if update.Message != nil && update.Message.Photo != nil {
			handleImageMessage(bot, update.Message)
		} else if update.Message != nil {
			go handleTextMessage(bot, update.Message)
		} else if update.CallbackQuery != nil {
			go handleCallbackQuery(bot, update.CallbackQuery)
		}
	}
}
