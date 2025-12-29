package handler

import (
	"encoding/json"
	"errors"
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
	Storage repository.Storage
	Logger  *zap.SugaredLogger
}

func New(baseURL string, storage repository.Storage, logger *zap.SugaredLogger) *Handler {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &Handler{
		BaseURL: baseURL,
		Storage: storage,
		Logger:  logger,
	}
}

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

	if errors.Is(err, repository.ErrNotExists) {
		jsonutils.WriteJSONError(w, jsonutils.NewJSONError(http.StatusNotFound, "Not Found", "short URL not found"))
		return
	}

	if err != nil {
		h.Logger.Errorw("INTERNAL ERROR", "error", err)
		jsonutils.WriteInternalError(w)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
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

	userID, ok := r.Context().Value(models.UserIDKey).(string)
	if !ok {
		h.Logger.Errorw("error", "details", "failed to parse id to string")
		jsonutils.WriteInternalError(w)
		return
	}

	res, err := h.Storage.SaveURL(ctx, postURLBody.Url, shortId, userID)
	if errors.Is(err, repository.ErrConflict) {
		res = res.ToFullURL(h.BaseURL)
		jsonutils.WriteJSON(w, http.StatusConflict, res)
		return
	}

	if err != nil {
		h.Logger.Errorw("INTERNAL ERROR", "error", err)
		jsonutils.WriteInternalError(w)
		return
	}

	res = res.ToFullURL(h.BaseURL)
	jsonutils.WriteJSON(w, http.StatusCreated, res)
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
	userID, ok := r.Context().Value(models.UserIDKey).(string)
	if !ok {
		h.Logger.Errorw("error", "details", "failed to parse id to string")
		jsonutils.WriteInternalError(w)
		return
	}

	batchResponse, err := h.Storage.SaveBatch(ctx, batchBody, userID)

	if err != nil {
		h.Logger.Errorw("SaveBatch error", "error", err)
		jsonutils.WriteInternalError(w)
		return
	}

	for i := range batchResponse {
		batchResponse[i].ShortURL = h.BaseURL + "/" + batchResponse[i].ShortURL
	}
	jsonutils.WriteJSON(w, http.StatusCreated, &batchResponse)
}

func (h *Handler) PingDB(w http.ResponseWriter, r *http.Request) {
	err := h.Storage.Ping(r.Context())
	if err != nil {
		h.Logger.Errorw("failed to connect to db", "error", err)
		jsonutils.WriteInternalError(w)
		return
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

	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return jsonutils.NewJSONError(http.StatusBadRequest, "Incorrect Content-Type header", "incorrect Content-Type header")
	}

	return nil
}
