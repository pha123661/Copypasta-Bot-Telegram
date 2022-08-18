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
	"sync"
	"time"

	hfapigo "github.com/TannerKvarfordt/hfapigo"
	gt "github.com/bas24/googletranslatefree"
)

var (
	SumSemaphore, CapSemaphore chan Empty
	SumCoolSema, CapCoolSema   sync.Mutex
	SumCool, CapCool           time.Duration
)

func InitVLP() {
	SumSemaphore = make(chan Empty, CONFIG.SETTING.CONCURRENT.SUM.LIMIT)
	CapSemaphore = make(chan Empty, CONFIG.SETTING.CONCURRENT.CAP.LIMIT)

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
			break
		} else {
			log.Printf("HF api \"%s\" not available: %s\n", CONFIG.API.HF.TOKENs[i][:8]+"...", err)
		}
	}

	if !success {
		CONFIG.API.HF.CURRENT_TOKEN = ""
		log.Panicln("No available hf api!")
	}
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
			log.Println(err)
			// input too long
			Summarization = ""
		} else {
			log.Println(err)
			log.Println("[HuggingFace] API dead, switching token...")
			SetHFAPI()
			return TextSummarization(Keyword, Content)
		}
	} else {
		Summarization = sresps[0].SummaryText
	}
	log.Println("[HuggingFace] Get request for", Keyword, "summarzation:", Summarization)
	return Summarization
}

func ImageCaptioning(Keyword, Image_URL string) string {
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

	// Encode image
	ImgEnc, err := DownloadImageToBase64(Image_URL)
	if err != nil {
		log.Println("[ImgSum]", err)
		return ""
	}

	// No image captioning inference api available -> reply on user-hosted space api
	// get caption
	url := "https://hf.space/embed/OFA-Sys/OFA-Image_Caption/+/api/predict/"
	jsonStr := fmt.Sprintf("{\"data\": [\"data:image/jpg;base64,%s\"]}", ImgEnc)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		log.Println("[ImgSum]", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("[ImgSum]", err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("[ImgSum] received non 200 response code: %d", resp.StatusCode)
		return ""
	}
	j := &struct {
		Data          []string  `json:"data"`
		Durations     []float32 `json:"durations"`
		Avg_durations []float32 `json:"avg_durations"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(j)
	if err != nil {
		log.Println("[ImgSum]", err)
		return ""
	}

	// translate
	CaptionZHTW, err := gt.Translate(j.Data[0], "en", "zh-TW")
	if err != nil {
		log.Println("[ImgSum]", err)
		return ""
	}
	log.Println("[HuggingFace] Get request for", Keyword, "caption:", CaptionZHTW)
	return CaptionZHTW
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
