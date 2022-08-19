package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// // Deletes[ChatID]["UID"] = smth
// var Deletes = make(map[int64]map[string]*DeleteEntity)
// QueuedDeletes[ChatID][MessageID][doc_id] = doc
var QueuedDeletes = make(map[int64]map[int]map[string]*DeleteEntity)

type DeleteEntity struct {
	// info
	HTB HokTseBun
	// status
	Confirmed bool
	Done      bool
}

func handleCommand(Message *tgbotapi.Message) {
	// handle commands
	switch Message.Command() {
	// public available functions
	case "start":
		// Startup
		NewChat(Message.Chat.ID)
	case "example": // short: EXP
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
			log.Println("[new]", err)
		}
	case "random", "randImage", "randText": // short: RAND
		randomHandler(Message)
	case "new", "add": // short: NEW, ADD
		Command_Args := strings.Fields(Message.CommandArguments())
		if len(Command_Args) <= 1 {
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("錯誤：指令格式爲 “/%s {關鍵字} {內容}”", Message.Command()))
			replyMsg.ReplyToMessageID = Message.MessageID
			replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("蛤 我不會啦 啥", "EXP HOWTXT")),
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("阿如果我想傳圖片/GIF/影片咧", "EXP HOWMEDIA")),
			)
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
			return
		}
		var Keyword string = Command_Args[0]
		var Content string = strings.TrimSpace(Message.Text[strings.Index(Message.Text, Command_Args[1]):])
		addHandler(Message, Keyword, Content, CONFIG.SETTING.TYPE.TXT)
	case "search": // short: SERC
		Command_Args := strings.Fields(Message.CommandArguments())
		if len(Command_Args) < 1 {
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("錯誤：指令格式爲 “/%s {關鍵字}”", Message.Command()))
			replyMsg.ReplyToMessageID = Message.MessageID
			replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("蛤 我不會啦 啥", "EXP SERC")))
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
			return
		}
		searchHandler(Message)
	case "delete": // short: DEL
		Command_Args := strings.Fields(Message.CommandArguments())
		if len(Command_Args) < 1 {
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("錯誤：指令格式爲 “/%s {關鍵字}”", Message.Command()))
			replyMsg.ReplyToMessageID = Message.MessageID
			replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("蛤 我不會啦 啥", "EXP DEL")))
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
			return
		}
		deleteHandler(Message)

	// internal usage
	// case "import":
	// 	DB.DropCollection(CONFIG.GetColbyChatID(Message.Chat.ID))
	// 	DB.ImportCollection(CONFIG.GetColbyChatID(Message.Chat.ID), Message.CommandArguments())
	// 	SendText(Message.Chat.ID, Message.CommandArguments(), Message.MessageID)
	case "refresh_beginner":
		if Message.CommandArguments() != CONFIG.API.TG.TOKEN {
			SendText(Message.Chat.ID, "亂什麼洨 幹你娘", Message.MessageID)
			return
		}
		DB2.Collection("Beginner").Drop(context.TODO())
		if err := ImportCollection(DB2, "Beginner", "./Beginner pack.json"); err != nil {
			SendText(Message.Chat.ID, "刷新失敗: "+err.Error(), Message.MessageID)
			log.Println(err)
			return
		}
		SendText(Message.Chat.ID, "成功刷新 Beginner DB", Message.MessageID)
	case "chatid":
		SendText(Message.Chat.ID, fmt.Sprintf("此聊天室的 ChatID: %d", Message.Chat.ID), 0)
	case "drop":
		if Message.CommandArguments() != fmt.Sprint(Message.Chat.ID) {
			SendText(Message.Chat.ID, "防呆: 請和指令一起送出 ChatID, 格式爲\"/drop {ChatID}\"", 0)
			return
		}
		ChatID, err := strconv.ParseInt(Message.CommandArguments(), 10, 64)
		if err != nil {
			SendText(Message.Chat.ID, fmt.Sprint(ChatID, err), 0)
			return
		}
		DB2.Collection(CONFIG.GetColbyChatID(ChatID)).Drop(context.TODO())
		SendText(Message.Chat.ID, fmt.Sprintf("成功刪除 %d", ChatID), 0)
	case "import":
		var SourceCol string
		var TargetCol string = CONFIG.GetColbyChatID(Message.Chat.ID)

		if Message.CommandArguments() == "Beginner" {
			SourceCol = "Beginner"
		} else {
			SourceChatID, err := strconv.ParseInt(Message.CommandArguments(), 10, 64)
			if err != nil {
				SendText(Message.Chat.ID, fmt.Sprintf("匯入失敗: %s", err.Error()), Message.MessageID)
				return
			}
			SourceCol = CONFIG.GetColbyChatID(SourceChatID)
		}

		// type 0: system attribute
		Filter := bson.D{
			{"$and",
				bson.A{
					bson.D{{"Type", 0}},
					bson.D{{"Keyword", "Import"}},
					bson.D{{"Content", SourceCol}},
				}},
		}
		SingleRst := DB2.Collection(TargetCol).FindOne(context.TODO(), Filter)
		if SingleRst.Err() != mongo.ErrNoDocuments {
			SendText(Message.Chat.ID, "你之前匯入過了~", Message.MessageID)
			return
		}
		Curser, err := DB2.Collection(SourceCol).Find(context.TODO(), bson.D{})
		defer Curser.Close(context.TODO())
		if err != nil {
			SendText(Message.Chat.ID, fmt.Sprintf("匯入失敗: %s", err.Error()), Message.MessageID)
			return
		}

		var docs []interface{}
		if err := Curser.All(context.TODO(), &docs); err != nil {
			SendText(Message.Chat.ID, fmt.Sprintf("匯入失敗: %s", err.Error()), Message.MessageID)
			return
		}

		_, err = DB2.Collection(TargetCol).InsertMany(context.TODO(), docs)
		if err != nil {
			SendText(Message.Chat.ID, fmt.Sprintf("匯入失敗: %s", err.Error()), Message.MessageID)
			return
		}

		SendText(Message.Chat.ID, fmt.Sprintf("成功從 %s 匯入 %d 筆資料", SourceCol, len(docs)), Message.MessageID)

		// update system attribute
		InsertHTB(TargetCol, &HokTseBun{Type: 0, Keyword: "Import", Content: SourceCol})

	case "echo":
		// Echo
		SendText(Message.Chat.ID, Message.CommandArguments(), Message.MessageID)

	default:
		SendText(Message.Chat.ID, fmt.Sprintf("錯誤：我不會 “/%s” 啦QQ", Message.Command()), Message.MessageID)
	}
}

