package models

import (
	"net/http"
	"net/url"

	"github.com/advn1/url-shortener/internal/jsonutils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type PostBody interface {
	Validate() error
}

type PostURLBody struct {
	Url string `json:"url"`
}

func (p PostURLBody) Validate() error {
	if p.Url == "" {
		return jsonutils.NewJSONError(http.StatusBadRequest, "Empty URL", "")
	}

	if _, err := url.ParseRequestURI(p.Url); err != nil {
		return jsonutils.NewJSONError(http.StatusBadRequest, "Invalid URL format", "")
	}
	return nil
}

type ShortURL struct {
	ID          uuid.UUID `json:"uuid"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	UserID      string    `json:"user_id"`
}

func (s ShortURL) ToFullURL(baseURL string) ShortURL {
	return ShortURL{
		ID:          s.ID,
		ShortURL:    baseURL + "/" + s.ShortURL,
		OriginalURL: s.OriginalURL,
		UserID:      s.UserID,
	}
}

type BatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchBody []BatchRequest

func (b BatchBody) Validate() error {
	if len(b) == 0 {
		return jsonutils.NewJSONError(http.StatusBadRequest, "Empty batch", "batch request cannot be empty")
	}

	for _, el := range b {
		if el.OriginalURL == "" || el.CorrelationID == "" {
			return jsonutils.NewJSONError(http.StatusBadRequest, "original_url or correlation_id cannot be empty", "")
		}
	}

	for _, el := range b {
		if _, err := url.ParseRequestURI(el.OriginalURL); err != nil {
			return jsonutils.NewJSONError(http.StatusBadRequest, "Invalid URL format", "")
		}
	}

	return nil
}

type BatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type ContextKey string

const UserIDKey ContextKey = "userID"

type UserClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type UserURLs struct {
	ShortURL string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (s UserURLs) ToFullURL(baseURL string) UserURLs {
	return UserURLs{
		ShortURL:    baseURL + "/" + s.ShortURL,
		OriginalURL: s.OriginalURL,
	}
}
