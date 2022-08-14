package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func init_image() {
	if _, err := os.Stat(CONFIG.IMAGE_LOCATION); errors.Is(err, os.ErrNotExist) {
		// no file
		_, err := os.Create(CONFIG.IMAGE_LOCATION)
		if err != nil {
			log.Panicln(err)
		}
		IMAGE_CACHE = make(map[string]HokSeToo)
		log.Printf("[ImageDB] Create new \"%s\" successfully\n", CONFIG.IMAGE_LOCATION)
	} else {
		// load from file
		dataFile, err := os.Open(CONFIG.IMAGE_LOCATION)
		if err != nil {
			log.Panicln(err)
		}
		// decode
		dataDecoder := gob.NewDecoder(dataFile)
		err = dataDecoder.Decode(&IMAGE_CACHE)
		if err != nil {
			log.Panicln(err)
		}
		dataFile.Close()

		fmt.Println("Loaded imagedb")
		fmt.Println("#####################")

		log.Printf("[ImageDB] Read from \"%s\" successfully\n", CONFIG.IMAGE_LOCATION)
	}
}

func newToo(FileID tgbotapi.FileID) HokSeToo {
	return HokSeToo{FileID: FileID, summarization: ""}
}

func addToo2Cache(keyword string, Too HokSeToo) error {
	// append
	IMAGE_CACHE[keyword] = Too

	// flush to disk; rename old_file -> save new file
	// rename
	os.Rename(CONFIG.IMAGE_LOCATION, CONFIG.IMAGE_LOCATION+".bak")
	// save
	dataFile, err := os.Create(CONFIG.IMAGE_LOCATION)
	if err != nil {
		return err
	}
	dataEncoder := gob.NewEncoder(dataFile)
	err = dataEncoder.Encode(IMAGE_CACHE)

	dataFile.Close()
	return err
}

func handleImageMessage(bot *tgbotapi.BotAPI, Message *tgbotapi.Message) {
	if Message.Caption == "" {
		return
	}

	var Keyword = Message.Caption

	var FileID tgbotapi.FileID
	var max_area int = 0

	for _, image := range Message.Photo {
		/*"photo": [
			{
				"file_id": "for_access",
				"file_unique_id": "for_tg_internal_use",
				"file_size": 1703,
				"width": 90,
				"height": 90
			},
		],*/
		if image.Width*image.Height >= max_area {
			max_area = image.Width * image.Height
			FileID = tgbotapi.FileID(image.FileID)
		}

	}

	// find existing file
	if OldFile, is_exist := IMAGE_CACHE[Keyword]; is_exist {
		Queued_Overrides[Keyword] = newOverwriteImage(FileID)

		PhotoConfig := tgbotapi.NewPhoto(Message.Chat.ID, OldFile.FileID)

		PhotoConfig.Caption = fmt.Sprintf("「%s」已存在，確認是否覆蓋？", Keyword)
		PhotoConfig.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("是", Keyword),
				tgbotapi.NewInlineKeyboardButtonData("否", "NIL"),
			),
		)
		if _, err := bot.Request(PhotoConfig); err != nil {
			log.Println(err)
		}
		return
	}

	// add to cache
	addToo2Cache(Keyword, newToo(FileID))

	PhotoConfig := tgbotapi.NewPhoto(Message.Chat.ID, FileID)
	PhotoConfig.Caption = fmt.Sprintf("成功新增圖片「%s」", Keyword)
	if _, err := bot.Request(PhotoConfig); err != nil {
		log.Println(err)
	}
}
