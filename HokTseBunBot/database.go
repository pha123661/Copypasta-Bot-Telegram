package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	DB         *mongo.Database
	ChatStatus map[int64]ChatStatusEntity
	CSLock     sync.RWMutex
	UserStatus map[int64]UserStatusEntity
	USLock     sync.RWMutex
)

type ChatStatusEntity struct {
	ChatID int64 `bson:"ChatID"`
	Global bool  `bson:"Global"`
}

type UserStatusEntity struct {
	UserID       int64 `bson:"UserID"`
	Contribution int   `bson:"Contribution"`
	Banned       bool  `bson:"Banned"`
}

type HokTseBun struct {
	UID           primitive.ObjectID `bson:"_id"`
	Type          int                `bson:"Type"`
	Keyword       string             `bson:"Keyword"`
	Summarization string             `bson:"Summarization"`
	Content       string             `bson:"Content"`
	From          int64              `bson:"From"`
	CreateTime    time.Time          `bson:"CreateTime"`
	URL           string             `bson:"URL"`
	FileUniqueID  string             `bson:"FileUniqueID"`
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
	// Create default collections
	DB.CreateCollection(context.TODO(), CONFIG.DB.GLOBAL_COL)
	DB.CreateCollection(context.TODO(), CONFIG.DB.USER_STATUS)
	DB.CreateCollection(context.TODO(), CONFIG.DB.CHAT_STATUS)
	// create index for every collection
	Collections, err = DB.ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		log.Panicln(err)
	}

	AddFileUIDForText := func(Col *mongo.Collection) {
		// Add file UID for text
		Filter := bson.M{
			"$and": bson.A{
				bson.M{"Type": 1},
				bson.M{"$or": bson.A{
					bson.M{"FileUniqueID": ""},
					bson.M{"FileUniqueID": bson.M{"$exists": false}},
				}},
			},
		}

		Curser, err := Col.Find(context.TODO(), Filter)
		defer func() { Curser.Close(context.TODO()) }()
		if err != nil {
			log.Panic(err)
		}

		for Curser.Next(context.TODO()) {
			var doc HokTseBun
			Curser.Decode(&doc)

			FileUniqueID := Sha256String(doc.Content)
			Update := bson.D{{Key: "$set", Value: bson.D{{Key: "FileUniqueID", Value: FileUniqueID}}}}
			Col.UpdateByID(context.TODO(), doc.UID, Update)
			fmt.Println("Added FUID")
		}
	}
	var wg sync.WaitGroup
	for _, Col_name := range Collections {
		Col := DB.Collection(Col_name)
		// Update index
		// Col.Indexes().DropAll(context.TODO())
		switch Col_name {
		case CONFIG.DB.CHAT_STATUS:
			_, err := Col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
				Keys:    bson.D{{Key: "ChatID", Value: 1}},
				Options: options.Index().SetUnique(true),
			})
			if err != nil {
				panic(err)
			}
		case CONFIG.DB.USER_STATUS:
			_, err := Col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
				Keys:    bson.D{{Key: "UserID", Value: 1}},
				Options: options.Index().SetUnique(true),
			})
			if err != nil {
				panic(err)
			}
		case CONFIG.DB.GLOBAL_COL:
			wg.Add(1)
			go func() {
				AddFileUIDForText(Col)
				wg.Done()
			}()
			_, err := Col.Indexes().CreateMany(context.TODO(), []mongo.IndexModel{
				// index 1
				{
					Keys: bson.D{{Key: "Type", Value: 1}},
				},
				// index 2
				{
					Keys: bson.D{{Key: "Type", Value: 1}, {Key: "Keyword", Value: 1}},
				},
				// index 3
				{
					Keys:    bson.D{{Key: "Type", Value: 1}, {Key: "Keyword", Value: 1}, {Key: "FileUniqueID", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
			})
			if err != nil {
				panic(err)
			}

		default:
			wg.Add(1)
			go func() {
				AddFileUIDForText(Col)
				wg.Done()
			}()
			_, err := Col.Indexes().CreateMany(context.TODO(), []mongo.IndexModel{
				// index 1
				{
					Keys: bson.D{{Key: "Type", Value: 1}},
				},
				// index 2
				{
					Keys: bson.D{{Key: "Type", Value: 1}, {Key: "Keyword", Value: 1}},
				},
				// index 3
				{
					Keys: bson.D{{Key: "Type", Value: 1}, {Key: "Keyword", Value: 1}, {Key: "FileUniqueID", Value: 1}},
				},
			})
			if err != nil {
				panic(err)
			}
		}
	}
	wg.Wait()

	BuildStatusMap()
}

