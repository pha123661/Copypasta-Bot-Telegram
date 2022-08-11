# Hok_tse_bun_tgbot
A Ho̍k tsè bûn (copypasta in taiwanese) bot for telegram  

# Config Setup
How to: Rename ``sample_config.toml`` to ``config.toml`` and modify it!

---

Setting: ``FILELOCATION``   
Description: The location to store your copypastas  
Default Value: ``"../HokSeBun_db"``  

---

Setting: ``SUMMARIZATION_LOCATION``  
Description: The location to store your extracted summarization  
Default value: ``"../HokSeBun_db/Sums"``  

---

Setting: ``LOG_FILE``  
Description: Name of your log file  
Default value: ``"../log.log"``  

---

Setting: ``TELEGRAM_API_TOKEN``  
Description: API token for your telegram bot  
Default value: ``"YOUR_TELEGRAM_API_TOKEN"`` (no this does not work)  

---

Setting: ``HUGGINGFACE_TOKENs``   
Description: A list of huggingface tokens, bot switchs token whenever it fails (ex: quota exceeded)  
Default value= ``["YOUR_HUGGINGFACETOKEN1", "YOUR_HUGGINGFACETOKEN2",]``  

---

Setting: ``HUGGINGFACE_MODEL``  
Description: The desired model to use, any model which supports summarization in your language should work  
Default value = ``"csebuetnlp/mT5_multilingual_XLSum"``  
Note: The model should support inference api to work.  

---

# Deploy
1. setup your config file following the section above  
2. run ``cd HokSeBunBot && go run main.go``

# Usage
Support three commands atm  
1. echo: echo  
2. new/add: insert new Ho̍k tsè bûn into the database  
3. random: select a Ho̍k tsè bûn randomly and post it  
4. search: fuzzy search every file in database  
