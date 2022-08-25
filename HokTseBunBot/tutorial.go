package main

import (
	"log"
	"os"
	"path"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func exampleHandler(Message *tgbotapi.Message) {
	replyMsg := tgbotapi.NewMessage(Message.Chat.ID, "請選擇要觀看的教學說明\n點擊指令按鈕可以查看指令用途")
	replyMsg.ReplyToMessageID = Message.MessageID
	replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("這個 bot 是幹嘛用的", "EXP WHATISTHIS")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("我要如何新增複製文?", "EXP HOWTXT")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("我要如何新增圖片/GIF/影片?", "EXP HOWMEDIA")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("什麼是私人/公共資料庫?", "EXP WHATISPUBLIC")),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("/add", "EXP ADD"),
			tgbotapi.NewInlineKeyboardButtonData("/random", "EXP RAND"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("/search", "EXP SERC"),
			tgbotapi.NewInlineKeyboardButtonData("/delete", "EXP DEL"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("/toggle", "EXP TOG"),
			tgbotapi.NewInlineKeyboardButtonData("✨/recent", "EXP RCNT"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("/status", "EXP STAT"),
			tgbotapi.NewInlineKeyboardButtonData("/dump", "EXP DUMP"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✨/nickname", "EXP NICK"),
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
