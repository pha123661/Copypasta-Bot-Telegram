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

var DB *mongo.Database

type HokTseBun struct {
	UID           primitive.ObjectID `bson:"_id" json:"_id"`
	Type          int                `bson:"Type"`
	Keyword       string             `bson:"Keyword"`
	Summarization string             `bson:"Summarization"`
	Content       string             `bson:"Content"`
	From          int64              `bson:"From"`
	CreateTime    time.Time          `bson:"CreateTime"`
	URL           string             `bson:"URL"`
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
	DB = DBClient.Database(CONFIG.DB.DB_NAME)

	Collections, err := DB.ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		log.Panicln(err)
	}

	var BeginnerExists bool
	for _, Col := range Collections {
		if Col == "Beginner" {
			BeginnerExists = true
			break
		}
	}

	if !BeginnerExists {
		if err := ImportCollection(DB, "Beginner", "./Beginner.json"); err != nil {
			log.Println("Begginer initialization failed!")
			log.Panicln(err)
			return
		} else {
			log.Println("Begginer initialized")
		}
	}
}

func InsertHTB(Collection string, HTB *HokTseBun) (primitive.ObjectID, error) {
	// Create doc
	// doc, err := bson.Marshal(HTB)
	// if err != nil {
	// 	log.Println(err)
	// }
	doc := bson.D{
		{Key: "Type", Value: HTB.Type},
		{Key: "Keyword", Value: HTB.Keyword},
		{Key: "Summarization", Value: HTB.Summarization},
		{Key: "Content", Value: HTB.Content},
		{Key: "From", Value: HTB.From},
		{Key: "CreateTime", Value: time.Now()},
		{Key: "URL", Value: HTB.URL},
	}

	// Insert doc
	Col := DB.Collection(Collection)
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
		log.Printf("[ImpCol], Col: %s, path: %s\n", Collection, path)
		log.Println("[ImpCol]", err)
		return err
	}
	jsonbytes, err = DeleteFieldFromJson("_id", jsonbytes)
	if err != nil {
		log.Printf("[ImpCol], Col: %s, path: %s\n", Collection, path)
		log.Println("[ImpCol]", err)
		return err
	}
	bson.UnmarshalExtJSON(jsonbytes, true, &docs)
	_, err = DB.Collection(Collection).InsertMany(context.TODO(), docs)
	return err
}
