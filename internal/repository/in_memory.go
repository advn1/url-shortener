package repository

import (
	"context"

	"github.com/advn1/url-shortener/internal/models"
	"github.com/advn1/url-shortener/internal/random"
	"github.com/google/uuid"
)

type InMemory struct {
	URLs        map[string]models.ShortURL
	ReverseURLs map[string]string
}

func InitInMemoryStorage() *InMemory {
	URLs := make(map[string]models.ShortURL, 1024)
	reverseURLs := make(map[string]string, 1024)
	return &InMemory{URLs: URLs, ReverseURLs: reverseURLs}
}

func (m *InMemory) SaveURL(ctx context.Context, original, short, userId string) (models.ShortURL, error) {
	shortID, ok := m.ReverseURLs[original]

	if ok {
		return m.URLs[shortID], ErrConflict
	}

	res := models.ShortURL{ID: uuid.New(), OriginalURL: original, ShortURL: short, UserID: userId}

	m.URLs[short] = res
	m.ReverseURLs[original] = short
	return res, nil
}

func (m *InMemory) GetOriginalURL(ctx context.Context, short string) (string, error) {
	shortURL, exists := m.URLs[short]
	if !exists {
		return "", ErrNotExists
	}

	return shortURL.OriginalURL, nil
}

func (m *InMemory) SaveBatch(ctx context.Context, batchRequest []models.BatchRequest, userId string) ([]models.BatchResponse, error) {
	var batchResponse []models.BatchResponse
	for _, batch := range batchRequest {
		if _, ok := m.ReverseURLs[batch.OriginalURL]; !ok {
			short := random.GenerateRandomUrl()
			UUID := uuid.New()

			m.URLs[short] = models.ShortURL{ID: UUID, OriginalURL: batch.OriginalURL, ShortURL: short, UserID: userId}
			m.ReverseURLs[batch.OriginalURL] = short
			batchResponse = append(batchResponse, models.BatchResponse{CorrelationID: batch.CorrelationID, ShortURL: short})
		} else {
			batchResponse = append(batchResponse, models.BatchResponse{CorrelationID: batch.CorrelationID, ShortURL: m.ReverseURLs[batch.OriginalURL]})
		}
	}

	return batchResponse, nil
}

func (m *InMemory) Ping(ctx context.Context) error {
	return nil
}
