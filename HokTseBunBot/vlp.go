package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	hfapigo "github.com/TannerKvarfordt/hfapigo"
	gt "github.com/bas24/googletranslatefree"
	"github.com/juliangruber/go-intersect"
	"github.com/yanyiwu/gojieba"
)

var (
	SumSemaphore, CapSemaphore chan Empty
	SumCoolSema, CapCoolSema   sync.Mutex
	SumCool, CapCool           time.Duration
	Jieba                      *gojieba.Jieba
)

func InitVLP() {
	SetHFAPI()

	SumSemaphore = make(chan Empty, CONFIG.SETTING.CONCURRENT.SUM.LIMIT)
	CapSemaphore = make(chan Empty, CONFIG.SETTING.CONCURRENT.CAP.LIMIT)

	Jieba = gojieba.NewJieba()
	Jieba.AddWord("笑死")
	Jieba.AddWord("複製文")

	SumCool = time.Duration(CONFIG.SETTING.CONCURRENT.SUM.COOLDOWN) * time.Millisecond
	CapCool = time.Duration(CONFIG.SETTING.CONCURRENT.CAP.COOLDOWN) * time.Millisecond
}

func SetHFAPI() {
	TestHFAPI := func() error {
		_, err := hfapigo.SendSummarizationRequest(
			CONFIG.API.HF.SUM_MODEL,
			&hfapigo.SummarizationRequest{
				Inputs:  []string{"據了解，死者是88歲老翁，案發當時他剛運動完，正要走回家，但沒有走斑馬線，而是直接橫越馬路，而無照騎車的少年在閃避違規臨停的"},
				Options: *hfapigo.NewOptions().SetWaitForModel(true),
			},
		)
		return err
	}

	var success bool = false

	rand.Seed(time.Now().UTC().UnixNano())
	perm := rand.Perm(len(CONFIG.API.HF.TOKENs))

	for _, i := range perm {
		log.Println("Testing HF api:", CONFIG.API.HF.TOKENs[i][:8]+"...")
		hfapigo.SetAPIKey(CONFIG.API.HF.TOKENs[0])

		if err := TestHFAPI(); err == nil {
			CONFIG.API.HF.CURRENT_TOKEN = CONFIG.API.HF.TOKENs[i]
			success = true
			log.Println("HF api:", CONFIG.API.HF.TOKENs[i][:8]+"...", "is available")
			break
		} else {
			log.Printf("HF api \"%s\" is not available: %s\n", CONFIG.API.HF.TOKENs[i][:8]+"...", err)
		}
	}

	if !success {
		CONFIG.API.HF.CURRENT_TOKEN = ""
		log.Panicln("No available HF api!")
	}
}

func TestHit(Query string, KeySlice ...string) bool {
	var UseHmm = true
	QuerySet := Jieba.Cut(Query, UseHmm)
	QuerySet = append(QuerySet, Query)
	// sort strings by length
	sort.Slice(KeySlice, func(i, j int) bool {
		return len(KeySlice[i]) < len(KeySlice[j])
	})

	for _, Key := range KeySlice {
		var KeySet []string

		KeySet = Jieba.Extract(Key, Max(10, len(Key)/100))
		KeySet = append(KeySet, Key)

		rst := removeDuplicateStr(intersect.Hash(QuerySet, KeySet))
		sum := 0
		for _, r := range rst {
			sum += utf8.RuneCountInString(r.(string))
		}
		if sum >= 2 {
			fmt.Println("############")
			fmt.Println(Query, Key)
			fmt.Println(QuerySet, KeySet)
			fmt.Println(rst)
			return true
		}
	}

	return false
}

