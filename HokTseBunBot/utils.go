package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/joho/godotenv/autoload"
	toml "github.com/pelletier/go-toml/v2"
	"go.mongodb.org/mongo-driver/bson"
)

var CONFIG cfg

// QueuedDeletes[ChatID][MessageID][doc_id] = doc
var QueuedDeletes = make(map[int64]map[int]map[string]*DeleteEntity)

type Dict map[string]interface{}
type Empty struct{}

type MutexMap struct {
	mut     *sync.RWMutex       // handle concurrent access of chanMap
	chanMap map[int](chan bool) // dynamic mutexes map
}

func NewMutextMap() *MutexMap {
	return &MutexMap{
		mut:     new(sync.RWMutex),
		chanMap: make(map[int](chan bool)),
	}
}

// Acquire a lock, with timeout
func (mm *MutexMap) Lock(id int) error {
	// get global lock to read from map and get a channel
	mm.mut.Lock()
	if _, ok := mm.chanMap[id]; !ok {
		mm.chanMap[id] = make(chan bool, 1)
	}
	ch := mm.chanMap[id]
	mm.mut.Unlock()

	// try to write to buffered channel
	ch <- true
	return nil

}

// release lock
func (mm *MutexMap) Release(id int) {
	mm.mut.Lock()
	ch := mm.chanMap[id]
	mm.mut.Unlock()
	<-ch
}

type DeleteEntity struct {
	// info
	HTB HokTseBun
	// status
	Global    bool
	Confirmed bool
	Done      bool
}

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
		LOG_FILE           string
		EXAMPLE_PIC_DIR    string
		EXAMPLE_TXT_DIR    string
		BOT_TALK_THRESHOLD float32
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
		IMGUR struct {
			CLIENTID string
			SECRET   string
		}
	}

	DB struct {
		DB_NAME, GLOBAL_DB_NAME, CFormat     string
		GLOBAL_COL, CHAT_STATUS, USER_STATUS string
	}
}

// Gets collection name of given ChatID
func (Config cfg) GetColbyChatID(ChatID int64) string {
	return fmt.Sprintf(CONFIG.DB.CFormat, ChatID)
}

// Gets Chinese name of given Type
func (Config cfg) GetNameByType(Type int) string {
	switch Type {
	case Config.SETTING.TYPE.TXT:
		return Config.SETTING.NAME.TXT
	case Config.SETTING.TYPE.IMG:
		return Config.SETTING.NAME.IMG
	case Config.SETTING.TYPE.ANI:
		return Config.SETTING.NAME.ANI
	case Config.SETTING.TYPE.VID:
		return Config.SETTING.NAME.VID
	default:
		return "大便"
	}
}

func InitConfig(CONFIG_PATH string) {
	// parse toml file
	tomldata, err := os.ReadFile(CONFIG_PATH)
	if err != nil {
		log.Panicln("[InitConfig]", err)
	}

	if err := toml.Unmarshal(tomldata, &CONFIG); err != nil {
		log.Panicln("[InitConfig]", err)
	}

	// get secret configs from environment variables
	CONFIG.API.HF.TOKENs = strings.Fields(os.Getenv("APIHFTOKENs"))
	CONFIG.API.TG.TOKEN = os.Getenv("APITGTOKEN")
	CONFIG.API.MONGO.URI = os.Getenv("APIMONGOURI")
	CONFIG.DB.DB_NAME = os.Getenv("DBDB_NAME")
	CONFIG.DB.GLOBAL_DB_NAME = os.Getenv("DBGLOBAL_DB_NAME")
	CONFIG.API.IMGUR.CLIENTID = os.Getenv("APIIMGURCLIENTID")
	CONFIG.API.IMGUR.SECRET = os.Getenv("APIIMGURSECRET")

	fmt.Println("********************\nConfig Loaded:")
	PrintStructAsTOML(CONFIG)
	fmt.Println("********************")
}

