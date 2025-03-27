package utils

import (
	"log"
	"os"
)

func EnsureDir(path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Printf("Failed to create directory %s: %v", path, err)
	}
}
