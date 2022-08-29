# What Is This?
A "Ho̍k tsè bûn" / "copypasta" / "複製文" bot for telegram  
Invite me: https://t.me/HokTseBunBot  

# Features
* Post related copypasta for you whenever the bot detects matching keyword  
* Generate summarization (or caption, for media) automaticly by utlizing top-of-the-line DL models  
  * Model used for text summarization: [csebuetnlp/mT5_multilingual_XLSum](https://huggingface.co/csebuetnlp/mT5_multilingual_XLSum)  
  * Model used for image captioning: [OFA-Sys/OFA-base](https://huggingface.co/OFA-Sys/OFA-base)

[`#ImageCaptioning`](https://paperswithcode.com/task/image-captioning)  
[`#TextGeneration`](https://paperswithcode.com/task/text-generation)  
[`#TextSummarization`](https://paperswithcode.com/task/text-summarization)  

# DEMO
### Insert Text   
![Imgur](https://imgur.com/s2z5lsH.jpg)  
### Insert Media  
![Imgur](https://imgur.com/WYzxE6R.jpg)  
### Post copypasta
![Imgur](https://imgur.com/uKzFxLT.jpg)  
and more......  

# Usage  
Supported commands atm:  
1. new/add: insert new Ho̍k tsè bûn into the database  
1. random: select a Ho̍k tsè bûn randomly and post it  
1. search: fuzzy search every file in database  
1. Automatically insert new media into the database whenever an media with caption is sent  
1. delete: delete copypasta
1. example: show tutorial  

# Deploy on [Replit.com](http://replit.com/)
1. Import the repo into replit
2. Setup your config file following section "Config Setup" and move it to ``/HokSeBunBot``
3. Run (first time only)
```go
go mod init HokSeBunBot  
go mod tidy
```
4. Make sure your replit run command is ``cd HokSeBunBot && go run .``
5. (Optional) In order to make the bot work in group chat, turn off privacy mode for you bot (by using BotFother).
6. Profit!  


# Config Setup
## Environment Variable
Environment Variable: ``API.TG.TOKEN``  
Description: API token for your telegram bot  
Default value: ``"YOUR_TELEGRAM_API_TOKEN"`` (no this does not work)  

---

Environment Variable: ``API.HT.TOKENs``   
Description: A list of huggingface tokens, bot switchs token whenever it fails (ex: quota exceeded)  
Default value= ``["YOUR_HUGGINGFACETOKEN1", "YOUR_HUGGINGFACETOKEN2",]``  

---

## [SETTING]

Setting: ``LOG_FILE``  
Description: Name of your log file  
Default value: ``"../log.log"``  

---

## [API]
### [API.HF]

---

Setting: ``SUM_MODEL``  
Description: The desired model to use for summarization, any model which supports summarization in your language should work  
Default value = ``"csebuetnlp/mT5_multilingual_XLSum"``  
Note: The model should support inference api to work.  

---

Setting: ``MT_MODEL``  
Description: The desired model to use for translation, any model which supports translation in your language should work  
Default value = ``"Helsinki-NLP/opus-mt-en-zh"``  
Note: This setting isn't working since the bot uses google translate

---

## [DB]

Setting: ``DIR``   
Description: The location to store your [clover](https://github.com/ostafen/clover) database  
Default Value: ``"../HokSeBun_db"``  

---

Setting: ``COLLECTION``   
Description: The collection name in your [clover](https://github.com/ostafen/clover) database  
Default Value: ``"Copypasta"``  
