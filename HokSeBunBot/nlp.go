package main

import (
	"log"
	"math/rand"
	"os"
	"path"
	"time"

	hfapigo "github.com/TannerKvarfordt/hfapigo"
)

func init_nlp() {
	setAvailableAPI()
}

func setAvailableAPI() {
	var success bool = false

	rand.Seed(time.Now().UTC().UnixNano())
	perm := rand.Perm(len(CONFIG.HUGGINGFACE_TOKENs))

	for _, i := range perm {
		log.Println("Testing HF api:", CONFIG.HUGGINGFACE_TOKENs[i][:8])
		hfapigo.SetAPIKey(CONFIG.HUGGINGFACE_TOKENs[0])

		if err := testHfAPI(); err == nil {
			success = true
			break
		} else {
			log.Printf("HF api \"%s\" not available: %s\n", CONFIG.HUGGINGFACE_TOKENs[i][:8], err)
		}
	}

	if !success {
		log.Panicln("No available hf api!")
	}
}

func testHfAPI() error {
	_, err := hfapigo.SendSummarizationRequest(
		CONFIG.HUGGINGFACE_MODEL,
		&hfapigo.SummarizationRequest{
			Inputs:  []string{"據了解，死者是88歲老翁，案發當時他剛運動完，正要走回家，但沒有走斑馬線，而是直接橫越馬路，而無照騎車的少年在閃避違規臨停的"},
			Options: *hfapigo.NewOptions().SetWaitForModel(true),
		},
	)
	return err
}

func getSingleSummarization(filename string, input string, forceUpdate bool) string {
	if _, err := os.Stat(path.Join(CONFIG.SUMMARIZATION_LOCATION, filename)); !forceUpdate && err == nil {
		bytes, err := os.ReadFile(path.Join(CONFIG.SUMMARIZATION_LOCATION, filename))
		if err != nil {
			log.Panicln(err)
		}
		return string(bytes)
	}

	var content string
	sresps, err := hfapigo.SendSummarizationRequest(
		CONFIG.HUGGINGFACE_MODEL,
		&hfapigo.SummarizationRequest{
			Inputs:  []string{input},
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
			setAvailableAPI()
			return getSingleSummarization(filename, input, forceUpdate)
		}
	} else {
		content = sresps[0].SummaryText
	}

	// write summarization
	file, err := os.Create(path.Join(CONFIG.SUMMARIZATION_LOCATION, filename))
	if err != nil {
		log.Println(err)
	}
	file.WriteString(content)
	file.Close()
	log.Println("[HuggingFace] Get request for", filename, "content:", content)
	return content
}
