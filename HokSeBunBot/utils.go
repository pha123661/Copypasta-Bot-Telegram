package main

import (
	"os"
	"path"
	"strings"
)

var CACHE = make(map[string]string)
var FILE_LOCATION string = "../HokSeBun_db" // should be changed

func delExtension(fileName string) string {
	// utility for removing file extension from filename
	if pos := strings.LastIndexByte(fileName, '.'); pos != -1 {
		return fileName[:pos]
	}
	return fileName
}

func build_cache() {
	// updates cache with existing files
	files, err := os.ReadDir(FILE_LOCATION)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		text, _ := os.ReadFile(path.Join(FILE_LOCATION, file.Name()))
		CACHE[delExtension(file.Name())] = string(text) // text is []byte
	}
}
