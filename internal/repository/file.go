package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/advn1/url-shortener/internal/models"
	"github.com/advn1/url-shortener/internal/random"
	"github.com/google/uuid"
)

type FileStorage struct {
	File        *os.File
	URLs        map[string]models.ShortURL
	ReverseURLs map[string]string
}

func InitFileStorage(path string) (*FileStorage, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		return &FileStorage{}, fmt.Errorf("couldn't open a file: %w", err)
	}

	scanner := bufio.NewScanner(file)
	shortURLMap := make(map[string]models.ShortURL, 1024)
	reverseURLMap := make(map[string]string, 1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var jsonLine models.ShortURL

		err := json.Unmarshal(line, &jsonLine)
		if err != nil {
			return &FileStorage{}, fmt.Errorf("couldn't unmarshal line: %w", err)
		}

		shortURLMap[jsonLine.ShortURL] = jsonLine
		reverseURLMap[jsonLine.OriginalURL] = jsonLine.ShortURL
	}

	return &FileStorage{File: file, URLs: shortURLMap, ReverseURLs: reverseURLMap}, nil
}

func (f *FileStorage) SaveURL(ctx context.Context, original, short, userId string) (models.ShortURL, error) {
	shortID, ok := f.ReverseURLs[original]

	if ok {
		return f.URLs[shortID], ErrConflict
	}

	res := models.ShortURL{ID: uuid.New(), OriginalURL: original, ShortURL: short, UserID: userId}
	err := f.writeToFile(res)
	if err != nil {
		return models.ShortURL{}, err
	}

	f.URLs[short] = res
	f.ReverseURLs[original] = short
	return res, nil

}

func (f *FileStorage) GetOriginalURL(ctx context.Context, short string) (string, error) {
	shortURL, exists := f.URLs[short]
	if !exists {
		return "", ErrNotExists
	}

	return shortURL.OriginalURL, nil
}

func (f *FileStorage) SaveBatch(ctx context.Context, batchRequest []models.BatchRequest, userId string) ([]models.BatchResponse, error) {
	var batchResponse []models.BatchResponse
	for _, batch := range batchRequest {
		if _, ok := f.ReverseURLs[batch.OriginalURL]; !ok {
			short := random.GenerateRandomUrl()
			UUID := uuid.New()
			err := f.writeToFile(models.ShortURL{ID: UUID, ShortURL: short, OriginalURL: batch.OriginalURL, UserID: userId})
			if err != nil {
				return batchResponse, err
			}

			f.URLs[short] = models.ShortURL{ID: UUID, OriginalURL: batch.OriginalURL, ShortURL: short}
			f.ReverseURLs[batch.OriginalURL] = short
			batchResponse = append(batchResponse, models.BatchResponse{CorrelationID: batch.CorrelationID, ShortURL: short})
		} else {
			batchResponse = append(batchResponse, models.BatchResponse{CorrelationID: batch.CorrelationID, ShortURL: f.ReverseURLs[batch.OriginalURL]})
		}
	}

	return batchResponse, nil
}

func (f *FileStorage) GetUserURLs(ctx context.Context, userId string) ([]models.UserURLs, error) {
	urls := make([]models.UserURLs, 0, 10)
	for _, v := range f.URLs {
		if v.UserID == userId {
			urls = append(urls, models.UserURLs{ShortURL: v.ShortURL, OriginalURL: v.OriginalURL})
		}
	}

	return urls, nil
}

func (f *FileStorage) Ping(ctx context.Context) error {
	if f.File == nil {
		return fmt.Errorf("file not open")
	}
	_, err := f.File.Stat()
	return err
}

func (f *FileStorage) writeToFile(record models.ShortURL) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	data = append(data, '\n')

	if _, err := f.File.Write(data); err != nil {
		return fmt.Errorf("write to file error: %w", err)
	}
	return nil
}

func (f *FileStorage) Close() error {
	if f.File == nil {
		return nil
	}
	return f.File.Close()
}
