package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/lithammer/fuzzysearch/fuzzy"
	c "github.com/ostafen/clover/v2"
)

// Deletes[ChatID]["UID"] = smth
var Deletes = make(map[int64]map[string]*DeleteEntity)

type DeleteEntity struct {
	ToBeDeleteMessageIDs []int // pointer is used to modify it afterwards
	Keyword              string
	Confirmed            bool
	Done                 bool
}

func handleCommand(Message *tgbotapi.Message) {
	// handle commands
	switch Message.Command() {
	case "start":
		// Startup
		SendText(Message.Chat.ID, "歡迎使用，使用方式可以參考我的github: https://github.com/pha123661/Hok_tse_bun_tgbot", 0)
	case "echo":
		// Echo
		SendText(Message.Chat.ID, Message.CommandArguments(), Message.MessageID)
	case "random", "randomImage", "randomText":
		var Query *c.Query
		switch Message.Command() {
		case "random":
			Query = c.NewQuery(CONFIG.DB.COLLECTION)
		case "randomImage":
			Criteria := c.Field("Type").Eq(2)
			Query = c.NewQuery(CONFIG.DB.COLLECTION).Where(Criteria)
		case "randomText":
			Criteria := c.Field("Type").Eq(1)
			Query = c.NewQuery(CONFIG.DB.COLLECTION).Where(Criteria)
		default:
			Query = c.NewQuery(CONFIG.DB.COLLECTION)
		}
		docs, err := DB.FindAll(Query)
		if err != nil {
			log.Println("[random]", err)
			return
		}
		if len(docs) <= 0 {
			SendText(Message.Chat.ID, "資料庫沒東西是在抽屁", 0)
			return
		}
		RandomIndex := rand.Intn(len(docs))

		var HTB *HokTseBun = &HokTseBun{}
		for idx, doc := range docs {
			if idx == RandomIndex {
				doc.Unmarshal(HTB)
				break
			}
		}
		switch {
		case HTB.IsText():
			SendText(Message.Chat.ID, fmt.Sprintf("幫你從 %d 坨大便中精心選擇了「%s」：\n%s", len(docs), HTB.Keyword, HTB.Content), 0)
		case HTB.IsImage():
			SendMultiMedia(Message.Chat.ID, fmt.Sprintf("幫你從 %d 坨大便中精心選擇了「%s」", len(docs), HTB.Keyword), HTB.Content, HTB.Type)
		}

	case "new", "add": // new hok tse bun
		// Parse command
		Command_Args := strings.Fields(Message.CommandArguments())
		if len(Command_Args) <= 1 {
			SendText(Message.Chat.ID, fmt.Sprintf("錯誤：新增格式爲 “/%s {關鍵字} {內容}”", Message.Command()), Message.MessageID)
			return
		}
		var Keyword string = Command_Args[0]
		var Content string = strings.TrimSpace(Message.Text[strings.Index(Message.Text, Command_Args[1]):])

		if utf8.RuneCountInString(Keyword) >= 30 {
			SendText(Message.Chat.ID, fmt.Sprintf("關鍵字長度不可大於 30, 目前爲 %d 字”", utf8.RuneCountInString(Keyword)), Message.MessageID)
			return
		}

		// find existing images
		Criteria := c.Field("Keyword").Eq(Keyword).And(c.Field("Content").Eq(Content))
		if doc, _ := DB.FindFirst(c.NewQuery(CONFIG.DB.COLLECTION).Where(Criteria)); doc != nil {
			SendText(Message.Chat.ID, "傳過了啦 腦霧?", Message.MessageID)
			return
		}

		// Create tmp message
		to_be_delete_message := SendText(Message.Chat.ID, "運算中，請稍後……", Message.MessageID)
		// Insert HTB
		var Sum = TextSummarization(Keyword, Content)
		var err error
		// Delete tmp message
		bot.Request(tgbotapi.NewDeleteMessage(Message.Chat.ID, to_be_delete_message.MessageID))
		_, err = InsertHTB(
			CONFIG.DB.COLLECTION,
			&HokTseBun{
				Type:          1,
				Keyword:       Keyword,
				Summarization: Sum,
				Content:       Content,
				URL:           "", // text
				From:          Message.From.ID,
			},
		)
		// send response to user
		if err != nil {
			log.Println("[new]", err)
			SendText(Message.Chat.ID, fmt.Sprintf("新增複製文「%s」失敗：%s", Keyword, err), Message.MessageID)
		} else {
			SendText(Message.Chat.ID, fmt.Sprintf("新增複製文「%s」成功，\n自動生成的摘要如下：「%s」", Keyword, Sum), Message.MessageID)
		}
	case "search":
		var (
			Query       string = Message.CommandArguments()
			ResultCount int    = 0
			MaxResults         = 25
		)

		SendText(Message.From.ID, fmt.Sprintf("「%s」的搜尋結果如下：", Query), 0)

		if Message.Chat.ID != Message.From.ID {
			SendText(Message.Chat.ID, "正在搜尋中…… 請稍後", 0)
		}

		// search
		docs, _ := DB.FindAll(c.NewQuery(CONFIG.DB.COLLECTION))
		HTB := &HokTseBun{}
		for _, doc := range docs {
			if ResultCount >= MaxResults {
				ResultCount++
				break
			}
			doc.Unmarshal(HTB)
			if fuzzy.Match(Query, HTB.Keyword) || fuzzy.Match(HTB.Keyword, Query) || fuzzy.Match(Query, HTB.Summarization) || (fuzzy.Match(Query, HTB.Content) && HTB.IsText()) {
				switch {
				case HTB.IsText():
					SendText(Message.From.ID, fmt.Sprintf("名稱：「%s」\n摘要：「%s」\n內容：「%s」", HTB.Keyword, HTB.Summarization, HTB.Content), 0)
				case HTB.IsMultiMedia():
					SendMultiMedia(Message.From.ID, fmt.Sprintf("名稱：「%s」\n描述：「%s」", HTB.Keyword, HTB.Summarization), HTB.Content, HTB.Type)
				}
				ResultCount++
			}
		}

		if ResultCount <= MaxResults {
			SendText(Message.From.ID, fmt.Sprintf("搜尋完成，共 %d 筆吻合\n", ResultCount), 0)
			if Message.Chat.ID != Message.From.ID {
				SendText(Message.Chat.ID, fmt.Sprintf("搜尋完成，共 %d 筆吻合\n(結果在與bot的私訊中)", ResultCount), 0)
			}
		} else {
			SendText(Message.From.ID, fmt.Sprintf("搜尋完成，結果超過上限%d筆，請嘗試更換關鍵字", MaxResults), 0)
			if Message.Chat.ID != Message.From.ID {
				SendText(Message.Chat.ID, fmt.Sprintf("搜尋完成，結果超過上限%d筆，請嘗試更換關鍵字\n(結果在與bot的私訊中)", MaxResults), 0)
			}
		}
	case "delete":
		var BeDeletedKeyword = Message.CommandArguments()
		if BeDeletedKeyword == "" {
			SendText(Message.Chat.ID, "請輸入關鍵字", Message.MessageID)
			return
		}
		Criteria := c.Field("Keyword").Eq(BeDeletedKeyword)
		docs, err := DB.FindAll(c.NewQuery(CONFIG.DB.COLLECTION).Where(Criteria))
		if err != nil {
			log.Println("[delete]", err)
		}
		if len(docs) <= 0 {
			SendText(Message.Chat.ID, "沒有文章符合關鍵字", Message.MessageID)
			return
		}

		ReplyMarkup := make([][]tgbotapi.InlineKeyboardButton, 0, len(docs))
		HTB := &HokTseBun{}

		tmp_modified_entries := make([]*DeleteEntity, 0, len(docs))

		for idx, doc := range docs {
			doc.Unmarshal(HTB)
			var ShowEntry string
			switch {
			case HTB.IsText():
				ShowEntry = fmt.Sprintf("%d. %s", idx+1, TruncateString(HTB.Content, 20))
			case HTB.IsImage():
				type_prompt := "圖片："
				ShowEntry = fmt.Sprintf("%d. %s%s", idx+1, type_prompt, TruncateString(HTB.Summarization, 15-utf8.RuneCountInString(type_prompt)))
			case !HTB.IsImage() && HTB.IsMultiMedia():
				type_prompt := "動圖："
				ShowEntry = fmt.Sprintf("%d. %s%s", idx+1, type_prompt, TruncateString(HTB.Summarization, 15-utf8.RuneCountInString(type_prompt)))
			}
			ReplyMarkup = append(ReplyMarkup, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(ShowEntry, HTB.UID)))

			if _, ok := Deletes[Message.Chat.ID]; !ok {
				Deletes[Message.Chat.ID] = make(map[string]*DeleteEntity)
			}
			Deletes[Message.Chat.ID][HTB.UID] = &DeleteEntity{Keyword: HTB.Keyword, ToBeDeleteMessageIDs: make([]int, 0, 5)}
			tmp_modified_entries = append(tmp_modified_entries, Deletes[Message.Chat.ID][HTB.UID])
		}
		ReplyMarkup = append(ReplyMarkup, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("取消", "NIL")))
		replyMsg := tgbotapi.NewMessage(Message.Chat.ID, "請選擇要刪除以下哪一篇文章？")
		replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(ReplyMarkup...)

		Msg, err := bot.Send(replyMsg)
		if err != nil {
			log.Println("[delete]", err)
		}
		for _, DE := range tmp_modified_entries {
			DE.ToBeDeleteMessageIDs = append(DE.ToBeDeleteMessageIDs, Msg.MessageID)
		}
		fmt.Printf("%+v\n", Deletes)
		for _, v := range Deletes {
			for _, vv := range v {
				fmt.Printf("%+v\n", vv)
			}
		}
	default:
		SendText(Message.Chat.ID, fmt.Sprintf("錯誤：我不會 “/%s” 啦", Message.Command()), Message.MessageID)
	}
}

