package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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
	URLs         map[string]string
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

// handler POST URL
func (h *Handler) HandlePost(w http.ResponseWriter, r *http.Request) {
	h.logger.Infow("HandlePost called", "path", r.URL.Path)

	if r.Method == http.MethodPost {
		w.Header().Set("Content-Type", "text/plain")
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		url, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(url) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		stringUrl := string(url)

		if !strings.HasPrefix(stringUrl, "http://") && !strings.HasPrefix(stringUrl, "https://") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		encodedUrl := GenerateRandomUrl()
		h.URLs[encodedUrl] = string(url)

		fullUrl := h.BaseURL + "/" + encodedUrl

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
			jsonutils.WriteJSONError(w, http.StatusBadRequest, "Empty ID", "short URL ID cannot be empty")
			return
		}

		var originalUrl string

		if h.dbConnection != nil {
			// fetch from database
			err := h.dbConnection.QueryRow("SELECT original_url FROM urls WHERE short_url = $1", stringId).Scan(&originalUrl)
			if err != nil {
				if err == sql.ErrNoRows {
					h.logger.Infow("DB fetch", "error", fmt.Sprintf("id: \"%v\" doesn't exists", stringId))
					jsonutils.WriteJSONError(w, http.StatusBadRequest, "Non existing ID", "provided short URL ID doesn't exists")
					return
				} else {
					h.logger.Errorw("DB fetch", "error", err, "id", stringId)
					jsonutils.WriteJSONError(w, http.StatusInternalServerError, "Internal Server Error", "")
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

type PostURLResponse struct {
	Uuid        uuid.UUID `json:"uuid"`
	ShortUrl    string    `json:"short_url"`
	OriginalUrl string    `json:"original_url"`
}

func (h *Handler) HandlePostRESTApi(w http.ResponseWriter, r *http.Request) {
	h.logger.Infow("HandlePostRESTApi called", "path", r.URL.Path)

	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		h.logger.Errorw("error", "message", "method not allowed")
		jsonutils.WriteJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "method not allowed")
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		jsonutils.WriteJSONError(w, http.StatusBadRequest, "Incorrect Content-Type header", "incorrect Content-Type header")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonutils.WriteJSONError(w, http.StatusInternalServerError, "Failed to read request body", "failed to read request body")
		return
	}

	if len(body) == 0 {
		jsonutils.WriteJSONError(w, http.StatusBadRequest, "Empty POST request body", "empty POST request body")
		return
	}

	var postURLBody PostURLBody
	if err := json.Unmarshal(body, &postURLBody); err != nil {
		jsonutils.WriteJSONError(w, http.StatusBadRequest, "Invalid JSON format", "")
		return
	}

	if postURLBody.Url == "" {
		jsonutils.WriteJSONError(w, http.StatusBadRequest, "Empty URL", "")
		return
	}

	if _, err := url.ParseRequestURI(postURLBody.Url); err != nil {
		jsonutils.WriteJSONError(w, http.StatusBadRequest, "Invalid URL format", "")
		return
	}

	encodedUrl := GenerateRandomUrl()
	h.URLs[encodedUrl] = postURLBody.Url

	result := PostURLResponse{Uuid: uuid.New(), ShortUrl: encodedUrl, OriginalUrl: postURLBody.Url}

	jsonResult, err := json.Marshal(&result)
	if err != nil {
		jsonutils.WriteJSONError(w, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	if h.dbConnection != nil {
		// save to database
		_, err = h.dbConnection.ExecContext(context.Background(), "INSERT INTO urls (id, original_url, short_url) VALUES ($1, $2, $3)", result.Uuid, result.OriginalUrl, result.ShortUrl)
		if err != nil {
			h.logger.Errorw("DB Exec query", "error", err, "values", result)
			jsonutils.WriteJSONError(w, http.StatusInternalServerError, "Internal Server Error", "cannot save to database")
			return
		}	
	} else if h.StoragePath != "" {
		// save to file
		message, code, err := saveToFile(jsonResult, h.StoragePath)
		if err != nil {
			jsonutils.WriteJSONError(w, code, "Internal Server Error", message)
			return
		}	
	} else {
		h.URLs[encodedUrl] = postURLBody.Url
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(jsonResult)
}

func saveToFile(jsonResult []byte, storagePath string) (string, int, error) {
	jsonResult = append(jsonResult, '\n')

	file, err := os.OpenFile(storagePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0664)
	if err != nil {
		return "couldn't open storage file " + storagePath, http.StatusInternalServerError, err
	}
	defer file.Close()

	_, err = file.Write(jsonResult)
	if err != nil {
		return "couldn't write to a storage file", http.StatusInternalServerError, err
	}

	return "", 0, nil
}

func (h *Handler) PingBD(w http.ResponseWriter, r *http.Request) {
	err := h.dbConnection.Ping()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}