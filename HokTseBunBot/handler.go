package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"strconv"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ChannelCriticalLock = NewMutextMap()

func recentHandler(Message *tgbotapi.Message) {
	ChatID := Message.Chat.ID

	if !ChatStatus[ChatID].Global {
		SendText(ChatID, fmt.Sprintf("執行失敗:%s", "此指令只能在公共模式下執行"), Message.MessageID)
		return
	}

	// Parse args
	num, err := strconv.Atoi(Message.CommandArguments())
	if err != nil {
		num = 3
	}
	num = Min(10, num)

	Filter := bson.M{"From": bson.M{"$ne": Message.From.ID}}
	opts := options.Find().SetLimit(int64(num))
	Curser, err := DB.Collection(CONFIG.DB.GLOBAL_COL).Find(context.TODO(), Filter, opts)
	if err != nil {
		log.Printf("%v\n", Message)
		log.Println("[recent]", err)
		SendText(ChatID, fmt.Sprintf("執行失敗:%s", err), Message.MessageID)
		return
	}

	for Curser.Next(context.TODO()) {
		var HTB HokTseBun
		Curser.Decode(&HTB)
		switch {
		case HTB.IsText():
			SendText(ChatID, fmt.Sprintf("來自：「%s」\n名稱：「%s」\n摘要：「%s」\n內容：「%s」", GetMaskedNameByID(HTB.From), HTB.Keyword, HTB.Summarization, HTB.Content), 0)
		case HTB.IsMultiMedia():
			SendMultiMedia(ChatID, fmt.Sprintf("來自：「%s」\n名稱：「%s」\n摘要：「%s」", GetMaskedNameByID(HTB.From), HTB.Keyword, HTB.Summarization), HTB.Content, HTB.Type)
		}
	}
}

func statusHandler(Message *tgbotapi.Message) {
	// Send learderboard info
	LeaderBoard, _ := GetLBInfo(3)
	SendText(Message.Chat.ID, LeaderBoard, 0)
	// Send user status
	var content string
	if ChatStatus[Message.Chat.ID].Global {
		content = fmt.Sprintf("目前處於 公共模式\n貢獻值爲 %d", UserStatus[Message.From.ID].Contribution)
	} else {
		content = fmt.Sprintf("目前處於 私人模式\n貢獻值爲 %d", UserStatus[Message.From.ID].Contribution)
	}
	SendText(Message.Chat.ID, content, 0)
}

func toggleHandler(Message *tgbotapi.Message) {
	// check if exist already
	if v, ok := ChatStatus[Message.Chat.ID]; ok && v.Global {
		// Close
		TmpChatStatus := ChatStatusEntity{Global: false, ChatID: Message.Chat.ID}
		err := UpdateChatStatus(TmpChatStatus)
		if err != nil {
			log.Println("[toggleG]", err)
			SendText(Message.Chat.ID, "關閉公共模式失敗:"+err.Error(), 0)
			return
		}
		ChatStatus[Message.Chat.ID] = TmpChatStatus

		SendText(Message.Chat.ID, "已關閉公共模式", 0)
		return
	} else if !ok {
		// First time entering public mode
		content := `第一次進入公共模式，請注意：
		1. 這裡的資料庫是所有人共享的
		2. 只能刪除自己新增的東西
		3. 我不想管裡面有啥 但你亂加東西讓我管 我就ban你
		4. 可以再次使用 /toggle 來退出`
		SendText(Message.Chat.ID, content, 0)
	}
	// Open
	if UserStatus[Message.From.ID].Banned {
		SendText(Message.Chat.ID, "你被ban了 不能開啓公共模式 覺得莫名奇妙的話也可能是bug 請找作者🤷", 0)
		return
	}

	TmpChatStatus := ChatStatusEntity{Global: true, ChatID: Message.Chat.ID}
	err := UpdateChatStatus(TmpChatStatus)
	if err != nil {
		log.Println("[toggleG]", err)
		SendText(Message.Chat.ID, "開啓公共模式失敗:"+err.Error(), 0)
		return
	}
	ChatStatus[Message.Chat.ID] = TmpChatStatus

	SendText(Message.Chat.ID, "已開啓公共模式", 0)
}

