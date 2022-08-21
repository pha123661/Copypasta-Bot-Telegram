package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	toml "github.com/pelletier/go-toml/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var CONFIG cfg

type cfg struct {
	SETTING struct {
		TYPE struct {
			TXT, IMG, ANI, VID int
		}
		NAME struct {
			TXT, IMG, ANI, VID string
		}
		CONCURRENT struct {
			SUM, CAP struct {
				COOLDOWN int // ms
				LIMIT    int
			}
		}
		LOG_FILE        string
		EXAMPLE_PIC_DIR string
		EXAMPLE_TXT_DIR string
	}

	API struct {
		TG struct {
			TOKEN string
		}
		HF struct {
			TOKENs              []string
			CURRENT_TOKEN       string
			SUM_MODEL, MT_MODEL string
		}
		MONGO struct {
			USER string
			PASS string
			URI  string
		}
	}

	DB struct {
		DB_NAME, CFormat                     string
		GLOBAL_COL, CHAT_STATUS, USER_STATUS string
	}
}

func InitConfig(CONFIG_PATH string) {
	// parse toml file
	tomldata, err := os.ReadFile(CONFIG_PATH)
	if err != nil {
		panic("[InitConfig]" + err.Error())
	}

	if err := toml.Unmarshal(tomldata, &CONFIG); err != nil {
		panic("[InitConfig]" + err.Error())
	}

	// get secret configs from environment variables
	CONFIG.API.HF.TOKENs = strings.Fields(os.Getenv("API.HF.TOKENs"))
	CONFIG.API.TG.TOKEN = os.Getenv("API.TG.TOKEN")
	CONFIG.API.MONGO.URI = os.Getenv("API.MONGO.URI")
	CONFIG.DB.DB_NAME = os.Getenv("DB.DB_NAME")

	fmt.Println("********************\nConfig Loaded:")
	PrintStructAsTOML(CONFIG)
	fmt.Println("********************")
}

func PrintStructAsTOML(v interface{}) error {
	buf := bytes.Buffer{}
	enc := toml.NewEncoder(&buf)
	enc.SetIndentTables(true)
	if err := enc.Encode(v); err != nil {
		return err
	}
	fmt.Println(buf.String())
	return nil
}

var bot *tgbotapi.BotAPI

func main() {
	InitConfig("../../HokTseBunBot/config.toml")
	DBClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(CONFIG.API.MONGO.URI))
	if err != nil {
		panic("[InitConfig]" + err.Error())
	}
	DB := DBClient.Database(CONFIG.DB.DB_NAME)

	Collections, err := DB.ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		panic("[InitConfig]" + err.Error())
	}

	bot, _ = tgbotapi.NewBotAPI(CONFIG.API.TG.TOKEN)

	B, err := os.ReadFile("./WHATTOSAY.txt")
	if err != nil || len(B) == 0 {
		fmt.Println(err)
		return
	}
	Text := string(B)
	fmt.Println("是否要傳送：")
	fmt.Println("==============")
	fmt.Println(Text)
	fmt.Println("==============")
	fmt.Println("請按Enter確認")

	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

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

			SendText(ChatID, Text, 0)
		}
	}
	os.Truncate("./WHATTOSAY.txt", 0)
}

func SendText(ChatID int64, Content string, ReplyMsgID int) tgbotapi.Message {
	replyMsg := tgbotapi.NewMessage(ChatID, Content)
	if ReplyMsgID != 0 {
		replyMsg.ReplyToMessageID = ReplyMsgID
	}
	replyMsg.DisableNotification = true
	Msg, err := bot.Send(replyMsg)
	if err != nil {
		fmt.Printf("[SendTR] ChatID: %d, Content:%s, MeplyMsgID: %d\n", ChatID, Content, ReplyMsgID)
		fmt.Println("[SendTR]", err)
	}
	return Msg
}