func BuildStatusMap() {
	CSLock.Lock()
	// Import ChatStatus
	ChatStatus = make(map[int64]ChatStatusEntity)
	Curser, err := DB.Collection(CONFIG.DB.CHAT_STATUS).Find(context.TODO(), bson.D{})
	defer func() { Curser.Close(context.TODO()) }()
	if err != nil {
		log.Panic(err)
	}
	for Curser.Next(context.TODO()) {
		CS := ChatStatusEntity{}
		Curser.Decode(&CS)
		ChatStatus[CS.ChatID] = CS
	}
	CSLock.Unlock()

	USLock.Lock()
	// Import UserStatus
	UserStatus = make(map[int64]UserStatusEntity)
	Curser, err = DB.Collection(CONFIG.DB.USER_STATUS).Find(context.TODO(), bson.D{})
	defer func() { Curser.Close(context.TODO()) }()
	if err != nil {
		log.Panic(err)
	}
	for Curser.Next(context.TODO()) {
		US := UserStatusEntity{}
		Curser.Decode(&US)
		UserStatus[US.UserID] = US
	}
	USLock.Unlock()

}

func GetLBInfo(num int64) (string, error) {
	opts := options.Find().SetLimit(num).SetSort(bson.D{{Key: "Contribution", Value: -1}})
	Curser, err := DB.Collection(CONFIG.DB.USER_STATUS).Find(context.TODO(), bson.D{}, opts)
	defer func() { Curser.Close(context.TODO()) }()
	if err != nil {
		return "", err
	}

	var (
		Ret strings.Builder
		Idx int = 0
		US  UserStatusEntity
	)
	Ret.WriteString("目前排行榜：\n")
	for Curser.Next(context.TODO()) {
		Idx++
		Curser.Decode(&US)
		Con := US.Contribution
		UserID := US.UserID

		Name := GetMaskedNameByID(UserID)
		Ret.WriteString(fmt.Sprintf("%d. %v, 貢獻值:%d\n", Idx, Name, Con))
	}
	return Ret.String(), nil
}

func UpdateChatStatus(CS ChatStatusEntity) error {
	COL := DB.Collection(CONFIG.DB.CHAT_STATUS)
	Filter := bson.D{{Key: "ChatID", Value: CS.ChatID}}
	Update := bson.D{{Key: "$set", Value: bson.D{{Key: "Global", Value: CS.Global}}}}
	opts := options.FindOneAndUpdate().SetUpsert(true)

	// Update
	SRst := COL.FindOneAndUpdate(context.TODO(), Filter, Update, opts)
	if SRst.Err() != nil && SRst.Err() != mongo.ErrNoDocuments {
		return SRst.Err()
	}
	return nil
}

func AddUserContribution(UserID int64, DeltaContribution int) (int, error) {
	COL := DB.Collection(CONFIG.DB.USER_STATUS)
	Filter := bson.D{{Key: "UserID", Value: UserID}}
	Update := bson.D{{Key: "$inc", Value: bson.D{{Key: "Contribution", Value: DeltaContribution}}}}
	comment := fmt.Sprintf("Increment %d contribution by %d", UserID, DeltaContribution)
	opts := options.FindOneAndUpdate().SetUpsert(true).SetComment(comment)

	// Update
	SRst := COL.FindOneAndUpdate(context.TODO(), Filter, Update, opts)
	if SRst.Err() != nil && SRst.Err() != mongo.ErrNoDocuments {
		return 0, SRst.Err()
	}

	NewUS := &UserStatusEntity{}
	SRst.Decode(NewUS)
	NewUS.Contribution += DeltaContribution

	// Update cache
	USLock.Lock()
	UserStatus[UserID] = *NewUS
	USLock.Unlock()

	return NewUS.Contribution, nil
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
		{Key: "FileUniqueID", Value: HTB.FileUniqueID},
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
