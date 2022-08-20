package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func exampleHandler(Message *tgbotapi.Message) {
	replyMsg := tgbotapi.NewMessage(Message.Chat.ID, "請按按鈕選擇要觀看的教學範例:")
	replyMsg.ReplyToMessageID = Message.MessageID
	replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("這個 bot 是幹嘛用的", "EXP WHATISTHIS")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("我要如何新增複製文?", "EXP HOWTXT")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("我要如何新增圖片/GIF/影片?", "EXP HOWMEDIA")),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("/add 指令", "EXP ADD"),
			tgbotapi.NewInlineKeyboardButtonData("/new 指令", "EXP ADD"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("/random 指令", "EXP RAND"),
			tgbotapi.NewInlineKeyboardButtonData("/search 指令", "EXP SERC"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("/delete 指令", "EXP DEL"),
			tgbotapi.NewInlineKeyboardButtonData("/example 指令", "EXP EXP"),
		),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✖️取消", "NIL_WITH_REACT")),
	)
	if _, err := bot.Send(replyMsg); err != nil {
		log.Printf("[exp], %+v\n", Message)
		log.Println("[exp]", err)
		return
	}
}

func randomHandler(Message *tgbotapi.Message) {
	var Filter bson.D
	switch Message.Command() {
	case "randImage":
		Filter = bson.D{{Key: "Type", Value: 2}}
	case "randText":
		Filter = bson.D{{Key: "Type", Value: 1}}
	default:
		Filter = bson.D{{Key: "Type", Value: bson.D{{Key: "$ne", Value: 0}}}}
	}

	// Get Docs length
	num, err := DB.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).CountDocuments(context.TODO(), Filter)
	if err != nil {
		log.Printf("[random], %+v\n", Message)
		log.Println("[random]", err)
		SendText(Message.Chat.ID, fmt.Sprintf("錯誤：%s", err), 0)
		return
	}
	if num == 0 {
		SendText(Message.Chat.ID, "資料庫沒東西是在抽屁", 0)
		return
	}

	// Get Curser
	Curser, err := DB.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).Find(context.TODO(), Filter)
	defer func() { Curser.Close(context.TODO()) }()
	if err != nil {
		log.Printf("[random], %+v\n", Message)
		log.Println("[random]", err)
		SendText(Message.Chat.ID, fmt.Sprintf("錯誤：%s", err), 0)
		return
	}

	var HTB *HokTseBun = &HokTseBun{}
	RandomIndex := rand.Int63n(num)

	for Curser.Next(context.TODO()) {
		if RandomIndex <= 0 {
			Curser.Decode(HTB)
			break
		}
		RandomIndex--
	}

	switch {
	case HTB.IsText():
		SendText(Message.Chat.ID, fmt.Sprintf("幫你從 %d 坨大便中精心選擇了「%s」：\n%s", num, HTB.Keyword, HTB.Content), 0)
	case HTB.IsMultiMedia():
		SendMultiMedia(Message.Chat.ID, fmt.Sprintf("幫你從 %d 坨大便中精心選擇了「%s」", num, HTB.Keyword), HTB.Content, HTB.Type)
	default:
		log.Printf("[random], 發生了奇怪的錯誤，送不出去這個東西：%+v\nMsg:%+v\n", HTB, Message)
		SendText(Message.Chat.ID, fmt.Sprintf("發生了奇怪的錯誤，送不出去這個東西：%+v", HTB), 0)
	}
}

