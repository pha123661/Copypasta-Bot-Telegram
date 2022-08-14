package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	toml "github.com/BurntSushi/toml"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	pb "github.com/schollz/progressbar/v2"
)

var CACHE = make(map[string]HokSeBun)
var IMAGE_CACHE map[string]HokSeToo
var CONFIG Config_Type

type HokSeBun struct {
	content       string
	summarization string
}
type HokSeToo struct {
	FileID        tgbotapi.FileID
	summarization string
}
type Config_Type struct {
	DB_LOCATION        string
	LOG_FILE           string
	TELEGRAM_API_TOKEN string
	HUGGINGFACE_TOKENs []string
	HUGGINGFACE_MODEL  string

	// to be filled by program
	FILE_LOCATION          string
	SUMMARIZATION_LOCATION string
	IMAGE_LOCATION         string
}

func init_utils() {
	// read config
	CONFIG = initConfig("./config.toml")
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

func initConfig(CONFIG_PATH string) Config_Type {
	// parse toml file
	tomldata, err := os.ReadFile(CONFIG_PATH)
	if err != nil {
		log.Panicln(err)
	}
	if _, err := toml.Decode(string(tomldata), &CONFIG); err != nil {
		log.Panicln(err)
	}

	// add specific locations
	CONFIG.FILE_LOCATION = filepath.Join(CONFIG.DB_LOCATION, "Text")
	CONFIG.SUMMARIZATION_LOCATION = filepath.Join(CONFIG.DB_LOCATION, "Sum")
	CONFIG.IMAGE_LOCATION = filepath.Join(CONFIG.DB_LOCATION, "Image", "ImageDB.gob")

	var CreateIfNotExist = func(path string) error {
		var error error
		if _, err := os.Stat(path); os.IsNotExist(err) {
			error = os.Mkdir(path, 0755)
		}
		return error
	}

	CreateIfNotExist(CONFIG.DB_LOCATION)
	CreateIfNotExist(CONFIG.FILE_LOCATION)
	CreateIfNotExist(CONFIG.SUMMARIZATION_LOCATION)
	CreateIfNotExist(filepath.Dir(CONFIG.IMAGE_LOCATION)) // since IMAGE_LOCATION stands for a gob file

	return CONFIG
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func Min(a int, b int) int {
	if a < b {
		return a
	}
	return b
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
	fmt.Println("Building Cache..., this may take some time")
	log.Println("Building Cache...")
	// updates cache with existing files
	files, err := os.ReadDir(CONFIG.FILE_LOCATION)
	if err != nil {
		log.Panicln(err)
	}
	bar := pb.New(len(files))
	for _, file := range files {
		bar.Add(1)
		if file.IsDir() {
			continue
		}
		text, _ := os.ReadFile(path.Join(CONFIG.FILE_LOCATION, file.Name()))
		CACHE[delExtension(file.Name())] = HokSeBun{content: string(text), summarization: getSingleSummarization(file.Name(), string(text), false)} // text is []byte
	}
	fmt.Println("Building Cache Done")
	log.Println("Building Cache Done")
}