func randomHandler(Message *tgbotapi.Message) {
	var Filter bson.D
	switch Message.Command() {
	case "randImage":
		Filter = bson.D{{"Type", 2}}
	case "randText":
		Filter = bson.D{{"Type", 1}}
	default:
		Filter = bson.D{{"Type", bson.D{{"$ne", 0}}}}
	}
	Curser, err := DB2.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).Find(context.TODO(), Filter)
	defer Curser.Close(context.TODO())
	if err != nil {
		log.Println("[random]", err)
		SendText(Message.Chat.ID, fmt.Sprintf("錯誤：%s", err), 0)
		return
	}
	if Curser.Next(context.TODO()) {
		SendText(Message.Chat.ID, "資料庫沒東西是在抽屁", 0)
		return
	}

	num, err := DB2.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).EstimatedDocumentCount(context.TODO())
	RandomIndex := rand.Int63n(num)

	var HTB *HokTseBun = &HokTseBun{}

	for Curser.Next(context.TODO()) {
		if RandomIndex == 0 {
			Curser.Decode(HTB)
			PrintStructAsTOML(HTB)
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
		SendText(Message.Chat.ID, fmt.Sprintf("發生了奇怪的錯誤，送不出去這個東西：%+v", HTB), 0)
	}
}

func addHandler(Message *tgbotapi.Message, Keyword, Content string, Type int) {
	// check Keyword length
	if utf8.RuneCountInString(Keyword) >= 30 {
		SendText(Message.Chat.ID, fmt.Sprintf("關鍵字長度不可大於 30, 目前爲 %d 字”", utf8.RuneCountInString(Keyword)), Message.MessageID)
		return
	}
	// check content length
	if utf8.RuneCountInString(Content) >= 4000 {
		SendText(Message.Chat.ID, fmt.Sprintf("內容長度不可大於 4000, 目前爲 %d 字”", utf8.RuneCountInString(Content)), Message.MessageID)
		return
	}
	// find existing files
	Filter := bson.D{{
		"$and", bson.A{bson.D{{"Type", Type}}, bson.D{{"Keyword", Keyword}}, bson.D{{"Content", Content}}},
	}}
	if Rst := DB2.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).FindOne(context.TODO(), Filter); Rst.Err() != nil {
		SendText(Message.Chat.ID, "傳過了啦 腦霧?", Message.MessageID)
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
			log.Println("[HandleImg]", err)
			SendText(Message.Chat.ID, fmt.Sprintf("新增%s「%s」失敗：%s", CONFIG.GetNameByType(CONFIG.SETTING.TYPE.IMG), Keyword, err), Message.MessageID)
			Sum = ""
		} else {
			Sum = ImageCaptioning(Keyword, URL)
		}
	case CONFIG.SETTING.TYPE.ANI:
		// get url
		URL, err = bot.GetFileDirectURL(Content)
		if err != nil {
			log.Println("[handleAni]", err)
		}
		// get caption by thumbnail
		Thumb_URL, err := bot.GetFileDirectURL(Message.Animation.Thumbnail.FileID)
		if err != nil {
			log.Println("[handleAni]", err)
		}
		Sum = ImageCaptioning(Keyword, Thumb_URL)
	case CONFIG.SETTING.TYPE.VID:
		// get url
		URL, err = bot.GetFileDirectURL(Content)
		if err != nil {
			log.Println("[handleVid]", err)
		}
		// get caption by thumbnail
		Thumb_URL, err := bot.GetFileDirectURL(Message.Video.Thumbnail.FileID)
		if err != nil {
			log.Println("[handleVid]", err)
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
		log.Println("[new]", err)
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
	Filter := bson.D{{"Type", bson.D{{"$ne", 0}}}}
	opts := options.Find().SetSort(bson.D{{"Type", 1}})
	Curser, err := DB2.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).Find(context.TODO(), Filter, opts)
	defer Curser.Close(context.TODO())
	if err != nil {
		SendText(Message.Chat.ID, "搜尋失敗:"+err.Error(), Message.MessageID)
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

	Filter := bson.D{{"Keyword", BeDeletedKeyword}}
	opts := options.Find().SetSort(bson.D{{"Type", 1}})
	Curser, err := DB2.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).Find(context.TODO(), Filter, opts)
	defer Curser.Close(context.TODO())
	if err != nil {
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
		ReplyMarkup = append(ReplyMarkup, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(ShowEntry, "DEL_"+HTB.UID)))
		TB_HTB["DEL_"+HTB.UID] = &DeleteEntity{HTB: *HTB}
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
		Filter := bson.D{{"Keyword", bson.D{{"$ne", 0}}}}
		opts := options.Find().SetSort(bson.D{{"Type", 1}})
		Curser, err := DB2.Collection(CONFIG.GetColbyChatID(Message.Chat.ID)).Find(context.TODO(), Filter, opts)
		defer Curser.Close(context.TODO())

		if err != nil {
			log.Println("[Normal]", err)
			return
		}
		for Curser.Next(context.TODO()) {
			HTB := &HokTseBun{}
			Curser.Decode(HTB)

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

	addHandler(Message, Keyword, Content, CONFIG.SETTING.TYPE.IMG)
}

