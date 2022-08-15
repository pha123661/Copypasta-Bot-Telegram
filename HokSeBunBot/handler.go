package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/lithammer/fuzzysearch/fuzzy"
	c "github.com/ostafen/clover/v2"
)

func handleCommand(bot *tgbotapi.BotAPI, Message *tgbotapi.Message) {
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
		var HTB *HokTseBun = &HokTseBun{}
		for idx, doc := range docs {
			if idx == RandomIndex {
				doc.Unmarshal(HTB)
				Keyword = HTB.Keyword
				Content = HTB.Content
				Type = HTB.Type
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
			// Queue changes
			Queued_Overwrites[Keyword] = &OverwriteEntity{
				Type:    1,
				Keyword: Keyword,
				Content: Content,
				From:    Message.From.ID,
				Done:    false,
			}

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
func handleTextMessage(bot *tgbotapi.BotAPI, Message *tgbotapi.Message) {
	if Message.Text == "" || Message.Text == " " {
		return
	}

	// asyc search
	go func() {
		// helper functions
		SendTextResult := func(ChatID int64, Content string) {
			replyMsg := tgbotapi.NewMessage(ChatID, Content)
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
		}
		SendImageResult := func(ChatID int64, Keyword string, Content string) {
			FileID := tgbotapi.FileID(Content)
			PhotoConfig := tgbotapi.NewPhoto(ChatID, FileID)
			PhotoConfig.Caption = Keyword
			if _, err := bot.Request(PhotoConfig); err != nil {
				log.Println(err)
			}
		}

		var (
			Query = Message.Text
			Limit = Min(500, 100*utf8.RuneCountInString(Query))
		)
		const RunesPerImage = 200
		docs, err := DB.FindAll(c.NewQuery(CONFIG.DB.COLLECTION))
		if err != nil {
			log.Println("[Normal]", err)
			return
		}

		for _, doc := range docs {
			HTB := &HokTseBun{}
			doc.Unmarshal(HTB)

			switch {
			case utf8.RuneCountInString(Query) >= 3:
				if fuzzy.Match(HTB.Keyword, Query) || (fuzzy.Match(Query, HTB.Keyword) && Abs(len(Query)-len(HTB.Keyword)) <= 3) || fuzzy.Match(Query, HTB.Summarization) {
					switch HTB.Type {
					case 1:
						// text
						go SendTextResult(Message.Chat.ID, HTB.Content)
						Limit -= utf8.RuneCountInString(HTB.Content)
					case 2:
						// image
						go SendImageResult(Message.Chat.ID, HTB.Keyword, HTB.Content)
						Limit -= RunesPerImage
					}
				}
			case utf8.RuneCountInString(Query) >= 2:
				if strings.Contains(Query, HTB.Keyword) || strings.Contains(HTB.Keyword, Query) {
					switch HTB.Type {
					case 1:
						// text
						go SendTextResult(Message.Chat.ID, HTB.Content)
						Limit -= utf8.RuneCountInString(HTB.Content)
					case 2:
						// image
						go SendImageResult(Message.Chat.ID, HTB.Keyword, HTB.Content)
						Limit -= RunesPerImage
					}
				}
			case utf8.RuneCountInString(Query) == 1:
				if utf8.RuneCountInString(HTB.Keyword) == 1 && Query == HTB.Keyword {
					switch HTB.Type {
					case 1:
						// text
						go SendTextResult(Message.Chat.ID, HTB.Content)
						Limit -= utf8.RuneCountInString(HTB.Content)
					case 2:
						// image
						go SendImageResult(Message.Chat.ID, HTB.Keyword, HTB.Content)
						Limit -= RunesPerImage
					}
				}
			}
			if Limit <= 0 {
				break
			}
		}
	}()
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
			log.Println("[newImage]", err)
		}
		return
	}

	InsertCP(Message.From.ID, Keyword, Content, 2)
	replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("新增圖片「%s」成功", Keyword))
	replyMsg.ReplyToMessageID = Message.MessageID
	if _, err := bot.Send(replyMsg); err != nil {
		log.Println("[newImage]", err)
	}

}

func handleCallbackQuery(bot *tgbotapi.BotAPI, CallbackQuery *tgbotapi.CallbackQuery) {
	// close the inline keyboard
	editMsg := tgbotapi.NewEditMessageReplyMarkup(
		CallbackQuery.Message.Chat.ID,
		CallbackQuery.Message.MessageID,
		tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: make([][]tgbotapi.InlineKeyboardButton, 0),
		},
	)
	bot.Send(editMsg)
	if CallbackQuery.Data == "NIL" {
		// 否
		// show respond
		callback := tgbotapi.NewCallback(CallbackQuery.ID, "不新增")
		if _, err := bot.Request(callback); err != nil {
			log.Println("[CallBQ]", err)
		}

		replyMsg := tgbotapi.NewMessage(CallbackQuery.Message.Chat.ID, "其實不按否也沒差啦 哈哈")
		replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
		if _, err := bot.Send(replyMsg); err != nil {
			log.Println("[CallBQ]", err)
		}
	} else {
		// 是
		// show respond
		callback := tgbotapi.NewCallback(CallbackQuery.ID, "正在新增中……")
		if _, err := bot.Request(callback); err != nil {
			log.Println("[CallBQ]", err)
		}

		OW_Entity := Queued_Overwrites[CallbackQuery.Data]
		if OW_Entity.Done {
			return
		}
		OW_Entity.Done = true

		replyMsg := tgbotapi.NewMessage(CallbackQuery.Message.Chat.ID, "運算中，請稍後……")
		replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
		to_be_delete_message, err := bot.Send(replyMsg)
		if err != nil {
			log.Println("[CallBQ]", err)
		}
		to_be_delete_message_id := to_be_delete_message.MessageID

		Sum, err := InsertCP(
			OW_Entity.From,
			OW_Entity.Keyword,
			OW_Entity.Content,
			OW_Entity.Type,
		)
		if err != nil {
			log.Println("[CallBQ]", err)
		}

		// delete tmp message
		bot.Request(tgbotapi.NewDeleteMessage(CallbackQuery.Message.Chat.ID, to_be_delete_message_id))

		// send response to user
		replyMsg = tgbotapi.NewMessage(CallbackQuery.Message.Chat.ID, fmt.Sprintf("更新複製文「%s」成功，自動生成的摘要如下：「%s」", OW_Entity.Keyword, Sum))
		replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
		if _, err := bot.Send(replyMsg); err != nil {
			log.Println("[CallBQ]", err)
		}
	}
}
