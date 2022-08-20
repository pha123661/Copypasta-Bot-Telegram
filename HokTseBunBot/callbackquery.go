package main

import (
	"context"
	"fmt"
	"log"
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
		SendExample(ChatID, Command)
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
			// check did not toggle between deletion
			if ChatStatus[CallbackQuery.Message.Chat.ID].Global != DEntity.Global {
				SendText(CallbackQuery.Message.Chat.ID, "很皮哦 delete 時不能 toggle", 0)

				if CallbackQuery.Message.ReplyToMessage != nil {
					bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.ReplyToMessage.MessageID))
				}
				bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.MessageID))
				delete(QueuedDeletes[ChatID], CallbackQuery.Message.MessageID)
				return
			}

			var CollectionName string

			if ChatStatus[CallbackQuery.Message.Chat.ID].Global {
				CollectionName = CONFIG.DB.GLOBAL_COL
			} else {
				CollectionName = CONFIG.GetColbyChatID(CallbackQuery.Message.Chat.ID)
			}

			// Find by id
			result := DB.Collection(CollectionName).FindOne(context.Background(), bson.M{"_id": DEntity.HTB.UID})
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
			// check did not toggle between deletion
			if ChatStatus[CallbackQuery.Message.Chat.ID].Global != DEntity.Global {
				SendText(CallbackQuery.Message.Chat.ID, "很皮哦 在 delete 時不能 toggle", 0)

				if CallbackQuery.Message.ReplyToMessage != nil {
					bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.ReplyToMessage.MessageID))
				}
				bot.Request(tgbotapi.NewDeleteMessage(ChatID, CallbackQuery.Message.MessageID))
				delete(QueuedDeletes[ChatID], CallbackQuery.Message.ReplyToMessage.MessageID)
				return
			}

			var CollectionName string

			if ChatStatus[CallbackQuery.Message.Chat.ID].Global {
				CollectionName = CONFIG.DB.GLOBAL_COL
			} else {
				CollectionName = CONFIG.GetColbyChatID(CallbackQuery.Message.Chat.ID)
			}

			result := DB.Collection(CollectionName).FindOneAndDelete(context.Background(), bson.M{"_id": DEntity.HTB.UID})
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