func handleTextMessage(Message *tgbotapi.Message) {
	if Message.Text == "" || Message.Text == " " {
		return
	}

	// asyc search
	go func() {
		var (
			Query         = Message.Text
			Limit         = Min(500, 100*utf8.RuneCountInString(Query))
			RunesPerImage = 200
		)

		docs, err := DB.FindAll(c.NewQuery(CONFIG.DB.COLLECTION))
		if err != nil {
			log.Println("[Normal]", err)
			return
		}

		for _, doc := range docs {
			HTB := &HokTseBun{}
			doc.Unmarshal(HTB)

			HIT := false
			switch {
			case utf8.RuneCountInString(Query) >= 3:
				if fuzzy.Match(HTB.Keyword, Query) || (fuzzy.Match(Query, HTB.Keyword) && Abs(len(Query)-len(HTB.Keyword)) <= 3) || fuzzy.Match(Query, HTB.Summarization) {
					HIT = true
				}
			case utf8.RuneCountInString(Query) >= 2:
				if strings.Contains(Query, HTB.Keyword) || strings.Contains(HTB.Keyword, Query) {
					HIT = true
				}
			case utf8.RuneCountInString(Query) == 1:
				if utf8.RuneCountInString(HTB.Keyword) == 1 && Query == HTB.Keyword {
					HIT = true
				}
			}
			if HIT {
				switch {
				case HTB.IsText():
					// text
					go SendText(Message.Chat.ID, HTB.Content, 0)
					Limit -= utf8.RuneCountInString(HTB.Content)
				case HTB.IsMultiMedia():
					// image
					go SendMultiMedia(Message.Chat.ID, "", HTB.Content, HTB.Type)
					Limit -= RunesPerImage
				}
			}

			if Limit <= 0 {
				break
			}
		}
	}()
}