func TruncateString(text string, width int) string {
	text = strings.TrimSpace(text)
	width = width - utf8.RuneCountInString("……")
	if utf8.RuneCountInString(text) > width {
		r := []rune(text)[:width]
		text = string(r) + "……"
	}
	return text
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func Min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a int, b int) int {
	if a > b {
		return a
	}
	return b
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

func FindNthSubstr(haystack, needle string, n int) int {
	start := strings.Index(haystack, needle)
	for start >= 0 && n > 1 {
		if start+len(needle) >= len(haystack) {
			return -1
		}
		start = strings.Index(haystack[start+len(needle):], needle) + start + len(needle)
		n--
	}
	return start
}

// Sends text message, set ReplyMsgID=0 to disable reply
func SendText(ChatID int64, Content string, ReplyMsgID int) tgbotapi.Message {
	replyMsg := tgbotapi.NewMessage(ChatID, Content)
	if ReplyMsgID != 0 {
		replyMsg.ReplyToMessageID = ReplyMsgID
	}
	replyMsg.DisableNotification = true
	Msg, err := bot.Send(replyMsg)
	if err != nil {
		log.Printf("[SendTR] ChatID: %d, Content:%s, MeplyMsgID: %d\n", ChatID, Content, ReplyMsgID)
		log.Println("[SendTR]", err)
	}
	return Msg
}

// Sends media message
func SendMultiMedia(ChatID int64, Caption string, FileID_Str string, Type int) *tgbotapi.APIResponse {
	var Msg *tgbotapi.APIResponse
	var err error
	var File tgbotapi.RequestFileData

	if _, err := bot.GetFile(tgbotapi.FileConfig{FileID: FileID_Str}); err != nil {
		// content is not FileID, but is URL
		// URL := FileID_Str
		// if Caption != "" {
		// 	SendText(ChatID, fmt.Sprintf("%s\n%s", Caption, URL), 0)
		// } else {
		// 	SendText(ChatID, URL, 0)
		// }
		// return nil
		File = tgbotapi.FileURL(FileID_Str)
	} else {
		File = tgbotapi.FileID(FileID_Str)
	}

	switch Type {
	case 1:
		log.Println("[SendIR] Sending text by SendMultiMedia")
		return nil

	case 2:
		Config := tgbotapi.NewPhoto(ChatID, File)
		if Caption != "" {
			Config.Caption = Caption
		}
		Config.DisableNotification = true
		Msg, err = bot.Request(Config)
		if !Msg.Ok {
			log.Printf("[SendIR] ChatID: %d, Caption:%s, FileID_Str: %s, Type: %d\n", ChatID, Caption, FileID_Str, Type)
			log.Println("[SendIR]", Msg.ErrorCode, Msg.Description, fmt.Sprintf("%+v", Config))
			SendText(ChatID, "傳不出來 tg在搞", 0)
		} else if err != nil {
			log.Printf("[SendIR] ChatID: %d, Caption:%s, FileID_Str: %s, Type: %d\n", ChatID, Caption, FileID_Str, Type)
			log.Println("[SendIR]", err)
			SendText(ChatID, "傳送失敗： "+err.Error(), 0)
		}

	case 3:
		Config := tgbotapi.NewAnimation(ChatID, File)
		if Caption != "" {
			Config.Caption = Caption
		}
		Config.DisableNotification = true
		Msg, err = bot.Request(Config)
		if !Msg.Ok {
			log.Printf("[SendIR] ChatID: %d, Caption:%s, FileID_Str: %s, Type: %d\n", ChatID, Caption, FileID_Str, Type)
			log.Println("[SendIR]", Msg.ErrorCode, Msg.Description, fmt.Sprintf("%+v", Config))
			SendText(ChatID, "傳不出來 tg在搞", 0)
		} else if err != nil {
			log.Printf("[SendIR] ChatID: %d, Caption:%s, FileID_Str: %s, Type: %d\n", ChatID, Caption, FileID_Str, Type)
			log.Println("[SendIR]", err)
			SendText(ChatID, "傳送失敗： "+err.Error(), 0)
		}

	case 4:
		Config := tgbotapi.NewVideo(ChatID, File)
		if Caption != "" {
			Config.Caption = Caption
		}
		Config.DisableNotification = true
		Msg, err = bot.Request(Config)
		if !Msg.Ok {
			log.Printf("[SendIR] ChatID: %d, Caption:%s, FileID_Str: %s, Type: %d\n", ChatID, Caption, FileID_Str, Type)
			log.Println("[SendIR]", Msg.ErrorCode, Msg.Description, fmt.Sprintf("%+v", Config))
			SendText(ChatID, "傳不出來 tg在搞", 0)
		} else if err != nil {
			log.Printf("[SendIR] ChatID: %d, Caption:%s, FileID_Str: %s, Type: %d\n", ChatID, Caption, FileID_Str, Type)
			log.Println("[SendIR]", err)
			SendText(ChatID, "傳送失敗： "+err.Error(), 0)
		}

	}
	return Msg
}

func DeleteFieldFromJson(field string, jsonbytes []byte) ([]byte, error) {
	var docs_interface interface{}
	err := json.Unmarshal(jsonbytes, &docs_interface)
	if err != nil {
		return make([]byte, 0), err
	}

	for i := 0; i < len(docs_interface.([]interface{})); i++ {
		delete(docs_interface.([]interface{})[i].(map[string]interface{}), "_id")
	}

	return json.Marshal(docs_interface)
}

func GetUserNameByID(ChatID int64) ([]rune, error) {
	C, err := bot.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: ChatID}})
	if err != nil {
		return make([]rune, 0), err
	}
	switch C.Type {
	case "private":
		return []rune(C.UserName), nil
	case "group", "supergroup":
		return []rune(C.Title), nil
	default:
		return []rune(C.Title + C.FirstName + C.LastName + C.UserName), nil
	}
}

func GetMaskedNameByID(FromID int64) string {
	Filter1 := bson.M{"$and": bson.A{
		bson.M{"TGUserID": FromID},
		bson.M{"Nickname": bson.M{"$exists": true}},
		bson.M{"Nickname": bson.M{"$ne": ""}},
	}}
	Filter2 := bson.M{"$and": bson.A{
		bson.M{"DCUserID": FromID},
		bson.M{"Nickname": bson.M{"$exists": true}},
		bson.M{"Nickname": bson.M{"$ne": ""}},
	}}
	Filter := bson.M{"$or": bson.A{Filter1, Filter2}}

	if SRst := GLOBAL_DB.Collection(CONFIG.DB.USER_STATUS).FindOne(context.TODO(), Filter); SRst.Err() == nil {
		// has nickname
		var US UserStatusEntity
		SRst.Decode(&US)
		return US.Nickname
	}
	NameRune, err := GetUserNameByID(FromID)
	if err != nil {
		return "DC使用者"
	}

	Mask := strings.Repeat("*", Max(len(NameRune)-6, 2))
	UnmaskIdx := (len(NameRune) - utf8.RuneCountInString(Mask)) / 2

	return string(NameRune[:UnmaskIdx]) + Mask + string(NameRune[UnmaskIdx+utf8.RuneCountInString(Mask):])
}

func Sha256String(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

// Max Priority queue
type HTB_pq struct {
	HTB      HokTseBun
	priority float32
}

func (t *HTB_pq) Less(other interface{}) bool {
	return t.priority > other.(*HTB_pq).priority
}
