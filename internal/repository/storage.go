package repository

import (
	"context"

	"github.com/advn1/url-shortener/internal/models"
)

type Storage interface {
	SaveURL(ctx context.Context, original, short string) (models.ShortURL, error)
	GetOriginalURL(ctx context.Context, short string) (string, error)
	SaveBatch(ctx context.Context, batchRequest []models.BatchRequest) ([]models.BatchResponse, error)
	Ping(ctx context.Context) error
}
