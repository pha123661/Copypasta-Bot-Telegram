package main

import (
	"log"
	"time"

	c "github.com/ostafen/clover/v2"
)

var DB *c.DB

type Dict map[string]interface{}

func InitDB() {
	// Open DB and create documents
	var err error
	DB, err = c.Open(CONFIG.DB.DIR)
	if err != nil {
		log.Panicln("[InitDB]", err)
	}
	DB.CreateCollection(CONFIG.DB.COLLECTION)
}

func InsertCP(FromID int, Keyword, Content string, Type int) error {
	var Summarization string
	switch Type {
	case 0:
		// Reserved
	case 1:
		// Text
		Summarization = GetOneSummarization(Keyword, Content)
	case 2:
		// Image
	}
	doc := c.NewDocument()
	doc.SetAll(Dict{
		"Type":          Type,
		"Keyword":       Keyword,
		"Summarization": Summarization,
		"Content":       Content,
		"From":          FromID,
		"CreateTime":    time.Now().Format("2006-01-02 15:04:05"),
	})

	_, err := DB.InsertOne(CONFIG.DB.COLLECTION, doc)
	if err != nil {
		log.Println("[InsertCP]", err)
	}
	return err
}
