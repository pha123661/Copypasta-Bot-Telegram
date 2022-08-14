package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var IMAGE_CACHE map[string]HokSeToo

type HokSeToo struct {
	FileID        tgbotapi.FileID
	summarization string
}

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
	// if Message.Caption == "" {
	// 	return
	// }

	var FileID tgbotapi.FileID
	var max_area int = 0

	for _, image := range Message.Photo {
		/*
			"photo": [
					{
						"file_id": "for_access",
						"file_unique_id": "for_tg_internal_use",
						"file_size": 1703,
						"width": 90,
						"height": 90
					},
				],
		*/
		if image.Width*image.Height >= max_area {
			max_area = image.Width * image.Height
			FileID = tgbotapi.FileID(image.FileID)
		}

	}
	// add to cache
	addToo2Cache("123", newToo(FileID))

	PhotoConfig := tgbotapi.NewPhoto(Message.Chat.ID, FileID)
	PhotoConfig.Caption = fmt.Sprintf("成功新增圖片「%s」", "123")

	bot.Request(PhotoConfig)
}
