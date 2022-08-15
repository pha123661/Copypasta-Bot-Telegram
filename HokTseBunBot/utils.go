package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode/utf8"

	toml "github.com/BurntSushi/toml"
)

var CONFIG Config_Type

type Config_Type struct {
	SETTING struct {
		LOG_FILE string
	}

	API struct {
		TG struct {
			TOKEN string
		}
		HF struct {
			TOKENs        []string
			CURRENT_TOKEN string
			MODEL         string
		}
	}

	DB struct {
		DIR        string
		COLLECTION string
	}
}

func InitConfig(CONFIG_PATH string) {
	// parse toml file
	tomldata, err := os.ReadFile(CONFIG_PATH)
	if err != nil {
		log.Panicln("[InitConfig]", err)
	}
	if _, err := toml.Decode(string(tomldata), &CONFIG); err != nil {
		log.Panicln("[InitConfig]", err)
	}

	buf := new(bytes.Buffer)
	toml.NewEncoder(buf).Encode(CONFIG)
	fmt.Printf("********************\nConfig Loaded:\n%s\n********************\n", buf.String())

	// var CreateDirIfNotExist = func(path string) {
	// 	if _, err := os.Stat(path); os.IsNotExist(err) {
	// 		errr := os.Mkdir(path, 0755)
	// 		if errr != nil {
	// 			log.Panicln("[InitConfig]", errr)
	// 		}
	// 	}
	// }

	// CreateDirIfNotExist(CONFIG.DB.DIR)
}

func TruncateString(text string, width int) string {
	text = strings.TrimSpace(text)
	width = width - utf8.RuneCountInString("……")
	if utf8.RuneCountInString(text) > width {
		r := []rune(text)[:width]
		text = string(r) + "……"
	}
	return text
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
