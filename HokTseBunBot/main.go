package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	xurls "mvdan.cc/xurls/v2"
)

/*
Doc format:

	Dict{
			"Type":          Type,
			"Keyword":       Keyword,
			"Summarization": Summarization,
			"Content":       Content,
			"From":          FromID,
			"CreateTime":    time.Now(),
		}
*/

var bot *tgbotapi.BotAPI

func init() {
	InitConfig("./config.toml")
	// setup log log_file
	log_file, err := os.OpenFile(CONFIG.SETTING.LOG_FILE, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	log_file_and_stdout := io.MultiWriter(os.Stdout, log_file)
	log.SetOutput(log_file_and_stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("*** Starting Server ***")
}

func main() {
	// keep alive
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Welcome to new server!")
		})
		http.ListenAndServe(":5050", nil)
	}()

	var err error
	// start bot
	bot, err = tgbotapi.NewBotAPI(CONFIG.API.TG.TOKEN)
	if err != nil {
		log.Panicln(err)
	}
	bot.Debug = true

	InitVLP()
	InitDB()

	// update config
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	log.Println("***", "Sucessful logged in as", bot.Self.UserName, "***")
	// get messages
	updates := bot.GetUpdatesChan(updateConfig)
	for update := range updates {
		switch {
		case update.Message != nil:
			switch {
			case update.Message.Photo != nil || update.Message.Animation != nil || update.Message.Video != nil:
				go MediaMessage(update.Message)
			case update.Message.IsCommand():
				go ParseCommand(update.Message)
			case update.Message.Text != "":
				// long messages are ignored
				if utf8.RuneCountInString(update.Message.Text) >= 200 {
					break
				}
				// messages contain url are ignored
				if xurls.Relaxed().FindString(update.Message.Text) != "" {
					break
				}
				go NormalTextMessage(update.Message)
			}
		case update.CallbackQuery != nil:
			// handle callback query
			go CallQ(update.CallbackQuery)
		case update.MyChatMember != nil:
			if update.MyChatMember.NewChatMember.Status == "restricted" || update.MyChatMember.NewChatMember.Status == "kicked" || update.MyChatMember.NewChatMember.Status == "left" {
				log.Println("[Kicked] Get Kicked by", update.MyChatMember.Chat.ID, update.MyChatMember.Chat.UserName, update.MyChatMember.Chat.Title)
			} else {
				log.Println("[Joining] Joining", update.MyChatMember.Chat.ID, update.MyChatMember.Chat.UserName, update.MyChatMember.Chat.Title)
				if update.MyChatMember.Chat.Type == "group" || update.MyChatMember.Chat.Type == "supergroup" {
					// get invited in a group
					Content := `歡迎使用，請輸入或點擊 /example 以查看使用方式
我的github: https://github.com/pha123661/Hok_tse_bun_tgbot`
					SendText(update.MyChatMember.Chat.ID, Content, 0)
				}
			}
		}
	}
}

