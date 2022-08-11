package main

import (
	"os"
	"path"
	"strings"
	"unicode/utf8"

	toml "github.com/BurntSushi/toml"
)

var CACHE = make(map[string]string)
var CONFIG Config

type Config struct {
	TELEGRAM_API_TOKEN string
	FILE_LOCATION      string
}

func initConfig(CONFIG_PATH string) Config {
	tomldata, err := os.ReadFile(CONFIG_PATH)
	if err != nil {
		panic(err)
	}
	if _, err := toml.Decode(string(tomldata), &CONFIG); err != nil {
		panic(err)
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

func trim_string(str string, length int) string {
	length = length - utf8.RuneCountInString("……")
	if utf8.RuneCountInString(str) >= length {
		r := []rune(str)[:length]
		str = string(r) + "……"
	}
	return str
}

func build_cache() {
	// updates cache with existing files
	files, err := os.ReadDir(CONFIG.FILE_LOCATION)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		text, _ := os.ReadFile(path.Join(CONFIG.FILE_LOCATION, file.Name()))
		CACHE[delExtension(file.Name())] = string(text) // text is []byte
	}
}