func handleImageMessage(Message *tgbotapi.Message) {
	if Message.Caption == "" {
		return
	}

	var (
		Keyword     string = strings.TrimSpace(Message.Caption)
		Content     string
		max_area    int = 0
		MaxFileSize int = 20 * 1000 * 1000
		FileSize    int
	)

	for _, image := range Message.Photo {
		if image.Width*image.Height >= max_area {
			max_area = image.Width * image.Height
			Content = image.FileID
			FileSize = image.FileSize
		}
	}
	if FileSize >= MaxFileSize {
		SendText(
			Message.Chat.ID,
			fmt.Sprintf("新增失敗，目前檔案大小爲 %.2f MB，檔案大小上限爲 %.2f MB", float32(FileSize)/1000.0/1000.0, float32(MaxFileSize)/1000.0/1000.0),
			Message.MessageID,
		)
		return
	}

	// find existing images
	Criteria := c.Field("Keyword").Eq(Keyword).And(c.Field("Content").Eq(Content))
	if doc, _ := DB.FindFirst(c.NewQuery(CONFIG.DB.COLLECTION).Where(Criteria)); doc != nil {
		SendText(Message.Chat.ID, "傳過了啦 腦霧?", Message.MessageID)
		return
	}

	var Cap string
	// Send tmp message
	to_be_delete_message := SendText(Message.Chat.ID, "運算中，請稍後……", Message.MessageID)
	URL, err := bot.GetFileDirectURL(Content)
	if err != nil {
		log.Println("[HandleImg]", err)
		Cap = ""
	} else {
		Cap = ImageCaptioning(Keyword, URL)
	}
	// Delete tmp message
	bot.Request(tgbotapi.NewDeleteMessage(Message.Chat.ID, to_be_delete_message.MessageID))

	_, err = InsertHTB(
		CONFIG.DB.COLLECTION,
		&HokTseBun{
			Type:          2,
			Keyword:       Keyword,
			Summarization: Cap,
			Content:       Content,
			URL:           URL,
			From:          Message.From.ID,
		},
	)
	// send response to user
	if err != nil {
		log.Println("[HandleImg]", err)
		SendText(Message.Chat.ID, fmt.Sprintf("新增圖片「%s」失敗：%s", Keyword, err), Message.MessageID)
	} else {
		SendText(Message.Chat.ID, fmt.Sprintf("新增圖片「%s」成功，\n自動生成的描述如下：「%s」", Keyword, Cap), Message.MessageID)
	}

}

