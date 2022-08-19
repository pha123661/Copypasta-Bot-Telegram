package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB2 *mongo.Database

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

func (HTB *HokTseBun) IsAnimation() bool {
	return (HTB.Type == 3)
}

func (HTB *HokTseBun) IsVideo() bool {
	return (HTB.Type == 4)
}

func (HTB *HokTseBun) IsMultiMedia() bool {
	return HTB.IsImage() || HTB.IsAnimation() || HTB.IsVideo()
}

func InitDB() {
	var err error
	DBClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(CONFIG.API.MONGO.URI))
	if err != nil {
		log.Panicln(err)
	}
	DB2 = DBClient.Database(CONFIG.DB.DB_NAME)
}

func InsertHTB(Collection string, HTB *HokTseBun) (primitive.ObjectID, error) {
	// Create doc
	HTB.CreateTime = time.Now()
	doc, err := bson.Marshal(HTB)
	if err != nil {
		log.Println(err)
	}

	// Insert doc
	Col := DB2.Collection(Collection)
	InRst, err := Col.InsertOne(context.TODO(), doc)
	if err != nil {
		return primitive.ObjectID{}, err
	}
	return InRst.InsertedID.(primitive.ObjectID), nil
}

func ImportCollection(DB *mongo.Database, Collection, path string) error {
	var docs []interface{}
	jsonbytes, err := os.ReadFile(path)
	if err != nil {
		log.Println(err)
		return err
	}
	bson.UnmarshalExtJSON(jsonbytes, true, docs)
	DB.Collection(Collection).InsertMany(context.TODO(), docs)
	return nil
}
