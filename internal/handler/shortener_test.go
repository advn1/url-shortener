package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)


func TestPostURL_Success(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()
	
	h := New("http://localhost:8080", sugar)
	originalURL := "https://youtube.com"

	body := strings.NewReader(originalURL)

	r := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()
	h.HandlePost(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Errorf("incorrect status code. Got %v, wanted %v", res.StatusCode, http.StatusCreated)
	}

	if res.Header.Get("Content-Type") != "text/plain" {
		t.Errorf("incorrect Content-Type Header. Got %v, wanted %v", res.Header.Get("Content-Type"), "text/plain")
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("error reading response body: %v", err)
	}

	shortURL := string(data)

	if !strings.HasPrefix(shortURL, h.BaseURL) {
		t.Errorf("expected short URL to start with %s, got %s", h.BaseURL, shortURL)
	}

	splitted := strings.Split(shortURL, "/")
	id := splitted[len(splitted)-1]

	if originalURL != h.URLs[id] {
		t.Errorf("failed to save shortened url.")
	}
}

func TestPostURL_EmptyURL(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()
	
	h := New("http://localhost:8080", sugar)

	originalURL := ""
	body := strings.NewReader(originalURL)

	r := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()
	h.HandlePost(w, r)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("incorrect status code. Got %v, wanted %v", res.StatusCode, http.StatusBadRequest)
	}
}

func TestGetURL_Success(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()
	
	h := New("http://localhost:8080", sugar)

	savedUrls := h.URLs
	testURL_DB := map[string]string{
		"e1ef4c662c790d8e4f72": "https://google.com",
	}

	h.URLs = testURL_DB
	defer func() {
		h.URLs = savedUrls
	}()

	r := httptest.NewRequest("GET", "/"+"e1ef4c662c790d8e4f72", nil)
	w := httptest.NewRecorder()

	h.HandleGetById(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("incorrect status code. Got %v, wanted %v", res.StatusCode, http.StatusTemporaryRedirect)
	}

	location := res.Header.Get("Location")
	if location != "https://google.com" {
		t.Errorf("incorrect Location header. Got %v, wanted %v", location, "https://google.com")
	}
}

func TestGetURL_NonExistID(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()
	
	h := New("http://localhost:8080", sugar)
	nonExistentID := "5f4e167e355b7b52571c"

	r := httptest.NewRequest("GET", "/"+nonExistentID, nil)
	w := httptest.NewRecorder()

	h.HandleGetById(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("incorrect status code. Got %v, wanted %v", res.StatusCode, http.StatusBadRequest)
	}
}

func TestGetURL_EmptyID(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()
	
	h := New("http://localhost:8080", sugar)
	nonExistentID := ""

	r := httptest.NewRequest("GET", "/"+nonExistentID, nil)
	w := httptest.NewRecorder()

	h.HandleGetById(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("incorrect status code. Got %v, wanted %v", res.StatusCode, http.StatusBadRequest)
	}
}

func TestPostURL_InvalidURL(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()
	
	h := New("http://localhost:8080", sugar)

	invalidURL := "ftp://example.com" // not http or https protocol
	body := strings.NewReader(invalidURL)

	r := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()
	h.HandlePost(w, r)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected %v for invalid URL, got %v", http.StatusBadRequest, res.StatusCode)
	}
}