func removeDuplicateStr(strSlice []interface{}) []interface{} {
	allKeys := make(map[interface{}]bool)
	var list []interface{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func TextSummarization(Keyword, Content string) string {
	// cooldown
	SumCoolSema.Lock()
	time.Sleep(SumCool)
	SumCoolSema.Unlock()

	SumSemaphore <- Empty{} // acquire
	defer func() {
		<-SumSemaphore // release
	}()
	var Summarization string
	sresps, err := hfapigo.SendSummarizationRequest(
		CONFIG.API.HF.SUM_MODEL,
		&hfapigo.SummarizationRequest{
			Inputs:  []string{Content},
			Options: *hfapigo.NewOptions().SetWaitForModel(true),
		},
	)
	if err != nil {
		if err.Error() == `{"error":"Service Unavailable"}` {
			log.Printf("[TxtSum] Keyword:%s, Content: %s\n", Keyword, Content)
			log.Println("[TxtSum]", err)
			// input too long
			return ""
		} else {
			log.Printf("[TxtSum] Keyword:%s, Content: %s\n", Keyword, Content)
			log.Println("[TxtSum]", err)
			log.Println("[HuggingFace] API dead, switching token...")
			SetHFAPI()
			return TextSummarization(Keyword, Content)
		}
	} else {
		Summarization = sresps[0].SummaryText
	}
	log.Println("[TxtSum] Get request for", Keyword, "summarzation:", Summarization)
	return Summarization
}

func ImageCaptioning(Keyword, ImgEnc string) (string, error) {
	/*
		Preprocessing:
		1. Download image (as []byte) by given url
		2. base64encode the image

		Pipeline:
		1. "Image captioning" (image -> english)
		2. "Google translation" (english -> traditional chinese)
		// 2. "Machine translation" (image -> simplified chinese)
		// 3. "OpenCC" (simplified chinese -> traditional chinese)
	*/

	// cooldown
	CapCoolSema.Lock()
	time.Sleep(CapCool)
	CapCoolSema.Unlock()

	CapSemaphore <- Empty{} // acquire
	defer func() {
		<-CapSemaphore // release
	}()

	// No image captioning inference api available -> reply on user-hosted space api
	// get caption
	url := "https://hf.space/embed/OFA-Sys/OFA-Image_Caption/+/api/predict/"
	jsonStr := fmt.Sprintf("{\"data\": [\"data:image/jpg;base64,%s\"]}", ImgEnc)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		log.Printf("[ImgSum] Keyword:%s\n", Keyword)
		log.Println("[ImgSum]", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ImgSum] Keyword:%s\n", Keyword)
		log.Println("[ImgSum]", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("[ImgSum] Keyword:%s\n", Keyword)
		log.Printf("[ImgSum] received non 200 response code: %d", resp.StatusCode)
		return "", err
	}
	j := &struct {
		Data          []string  `json:"data"`
		Durations     []float32 `json:"durations"`
		Avg_durations []float32 `json:"avg_durations"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(j)
	if err != nil {
		log.Printf("[ImgSum] Keyword:%s\n", Keyword)
		log.Println("[ImgSum]", err)
		return "", err
	}

	// translate
	CaptionZHTW, err := gt.Translate(j.Data[0], "en", "zh-TW")
	if err != nil {
		log.Printf("[ImgSum] Keyword:%s\n", Keyword)
		log.Println("[ImgSum]", err)
		return "", err
	}
	log.Println("[ImgSum] Get request for", Keyword, "caption:", CaptionZHTW)
	return CaptionZHTW, nil
}

// helper functions
func DownloadImageToBase64(URL string) (string, error) {
	resp, err := http.Get(URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("received non 200 response code: %d", resp.StatusCode)
	}

	// read resp body -> []byte
	image, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	b64str := base64.StdEncoding.EncodeToString(image)

	return b64str, nil
}

type ImgurResponse struct {
	Data    ImageData `json:"data"`
	Status  int       `json:"status"`
	Success bool      `json:"success"`
}

type ImageData struct {
	Account_id int    `json:"account_id"`
	Animated   bool   `json:"animated"`
	Bandwidth  int    `json:"bandwidth"`
	DateTime   int    `json:"datetime"`
	Deletehash string `json:"deletehash"`
	Favorite   bool   `json:"favorite"`
	Height     int    `json:"height"`
	Id         string `json:"id"`
	In_gallery bool   `json:"in_gallery"`
	Is_ad      bool   `json:"is_ad"`
	Link       string `json:"link"`
	Name       string `json:"name"`
	Size       int    `json:"size"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	Views      int    `json:"views"`
	Width      int    `json:"width"`
}

func UploadToImgur(ImgEnc string) string {
	parameters := url.Values{"image": {ImgEnc}}
	req, err := http.NewRequest("POST", "https://api.imgur.com/3/image", strings.NewReader(parameters.Encode()))
	if err != nil {
		log.Println(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Client-ID "+CONFIG.API.IMGUR.CLIENTID)
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
	}

	var imgurResponse ImgurResponse
	json.NewDecoder(r.Body).Decode(&imgurResponse)
	if imgurResponse.Success {
		return imgurResponse.Data.Link
	}
	return ""
}

func MTen2zhcn(EN string) (string, error) {
	tresps, err := hfapigo.SendTranslationRequest(
		CONFIG.API.HF.MT_MODEL,
		&hfapigo.TranslationRequest{
			Inputs:  []string{EN},
			Options: *hfapigo.NewOptions().SetWaitForModel(true),
		},
	)
	if err != nil {
		return "", err
	}
	return tresps[0].TranslationText, nil
}