func handleAnimatedMessage(Message *tgbotapi.Message) {
	if Message.Caption == "" {
		return
	}
	// check file size
	var MaxFileSize int = 20 * 1000 * 1000
	var FileSize int
	if Message.Animation != nil {
		FileSize = Message.Animation.FileSize
	} else if Message.Video != nil {
		FileSize = Message.Video.FileSize
	}
	if FileSize >= MaxFileSize {
		SendText(
			Message.Chat.ID,
			fmt.Sprintf("新增失敗，目前檔案大小爲 %.2f MB，檔案大小上限爲 %.2f MB", float32(FileSize)/1000.0/1000.0, float32(MaxFileSize)/1000.0/1000.0),
			Message.MessageID,
		)
		return
	}

	var (
		Keyword string = strings.TrimSpace(Message.Caption)
		Content string
		Type    int
		URL     string
		Cap     string
		err     error
	)
	// Send tmp message
	to_be_delete_message := SendText(Message.Chat.ID, "運算中，請稍後……", Message.MessageID)

	switch {
	case Message.Animation != nil:
		Content = Message.Animation.FileID
		Type = 3
		// get url
		URL, err = bot.GetFileDirectURL(Content)
		if err != nil {
			log.Println("[handleAnima]", err)
		}
		// get caption by thumbnail
		Thumb_URL, err := bot.GetFileDirectURL(Message.Animation.Thumbnail.FileID)
		if err != nil {
			log.Println("[handleAnima]", err)
		}
		Cap = ImageCaptioning(Keyword, Thumb_URL)

	case Message.Video != nil:
		Content = Message.Video.FileID
		Type = 4
		// get url
		URL, err = bot.GetFileDirectURL(Content)
		if err != nil {
			log.Println("[handleAnima]", err)
		}
		// get caption by thumbnail
		Thumb_URL, err := bot.GetFileDirectURL(Message.Video.Thumbnail.FileID)
		if err != nil {
			log.Println("[handleAnima]", err)
		}
		Cap = ImageCaptioning(Keyword, Thumb_URL)
	}
	// Delete tmp message
	bot.Request(tgbotapi.NewDeleteMessage(Message.Chat.ID, to_be_delete_message.MessageID))

	_, err = InsertHTB(
		CONFIG.DB.COLLECTION,
		&HokTseBun{
			Type:          Type,
			Keyword:       Keyword,
			Summarization: Cap,
			Content:       Content,
			URL:           URL,
			From:          Message.From.ID,
		},
	)

	if err != nil {
		SendText(Message.Chat.ID, fmt.Sprintf("新增動圖「%s」失敗：%s", Keyword, err), Message.MessageID)
	} else {
		SendText(Message.Chat.ID, fmt.Sprintf("新增動圖「%s」成功，\n自動生成的描述如下：「%s」", Keyword, Cap), Message.MessageID)
	}
}

