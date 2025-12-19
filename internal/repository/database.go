package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/advn1/url-shortener/internal/models"
	"github.com/advn1/url-shortener/internal/random"
	"github.com/google/uuid"
)

type DatabaseStorage struct {
	DB *sql.DB
}

func (d DatabaseStorage) SaveURL(ctx context.Context, original, short string) (models.ShortURL, error) {
	var result models.ShortURL
	// --- check if short URL already exists
	foundUUID, foundURL, exists, err := findExistingURL(ctx, d.DB, original)
	if err != nil {
		return result, fmt.Errorf("failed to query select statement: %w", err)
	}

	if exists {
		parsedUUID, err := uuid.Parse(foundUUID)
		if err != nil {
			return result, ErrInternal
		}
		result = models.ShortURL{ID: parsedUUID, ShortURL: foundURL, OriginalURL: original}

		return result, ErrConflict
	}

	result = models.ShortURL{ID: uuid.New(), ShortURL: short, OriginalURL: original}

	// --- save to database ---
	res, err := saveToDatabase(ctx, d.DB, []models.ShortURL{result})
	if err != nil {
		return result, fmt.Errorf("failed to save to db: %w", err)
	}

	// --- prevent race conditions ---
	if rows, _ := res.RowsAffected(); rows == 0 {
		foundUUID, foundURL, _, _ := findExistingURL(ctx, d.DB, result.OriginalURL)
		parsedUUID, err := uuid.Parse(foundUUID)
		if err != nil {
			return result, fmt.Errorf("failed to parse UUID from DB: %w", err)
		}
		result := models.ShortURL{ID: parsedUUID, ShortURL: foundURL, OriginalURL: original}

		return result, ErrConflict
	}
	return result, nil
}

func (d DatabaseStorage) GetOriginalURL(ctx context.Context, short string) (string, error) {
	var originalURL string
	// fetch from database
	err := d.DB.QueryRowContext(ctx, "SELECT original_url FROM urls WHERE short_url = $1", short).Scan(&originalURL)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotExists
	}

	if err != nil {
		return "", fmt.Errorf("failed to query select statement: %w", err)
	}
	return originalURL, nil
}

func (d DatabaseStorage) SaveBatch(ctx context.Context, batchRequest []models.BatchRequest) ([]models.BatchResponse, error) {

	var batchResponse []models.BatchResponse
	var itemsToSave []models.ShortURL
	var allOriginalURLs []string

	for _, batchBody := range batchRequest {
		shortUrl := random.GenerateRandomUrl()
		batchResponse = append(batchResponse, models.BatchResponse{CorrelationID: batchBody.CorrelationID, ShortURL: shortUrl})
		itemsToSave = append(itemsToSave, models.ShortURL{ID: uuid.New(), OriginalURL: batchBody.OriginalURL, ShortURL: shortUrl})
		allOriginalURLs = append(allOriginalURLs, batchBody.OriginalURL)
	}

	shortAndOriginal, err := saveToDatabaseBatch(ctx, d.DB, allOriginalURLs, itemsToSave)
	if err != nil {
		return nil, fmt.Errorf("failed save to database: %w", err)
	}

	dbMap := make(map[string]string, len(shortAndOriginal))
	for _, el := range shortAndOriginal {
		dbMap[el.originalURL] = el.shortURL
	}

	for i := range len(batchRequest) {
		val, exists := dbMap[batchRequest[i].OriginalURL]
		if exists {
			batchResponse[i].ShortURL = val
		}
	}

	return batchResponse, nil
}

func (d DatabaseStorage) Ping(ctx context.Context) error {
	return d.DB.PingContext(ctx)
}

type ShortAndOriginal struct {
	shortURL    string
	originalURL string
}

func saveToDatabaseBatch(ctx context.Context, db *sql.DB, allOriginalURLS []string, itemsToSave []models.ShortURL) ([]ShortAndOriginal, error) {
	// begin transaction
	// insert new urls in db, ignore already existing
	// after inserting
	// grab all short and original where original in user's original urls
	// this is needeed to then replace new short urls where original urls already has short url
	// note: not safe from race conditions

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed save to database: %w", err)
	}

	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT INTO urls (uuid, original_url, short_url) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO NOTHING")
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer stmt.Close()

	// insert to database
	for _, item := range itemsToSave {
		_, err := stmt.ExecContext(ctx, item.ID, item.OriginalURL, item.ShortURL)
		if err != nil {
			return nil, fmt.Errorf("failed to execute insert statement: %w", err)
		}
	}

	var shortAndOriginal []ShortAndOriginal
	rows, err := tx.QueryContext(ctx, "SELECT short_url, original_url FROM urls WHERE original_url = ANY($1)", allOriginalURLS)
	if err != nil {
		return nil, fmt.Errorf("failed to query select statement: %w", err)
	}

	defer rows.Close()

	// assign existing short and original
	var i = 0
	for rows.Next() {
		var shortURL, originalURL string

		rows.Scan(&shortURL, &originalURL)
		shortAndOriginal = append(shortAndOriginal, ShortAndOriginal{shortURL: shortURL, originalURL: originalURL})
		i++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return shortAndOriginal, nil
}

func saveToDatabase(ctx context.Context, db *sql.DB, itemsToSave []models.ShortURL) (sql.Result, error) {
	stmt, err := db.PrepareContext(ctx, "INSERT INTO urls (uuid, original_url, short_url) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO NOTHING")
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}

	defer stmt.Close()

	var latestRes sql.Result
	for _, item := range itemsToSave {
		res, err := stmt.ExecContext(ctx, item.ID, item.OriginalURL, item.ShortURL)
		if err != nil {
			return nil, fmt.Errorf("failed to insert to db: %w", err)
		}
		latestRes = res
	}

	return latestRes, nil
}

func findExistingURL(ctx context.Context, db *sql.DB, url string) (string, string, bool, error) {
	var UUID, foundShortURL string
	err := db.QueryRowContext(
		ctx,
		"SELECT uuid, short_url FROM urls WHERE original_url = $1",
		url,
	).Scan(&UUID, &foundShortURL)

	if err == nil {
		return UUID, foundShortURL, true, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, nil
	}

	return "", "", false, err
}
