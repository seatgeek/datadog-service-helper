package config

import (
	"os"

	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func WriteIfChange(service string, filePath string, data []byte, currentHash string) (bool, string) {
	text := string(data)
	text = "---\n" + text

	// turn the text back to bytes for hashing
	data = []byte(text)

	// compare hash of the new content vs file on disk
	newHash := HashBytes(data)
	if newHash == currentHash {
		logger.Infof("[%s] File hash is the same, NOOP", service)
		return false, newHash
	}

	// open file for write (truncated)
	file, err := os.Create(filePath)
	defer file.Close()
	if err != nil {
		logger.Fatalf("[%s] Could not create file %s: %s", service, filePath, err)
		return false, newHash
	}

	// write file to disk
	if _, err := file.Write(data); err != nil {
		logger.Errorf("[%s] Could not write file %s: %s", service, filePath, err)
		return false, newHash
	}

	logger.Infof("[%s] Successfully updated file: %s (old: %s | new: %s)", service, filePath, currentHash, newHash)
	return false, newHash
}
