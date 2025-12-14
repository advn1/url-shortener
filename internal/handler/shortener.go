package handler

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/advn1/url-shortener/internal/jsonutils"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	BaseURL      string
	URLs         map[string]string // add mutex in future to provide thread safe writing/reading map
	StoragePath  string
	dbConnection *sql.DB
	logger       *zap.SugaredLogger
}

func New(baseURL string, urls map[string]string, storagePath string, db *sql.DB, sugar *zap.SugaredLogger) *Handler {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &Handler{
		BaseURL:      baseURL,
		URLs:         urls,
		StoragePath:  storagePath,
		dbConnection: db,
		logger:       sugar,
	}
}

// generate random url using rand package
func GenerateRandomUrl() string {
	randomUrl := make([]byte, 10) // for now buffer size is fixed size
	rand.Read(randomUrl)          // Read function returns an error but it's always nil

	encodedUrl := hex.EncodeToString(randomUrl)
	return encodedUrl
}

// deprecated
// handler POST URL
func (h *Handler) HandlePost(w http.ResponseWriter, r *http.Request) {
	h.logger.Infow("HandlePost called", "path", r.URL.Path)

	if r.Method == http.MethodPost {
		w.Header().Set("Content-Type", "text/plain")
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		bodyUrl, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(bodyUrl) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		stringUrl := string(bodyUrl)
		if _, err := url.ParseRequestURI(stringUrl); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var existingShortURL string

		if h.dbConnection != nil {
			err := h.dbConnection.QueryRowContext(context.Background(),
				"SELECT short_url FROM urls WHERE original_url = $1", stringUrl).Scan(&existingShortURL)

			if err == nil {
				h.logger.Infow("URL already exists in DB", "original", stringUrl)
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(h.BaseURL + "/" + existingShortURL))
				return
			}
		} else {
			for short, original := range h.URLs {
				if original == stringUrl {
					existingShortURL = short
					break
				}
			}
		}

		if existingShortURL != "" {
			h.logger.Infow("URL already exists in Map", "original", stringUrl)
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(h.BaseURL + "/" + existingShortURL))
			return
		}

		encodedUrl := GenerateRandomUrl()
		fullUrl := h.BaseURL + "/" + encodedUrl
		xUuid := uuid.New()

		if h.dbConnection != nil {
			res, err := h.dbConnection.ExecContext(context.Background(),
				"INSERT INTO urls (uuid, original_url, short_url) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO NOTHING",
				xUuid, stringUrl, encodedUrl)

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if rA, _ := res.RowsAffected(); rA == 0 {
				var shortUrl string

				err := h.dbConnection.QueryRowContext(context.Background(),
					"SELECT (short_url) FROM urls WHERE original_url = $1",
					stringUrl).Scan(&shortUrl)

				if err != nil {
					if err == sql.ErrNoRows {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(h.BaseURL + "/" + shortUrl))
			}
		}

		if h.StoragePath != "" {
			_, err := saveToFile(h.StoragePath, []ShortURL{{ID: xUuid, ShortURL: encodedUrl, OriginalURL: stringUrl}})
			if err != nil {
				jsonutils.InternalError(w)
				return
			}

		}
		h.URLs[encodedUrl] = stringUrl

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fullUrl))
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handler GET URL by ID
func (h *Handler) HandleGetById(w http.ResponseWriter, r *http.Request) {
	h.logger.Infow("HandleGetById called", "path", r.URL.Path)

	if r.Method == http.MethodGet {
		stringId := strings.TrimPrefix(r.URL.Path, "/")
		stringId = strings.TrimSpace(stringId)

		if stringId == "" {
			jsonutils.WriteJSONError(w, jsonutils.NewJSONError(http.StatusBadRequest, "Empty ID", "short URL ID cannot be empty"))
			return
		}

		var originalUrl string

		if h.dbConnection != nil {
			// fetch from database
			err := h.dbConnection.QueryRow("SELECT original_url FROM urls WHERE short_url = $1", stringId).Scan(&originalUrl)
			if err != nil {
				if err == sql.ErrNoRows {
					h.logger.Infow("DB fetch", "error", fmt.Sprintf("id: \"%v\" doesn't exists", stringId))
					jsonutils.WriteJSONError(w, jsonutils.NewJSONError(http.StatusBadRequest, "Non existing ID", "provided short URL ID doesn't exists"))
					return
				} else {
					h.logger.Errorw("DB fetch", "error", err, "id", stringId)
					jsonutils.InternalError(w)
					return
				}
			}
		} else {
			url, exists := h.URLs[stringId]
			if !exists {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			originalUrl = url
		}

		http.Redirect(w, r, originalUrl, http.StatusTemporaryRedirect)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type PostURLBody struct {
	Url string `json:"url"`
}

type ShortURL struct {
	ID          uuid.UUID `json:"uuid"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
}

func (s ShortURL) ToFullURL(baseURL string) ShortURL {
	return ShortURL{
		ID:          s.ID,
		ShortURL:    baseURL + "/" + s.ShortURL,
		OriginalURL: s.OriginalURL,
	}
}

func (h *Handler) HandlePostRESTApi(w http.ResponseWriter, r *http.Request) {
	h.logger.Infow("HandlePostRESTApi called", "path", r.URL.Path)

	w.Header().Set("Content-Type", "application/json")

	// - request validation -
	err := validatePostRESTApi_Request(r, h)
	if err != nil {
		if jsonErr, ok := err.(jsonutils.JSONError); ok {
			jsonutils.WriteJSONError(w, jsonErr)
		} else {
			h.logger.Errorw("Unexpected validation error", "error", err)
			jsonutils.InternalError(w)
		}
		return
	}

	// - request body validation -
	var postURLBody PostURLBody
	err = validatePostRESTApi_Body(&postURLBody, r)
	if err != nil {
		if jsonErr, ok := err.(jsonutils.JSONError); ok {
			jsonutils.WriteJSONError(w, jsonErr)
		} else {
			h.logger.Errorw("Unexpected validation error", "error", err)
			jsonutils.InternalError(w)
		}
		return
	}

	// - generate response -
	shortID := GenerateRandomUrl()
	result := ShortURL{ID: uuid.New(), ShortURL: shortID, OriginalURL: postURLBody.Url}

	// -- chose storage --
	if h.dbConnection != nil {
		// --- check if short URL already exists
		foundUUID, foundURL, exists, err := findExistingURL(h.dbConnection, result.OriginalURL)
		if err != nil {
			h.logger.Errorw("DB SELECT query", "error", err)
			jsonutils.InternalError(w)
		}
		if exists {
			h.logger.Infow("DB already existing short URL",
				"error", "short URL already exists",
				"short_url", foundURL)

			result = ShortURL{ID: uuid.MustParse(foundUUID), ShortURL: foundURL, OriginalURL: postURLBody.Url}
			result = result.ToFullURL(h.BaseURL)

			jsonutils.WriteJSON(w, http.StatusConflict, result)
			return
		}

		// --- save to database ---
		res, err := h.dbConnection.
			ExecContext(
				context.Background(),
				"INSERT INTO urls (uuid, original_url, short_url) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO NOTHING",
				result.ID, result.OriginalURL, result.ShortURL,
			)
		if err != nil {
			h.logger.Errorw("DB Exec query", "error", err, "values", result)
			jsonutils.InternalError(w)
			return
		}

		// --- prevent race conditions ---
		if rows, _ := res.RowsAffected(); rows == 0 {
			foundUUID, foundURL, _, _ := findExistingURL(h.dbConnection, result.OriginalURL)

			result := ShortURL{ID: uuid.MustParse(foundUUID), ShortURL: foundURL, OriginalURL: postURLBody.Url}
			result = result.ToFullURL(h.BaseURL)

			jsonutils.WriteJSON(w, http.StatusConflict, result)
			return
		}
	} else if h.StoragePath != "" {
		// -- check if short URL already exists in map -- //
		var foundShortID string
		var found bool

		for shortURL, originalURL := range h.URLs {
			if originalURL == postURLBody.Url {
				foundShortID = shortURL
				found = true
				break
			}
		}

		if found {
			file, err := os.Open(h.StoragePath)
			if err != nil {
				jsonutils.InternalError(w)
			}

			defer file.Close()

			scanner := bufio.NewScanner(file)
			// --- find UUID for found short URL ---
			for scanner.Scan() {
				line := scanner.Bytes()
				if len(line) == 0 {
					continue
				}
				var foundObj ShortURL
				if err := json.Unmarshal(line, &foundObj); err != nil {
					h.logger.Errorw("JSON unmarshal error in file scan", "error", err)
					continue
				}
				// --- return found object ---
				if foundObj.ShortURL == foundShortID {
					res := ShortURL{ID: foundObj.ID, ShortURL: foundObj.ShortURL, OriginalURL: foundObj.OriginalURL}
					res = res.ToFullURL(h.BaseURL)
					jsonutils.WriteJSON(w, http.StatusConflict, res)
					return
				}
			}
			
			h.logger.Errorw("Data inconsistency: URL found in map but not in file", "short_url", foundShortID)
			jsonutils.InternalError(w)
			return
		}
		// -- store to file --
		_, err := saveToFile(h.StoragePath, []ShortURL{result})
		if err != nil {
			jsonutils.InternalError(w)
			return
		}

		h.URLs[shortID] = postURLBody.Url

	} else {
		// -- store in memory --
		// note: In case of duplicates:
		// It doesn't store original UUIDs. we just generate them on every request. its ok for in-memory storage.
		var foundShortID string
		var found bool

		for shortURL, originalURL := range h.URLs {
			if originalURL == postURLBody.Url {
				foundShortID = shortURL
				found = true
				break
			}
		}

		if found {
			result.ShortURL = foundShortID
		} else {
			h.URLs[shortID] = postURLBody.Url
		}
	}

	result = result.ToFullURL(h.BaseURL)
	jsonutils.WriteJSON(w, http.StatusCreated, result)
}

func validatePostRESTApi_Request(r *http.Request, h *Handler) error {
	if r.Method != http.MethodPost {
		h.logger.Errorw("error", "message", "method not allowed")
		return jsonutils.NewJSONError(http.StatusMethodNotAllowed, "Method not allowed", "method not allowed")
	}

	if r.Header.Get("Content-Type") != "application/json" {
		h.logger.Errorw("error", "message", "incorrect Content-Type header")
		return jsonutils.NewJSONError(http.StatusMethodNotAllowed, "Incorrect Content-Type header", "incorrect Content-Type header")
	}

	return nil
}

func validatePostRESTApi_Body(postBody *PostURLBody, r *http.Request) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return jsonutils.NewJSONError(http.StatusInternalServerError, "Failed to read request body", "failed to read request body")
	}

	r.Body = io.NopCloser(bytes.NewBuffer(body))

	if len(body) == 0 {
		return jsonutils.NewJSONError(http.StatusBadRequest, "Empty POST request body", "empty POST request body")
	}

	if err := json.Unmarshal(body, &postBody); err != nil {
		jsonErr := jsonutils.NewJSONError(http.StatusBadRequest, "Invalid JSON format", "")
		return jsonErr
	}

	if postBody.Url == "" {
		jsonErr := jsonutils.NewJSONError(http.StatusBadRequest, "Empty URL", "")
		return jsonErr
	}

	if _, err := url.ParseRequestURI(postBody.Url); err != nil {
		jsonErr := jsonutils.NewJSONError(http.StatusBadRequest, "Invalid URL format", "")
		return jsonErr
	}
	return nil
}

func findExistingURL(db *sql.DB, url string) (string, string, bool, error) {
	var uuid, foundShortURL string
	err := db.QueryRowContext(
		context.Background(),
		"SELECT uuid, short_url FROM urls WHERE original_url = $1",
		url,
	).Scan(&uuid, &foundShortURL)

	if err == nil {
		return uuid, foundShortURL, true, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, nil
	}

	return "", "", false, jsonutils.NewJSONError(http.StatusInternalServerError, "Internal Server Error", err.Error())
}

type BatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// note: refactor this endpoint, add checks for duplicate urls
func (h *Handler) HandlePostBatchRESTApi(w http.ResponseWriter, r *http.Request) {
	h.logger.Infow("HandlePostBatchRESTApi called", "path", r.URL.Path)

	w.Header().Set("Content-type", "application/json")
	if r.Method != http.MethodPost {
		h.logger.Errorw("error", "message", "method not allowed")
		jsonutils.WriteJSONError(w, jsonutils.NewJSONError(http.StatusMethodNotAllowed, "Method not allowed", "method not allowed"))
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		jsonutils.WriteJSONError(w, jsonutils.NewJSONError(http.StatusBadRequest, "Incorrect Content-Type header", "incorrect Content-Type header"))
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonutils.WriteJSONError(w, jsonutils.NewJSONError(http.StatusInternalServerError, "Failed to read request body", "failed to read request body"))
		return
	}

	if len(body) == 0 {
		jsonutils.WriteJSONError(w, jsonutils.NewJSONError(http.StatusBadRequest, "Empty POST request body", "empty POST request body"))
		return
	}

	var BatchBody []BatchRequest
	if err := json.Unmarshal(body, &BatchBody); err != nil {
		jsonutils.WriteJSONError(w, jsonutils.NewJSONError(http.StatusBadRequest, "Invalid JSON format", "invalid JSON format"))
		return
	}

	var batchResponse []BatchResponse
	var itemsToSave []ShortURL

	for _, batchBody := range BatchBody {
		shortUrl := GenerateRandomUrl()
		batchResponse = append(batchResponse, BatchResponse{CorrelationID: batchBody.CorrelationID, ShortURL: shortUrl})
		itemsToSave = append(itemsToSave, ShortURL{ID: uuid.New(), OriginalURL: batchBody.OriginalURL, ShortURL: shortUrl})
	}

	if h.dbConnection != nil {
		statusCode, msg, details, err := saveToDatabase(h.dbConnection, h.logger, itemsToSave)
		if err != nil {
			h.logger.Errorw("DB saving", "error", err, "message", msg, "values", itemsToSave)
			jsonutils.WriteJSONError(w, jsonutils.NewJSONError(statusCode, msg, details))
			return
		}
	} else if h.StoragePath != "" {
		_, err := saveToFile(h.StoragePath, itemsToSave)
		if err != nil {
			h.logger.Errorw("Saving to file", "error", err, "values", itemsToSave)
			jsonutils.InternalError(w)
			return
		}

		for _, item := range itemsToSave {
			h.URLs[item.ShortURL] = item.OriginalURL
		}
	} else {
		for _, item := range itemsToSave {
			h.URLs[item.ShortURL] = item.OriginalURL
		}
	}

	w.WriteHeader(http.StatusCreated)

	jsonBatchResult, err := json.Marshal(&batchResponse)
	if err != nil {
		h.logger.Errorw("Encoding json", "error", err, "values", batchResponse)
		jsonutils.InternalError(w)
		return
	}

	w.Write(jsonBatchResult)
}

func saveToDatabase(db *sql.DB, logger *zap.SugaredLogger, itemsToSave []ShortURL) (int, string, string, error) {
	tx, err := db.Begin()
	if err != nil {
		logger.Errorw("DB transaction begin", "error", err, "values", itemsToSave)
		return http.StatusInternalServerError, "Internal Server Error", "cannot save to database", err
	}

	defer tx.Rollback()

	stmt, err := tx.PrepareContext(context.Background(), "INSERT INTO urls (uuid, original_url, short_url) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO NOTHING")
	if err != nil {
		return http.StatusInternalServerError, "Internal Server Error", "something database", err
	}

	defer stmt.Close()

	for _, item := range itemsToSave {
		// save to database
		_, err := stmt.ExecContext(context.Background(), item.ID, item.OriginalURL, item.ShortURL)
		if err != nil {
			logger.Errorw("DB transaction Exec query", "error", err, "values", itemsToSave)
			return http.StatusInternalServerError, "Internal Server Error", "cannot save to database", err
		}

	}

	tx.Commit()

	return http.StatusOK, "", "", nil
}

func saveToFile(storagePath string, items []ShortURL) (string, error) {
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

func (h *Handler) PingBD(w http.ResponseWriter, r *http.Request) {
	err := h.dbConnection.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
