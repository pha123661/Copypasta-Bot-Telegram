package main

func init() {
	InitConfig("./config.toml")
}

func main() {
	// // keep alive
	// http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
	// 	fmt.Fprint(res, "Hello World!")
	// })
	// go http.ListenAndServe(":9000", nil)

	// // start bot
	// bot, err := tgbotapi.NewBotAPI(CONFIG.TELEGRAM_API_TOKEN)
	// if err != nil {
	// 	log.Panicln(err)
	// }
	// bot.Debug = true
	// fmt.Println("***", "Sucessful logged in as", bot.Self.UserName, "***")

	// // update config
	// updateConfig := tgbotapi.NewUpdate(0)
	// updateConfig.Timeout = 60

	// // get messages
	// updates := bot.GetUpdatesChan(updateConfig)
	// for update := range updates {
	// 	switch {
	// 	case update.Message != nil:
	// 		if update.Message.Photo != nil {
	// 			// handle image updates
	// 		} else {
	// 			// handle text updates
	// 		}
	// 	case update.CallbackQuery != nil:
	// 		// handle callback query

	// 	}
	// }
}
