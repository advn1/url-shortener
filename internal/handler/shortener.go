package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/advn1/url-shortener/internal/jsonutils"
	"github.com/advn1/url-shortener/internal/models"
	"github.com/advn1/url-shortener/internal/random"
	"github.com/advn1/url-shortener/internal/repository"
	"go.uber.org/zap"
)

type Handler struct {
	BaseURL string
	URLs    map[string]string // add mutex in future to provide thread safe writing/reading map
	Storage repository.Storage
	Logger  *zap.SugaredLogger
}

func New(baseURL string, urls map[string]string, storage repository.Storage, logger *zap.SugaredLogger) *Handler {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &Handler{
		BaseURL: baseURL,
		URLs:    urls,
		Storage: storage,
		Logger:  logger,
	}
}

// deprecated
// handler POST URL
// func (h *Handler) HandlePost(w http.ResponseWriter, r *http.Request) {
// 	h.logger.Infow("HandlePost called", "path", r.URL.Path)

// 	if r.Method == http.MethodPost {
// 		w.Header().Set("Content-Type", "text/plain")
// 		if r.URL.Path != "/" {
// 			w.WriteHeader(http.StatusNotFound)
// 			return
// 		}

// 		bodyUrl, err := io.ReadAll(r.Body)
// 		if err != nil {
// 			w.WriteHeader(http.StatusBadRequest)
// 			return
// 		}

// 		if len(bodyUrl) == 0 {
// 			w.WriteHeader(http.StatusBadRequest)
// 			return
// 		}

// 		stringUrl := string(bodyUrl)
// 		if _, err := url.ParseRequestURI(stringUrl); err != nil {
// 			w.WriteHeader(http.StatusBadRequest)
// 			return
// 		}
// 		var existingShortURL string

// 		if h.dbConnection != nil {
// 			err := h.dbConnection.QueryRowContext(context.Background(),
// 				"SELECT short_url FROM urls WHERE original_url = $1", stringUrl).Scan(&existingShortURL)

// 			if err == nil {
// 				h.logger.Infow("URL already exists in DB", "original", stringUrl)
// 				w.WriteHeader(http.StatusConflict)
// 				w.Write([]byte(h.BaseURL + "/" + existingShortURL))
// 				return
// 			}
// 		} else {
// 			for short, original := range h.URLs {
// 				if original == stringUrl {
// 					existingShortURL = short
// 					break
// 				}
// 			}
// 		}

// 		if existingShortURL != "" {
// 			h.logger.Infow("URL already exists in Map", "original", stringUrl)
// 			w.WriteHeader(http.StatusConflict)
// 			w.Write([]byte(h.BaseURL + "/" + existingShortURL))
// 			return
// 		}

// 		encodedUrl := GenerateRandomUrl()
// 		fullUrl := h.BaseURL + "/" + encodedUrl
// 		xUuid := uuid.New()

// 		if h.dbConnection != nil {
// 			res, err := h.dbConnection.ExecContext(context.Background(),
// 				"INSERT INTO urls (uuid, original_url, short_url) VALUES ($1, $2, $3) ON CONFLICT (original_url) DO NOTHING",
// 				xUuid, stringUrl, encodedUrl)

// 			if err != nil {
// 				w.WriteHeader(http.StatusInternalServerError)
// 				return
// 			}

// 			if rA, _ := res.RowsAffected(); rA == 0 {
// 				var shortUrl string

// 				err := h.dbConnection.QueryRowContext(context.Background(),
// 					"SELECT (short_url) FROM urls WHERE original_url = $1",
// 					stringUrl).Scan(&shortUrl)

// 				if err != nil {
// 					if err == sql.ErrNoRows {
// 						w.WriteHeader(http.StatusInternalServerError)
// 						return
// 					}
// 					w.WriteHeader(http.StatusInternalServerError)
// 					return
// 				}

// 				w.WriteHeader(http.StatusConflict)
// 				w.Write([]byte(h.BaseURL + "/" + shortUrl))
// 			}
// 		}

// 		if h.StoragePath != "" {
// 			_, err := storage.SaveToFile(h.StoragePath, []models.ShortURL{{ID: xUuid, ShortURL: encodedUrl, OriginalURL: stringUrl}})
// 			if err != nil {
// 				jsonutils.InternalError(w)
// 				return
// 			}

// 		}
// 		h.URLs[encodedUrl] = stringUrl

// 		w.WriteHeader(http.StatusCreated)
// 		w.Write([]byte(fullUrl))
// 	} else {
// 		w.WriteHeader(http.StatusMethodNotAllowed)
// 	}
// }