func handleCallbackQuery(CallbackQuery *tgbotapi.CallbackQuery) {
	// close the inline keyboard
	editMsg := tgbotapi.NewEditMessageReplyMarkup(
		CallbackQuery.Message.Chat.ID,
		CallbackQuery.Message.MessageID,
		tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: make([][]tgbotapi.InlineKeyboardButton, 0),
		},
	)
	fmt.Println(CallbackQuery.Message.Chat.ID)
	fmt.Println(CallbackQuery.Data)
	fmt.Printf("%+v\n", Deletes[CallbackQuery.Message.Chat.ID][CallbackQuery.Data])
	if _, err := bot.Send(editMsg); err != nil {
		log.Println("[CallQ]", err)
	}
	if CallbackQuery.Data == "NIL" {
		// 否
		// show respond
		callback := tgbotapi.NewCallback(CallbackQuery.ID, "不新增")
		if _, err := bot.Request(callback); err != nil {
			log.Println("[CallBQ]", err)
		}
		SendText(CallbackQuery.Message.Chat.ID, "其實不按也沒差啦 哈哈", 0)
	} else if DEntity, ok := Deletes[CallbackQuery.Message.Chat.ID][CallbackQuery.Data]; ok {
		var UID = CallbackQuery.Data
		if !DEntity.Confirmed {
			DEntity.Confirmed = true
			// find HTB
			doc, err := DB.FindById(CONFIG.DB.COLLECTION, UID)
			if err != nil {
				log.Println("[CallBQ]", err)
				return
			}
			if doc == nil {
				return
			}
			HTB := &HokTseBun{}
			doc.Unmarshal(HTB)

			raw_json := struct {
				MID int `json:"message_id"`
			}{}
			// send confirmation
			switch HTB.Type {
			case 1:
				replyMsg := tgbotapi.NewMessage(CallbackQuery.Message.Chat.ID, fmt.Sprintf("請再次確認是否要刪除「%s」：\n「%s」？", HTB.Keyword, HTB.Content))
				replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
				replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("是", UID),
						tgbotapi.NewInlineKeyboardButtonData("否", "NIL"),
					),
				)
				Msg, err := bot.Send(replyMsg)
				if err != nil {
					log.Println("[CallBQ]", err)
				}
				DEntity.ToBeDeleteMessageIDs = append(DEntity.ToBeDeleteMessageIDs, Msg.MessageID)

			case 2:
				Config := tgbotapi.NewPhoto(CallbackQuery.Message.Chat.ID, tgbotapi.FileID(HTB.Content))
				Config.Caption = fmt.Sprintf("請再次確認是否要刪除「%s」？", HTB.Keyword)
				Config.ReplyToMessageID = CallbackQuery.Message.MessageID
				Config.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("是", UID),
						tgbotapi.NewInlineKeyboardButtonData("否", "NIL"),
					),
				)
				resp, err := bot.Request(Config)
				if err != nil {
					log.Println("[CallBQ]", err)
				}
				json.Unmarshal(resp.Result, &raw_json)
				DEntity.ToBeDeleteMessageIDs = append(DEntity.ToBeDeleteMessageIDs, raw_json.MID)

			case 3:
				Config := tgbotapi.NewAnimation(CallbackQuery.Message.Chat.ID, tgbotapi.FileID(HTB.Content))
				Config.Caption = fmt.Sprintf("請再次確認是否要刪除「%s」？", HTB.Keyword)
				Config.ReplyToMessageID = CallbackQuery.Message.MessageID
				Config.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("是", UID),
						tgbotapi.NewInlineKeyboardButtonData("否", "NIL"),
					),
				)
				resp, err := bot.Request(Config)
				if err != nil {
					log.Println("[CallBQ]", err)
				}
				json.Unmarshal(resp.Result, &raw_json)
				DEntity.ToBeDeleteMessageIDs = append(DEntity.ToBeDeleteMessageIDs, raw_json.MID)

			case 4:
				Config := tgbotapi.NewVideo(CallbackQuery.Message.Chat.ID, tgbotapi.FileID(HTB.Content))
				Config.Caption = fmt.Sprintf("請再次確認是否要刪除「%s」？", HTB.Keyword)
				Config.ReplyToMessageID = CallbackQuery.Message.MessageID
				Config.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("是", UID),
						tgbotapi.NewInlineKeyboardButtonData("否", "NIL"),
					),
				)
				resp, err := bot.Request(Config)
				if err != nil {
					log.Println("[CallBQ]", err)
				}
				json.Unmarshal(resp.Result, &raw_json)
				DEntity.ToBeDeleteMessageIDs = append(DEntity.ToBeDeleteMessageIDs, raw_json.MID)
			}
		} else if !DEntity.Done {
			DEntity.Done = true
			if err := DB.DeleteById(CONFIG.DB.COLLECTION, UID); err != nil {
				log.Println("[CallBQ]", err)
				return
			}
			log.Printf("[DELETE] \"%s\" has been deleted!\n", DEntity.Keyword)
			SendText(CallbackQuery.Message.Chat.ID, fmt.Sprintf("已成功刪除「%s」", DEntity.Keyword), 0)
			for _, to_be_delete_message_id := range DEntity.ToBeDeleteMessageIDs {
				bot.Request(tgbotapi.NewDeleteMessage(CallbackQuery.Message.Chat.ID, to_be_delete_message_id))
			}
		}
	}

	// legacy code to deal with overwriting keyword:
	// type OverwriteEntity struct {
	// 	Type    int64
	// 	Keyword string
	// 	Content string
	// 	From    int64
	// 	Done    bool // prevent multiple clicks
	// }
	// var Queued_Overwrites = make(map[string]*OverwriteEntity) // Keyword: OverwriteEntity

	// in "new"/"add":
	// docs, err := DB.FindAll(c.NewQuery(CONFIG.DB.COLLECTION).Where(
	// 	c.Field("Keyword").Eq(Keyword)))
	// if err != nil {
	// 	log.Println("[new]", err)
	// 	return
	// }
	// if len(docs) > 0 {
	// 	// Queue changes
	// 	Queued_Overwrites[Keyword] = &OverwriteEntity{
	// 		Type:    1,
	// 		Keyword: Keyword,
	// 		Content: Content,
	// 		From:    Message.From.ID,
	// 		Done:    false,
	// 	}

	// 	Reply_Content := fmt.Sprintf("相同關鍵字的複製文已有 %d 篇（內容如下），是否繼續添加？", len(docs))
	// 	for idx, doc := range docs {
	// 		// same keyword & content
	// 		if doc.Get("Content").(string) == Content {
	// 			SendText(Message.Chat.ID, "傳過了啦 腦霧?", Message.MessageID)
	// 			return
	// 		}
	// 		Reply_Content += fmt.Sprintf("\n%d.「%s」", idx+1, TruncateString(doc.Get("Content").(string), 30))
	// 	}

	// 	replyMsg := tgbotapi.NewMessage(Message.Chat.ID, Reply_Content)
	// 	replyMsg.ReplyToMessageID = Message.MessageID
	// 	replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
	// 		tgbotapi.NewInlineKeyboardRow(
	// 			tgbotapi.NewInlineKeyboardButtonData("是", Keyword),
	// 			tgbotapi.NewInlineKeyboardButtonData("否", "NIL"),
	// 		),
	// 	)
	// 	if _, err := bot.Send(replyMsg); err != nil {
	// 		log.Println("[new]", err)
	// 	}
	// 	return
	// }

	// in handle CallBQ
	// else if OW_Entity, ok := Queued_Overwrites[CallbackQuery.Data]; ok {
	// 	// 是 & in overwrite
	// 	// show respond
	// 	if OW_Entity.Done {
	// 		return
	// 	}

	// 	callback := tgbotapi.NewCallback(CallbackQuery.ID, "正在新增中……")
	// 	if _, err := bot.Request(callback); err != nil {
	// 		log.Println("[CallBQ]", err)
	// 	}
	// 	OW_Entity.Done = true

	// 	to_be_delete_message := SendText(CallbackQuery.Message.Chat.ID, "運算中，請稍後……", CallbackQuery.Message.MessageID)

	// 	if err != nil {
	// 		log.Println("[CallBQ]", err)
	// 		return
	// 	}

	// 	// delete tmp message
	// 	bot.Request(tgbotapi.NewDeleteMessage(CallbackQuery.Message.Chat.ID, to_be_delete_message.MessageID))

	// 	// send response to user
	// 	SendText(CallbackQuery.Message.Chat.ID, fmt.Sprintf("新增複製文「%s」成功，\n自動生成的摘要如下：「%s」", OW_Entity.Keyword, Sum), CallbackQuery.Message.MessageID)
	// }
}