func randomHandler(Message *tgbotapi.Message) {
	var Filter bson.D
	switch Message.Command() {
	case "randimg":
		Filter = bson.D{{Key: "Type", Value: 2}}
	case "randtxt":
		Filter = bson.D{{Key: "Type", Value: 1}}
	default:
		Filter = bson.D{{Key: "Type", Value: bson.D{{Key: "$ne", Value: 0}}}}
	}

	var CollectionName string

	if ChatStatus[Message.Chat.ID].Global {
		CollectionName = CONFIG.DB.GLOBAL_COL
	} else {
		CollectionName = CONFIG.GetColbyChatID(Message.Chat.ID)
	}

	// Get Docs length
	num, err := DB.Collection(CollectionName).CountDocuments(context.TODO(), Filter)
	RandomIndex := rand.Int63n(num)
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
	opts := options.FindOne().SetSkip(RandomIndex)
	SRst := DB.Collection(CollectionName).FindOne(context.TODO(), Filter, opts)
	if SRst.Err() != nil {
		log.Printf("[random], %+v\n", Message)
		log.Println("[random]", err)
		SendText(Message.Chat.ID, fmt.Sprintf("錯誤：%s", err), 0)
		return
	}

	var HTB HokTseBun
	SRst.Decode(&HTB)

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

func dumpHandler(Message *tgbotapi.Message) {
	// only one handler running for each chat
	ChannelCriticalLock.Lock(int(Message.Chat.ID))
	defer ChannelCriticalLock.Release(int(Message.Chat.ID))

	// This command dumps copypasta that you sent in private db into public db
	Filter := bson.D{{Key: "From", Value: Message.From.ID}}
	Curser, err := DB.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).Find(context.TODO(), Filter)
	defer func() { Curser.Close(context.TODO()) }()
	if err != nil {
		log.Println("[dump]", Message)
		log.Println("[dump]", err)
		SendText(Message.Chat.ID, "傾卸失敗: "+err.Error(), 0)
		return
	}

	docs2insert := make([]interface{}, 0, 100)
	for Curser.Next(context.TODO()) {
		var doc interface{}
		Curser.Decode(&doc)
		docs2insert = append(docs2insert, doc)
	}

	opts := options.InsertMany().SetOrdered(false)
	MRst, err := DB.Collection(CONFIG.DB.GLOBAL_COL).InsertMany(context.TODO(), docs2insert, opts)
	if err == mongo.ErrEmptySlice {
		SendText(Message.Chat.ID, "沒有東西能傾倒", 0)
		return
	} else if err != nil && reflect.TypeOf(err) != reflect.TypeOf(mongo.BulkWriteException{}) {
		log.Println("[dump]", Message)
		log.Println("[dump]", err)
		SendText(Message.Chat.ID, "傾卸失敗: "+err.Error(), 0)
		return
	}

	NewCon, err := AddUserContribution(Message.From.ID, len(MRst.InsertedIDs))
	if err != nil {
		log.Printf("Message: %v\n", Message)
		log.Println("[UpdateUS]", err)
	}
	SendText(Message.Chat.ID, fmt.Sprintf("成功把%d坨大便倒進公共資料庫，目前累計貢獻%d坨", len(MRst.InsertedIDs), NewCon), 0)
}

func addHandler(Message *tgbotapi.Message, Keyword, Content, FileUniqueID string, Type int) {
	// only one handler running for each chat
	ChannelCriticalLock.Lock(int(Message.Chat.ID))
	defer ChannelCriticalLock.Release(int(Message.Chat.ID))

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

	var CollectionName string
	Global := ChatStatus[Message.Chat.ID].Global
	if Global {
		CollectionName = CONFIG.DB.GLOBAL_COL
	} else {
		CollectionName = CONFIG.GetColbyChatID(Message.Chat.ID)
	}

	// find existing files
	Filter := bson.D{{Key: "$and",
		Value: bson.A{bson.D{{Key: "Type", Value: Type}}, bson.D{{Key: "Keyword", Value: Keyword}}, bson.D{{Key: "FileUniqueID", Value: FileUniqueID}}},
	}}
	if Rst := DB.Collection(CollectionName).FindOne(context.TODO(), Filter); Rst.Err() != mongo.ErrNoDocuments {
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
	var (
		Sum string
		URL string
		err error
	)
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
			Sum, err = ImageCaptioning(Keyword, URL)
			if err != nil {
				SendText(Message.Chat.ID, fmt.Sprintf("新增%s「%s」失敗，可能我濫用API被ban了：%s", CONFIG.GetNameByType(CONFIG.SETTING.TYPE.IMG), Keyword, err), Message.MessageID)
			}
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
		Sum, err = ImageCaptioning(Keyword, Thumb_URL)
		if err != nil {
			SendText(Message.Chat.ID, fmt.Sprintf("新增%s「%s」失敗，可能我濫用API被ban了：%s", CONFIG.GetNameByType(CONFIG.SETTING.TYPE.IMG), Keyword, err), Message.MessageID)
		}
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
		Sum, err = ImageCaptioning(Keyword, Thumb_URL)
		if err != nil {
			SendText(Message.Chat.ID, fmt.Sprintf("新增%s「%s」失敗，可能我濫用API被ban了：%s", CONFIG.GetNameByType(CONFIG.SETTING.TYPE.IMG), Keyword, err), Message.MessageID)
		}
	}
	// Delete tmp message
	bot.Request(tgbotapi.NewDeleteMessage(Message.Chat.ID, to_be_delete_message.MessageID))

	_, err = InsertHTB(
		CollectionName,
		&HokTseBun{
			Type:          Type,
			Keyword:       Keyword,
			Summarization: Sum,
			Content:       Content,
			URL:           URL,
			From:          Message.From.ID,
			FileUniqueID:  FileUniqueID,
		},
	)

	// send response to user
	if err != nil {
		log.Printf("[add] Keyword: %s, Content: %s, Type: %d, Message: %+v\n", Keyword, Content, Type, Message)
		log.Println("[add]", err)
		SendText(Message.Chat.ID, fmt.Sprintf("新增%s「%s」失敗：%s", CONFIG.GetNameByType(Type), Keyword, err), Message.MessageID)
	} else {
		if Global {
			Con, _ := AddUserContribution(Message.From.ID, 1)
			SendText(Message.Chat.ID, fmt.Sprintf("新增%s「%s」成功，\n自動生成的摘要如下：「%s」\n目前貢獻值爲%d", CONFIG.GetNameByType(Type), Keyword, Sum, Con), Message.MessageID)
		} else {
			SendText(Message.Chat.ID, fmt.Sprintf("新增%s「%s」成功，\n自動生成的摘要如下：「%s」", CONFIG.GetNameByType(Type), Keyword, Sum), Message.MessageID)
		}
	}
}