func handleAnimatedMessage(Message *tgbotapi.Message) {
	if Message.Caption == "" {
		return
	}
	var (
		Keyword     string = strings.TrimSpace(Message.Caption)
		Content     string
		Type        int
		MaxFileSize int = 20 * 1000 * 1000
		FileSize    int
	)

	switch {
	case Message.Animation != nil:
		FileSize = Message.Animation.FileSize
		Content = Message.Animation.FileID
		Type = 3

	case Message.Video != nil:
		FileSize = Message.Video.FileSize
		Content = Message.Video.FileID
		Type = 4
	}

	// check file size
	if FileSize >= MaxFileSize {
		SendText(
			Message.Chat.ID,
			fmt.Sprintf("新增失敗，目前檔案大小爲 %.2f MB，檔案大小上限爲 %.2f MB", float32(FileSize)/1000.0/1000.0, float32(MaxFileSize)/1000.0/1000.0),
			Message.MessageID,
		)
		return
	}
	addHandler(Message, Keyword, Content, Type)
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
	if _, err := bot.Send(editMsg); err != nil {
		log.Println("[CallQ]", err)
	}

	var ChatID = CallbackQuery.Message.Chat.ID
	switch {
	// handle "取消"
	case CallbackQuery.Data == "NIL_WITH_REACT":
		// 否
		// show respond
		callback := tgbotapi.NewCallback(CallbackQuery.ID, "不新增")
		if _, err := bot.Request(callback); err != nil {
			log.Println("[CallBQ]", err)
		}
		SendText(ChatID, "其實不按也沒差啦🈹", 0)
		if CallbackQuery.Message.ReplyToMessage != nil {
			bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.ReplyToMessage.MessageID))
		}
		bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.MessageID))
	case CallbackQuery.Data[:3] == "EXP":
		Command := strings.Fields(CallbackQuery.Data)[1]
		// send text tutorial
		var Text string = "[指令用途] %s\n[指令格式] %s\n[需要注意] %s\n實際使用範例如下圖:"
		switch Command {
		case "WHATISTHIS":
			Text = "我是複製文bot, 你可以:\n1. 新增複製文或圖片給我, 我會自動新增摘要/說明\n2. 提到關鍵字的時候, 我會把複製文抓出來鞭\n3. 我有搜尋功能, 也可以當作資料庫用"
		case "HOWMEDIA":
			Text = "請直接傳圖片/GIF/影片, 並附上註解(傳的時候下方可以輸入註解), bot 會自動新增\n實際使用範例如下圖:"
		case "HOWTXT":
			Text = "可以使用 /add 指令, 使用方法如下:\n" + Text
			Command = "ADD"
			fallthrough
		case "ADD":
			Text = fmt.Sprintf(Text, "新增複製文 (文字)", "/add {關鍵字} {內容}", "\n1. /add 和 /new 功能完全一樣, 愛用哪個用哪個\n2. 關鍵字可以重複, 但不建議\n3. 關鍵字不可過長")
		case "DEL":
			Text = fmt.Sprintf(Text, "根據關鍵字, 選擇並刪除複製文", "/delete {關鍵字}", "\n1. 確認刪除後無法復原\n2. 會先列出所有相同關鍵字的內容供選擇, 不會全部刪除")
		case "EXP":
			Text = fmt.Sprintf(Text, "查詢指令如何使用", "/example", "無")
		case "RAND":
			Text = fmt.Sprintf(Text, "隨機選取一篇資料庫內容給你", "/random", "無")
		case "SERC":
			Text = fmt.Sprintf(Text, "根據給定關鍵字, 在資料庫內搜尋, 並將結果私訊給你", "/search 關鍵字", "\n1. 要先私訊(啓動/開始)bot, 它才能私訊你\n2. 搜尋範圍包含關鍵字, 摘要, 內容\n3. 因爲是模糊搜尋, 所以會搜尋出一大堆不相關的\n4. 爲了防止被TG ban, 結果超過25筆會被取消, 可以換個關鍵字再搜尋")
		}
		SendText(ChatID, Text, 0)
		// send example image
		File := tgbotapi.FilePath(path.Join(CONFIG.SETTING.EXAMPLE_PIC_DIR, Command+".jpg"))
		replyMsg := tgbotapi.NewPhoto(ChatID, File)
		bot.Request(replyMsg)

		// delete prompt
		bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.MessageID))

	// handle deletion
	case CallbackQuery.Data[:4] == "DEL_":
		var (
			ok      bool
			DEntity *DeleteEntity
			UID     string
		)

		if CallbackQuery.Message.ReplyToMessage != nil {
			DEntity, ok = QueuedDeletes[ChatID][CallbackQuery.Message.ReplyToMessage.MessageID][CallbackQuery.Data]
		} else {
			DEntity, ok = QueuedDeletes[ChatID][CallbackQuery.Message.MessageID][CallbackQuery.Data]
		}

		if !ok {
			SendText(CallbackQuery.Message.Chat.ID, "bot 不知道爲啥壞了 笑死 你可以找作者出來講", 0)
		}

		UID = DEntity.HTB.UID
		switch {
		case !DEntity.Confirmed:
			DEntity.Confirmed = true

			// find HTB
			doc, err := DB.FindById(CONFIG.GetColbyChatID(CallbackQuery.Message.Chat.ID), UID)
			if err != nil {
				log.Println("[CallBQ]", err)
				return
			}
			if doc == nil {
				return
			}
			HTB := &HokTseBun{}
			doc.Unmarshal(HTB)

			ReplyMarkup := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✔️確認", "DEL_"+UID),
					tgbotapi.NewInlineKeyboardButtonData("✖️取消", "NIL_WITH_REACT"),
				),
			)
			// send confirmation
			switch HTB.Type {
			case 1:
				replyMsg := tgbotapi.NewMessage(ChatID, fmt.Sprintf("請再次確認是否要刪除「%s」：\n「%s」？", HTB.Keyword, HTB.Content))
				replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
				replyMsg.ReplyMarkup = ReplyMarkup
				_, err := bot.Send(replyMsg)
				if err != nil {
					log.Println("[CallBQ]", err)
				}

			case 2:
				Config := tgbotapi.NewPhoto(ChatID, tgbotapi.FileID(HTB.Content))
				Config.Caption = fmt.Sprintf("請再次確認是否要刪除「%s」？", HTB.Keyword)
				Config.ReplyToMessageID = CallbackQuery.Message.MessageID
				Config.ReplyMarkup = ReplyMarkup
				_, err := bot.Request(Config)
				if err != nil {
					log.Println("[CallBQ]", err)
				}

			case 3:
				Config := tgbotapi.NewAnimation(ChatID, tgbotapi.FileID(HTB.Content))
				Config.Caption = fmt.Sprintf("請再次確認是否要刪除「%s」？", HTB.Keyword)
				Config.ReplyToMessageID = CallbackQuery.Message.MessageID
				Config.ReplyMarkup = ReplyMarkup
				_, err := bot.Request(Config)
				if err != nil {
					log.Println("[CallBQ]", err)
				}

			case 4:
				Config := tgbotapi.NewVideo(ChatID, tgbotapi.FileID(HTB.Content))
				Config.Caption = fmt.Sprintf("請再次確認是否要刪除「%s」？", HTB.Keyword)
				Config.ReplyToMessageID = CallbackQuery.Message.MessageID
				Config.ReplyMarkup = ReplyMarkup
				_, err := bot.Request(Config)
				if err != nil {
					log.Println("[CallBQ]", err)
				}
			}
		case !DEntity.Done:
			DEntity.Done = true
			if err := DB.DeleteById(CONFIG.GetColbyChatID(CallbackQuery.Message.Chat.ID), UID); err != nil {
				log.Println("[CallBQ]", err)
				return
			}
			log.Printf("[DELETE] \"%s\" has been deleted!\n", DEntity.HTB.Keyword)
			SendText(CallbackQuery.Message.Chat.ID, fmt.Sprintf("已成功刪除「%s」", DEntity.HTB.Keyword), 0)

			if CallbackQuery.Message.ReplyToMessage != nil {
				bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.ReplyToMessage.MessageID))
			}
			bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.MessageID))
			delete(QueuedDeletes[ChatID], CallbackQuery.Message.ReplyToMessage.MessageID)
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
	// 			tgbotapi.NewInlineKeyboardButtonData("✔️確認", Keyword),
	// 			tgbotapi.NewInlineKeyboardButtonData("✖️取消", "NIL_WITH_REACT"),
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
