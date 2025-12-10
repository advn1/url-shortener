package handler

import (
	"crypto/rand"
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
	BaseURL string
	URLs    map[string]string
	StoragePath string
	logger  *zap.SugaredLogger
}

func New(baseURL string, urls map[string]string, storagePath string, sugar *zap.SugaredLogger) *Handler {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &Handler{
		BaseURL: baseURL,
		URLs:    urls,
		StoragePath: storagePath,
		logger:  sugar,
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
	if r.Method == http.MethodGet {
		stringId := strings.TrimPrefix(r.URL.Path, "/")
		stringId = strings.TrimSpace(stringId)

		if stringId == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		originalUrl, exists := h.URLs[stringId]
		if !exists {
			w.WriteHeader(http.StatusBadRequest)
			return
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
	Uuid uuid.UUID `json:"uuid"`
	ShortUrl string `json:"short_url"`
	OriginalUrl string `json:"original_url"`
}


func (h *Handler) HandlePostRESTApi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
    if r.Method != http.MethodPost {
        jsonutils.WriteJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
        return
    }
    
    if r.Header.Get("Content-Type") != "application/json" {
        jsonutils.WriteJSONError(w, http.StatusBadRequest, "incorrect Content-Type header")
        return
    }

    body, err := io.ReadAll(r.Body)
    if err != nil {
        jsonutils.WriteJSONError(w, http.StatusInternalServerError, "failed to read request body")
        return
    }

    if len(body) == 0 {
        jsonutils.WriteJSONError(w, http.StatusBadRequest, "empty POST request body")
        return
    }

    var postURLBody PostURLBody
    if err := json.Unmarshal(body, &postURLBody); err != nil {
        jsonutils.WriteJSONError(w, http.StatusBadRequest, "invalid JSON format")
        return
    }

    if postURLBody.Url == "" {
        jsonutils.WriteJSONError(w, http.StatusBadRequest, "empty URL")
        return
    }

    if _, err := url.ParseRequestURI(postURLBody.Url); err != nil {
        jsonutils.WriteJSONError(w, http.StatusBadRequest, "invalid URL format")
        return
    }

	encodedUrl := GenerateRandomUrl()
	h.URLs[encodedUrl] = postURLBody.Url

	// shortenedUrl := h.BaseURL + "/" + encodedUrl
	shortenedUrl := encodedUrl

	result := PostURLResponse {Uuid: uuid.New(), ShortUrl: shortenedUrl, OriginalUrl: postURLBody.Url}

	jsonResult, err := json.Marshal(&result)
	if err != nil {
		jsonutils.WriteJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	message, code, err := saveToDatabase(jsonResult, h.StoragePath)
	if err != nil {
        jsonutils.WriteJSONError(w, code, fmt.Sprintf("%v: %v", message, err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(jsonResult)
}

func saveToDatabase(jsonResult []byte, storagePath string) (string, int, error) {
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