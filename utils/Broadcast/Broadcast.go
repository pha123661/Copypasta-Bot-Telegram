package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var bot *tgbotapi.BotAPI

func main() {

	DBClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(os.Getenv("API.MONGO.URI")))
	if err != nil {
		panic("[InitConfig]" + err.Error())
	}
	DB := DBClient.Database(os.Getenv("DB.DB_NAME"))

	Collections, err := DB.ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		panic("[InitConfig]" + err.Error())
	}

	bot, _ = tgbotapi.NewBotAPI(os.Getenv("API.TG.TOKEN"))

	B, err := os.ReadFile("./WHATTOSAY.md")
	if err != nil || len(B) == 0 {
		fmt.Println(err)
		return
	}
	Text := string(B)
	fmt.Println("已登入爲:", bot.Self.UserName)
	fmt.Println("是否要傳送：")
	fmt.Println("==============")
	fmt.Println(Text)
	fmt.Println("==============")
	fmt.Println("請按Enter確認")

	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	fmt.Println("請問要送給\n[1] \\in DB\n[2] \\in UserStatus")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	switch input {
	case "1":
		r, _ := regexp.Compile(`^-?[1-9]\d*`)
		for _, ColName := range Collections {
			switch ColName {
			// case CONFIG.DB.CHAT_STATUS, CONFIG.DB.USER_STATUS, CONFIG.DB.GLOBAL_COL, "Beginner":
			// 	continue
			default:

				tmp := r.Find([]byte(ColName))
				if tmp == nil {
					continue
				}
				ChatID, _ := strconv.ParseInt(string(tmp), 10, 64)
				Chat, err := bot.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: ChatID}})
				if err != nil {
					fmt.Println(ChatID, err)
					continue
				}
				fmt.Printf("%+v\n", Chat)

				replyMsg := tgbotapi.NewMessage(ChatID, Text)
				replyMsg.DisableNotification = true
				replyMsg.ParseMode = "Markdown"
				replyMsg.DisableWebPagePreview = true
				_, err = bot.Send(replyMsg)
				if err != nil {
					fmt.Printf("[SendTR] ChatID: %d, Content:%s\n", ChatID, Text)
					fmt.Println("[SendTR]", err)
				}
			}
		}
	case "2":
		Curser, err := DB.Collection("0_UserStatus").Find(context.TODO(), bson.D{})
		if err != nil {
			panic(err)
		}
		for Curser.Next(context.TODO()) {
			var v interface{}
			Curser.Decode(&v)
			ChatID := v.(bson.D)[1].Value
			fmt.Println(ChatID)
			replyMsg := tgbotapi.NewMessage(ChatID.(int64), Text)
			replyMsg.DisableNotification = true
			replyMsg.ParseMode = "Markdown"
			replyMsg.DisableWebPagePreview = true
			_, err = bot.Send(replyMsg)
			if err != nil {
				fmt.Printf("[SendTR] ChatID: %d, Content:%s\n", ChatID, Text)
				fmt.Println("[SendTR]", err)
			}
		}
	default:
		fmt.Println("Get", input)
		return
	}
	os.Rename("./WHATTOSAY.md", "./SENT.md")
	os.Create("./WHATTOSAY.md")
}
