package main

import (
	"log"
	"math/rand"
	"time"

	hfapigo "github.com/TannerKvarfordt/hfapigo"
)

func InitNLP() {
	SetHFAPI()
}

func SetHFAPI() {
	TestHFAPI := func() error {
		_, err := hfapigo.SendSummarizationRequest(
			CONFIG.API.HF.MODEL,
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
			success = true
			break
		} else {
			log.Printf("HF api \"%s\" not available: %s\n", CONFIG.API.HF.TOKENs[i][:8]+"...", err)
		}
	}

	if !success {
		log.Panicln("No available hf api!")
	}
}

func GetOneSummarization(Keyword string, Content string) string {
	var content string
	sresps, err := hfapigo.SendSummarizationRequest(
		CONFIG.API.HF.MODEL,
		&hfapigo.SummarizationRequest{
			Inputs:  []string{Content},
			Options: *hfapigo.NewOptions().SetWaitForModel(true),
		},
	)
	if err != nil {
		if err.Error() == `{"error":"Service Unavailable"}` {
			log.Println(err)
			// input too long
			content = ""
		} else {
			log.Println(err)
			log.Println("[HuggingFace] API dead, switching token...")
			SetHFAPI()
			return GetOneSummarization(Keyword, Content)
		}
	} else {
		content = sresps[0].SummaryText
	}
	log.Println("[HuggingFace] Get request for", Keyword, "content:", content)
	return content
}
