package main

import (
	"context"
	"fmt"
	"log"
	"path"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
)

func CallQ(CallbackQuery *tgbotapi.CallbackQuery) {
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
		callback := tgbotapi.NewCallback(CallbackQuery.ID, "不新增")
		if _, err := bot.Request(callback); err != nil {
			log.Println("[CallBQ]", err)
		}
		SendText(ChatID, "其實不按也沒差啦🈹", 0)
		if CallbackQuery.Message.ReplyToMessage != nil {
			bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.ReplyToMessage.MessageID))
			delete(QueuedDeletes[ChatID], CallbackQuery.Message.ReplyToMessage.MessageID)
		}
		bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.MessageID))
		delete(QueuedDeletes[ChatID], CallbackQuery.Message.MessageID)

	case CallbackQuery.Data[:3] == "EXP":
		Command := strings.Fields(CallbackQuery.Data)[1]
		// send text tutorial
		var Text string = "[指令用途] %s\n[指令格式] %s\n[需要注意] %s\n實際使用範例如下圖:"
		switch Command {
		case "WHATISTHIS":
			Text = "我是複製文bot, 你可以:\n1. 新增複製文或圖片/GIF/影片給我, 我會自動新增摘要/說明\n2. 提到關鍵字的時候, 我會把複製文抓出來鞭 (推薦群組使用\n3. 我有搜尋功能, 也可以當作資料庫用"
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
		replyMsg.DisableNotification = true
		bot.Request(replyMsg)

		// delete prompt
		bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.MessageID))

	// handle deletion
	case CallbackQuery.Data[:4] == "DEL_":
		var (
			ok      bool
			DEntity *DeleteEntity
		)

		if CallbackQuery.Message.ReplyToMessage != nil {
			DEntity, ok = QueuedDeletes[ChatID][CallbackQuery.Message.ReplyToMessage.MessageID][CallbackQuery.Data]
		} else {
			DEntity, ok = QueuedDeletes[ChatID][CallbackQuery.Message.MessageID][CallbackQuery.Data]
		}

		if !ok {
			SendText(CallbackQuery.Message.Chat.ID, "bot 不知道爲啥壞了 笑死 你可以找作者出來講 跟他說92行壞掉了", 0)
		}

		switch {
		case !DEntity.Confirmed:
			DEntity.Confirmed = true

			// Find by id
			result := DB.Collection(CONFIG.GetColbyChatID(CallbackQuery.Message.Chat.ID)).FindOne(context.Background(), bson.M{"_id": DEntity.HTB.UID})
			if result.Err() != nil {
				log.Println("[CallBQ]", result.Err())
				return
			}

			HTB := &HokTseBun{}
			result.Decode(HTB)

			ReplyMarkup := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✔️確認", CallbackQuery.Data),
					tgbotapi.NewInlineKeyboardButtonData("✖️取消", "NIL_WITH_REACT"),
				),
			)
			// send confirmation
			switch HTB.Type {
			case 1:
				replyMsg := tgbotapi.NewMessage(ChatID, fmt.Sprintf("請再次確認是否要刪除「%s」：\n「%s」？", HTB.Keyword, HTB.Content))
				replyMsg.ReplyToMessageID = CallbackQuery.Message.MessageID
				replyMsg.ReplyMarkup = ReplyMarkup
				replyMsg.DisableNotification = true
				_, err := bot.Send(replyMsg)
				if err != nil {
					log.Println("[CallBQ]", err)
				}

			case 2:
				Config := tgbotapi.NewPhoto(ChatID, tgbotapi.FileID(HTB.Content))
				Config.Caption = fmt.Sprintf("請再次確認是否要刪除「%s」？", HTB.Keyword)
				Config.ReplyToMessageID = CallbackQuery.Message.MessageID
				Config.ReplyMarkup = ReplyMarkup
				Config.DisableNotification = true
				_, err := bot.Request(Config)
				if err != nil {
					log.Println("[CallBQ]", err)
				}

			case 3:
				Config := tgbotapi.NewAnimation(ChatID, tgbotapi.FileID(HTB.Content))
				Config.Caption = fmt.Sprintf("請再次確認是否要刪除「%s」？", HTB.Keyword)
				Config.ReplyToMessageID = CallbackQuery.Message.MessageID
				Config.ReplyMarkup = ReplyMarkup
				Config.DisableNotification = true
				_, err := bot.Request(Config)
				if err != nil {
					log.Println("[CallBQ]", err)
				}

			case 4:
				Config := tgbotapi.NewVideo(ChatID, tgbotapi.FileID(HTB.Content))
				Config.Caption = fmt.Sprintf("請再次確認是否要刪除「%s」？", HTB.Keyword)
				Config.ReplyToMessageID = CallbackQuery.Message.MessageID
				Config.ReplyMarkup = ReplyMarkup
				Config.DisableNotification = true
				_, err := bot.Request(Config)
				if err != nil {
					log.Println("[CallBQ]", err)
				}
			}
		case !DEntity.Done:
			DEntity.Done = true
			result := DB.Collection(CONFIG.GetColbyChatID(CallbackQuery.Message.Chat.ID)).FindOneAndDelete(context.Background(), bson.M{"_id": DEntity.HTB.UID})
			if result.Err() != nil {
				log.Println("[CallBQ]", result.Err())
				return
			}
			SendText(CallbackQuery.Message.Chat.ID, fmt.Sprintf("已成功刪除「%s」", DEntity.HTB.Keyword), 0)

			if CallbackQuery.Message.ReplyToMessage != nil {
				bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.ReplyToMessage.MessageID))
			}
			bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.MessageID))
			delete(QueuedDeletes[ChatID], CallbackQuery.Message.ReplyToMessage.MessageID)
		}
	}
}
