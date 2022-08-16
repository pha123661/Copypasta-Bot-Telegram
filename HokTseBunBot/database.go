package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
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
	UID           string    `json:"_id"`
	URL           string    `json:"URL"`
}

func (HTB *HokTseBun) IsText() bool {
	return (HTB.Type == 1)
}

func (HTB *HokTseBun) IsImage() bool {
	return (HTB.Type == 2)
}

func InitDB() {
	// Open DB and create documents
	var err error
	DB, err = c.Open(CONFIG.DB.DIR)
	if err != nil {
		log.Panicln("[InitDB]", err)
	}
	// DB.CreateCollection(CONFIG.DB.COLLECTION)
	// DB.ExportCollection(CONFIG.DB.COLLECTION, fmt.Sprintf("../BACKUP_%s.json", CONFIG.DB.COLLECTION))
	DB.DropCollection(CONFIG.DB.COLLECTION)
	DB.ImportCollection(CONFIG.DB.COLLECTION, "../123123123.json")

	// update out-dated documents
	var wg = &sync.WaitGroup{}

	semaphore := make(chan struct{}, 5) // maximum limit of chan, blocks when full

	docs, _ := DB.FindAll(c.NewQuery(CONFIG.DB.COLLECTION))
	for _, doc := range docs {
		HTB := &HokTseBun{}
		err := doc.Unmarshal(HTB)
		if err != nil {
			fmt.Println(err)
		}
		// Update image captions
		if HTB.IsImage() && HTB.Summarization == "" {
			wg.Add(1)
			// add caption
			go func() {
				semaphore <- struct{}{} // acquire to work (channel), blocks when the channel is full
				defer func() {
					wg.Done()
					<-semaphore // release
					fmt.Printf("[Done] Image %s\n", HTB.Keyword)
				}()
				
				fmt.Printf("[Updating] Image %s\n", HTB.Keyword)
				if HTB.URL == "" {
					URL, err := bot.GetFileDirectURL(HTB.Content)
					if err != nil {
						return // give up
					}
					HTB.URL = URL
				}
				Cap := ImageCaptioning(HTB.Keyword, HTB.URL)

				HTB.Summarization = Cap
				tmp_map := &Dict{}
				tmp_bytes, _ := json.Marshal(HTB)
				json.Unmarshal(tmp_bytes, tmp_map)
				DB.UpdateById(CONFIG.DB.COLLECTION, HTB.UID, *tmp_map)
			}()
			time.Sleep(2 * time.Second)
		}
	}
	wg.Wait() // wait for all updates to finish
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
		Summarization = TextSummarization(Keyword, Content)
	case 2:
		// Image
		URL, err := bot.GetFileDirectURL(Content)
		if err != nil {
			log.Println("[InsertCP]", err)
			break // do not do summarization
		}
		Summarization = ImageCaptioning(Keyword, URL)
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
	return Summarization, nil
}