// handler GET URL by ID
func (h *Handler) HandleGetById(w http.ResponseWriter, r *http.Request) {
	h.Logger.Infow("HandleGetById called", "path", r.URL.Path)

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	stringId := strings.TrimPrefix(r.URL.Path, "/")
	stringId = strings.TrimSpace(stringId)

	if stringId == "" {
		jsonutils.WriteJSONError(w, jsonutils.NewJSONError(http.StatusBadRequest, "Empty ID", "short URL ID cannot be empty"))
		return
	}

	ctx := r.Context()
	originalURL, err := h.Storage.GetOriginalURL(ctx, stringId)

	switch err {
	case repository.InternalErr:
		h.Logger.Errorw("INTERNAL ERROR", "error", err)
		jsonutils.WriteInternalError(w)
	case repository.NotExistsErr:
		jsonutils.WriteJSONError(w, jsonutils.NewJSONError(http.StatusBadRequest, "Non existing ID", "provided short ID doesn't exists"))
	case nil:
		http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
	default:
		h.Logger.Errorw("UNKNOWN ERROR", "error", err)
		jsonutils.WriteInternalError(w)
	}
}

func (h *Handler) HandlePostRESTApi(w http.ResponseWriter, r *http.Request) {
	h.Logger.Infow("HandlePostRESTApi called", "path", r.URL.Path)

	w.Header().Set("Content-Type", "application/json")

	err := validatePostRequest(r)
	if err != nil {
		jsonutils.WriteHTTPError(w, err)
		return
	}

	var postURLBody models.PostURLBody

	err = getPostBody(&postURLBody, r.Body)
	if err != nil {
		jsonutils.WriteHTTPError(w, err)
		return
	}

	err = postURLBody.Validate()
	if err != nil {
		jsonutils.WriteHTTPError(w, err)
		return
	}

	shortId := random.GenerateRandomUrl()

	ctx := r.Context()
	res, err := h.Storage.SaveURL(ctx, postURLBody.Url, shortId)
	if err != nil {
		h.Logger.Errorw("SaveURL", "error", err, "body", postURLBody)
	}

	switch err {
	case repository.ConflictErr:
		res = res.ToFullURL(h.BaseURL)
		jsonutils.WriteJSON(w, http.StatusConflict, res)
	case nil:
		res = res.ToFullURL(h.BaseURL)
		jsonutils.WriteJSON(w, http.StatusOK, res)
	default:
		h.Logger.Errorw("INTERNAL ERROR", "error", err)
		jsonutils.WriteInternalError(w)
	}
}

func (h *Handler) HandlePostBatchRESTApi(w http.ResponseWriter, r *http.Request) {
	h.Logger.Infow("HandlePostBatchRESTApi called", "path", r.URL.Path)

	w.Header().Set("Content-type", "application/json")
	err := validatePostRequest(r)
	if err != nil {
		jsonutils.WriteHTTPError(w, err)
		return
	}

	var batchBody models.BatchBody
	err = getPostBody(&batchBody, r.Body)
	if err != nil {
		jsonutils.WriteHTTPError(w, err)
		return
	}

	err = batchBody.Validate()
	if err != nil {
		jsonutils.WriteHTTPError(w, err)
		return
	}

	ctx := r.Context()
	batchResponse, err := h.Storage.SaveBatch(ctx, batchBody)

	switch err {
	case nil:
		for i := range batchResponse {
			batchResponse[i].ShortURL = h.BaseURL + "/" + batchResponse[i].ShortURL
		}
		jsonutils.WriteJSON(w, http.StatusCreated, &batchResponse)
	default:
		h.Logger.Errorw("SaveBatch", "error", err, "values", batchBody)
		jsonutils.WriteInternalError(w)
	}
}

func (h *Handler) PingDB(w http.ResponseWriter, r *http.Request) {
	err := h.Storage.Ping(r.Context())
	if err != nil {
		h.Logger.Errorw("failed to connect to db: %w", err)
		jsonutils.WriteInternalError(w)
	}
	w.WriteHeader(http.StatusOK)
}

func getPostBody(toDecode any, body io.ReadCloser) error {
	defer body.Close()

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(toDecode); err != nil {
		if err == io.EOF {
			return jsonutils.NewJSONError(http.StatusBadRequest, "Empty request body", "request body cannot be empty")
		}
		return jsonutils.NewJSONError(http.StatusBadRequest, "Invalid JSON format", err.Error())
	}

	if _, err := decoder.Token(); err != io.EOF {
		return jsonutils.NewJSONError(http.StatusBadRequest, "Invalid JSON format", "Request body must contain only one JSON entity")
	}

	return nil
}

func validatePostRequest(r *http.Request) error {
	if r.Method != http.MethodPost {
		return jsonutils.NewJSONError(http.StatusMethodNotAllowed, "Method not allowed", "method not allowed")
	}

	if r.Header.Get("Content-Type") != "application/json" {
		return jsonutils.NewJSONError(http.StatusMethodNotAllowed, "Incorrect Content-Type header", "incorrect Content-Type header")
	}

	return nil
}
