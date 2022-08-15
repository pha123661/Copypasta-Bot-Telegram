package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	toml "github.com/BurntSushi/toml"
)

var CONFIG Config_Type

type Config_Type struct {
	LOCATION struct {
		// DB_DIR   string
		LOG_FILE string
	}

	API struct {
		TG struct {
			TOKEN string
		}
		HF struct {
			TOKENs []string
			MODEL  string
		}
		MONGODB struct {
			USER string
			PASS string
			URL  string
		}
	}

	// to be filled by program
	DB_FILE   string
	TEXT_DIR  string
	SUM_DIR   string
	IMAGE_DIR string
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
	// 	// add specific locations
	// 	CONFIG.DB_FILE = filepath.Join(CONFIG.DB_DIR, "DataBase.gob")
	// 	CONFIG.TEXT_DIR = filepath.Join(CONFIG.DB_DIR, "Text")
	// 	CONFIG.SUM_DIR = filepath.Join(CONFIG.DB_DIR, "Sum")
	// 	CONFIG.IMAGE_DIR = filepath.Join(CONFIG.DB_DIR, "Image")

	// 	var CreateDirIfNotExist = func(path string) {
	// 		if _, err := os.Stat(path); os.IsNotExist(err) {
	// 			errr := os.Mkdir(path, 0755)
	// 			if errr != nil {
	// 				log.Panicln("[InitConfig]", errr)
	// 			}
	// 		}
	// 	}

	// 	CreateDirIfNotExist(CONFIG.DB_DIR)
	// 	CreateDirIfNotExist(CONFIG.TEXT_DIR)
	// 	CreateDirIfNotExist(CONFIG.SUM_DIR)
	// 	CreateDirIfNotExist(CONFIG.IMAGE_DIR) // since IMAGE_LOCATION stands for a gob file
}