func addHandler(Message *tgbotapi.Message, Keyword, Content string, Type int) {
	switch {
	// check Keyword length
	case utf8.RuneCountInString(Keyword) >= 30:
		SendText(Message.Chat.ID, fmt.Sprintf("關鍵字長度不可大於 30, 目前爲 %d 字”", utf8.RuneCountInString(Keyword)), Message.MessageID)
		return
	// check content length
	case utf8.RuneCountInString(Content) >= 4000:
		SendText(Message.Chat.ID, fmt.Sprintf("內容長度不可大於 4000, 目前爲 %d 字”", utf8.RuneCountInString(Content)), Message.MessageID)
		return

	}

	// find existing files
	Filter := bson.D{{Key: "$and",
		Value: bson.A{bson.D{{Key: "Type", Value: Type}}, bson.D{{Key: "Keyword", Value: Keyword}}, bson.D{{Key: "Content", Value: Content}}},
	}}
	if Rst := DB.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).FindOne(context.TODO(), Filter); Rst.Err() != mongo.ErrNoDocuments {
		SendText(Message.Chat.ID, "傳過了啦 腦霧?", Message.MessageID)
		return
	} else if Rst.Err() != nil && Rst.Err() != mongo.ErrNoDocuments {
		log.Printf("[add] Keyword: %s, Content: %s, Type: %d, Message: %+v\n", Keyword, Content, Type, Message)
		log.Println("[add]", Rst.Err())
		SendText(Message.Chat.ID, fmt.Sprintf(fmt.Sprintf("新增%s「%s」失敗：%s", CONFIG.GetNameByType(Type), Keyword, Rst.Err()), Message.MessageID), 0)
		return
	}
	// Create tmp message
	to_be_delete_message := SendText(Message.Chat.ID, "運算中，請稍後……", Message.MessageID)
	// Insert HTB
	var Sum string
	var URL string
	var err error
	switch Type {
	case CONFIG.SETTING.TYPE.TXT:
		Sum = TextSummarization(Keyword, Content)
		URL = ""
	case CONFIG.SETTING.TYPE.IMG:
		URL, err := bot.GetFileDirectURL(Content)
		if err != nil {
			log.Printf("[add] Keyword: %s, Content: %s, Type: %d, Message: %+v\n", Keyword, Content, Type, Message)
			log.Println("[add]", err)
			SendText(Message.Chat.ID, fmt.Sprintf("新增%s「%s」失敗：%s", CONFIG.GetNameByType(CONFIG.SETTING.TYPE.IMG), Keyword, err), Message.MessageID)
			Sum = ""
		} else {
			Sum = ImageCaptioning(Keyword, URL)
		}
	case CONFIG.SETTING.TYPE.ANI:
		// get url
		URL, err = bot.GetFileDirectURL(Content)
		if err != nil {
			log.Printf("[add] Keyword: %s, Content: %s, Type: %d, Message: %+v\n", Keyword, Content, Type, Message)
			log.Println("[add]", err)
		}
		// get caption by thumbnail
		Thumb_URL, err := bot.GetFileDirectURL(Message.Animation.Thumbnail.FileID)
		if err != nil {
			log.Printf("[add] Keyword: %s, Content: %s, Type: %d, Message: %+v\n", Keyword, Content, Type, Message)
			log.Println("[add]", err)
		}
		Sum = ImageCaptioning(Keyword, Thumb_URL)
	case CONFIG.SETTING.TYPE.VID:
		// get url
		URL, err = bot.GetFileDirectURL(Content)
		if err != nil {
			log.Printf("[add] Keyword: %s, Content: %s, Type: %d, Message: %+v\n", Keyword, Content, Type, Message)
			log.Println("[add]", err)
		}
		// get caption by thumbnail
		Thumb_URL, err := bot.GetFileDirectURL(Message.Video.Thumbnail.FileID)
		if err != nil {
			log.Printf("[add] Keyword: %s, Content: %s, Type: %d, Message: %+v\n", Keyword, Content, Type, Message)
			log.Println("[add]", err)
		}
		Sum = ImageCaptioning(Keyword, Thumb_URL)
	}
	// Delete tmp message
	bot.Request(tgbotapi.NewDeleteMessage(Message.Chat.ID, to_be_delete_message.MessageID))

	_, err = InsertHTB(
		CONFIG.GetColbyChatID(Message.Chat.ID),
		&HokTseBun{
			Type:          Type,
			Keyword:       Keyword,
			Summarization: Sum,
			Content:       Content,
			URL:           URL,
			From:          Message.From.ID,
		},
	)
	// send response to user
	if err != nil {
		log.Printf("[add] Keyword: %s, Content: %s, Type: %d, Message: %+v\n", Keyword, Content, Type, Message)
		log.Println("[add]", err)
		SendText(Message.Chat.ID, fmt.Sprintf("新增%s「%s」失敗：%s", CONFIG.GetNameByType(Type), Keyword, err), Message.MessageID)
	} else {
		SendText(Message.Chat.ID, fmt.Sprintf("新增%s「%s」成功，\n自動生成的摘要如下：「%s」", CONFIG.GetNameByType(Type), Keyword, Sum), Message.MessageID)
	}
}

