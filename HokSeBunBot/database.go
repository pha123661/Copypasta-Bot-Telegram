package main

import (
	"fmt"
	"log"
	"time"

	c "github.com/ostafen/clover/v2"
)

var DB *c.DB

type Dict map[string]interface{}
type HokTseBun struct {
	Type          int       `json:"Type"`
	Keyword       string    `json:"Keyword"`
	Summarization string    `json:"Summarization"`
	Content       string    `json:"Content"`
	From          int64     `json:"From"`
	CreateTime    time.Time `json:"CreateTime"`
	Uid           string    `json:"_id"`
}

func InitDB() {
	// Open DB and create documents
	var err error
	DB, err = c.Open(CONFIG.DB.DIR)
	if err != nil {
		log.Panicln("[InitDB]", err)
	}
	DB.CreateCollection(CONFIG.DB.COLLECTION)
	DB.ExportCollection(CONFIG.DB.COLLECTION, fmt.Sprintf("../BACKUP_%s.json", CONFIG.DB.COLLECTION))

	// docs, _ := DB.FindAll(c.NewQuery(CONFIG.DB.COLLECTION))
	// for _, doc := range docs {
	// 	var todo *HokTseBun = &HokTseBun{}
	// 	fmt.Println(doc.Get("_id").(string))
	// 	err := doc.Unmarshal(todo)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		panic(err)
	// 	}
	// 	fmt.Printf("%+v\n", todo)
	// 	fmt.Println(doc)
	// }
}

func InsertCP(FromID int64, Keyword, Content string, Type int64) (string, error) {
	var Summarization string
	var URL string
	switch Type {
	case 0:
		// Reserved
		return "", fmt.Errorf(`"InsertCP" not implemented for type 0`)
	case 1:
		// Text
		Summarization = GetOneSummarization(Keyword, Content)
		URL = ""
	case 2:
		// Image
		Summarization = ""
		URL, _ = bot.GetFileDirectURL(Content)
	}
	doc := c.NewDocument()
	doc.SetAll(Dict{
		"Type":          Type, // clover only supports int64
		"Keyword":       Keyword,
		"Summarization": Summarization,
		"Content":       Content,
		"URL":           URL,
		"From":          FromID,
		"CreateTime":    time.Now(),
	})

	_, err := DB.InsertOne(CONFIG.DB.COLLECTION, doc)
	if err != nil {
		log.Println("[InsertCP]", err)
	}
	return Summarization, err
}
