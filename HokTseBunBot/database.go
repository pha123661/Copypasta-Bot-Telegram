package main

import (
	"context"
	"fmt"
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

	// Archive := "../HokTseBunArchive"
	// items, err := os.ReadDir(Archive)
	// if os.IsNotExist(err) {
	// 	fmt.Println("Skip importing")
	// 	return
	// }
	// if err != nil {
	// 	panic(err)
	// }

	// var wg sync.WaitGroup

	// for _, item_out := range items {
	// 	if item_out.IsDir() {
	// 		continue
	// 	}
	// 	tmp := strings.Split(item_out.Name(), "-")[0]
	// 	fmt.Println(item_out.Name())

	// 	wg.Add(1)
	// 	go func(item fs.DirEntry) {
	// 		ImportCollection(DB, item.Name()[len(tmp)+1+7:len(item.Name())-5], path.Join(Archive, item.Name()))
	// 		wg.Done()
	// 	}(item_out)
	// }
	// wg.Wait()
	// os.Rename(Archive, Archive+"_imported")

	// create index for every collection
	Collections, err = DB.ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		log.Panicln(err)
	}
	for _, Col_name := range Collections {
		if Col_name == CONFIG.DB.GLOBAL_COL || Col_name == CONFIG.DB.CHAT_STATUS || Col_name == CONFIG.DB.USER_STATUS {
			continue
		}
		Col := DB.Collection(Col_name)
		_, err := Col.Indexes().CreateMany(
			context.TODO(),
			[]mongo.IndexModel{
				// index 1
				{Keys: bson.D{
					{Key: "Type", Value: 1},
				}},
				// index 2
				{Keys: bson.D{
					{Key: "Type", Value: 1},
					{Key: "Keyword", Value: 1},
				}},
				// index 3
				{Keys: bson.D{
					{Key: "Type", Value: 1},
					{Key: "Keyword", Value: 1},
					{Key: "Content", Value: 1},
				}},
			},
		)
		if err != nil {
			panic(err)
		}
	}

	BuildStatusMap()
}

func BuildStatusMap() {
	// Import ChatStatus
	ChatStatus = make(map[int64]ChatStatusEntity)
	Curser, err := DB.Collection(CONFIG.DB.CHAT_STATUS).Find(context.TODO(), bson.D{})
	if err != nil {
		log.Panic(err)
	}
	for Curser.Next(context.TODO()) {
		CS := ChatStatusEntity{}
		Curser.Decode(&CS)
		ChatStatus[CS.ChatID] = CS
	}

	// Import UserStatus
	UserStatus = make(map[int64]UserStatusEntity)
	Curser, err = DB.Collection(CONFIG.DB.USER_STATUS).Find(context.TODO(), bson.D{})
	if err != nil {
		log.Panic(err)
	}
	for Curser.Next(context.TODO()) {
		US := UserStatusEntity{}
		Curser.Decode(&US)
		UserStatus[US.UserID] = US
	}

	fmt.Println(UserStatus, ChatStatus)
}

func UpdateChatStatus(CS ChatStatusEntity) error {
	COL := DB.Collection(CONFIG.DB.CHAT_STATUS)
	Filter := bson.D{{Key: "ChatID", Value: CS.ChatID}}
	Update := bson.D{{Key: "$set", Value: bson.D{{Key: "Global", Value: CS.Global}}}}

	// Update
	SRst := COL.FindOneAndUpdate(context.TODO(), Filter, Update)
	// Not in collection
	if SRst.Err() == mongo.ErrNoDocuments {
		_, err := COL.InsertOne(context.TODO(), bson.M{"ChatID": CS.ChatID, "Global": CS.Global})
		if err != nil {
			return err
		}
	} else if SRst.Err() != nil {
		return SRst.Err()
	}
	return nil
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