func searchHandler(Message *tgbotapi.Message) {
	var (
		Query       string = Message.CommandArguments()
		ResultCount int    = 0
		MaxResults  int    = 25
	)

	if utf8.RuneCountInString(Query) >= 200 || utf8.RuneCountInString(Query) == 0 {
		SendText(Message.Chat.ID, fmt.Sprintf("關鍵字要介於1 ~ 200字，不然我的CPU要燒了，目前爲%d字", utf8.RuneCountInString(Query)), 0)
		return
	}

	SendText(Message.From.ID, fmt.Sprintf("「%s」的搜尋結果如下：", Query), 0)

	if Message.Chat.ID != Message.From.ID {
		SendText(Message.Chat.ID, "正在搜尋中…… 請稍後", 0)
	}

	// search
	Filter := bson.D{{Key: "Type", Value: bson.D{{Key: "$ne", Value: 0}}}}
	opts := options.Find().SetSort(bson.D{{Key: "Type", Value: 1}})
	Curser, err := DB.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).Find(context.TODO(), Filter, opts)
	defer func() { Curser.Close(context.TODO()) }()
	if err != nil {
		log.Printf("[search] Message: %+v\n", Message)
		log.Println("[search]", err)
		SendText(Message.Chat.ID, "搜尋失敗:"+err.Error(), Message.MessageID)
		return
	}

	HTB := &HokTseBun{}
	for Curser.Next(context.TODO()) {
		if ResultCount >= MaxResults {
			ResultCount++
			break
		}
		Curser.Decode(HTB)
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
		SendText(Message.From.ID, fmt.Sprintf("搜尋完成，結果超過%d筆上限，請嘗試更換關鍵字", MaxResults), 0)
		if Message.Chat.ID != Message.From.ID {
			SendText(Message.Chat.ID, fmt.Sprintf("搜尋完成，結果超過%d筆上限，請嘗試更換關鍵字\n(結果在與bot的私訊中)", MaxResults), 0)
		}
	}
}

func deleteHandler(Message *tgbotapi.Message) {
	var BeDeletedKeyword = Message.CommandArguments()
	if utf8.RuneCountInString(BeDeletedKeyword) >= 30 {
		SendText(Message.Chat.ID, fmt.Sprintf("關鍵字長度不可大於 30, 目前爲 %d 字”", utf8.RuneCountInString(BeDeletedKeyword)), Message.MessageID)
		return
	}

	Filter := bson.D{{Key: "Keyword", Value: BeDeletedKeyword}}
	opts := options.Find().SetSort(bson.D{{Key: "Type", Value: 1}})
	Curser, err := DB.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).Find(context.TODO(), Filter, opts)
	defer func() { Curser.Close(context.TODO()) }()
	if err != nil {
		log.Printf("[delete] Message: %+v\n", Message)
		log.Println("[delete]", err)
		SendText(Message.Chat.ID, fmt.Sprintf("刪除「%s」失敗：%s", BeDeletedKeyword, err), Message.MessageID)
		return
	}
	// if len(docs) <= 0 {
	// 	SendText(Message.Chat.ID, "沒有大便符合關鍵字", Message.MessageID)
	// 	return
	// }

	ReplyMarkup := make([][]tgbotapi.InlineKeyboardButton, 0)
	TB_HTB := make(map[string]*DeleteEntity)
	var idx int = 1
	for Curser.Next(context.TODO()) {
		HTB := &HokTseBun{}
		Curser.Decode(HTB)
		var ShowEntry string
		switch {
		case HTB.IsText():
			ShowEntry = fmt.Sprintf("%d. %s", idx, TruncateString(HTB.Content, 20))
		case HTB.IsImage():
			type_prompt := "圖片："
			ShowEntry = fmt.Sprintf("%d. %s%s", idx, type_prompt, TruncateString(HTB.Summarization, 15-utf8.RuneCountInString(type_prompt)))
		case !HTB.IsImage() && HTB.IsMultiMedia():
			type_prompt := "動圖："
			ShowEntry = fmt.Sprintf("%d. %s%s", idx, type_prompt, TruncateString(HTB.Summarization, 15-utf8.RuneCountInString(type_prompt)))
		}
		ReplyMarkup = append(ReplyMarkup, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(ShowEntry, "DEL_"+HTB.UID.Hex())))
		TB_HTB["DEL_"+HTB.UID.Hex()] = &DeleteEntity{HTB: *HTB}
	}
	ReplyMarkup = append(ReplyMarkup, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✖️取消", "NIL_WITH_REACT")))

	replyMsg := tgbotapi.NewMessage(Message.Chat.ID, "請選擇要刪除以下哪個？")
	replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(ReplyMarkup...)

	Msg, err := bot.Send(replyMsg)
	if err != nil {
		log.Println("[delete]", err)
	}
	if _, ok := QueuedDeletes[Message.Chat.ID]; !ok {
		QueuedDeletes[Message.Chat.ID] = make(map[int]map[string]*DeleteEntity)
	}
	QueuedDeletes[Message.Chat.ID][Msg.MessageID] = TB_HTB
}
