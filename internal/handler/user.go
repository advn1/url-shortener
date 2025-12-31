package handler

import (
	"net/http"

	"github.com/advn1/url-shortener/internal/jsonutils"
	"github.com/advn1/url-shortener/internal/models"
)

func (h *Handler) HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {
	h.Logger.Infow("GetUserURLs called", "path", r.URL.Path)

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	userId, ok := r.Context().Value(models.UserIDKey).(string)
	if !ok {
		h.Logger.Errorw("error", "details", "failed to parse id to string")
		jsonutils.WriteInternalError(w)
		return
	}

	urls, err := h.Storage.GetUserURLs(ctx, userId)
	if err != nil {
		h.Logger.Errorw("error", "details", err.Error())
		jsonutils.WriteInternalError(w)
		return
	}

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	for i, _ := range urls {
		urls[i] = urls[i].ToFullURL(h.BaseURL)
	}

	jsonutils.WriteJSON(w, http.StatusOK, urls)
}