func ParseCommand(Message *tgbotapi.Message) {
	// handle commands
	switch Message.Command() {
	// public available functions
	case "start":
		// Startup
		Content := `歡迎使用，請輸入或點擊 /example 以查看使用方式
我的github: https://github.com/pha123661/Hok_tse_bun_tgbot`
		SendText(Message.Chat.ID, Content, 0)

	case "example": // short: EXP
		exampleHandler(Message)

	case "random", "randimg", "randtxt": // short: RAND
		randomHandler(Message)

	case "toggle": // short: TOG
		toggleHandler(Message)

	case "status": // short: STAT
		statusHandler(Message)

	case "dump": // short DUMP
		dumpHandler(Message)

	case "recent": // short: RCNT
		recentHandler(Message)

	case "new", "add": // short: NEW, ADD
		Command_Args := strings.Fields(Message.CommandArguments())
		if len(Command_Args) <= 1 {
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("錯誤：指令格式爲 “/%s {關鍵字} {內容}”", Message.Command()))
			replyMsg.ReplyToMessageID = Message.MessageID
			replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("蛤 我不會啦 啥", "EXP HOWTXT")),
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("阿如果我想傳圖片/GIF/影片咧", "EXP HOWMEDIA")),
			)
			replyMsg.DisableNotification = true
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
			return
		}

		var Index int
		if strings.Contains(Command_Args[0], Command_Args[1]) {
			Index = FindNthSubstr(Message.Text, Command_Args[1], 2)
			if Index == -1 {
				Index = strings.Index(Message.Text, Command_Args[1])
			}
		} else {
			Index = strings.Index(Message.Text, Command_Args[1])
		}
		Keyword := Command_Args[0]
		Content := strings.TrimSpace(Message.Text[Index:])
		addHandler(Message, Keyword, Content, CONFIG.SETTING.TYPE.TXT)

	case "search": // short: SERC
		Command_Args := strings.Fields(Message.CommandArguments())
		if len(Command_Args) < 1 {
			replyMsg := tgbotapi.NewMessage(Message.Chat.ID, fmt.Sprintf("錯誤：指令格式爲 “/%s {關鍵字}”", Message.Command()))
			replyMsg.ReplyToMessageID = Message.MessageID
			replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("蛤 我不會啦 啥", "EXP SERC")))
			replyMsg.DisableNotification = true
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
			replyMsg.DisableNotification = true
			if _, err := bot.Send(replyMsg); err != nil {
				log.Println(err)
			}
			return
		}
		deleteHandler(Message)

	// internal usag
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
			log.Println(err)
			return
		}
		DB.Collection(CONFIG.GetColbyChatID(ChatID)).Drop(context.TODO())
		SendText(Message.Chat.ID, fmt.Sprintf("成功刪除 %d", ChatID), 0)

	case "import":

		var (
			SourceCol *mongo.Collection
			TargetCol = DB.Collection(CONFIG.GetColbyChatID(Message.Chat.ID))
		)

		if Message.CommandArguments() == "Beginner" {
			SourceCol = DB.Collection("Beginner")
		} else {
			SourceChatID, err := strconv.ParseInt(Message.CommandArguments(), 10, 64)
			if err != nil {
				SendText(Message.Chat.ID, fmt.Sprintf("匯入失敗: %s", err.Error()), Message.MessageID)
				log.Println(err)
				return
			}

			SourceCol = DB.Collection(CONFIG.GetColbyChatID(SourceChatID))
			// check if sourcecol exist
			Collections, err := DB.ListCollectionNames(context.TODO(), bson.D{})
			if err != nil {
				log.Panicln(err)
			}
			var i int
			for i = 0; i < len(Collections); i++ {
				if Collections[i] == CONFIG.GetColbyChatID(SourceChatID) {
					break
				}
			}
			if i == len(Collections) {
				SendText(Message.Chat.ID, fmt.Sprintf("匯入失敗: %d 不存在資料庫", SourceChatID), Message.MessageID)
				log.Printf("匯入失敗: %d 不存在資料庫", SourceChatID)
				return
			}
		}

		// type 0: system attribute
		Filter := bson.D{
			{Key: "$and",
				Value: bson.A{
					bson.D{{Key: "Type", Value: 0}},
					bson.D{{Key: "Keyword", Value: "Import"}},
					bson.D{{Key: "Content", Value: SourceCol.Name()}},
				}},
		}
		SingleRst := TargetCol.FindOne(context.TODO(), Filter)
		if SingleRst.Err() != mongo.ErrNoDocuments {
			SendText(Message.Chat.ID, "你之前匯入過了~", Message.MessageID)
			return
		}

		var docs []interface{}
		Curser, err := SourceCol.Find(context.TODO(), bson.D{})
		defer Curser.Close(context.TODO())
		if err != nil {
			SendText(Message.Chat.ID, fmt.Sprintf("匯入失敗: %s", err.Error()), Message.MessageID)
			log.Println(err)
			return
		}
		if err := Curser.All(context.TODO(), &docs); err != nil {
			SendText(Message.Chat.ID, fmt.Sprintf("匯入失敗: %s", err.Error()), Message.MessageID)
			log.Println(err)
			return
		}

		// Create tmp message
		to_be_delete_message := SendText(Message.Chat.ID, "匯入中，請稍後……", Message.MessageID)
		defer bot.Request(tgbotapi.NewDeleteMessage(Message.Chat.ID, to_be_delete_message.MessageID))

		_, err = TargetCol.InsertMany(context.TODO(), docs)
		if err != nil {
			SendText(Message.Chat.ID, fmt.Sprintf("匯入失敗: %s", err.Error()), Message.MessageID)
			log.Println(err)
			return
		}
		// update system attribute
		InsertHTB(TargetCol, &HokTseBun{Type: 0, Keyword: "Import", Content: SourceCol.Name()})
		SendText(Message.Chat.ID, fmt.Sprintf("成功從 %s 匯入 %d 筆資料", SourceCol.Name(), len(docs)), Message.MessageID)

	case "echo":
		// Echo
		SendText(Message.Chat.ID, Message.CommandArguments(), Message.MessageID)

	// authorized use
	case "refresh_beginner":
		if Message.CommandArguments() != CONFIG.API.TG.TOKEN {
			SendText(Message.Chat.ID, "亂什麼洨 幹你娘", Message.MessageID)
			return
		}
		DB.Collection("Beginner").Drop(context.TODO())
		if err := ImportCollection(DB, "Beginner", "./Beginner.json"); err != nil {
			SendText(Message.Chat.ID, "刷新失敗: "+err.Error(), Message.MessageID)
			log.Println(err)
			return
		}
		SendText(Message.Chat.ID, "成功刷新 Beginner DB", Message.MessageID)

	default:
		SendText(Message.Chat.ID, fmt.Sprintf("錯誤：我不會 “/%s” 啦QQ", Message.Command()), Message.MessageID)
	}
}

