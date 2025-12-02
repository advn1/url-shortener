package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Handler struct {
	BaseURL string
	URLs map[string]string
}

func New(b string) *Handler {
	b = strings.TrimSuffix(b, "/")
	return &Handler{
		BaseURL: b,
		URLs: make(map[string]string),
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
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		url, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(url) == 0 {
			fmt.Println("empty body")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		stringUrl := string(url)

		if !strings.HasPrefix(stringUrl, "http://") && !strings.HasPrefix(stringUrl, "https://") {
			fmt.Println("incorrect URL in body")
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
		stringId := strings.TrimPrefix(r.URL.Path,"/")
		stringId = strings.TrimSpace(stringId)

		fmt.Println(stringId)
		if stringId == "" {
			fmt.Println("empty id")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		originalUrl, exists := h.URLs[stringId]
		fmt.Println(h.URLs)
		fmt.Println("original url", h.URLs[stringId])
		if !exists {
			fmt.Println("id doesn't exists in db")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, originalUrl, http.StatusTemporaryRedirect)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
