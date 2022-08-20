package main

import (
	"log"
	"os"
	"path"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("/toggle 指令", "EXP TOG"),
			tgbotapi.NewInlineKeyboardButtonData("/status 指令", "EXP STAT"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("/dump 指令", "EXP DUMP"),
			tgbotapi.NewInlineKeyboardButtonData("✖️取消", "NIL_WITH_REACT"),
		),
	)
	replyMsg.DisableNotification = true
	if _, err := bot.Send(replyMsg); err != nil {
		log.Printf("[exp], %+v\n", Message)
		log.Println("[exp]", err)
		return
	}
}

func SendExample(ChatID int64, Command string) {
	// send text tutorial
	var Buff []byte
	switch Command {
	}
	Buff, _ = os.ReadFile(path.Join(CONFIG.SETTING.EXAMPLE_TXT_DIR, Command+".txt"))
	SendText(ChatID, string(Buff), 0)
	// send example image
	File := tgbotapi.FilePath(path.Join(CONFIG.SETTING.EXAMPLE_PIC_DIR, Command+".jpg"))
	replyMsg := tgbotapi.NewPhoto(ChatID, File)
	replyMsg.DisableNotification = true
	bot.Request(replyMsg)
}