func NormalTextMessage(Message *tgbotapi.Message) {
	if Message.Text == "" || Message.Text == " " {
		return
	}

	CSLock.RLock()
	CSE := ChatStatus[Message.Chat.ID]
	CSLock.RUnlock()

	var Col *mongo.Collection
	if CSE.Global {
		Col = GLOBAL_DB.Collection(CONFIG.DB.GLOBAL_COL)
	} else {
		Col = DB.Collection(CONFIG.GetColbyChatID(Message.Chat.ID))
	}

	// asyc search
	go func() {
		var (
			Query         = Message.Text
			Limit         = Min(500, 100*utf8.RuneCountInString(Query))
			RunesPerImage = 200
		)
		Filter := bson.D{{Key: "Keyword", Value: bson.D{{Key: "$ne", Value: 0}}}}
		opts := options.Find().SetSort(bson.D{{Key: "Type", Value: 1}})
		Curser, err := Col.Find(context.TODO(), Filter, opts)
		defer func() { Curser.Close(context.TODO()) }()
		if err != nil {
			log.Printf("[Normal] Message: %+v\n", Message)
			log.Println("[Normal]", err)
			return
		}

		for Curser.Next(context.TODO()) {
			HTB := &HokTseBun{}
			Curser.Decode(HTB)

			// HIT := false
			// switch {
			// case utf8.RuneCountInString(Query) >= 3:
			// 	if fuzzy.Match(HTB.Keyword, Query) || (fuzzy.Match(Query, HTB.Keyword) && Abs(len(Query)-len(HTB.Keyword)) <= 3) || fuzzy.Match(Query, HTB.Summarization) {
			// 		HIT = true
			// 	}
			// case utf8.RuneCountInString(Query) >= 2:
			// 	if strings.Contains(Query, HTB.Keyword) || strings.Contains(HTB.Keyword, Query) {
			// 		HIT = true
			// 	}
			// case utf8.RuneCountInString(Query) == 1:
			// 	if utf8.RuneCountInString(HTB.Keyword) == 1 && Query == HTB.Keyword {
			// 		HIT = true
			// 	}
			// }
			HIT := TestHit(Query, HTB.Keyword, HTB.Summarization)
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

func MediaMessage(Message *tgbotapi.Message) {
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
	case Message.Photo != nil:
		max_area := 0
		for _, image := range Message.Photo {
			if image.Width*image.Height >= max_area {
				max_area = image.Width * image.Height
				FileSize = image.FileSize
				Content = image.FileID
			}
		}
		Type = CONFIG.SETTING.TYPE.IMG

	case Message.Animation != nil:
		FileSize = Message.Animation.FileSize
		Content = Message.Animation.FileID
		Type = CONFIG.SETTING.TYPE.ANI

	case Message.Video != nil:
		FileSize = Message.Video.FileSize
		Content = Message.Video.FileID
		Type = CONFIG.SETTING.TYPE.VID
	}

	if strings.Contains(Message.Caption, "/search") {
		searchMediaHandler(Message.Chat.ID, Message.From.ID, Content, Type)
		return
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
