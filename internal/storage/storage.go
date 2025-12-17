package storage

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/advn1/url-shortener/internal/models"
	"go.uber.org/zap"
)

func _SaveToFile(storagePath string, items []models.ShortURL) (string, error) {
	file, err := os.OpenFile(storagePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0664)
	if err != nil {
		return "couldn't open storage file " + storagePath, err
	}

	defer file.Close()

	for _, obj := range items {
		jsonObj, err := json.Marshal(obj)
		if err != nil {
			return "Internal Server Error", err
		}

		_, err = file.Write(jsonObj)
		if err != nil {
			return "couldn't write to a storage file", err
		}

		_, err = file.Write([]byte{'\n'})
		if err != nil {
			return "couldn't write to a storage file", err
		}
	}

	return "", nil
}

func _FindShortURL_File(file *os.File, cmpShortID string, baseURL string, logger *zap.SugaredLogger) (models.ShortURL, bool) {
	scanner := bufio.NewScanner(file)
	// --- find UUID for found short URL ---
	for scanner.Scan() {
		line := scanner.Bytes()

		if len(line) == 0 {
			continue
		}

		var foundObj models.ShortURL
		if err := json.Unmarshal(line, &foundObj); err != nil {
			logger.Errorw("JSON unmarshal error in file scan", "error", err)
			continue
		}

		// --- return found object ---
		if foundObj.ShortURL == cmpShortID {
			res := foundObj.ToFullURL(baseURL)
			return res, true
		}
	}
	return models.ShortURL{}, false
}