func searchHandler(Message *tgbotapi.Message) {
	var (
		Query       string = Message.CommandArguments()
		ResultCount int    = 0
		MaxResults  int    = 25
	)

	var CollectionName string

	if ChatStatus[Message.Chat.ID].Global {
		CollectionName = CONFIG.DB.GLOBAL_COL
	} else {
		CollectionName = CONFIG.GetColbyChatID(Message.Chat.ID)
	}

	if utf8.RuneCountInString(Query) >= 200 || utf8.RuneCountInString(Query) == 0 {
		SendText(Message.Chat.ID, fmt.Sprintf("關鍵字要介於1 ~ 200字，不然我的CPU要燒了，目前爲%d字", utf8.RuneCountInString(Query)), 0)
		return
	}

	SendText(Message.From.ID, fmt.Sprintf("「%s」的搜尋結果如下：", Query), 0)

	var to_be_delete_message tgbotapi.Message
	if Message.Chat.ID != Message.From.ID {
		to_be_delete_message = SendText(Message.Chat.ID, "正在搜尋中…… 請稍後", 0)
	}

	// search
	Filter := bson.D{{Key: "Type", Value: bson.D{{Key: "$ne", Value: 0}}}}
	opts := options.Find().SetSort(bson.D{{Key: "Type", Value: 1}})
	Curser, err := DB.Collection(CollectionName).Find(context.TODO(), Filter, opts)
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

	// Delete tmp message
	bot.Request(tgbotapi.NewDeleteMessage(Message.Chat.ID, to_be_delete_message.MessageID))

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

func searchMediaHandler(ChatID, FromID int64, FileID_str, FileUniqueID string, Type int) {
	var CollectionName string
	if ChatStatus[ChatID].Global {
		CollectionName = CONFIG.DB.GLOBAL_COL
	} else {
		CollectionName = CONFIG.GetColbyChatID(ChatID)
	}

	SendMultiMedia(FromID, "此圖片的搜尋結果如下：", FileID_str, Type)

	// create tmp message
	var to_be_delete_message tgbotapi.Message
	if ChatID != FromID {
		to_be_delete_message = SendText(ChatID, "正在搜尋中…… 請稍後, 圖片只會搜尋完全相同的圖片", 0)
	}

	// search for same media in db
	Filter := bson.D{{Key: "$and", Value: bson.A{
		bson.D{{Key: "Type", Value: Type}},
		bson.D{{Key: "FileUniqueID", Value: FileUniqueID}},
	}}}
	Curser, err := DB.Collection(CollectionName).Find(context.TODO(), Filter)
	defer func() { Curser.Close(context.TODO()) }()
	if err != nil {
		log.Printf("[search] ChatID: %d, FilUID: %s, Type: %d\n", ChatID, FileUniqueID, Type)
		log.Println("[search]", err)
		SendText(ChatID, "搜尋失敗:"+err.Error(), 0)
		return
	}

	var (
		HTB         HokTseBun
		ResultCount = 0
		MaxResults  = 25
	)
	for Curser.Next(context.TODO()) {
		Curser.Decode(&HTB)
		SendText(FromID, fmt.Sprintf("名稱：「%s」\n描述：「%s」", HTB.Keyword, HTB.Summarization), 0)
		ResultCount++
	}

	// Delete tmp message
	bot.Request(tgbotapi.NewDeleteMessage(ChatID, to_be_delete_message.MessageID))

	if ResultCount <= MaxResults {
		SendText(FromID, fmt.Sprintf("搜尋完成，共 %d 筆吻合\n", ResultCount), 0)
		if ChatID != FromID {
			SendText(ChatID, fmt.Sprintf("搜尋完成，共 %d 筆吻合\n(結果在與bot的私訊中)", ResultCount), 0)
		}
	} else {
		SendText(FromID, fmt.Sprintf("搜尋完成，結果超過%d筆上限，請嘗試更換關鍵字", MaxResults), 0)
		if ChatID != FromID {
			SendText(ChatID, fmt.Sprintf("搜尋完成，結果超過%d筆上限，請嘗試更換關鍵字\n(結果在與bot的私訊中)", MaxResults), 0)
		}
	}
}

func deleteHandler(Message *tgbotapi.Message) {
	var BeDeletedKeyword = Message.CommandArguments()
	if utf8.RuneCountInString(BeDeletedKeyword) >= 30 {
		SendText(Message.Chat.ID, fmt.Sprintf("關鍵字長度不可大於 30, 目前爲 %d 字”", utf8.RuneCountInString(BeDeletedKeyword)), Message.MessageID)
		return
	}

	var (
		CollectionName string
		Filter         bson.D
	)
	Global := ChatStatus[Message.Chat.ID].Global

	if Global {
		CollectionName = CONFIG.DB.GLOBAL_COL
		Filter = bson.D{{Key: "$and",
			Value: bson.A{bson.D{{Key: "Keyword", Value: BeDeletedKeyword}}, bson.D{{Key: "From", Value: Message.From.ID}}},
		}}
	} else {
		CollectionName = CONFIG.GetColbyChatID(Message.Chat.ID)
		Filter = bson.D{{Key: "Keyword", Value: BeDeletedKeyword}}
	}

	num, err := DB.Collection(CollectionName).CountDocuments(context.TODO(), Filter)
	if err != nil {
		log.Printf("[delete] Message: %+v\n", Message)
		log.Println("[delete]", err)
		SendText(Message.Chat.ID, fmt.Sprintf("刪除「%s」失敗：%s", BeDeletedKeyword, err), Message.MessageID)
		return
	}
	if num <= 0 {
		if Global {
			SendText(Message.Chat.ID, "沒有大便符合關鍵字/是別人新增的", Message.MessageID)
		} else {
			SendText(Message.Chat.ID, "沒有大便符合關鍵字", Message.MessageID)
		}
		return
	}

	opts := options.Find().SetSort(bson.D{{Key: "Type", Value: 1}})
	Curser, err := DB.Collection(CollectionName).Find(context.TODO(), Filter, opts)
	defer func() { Curser.Close(context.TODO()) }()
	if err != nil {
		log.Printf("[delete] Message: %+v\n", Message)
		log.Println("[delete]", err)
		SendText(Message.Chat.ID, fmt.Sprintf("刪除「%s」失敗：%s", BeDeletedKeyword, err), Message.MessageID)
		return
	}

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
		case HTB.IsMultiMedia():
			type_prompt := CONFIG.GetNameByType(HTB.Type) + "："
			ShowEntry = fmt.Sprintf("%d. %s%s", idx, type_prompt, TruncateString(HTB.Summarization, 15-utf8.RuneCountInString(type_prompt)))
		}
		ReplyMarkup = append(ReplyMarkup, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(ShowEntry, "DEL_"+HTB.UID.Hex())))
		TB_HTB["DEL_"+HTB.UID.Hex()] = &DeleteEntity{HTB: *HTB, Global: Global}
	}
	ReplyMarkup = append(ReplyMarkup, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✖️取消", "NIL_WITH_REACT")))

	replyMsg := tgbotapi.NewMessage(Message.Chat.ID, "請選擇要刪除以下哪個？")
	replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(ReplyMarkup...)
	replyMsg.DisableNotification = true

	Msg, err := bot.Send(replyMsg)
	if err != nil {
		log.Println("[delete]", err)
	}
	if _, ok := QueuedDeletes[Message.Chat.ID]; !ok {
		QueuedDeletes[Message.Chat.ID] = make(map[int]map[string]*DeleteEntity)
	}
	QueuedDeletes[Message.Chat.ID][Msg.MessageID] = TB_HTB
}
