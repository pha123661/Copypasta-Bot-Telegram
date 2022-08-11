package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	toml "github.com/BurntSushi/toml"
)

var CACHE = make(map[string]HokSeBun)
var CONFIG Config

func init_utils() {
	// read config
	CONFIG = initConfig("../config.toml")
	fmt.Println("#####################")
	fmt.Println("Loaded config:")
	fmt.Printf("%+v\n", CONFIG)
	fmt.Println("#####################")
	if CONFIG.FILE_LOCATION == "" || CONFIG.TELEGRAM_API_TOKEN == "" {
		fmt.Println("Please setup your config properly: Missing fields")
		os.Exit(0)
	}
	if len(CONFIG.HUGGINGFACE_TOKENs) == 0 || CONFIG.SUMMARIZATION_LOCATION == "" {
		fmt.Println("Please setup your config properly: NLP components will be disabled")
	}
}

type Config struct {
	TELEGRAM_API_TOKEN     string
	FILE_LOCATION          string
	SUMMARIZATION_LOCATION string
	HUGGINGFACE_TOKENs     []string
	HUGGINGFACE_MODEL      string
}

type HokSeBun struct {
	content       string
	summarization string
}

func initConfig(CONFIG_PATH string) Config {
	tomldata, err := os.ReadFile(CONFIG_PATH)
	if err != nil {
		log.Panicln(err)
	}
	if _, err := toml.Decode(string(tomldata), &CONFIG); err != nil {
		log.Panicln(err)
	}
	return CONFIG
}

func delExtension(fileName string) string {
	// utility for removing file extension from filename
	if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
		return fileName[:pos]
	}
	return fileName
}

func trimString(str string, length int) string {
	length = length - utf8.RuneCountInString("……")
	if utf8.RuneCountInString(str) >= length {
		r := []rune(str)[:length]
		str = string(r) + "……"
	}
	return str
}

func buildCache() {
	// updates cache with existing files
	files, err := os.ReadDir(CONFIG.FILE_LOCATION)
	if err != nil {
		log.Panicln(err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		text, _ := os.ReadFile(path.Join(CONFIG.FILE_LOCATION, file.Name()))
		CACHE[delExtension(file.Name())] = HokSeBun{content: string(text), summarization: getSingleSummarization(delExtension(file.Name()), string(text))} // text is []byte
	}
}
